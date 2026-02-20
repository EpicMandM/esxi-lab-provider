package service

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"golang.org/x/crypto/curve25519"
)

// WireGuardConfig holds the WireGuard tunnel configuration from user_config.toml
type WireGuardConfig struct {
	Enabled             bool     `toml:"enabled"`
	ServerPublicKey     string   `toml:"server_public_key"`
	ServerEndpoint      string   `toml:"server_endpoint"`
	ServerTunnelNetwork string   `toml:"server_tunnel_network"`
	AllowedIPs          []string `toml:"allowed_ips"`
	MTU                 int      `toml:"mtu"`
	ClientAddresses     []string `toml:"client_addresses"`
	Keepalive           int      `toml:"keepalive"`
	// OPNsense API configuration (overridden from environment variables)
	OPNsenseURL       string `toml:"opnsense_url"`
	OPNsenseAPIKey    string `toml:"-"`
	OPNsenseAPISecret string `toml:"-"`
	OPNsenseInsecure  bool   `toml:"opnsense_insecure"`
	AutoRegisterPeers bool   `toml:"auto_register_peers"`
}

// WireGuardService manages WireGuard tunnel configurations.
//
// Security note: private keys are held in plaintext in process memory for the
// lifetime of the service. Keys are generated on-the-fly and are NOT persisted
// to disk, so they will be lost if the process restarts (clients must then
// regenerate). If the process memory is compromised, these keys are exposed.
type WireGuardService struct {
	config         *WireGuardConfig
	privateKeys    map[string]string
	opnsenseClient OPNsenseAPI
}

// NewWireGuardService creates a new WireGuard service.
// opnsense may be nil when auto-registration is not needed.
func NewWireGuardService(config *WireGuardConfig, opnsense OPNsenseAPI) *WireGuardService {
	return &WireGuardService{
		config:         config,
		privateKeys:    make(map[string]string),
		opnsenseClient: opnsense,
	}
}

// GenerateKeyPair generates a new WireGuard private/public key pair
func GenerateKeyPair() (privateKey, publicKey string, err error) {
	var privKeyBytes [32]byte
	if _, err := rand.Read(privKeyBytes[:]); err != nil {
		return "", "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Clamp the private key as per Curve25519 requirements
	privKeyBytes[0] &= 248
	privKeyBytes[31] &= 127
	privKeyBytes[31] |= 64

	var pubKeyBytes [32]byte
	curve25519.ScalarBaseMult(&pubKeyBytes, &privKeyBytes)

	privateKey = base64.StdEncoding.EncodeToString(privKeyBytes[:])
	publicKey = base64.StdEncoding.EncodeToString(pubKeyBytes[:])

	return privateKey, publicKey, nil
}

// RotateUserKey generates a new WireGuard key pair for a user
func (w *WireGuardService) RotateUserKey(username string) (privateKey, publicKey string, err error) {
	privKey, pubKey, err := GenerateKeyPair()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate key pair for %s: %w", username, err)
	}

	w.privateKeys[username] = privKey

	return privKey, pubKey, nil
}

// GenerateClientConfig generates a WireGuard client configuration file content
func (w *WireGuardService) GenerateClientConfig(username string, userIndex int) (string, error) {
	if !w.config.Enabled {
		return "", fmt.Errorf("WireGuard is not enabled in configuration")
	}

	if userIndex < 0 || userIndex >= len(w.config.ClientAddresses) {
		return "", fmt.Errorf("invalid user index %d for WireGuard client addresses", userIndex)
	}

	privateKey, ok := w.privateKeys[username]
	if !ok {
		return "", fmt.Errorf("no private key found for user %s", username)
	}

	var sb strings.Builder

	sb.WriteString("[Interface]\n")
	fmt.Fprintf(&sb, "PrivateKey = %s\n", privateKey)
	fmt.Fprintf(&sb, "Address = %s\n", w.config.ClientAddresses[userIndex])

	if w.config.MTU > 0 {
		fmt.Fprintf(&sb, "MTU = %d\n", w.config.MTU)
	}

	sb.WriteString("\n[Peer]\n")
	fmt.Fprintf(&sb, "PublicKey = %s\n", w.config.ServerPublicKey)
	fmt.Fprintf(&sb, "Endpoint = %s\n", w.config.ServerEndpoint)

	if len(w.config.AllowedIPs) > 0 {
		fmt.Fprintf(&sb, "AllowedIPs = %s\n", strings.Join(w.config.AllowedIPs, ", "))
	}

	if w.config.Keepalive > 0 {
		fmt.Fprintf(&sb, "PersistentKeepalive = %d\n", w.config.Keepalive)
	}

	return sb.String(), nil
}

// ValidateConfig validates the WireGuard configuration
func (w *WireGuardService) ValidateConfig() error {
	if !w.config.Enabled {
		return nil
	}

	if w.config.ServerPublicKey == "" {
		return fmt.Errorf("server_public_key is required")
	}

	if w.config.ServerEndpoint == "" {
		return fmt.Errorf("server_endpoint is required")
	}

	if len(w.config.ClientAddresses) == 0 {
		return fmt.Errorf("client_addresses cannot be empty")
	}

	decoded, err := base64.StdEncoding.DecodeString(w.config.ServerPublicKey)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf("invalid server_public_key format (must be 32-byte base64)")
	}

	return nil
}

// OPNsenseClient represents a client for OPNsense WireGuard API
type OPNsenseClient struct {
	baseURL   string
	apiKey    string
	apiSecret string
	client    *http.Client
}

// NewOPNsenseClient creates a new OPNsense API client.
//
// If httpClient is nil a default client is constructed. When insecure is true
// TLS certificate verification is skipped — this is necessary when the
// OPNsense appliance uses a self-signed certificate on a private network, but
// it makes the connection vulnerable to man-in-the-middle attacks. Set
// insecure to false and use a trusted certificate in production environments.
func NewOPNsenseClient(url, apiKey, apiSecret string, httpClient *http.Client, insecure bool) *OPNsenseClient {
	if httpClient == nil {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec // user-controlled flag
			},
		}
	}
	return &OPNsenseClient{
		baseURL:   url,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		client:    httpClient,
	}
}

// PeerRow represents a WireGuard peer from the OPNsense search response
type PeerRow struct {
	UUID          string `json:"uuid"`
	Name          string `json:"name"`
	PubKey        string `json:"pubkey"`
	TunnelAddress string `json:"tunneladdress"`
	Servers       string `json:"servers"`
}

// searchResponse represents the OPNsense search API response
type searchResponse struct {
	Rows []PeerRow `json:"rows"`
}

// normalizeTunnelAddress strips the CIDR suffix (e.g. "/32") so that
// addresses from different sources can be compared reliably.
func normalizeTunnelAddress(addr string) string {
	addr = strings.TrimSpace(addr)
	if idx := strings.Index(addr, "/"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

// SearchPeerByTunnelAddress finds an existing peer by its tunnel address.
// The comparison ignores CIDR notation so "172.17.18.103/32" matches "172.17.18.103".
func (c *OPNsenseClient) SearchPeerByTunnelAddress(tunnelAddress string) (*PeerRow, error) {
	url := fmt.Sprintf("%s/api/wireguard/client/search_client", c.baseURL)

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(`{}`))
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.apiKey, c.apiSecret)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search peers: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search peers returned status %d", resp.StatusCode)
	}

	var result searchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	normalizedTarget := normalizeTunnelAddress(tunnelAddress)
	for _, row := range result.Rows {
		if normalizeTunnelAddress(row.TunnelAddress) == normalizedTarget {
			return &row, nil
		}
	}

	return nil, nil
}

// UpdatePeer updates an existing WireGuard peer's public key.
// The servers field MUST be included to preserve the peer's attachment to
// the WireGuard server — OPNsense clears omitted fields on set_client.
func (c *OPNsenseClient) UpdatePeer(uuid, name, publicKey, tunnelAddress, servers string) error {
	url := fmt.Sprintf("%s/api/wireguard/client/set_client/%s", c.baseURL, uuid)

	clientPayload := map[string]interface{}{
		"enabled":       "1",
		"name":          name,
		"pubkey":        publicKey,
		"tunneladdress": tunnelAddress,
		"servers":       servers,
	}

	payload := map[string]interface{}{
		"client": clientPayload,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.apiKey, c.apiSecret)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to update peer: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := checkMutationResponse(resp); err != nil {
		return fmt.Errorf("failed to update peer %s: %w", uuid, err)
	}

	return c.applyChanges()
}

// CreatePeer creates a new WireGuard peer in OPNsense
func (c *OPNsenseClient) CreatePeer(name, publicKey, tunnelAddress string, keepalive int) error {
	url := fmt.Sprintf("%s/api/wireguard/client/add_client", c.baseURL)

	clientPayload := map[string]interface{}{
		"enabled":       "1",
		"name":          name,
		"pubkey":        publicKey,
		"tunneladdress": tunnelAddress,
	}
	if keepalive > 0 {
		clientPayload["keepalive"] = fmt.Sprintf("%d", keepalive)
	}

	payload := map[string]interface{}{
		"client": clientPayload,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.apiKey, c.apiSecret)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if err := checkMutationResponse(resp); err != nil {
		return fmt.Errorf("failed to create peer: %w", err)
	}

	return c.applyChanges()
}

// mutationResponse represents the OPNsense response for set/add operations.
// On success: {"result": "saved", "uuid": "..."}
// On validation error: {"result": "", "validations": {"client.pubkey": "..."}}
type mutationResponse struct {
	Result      string                 `json:"result"`
	UUID        string                 `json:"uuid"`
	Validations map[string]interface{} `json:"validations"`
}

// checkMutationResponse reads and validates an OPNsense set/add API response.
// OPNsense returns HTTP 200 even for validation errors, so we must inspect
// the body to detect failures.
func checkMutationResponse(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("OPNsense returned status %d: %s", resp.StatusCode, string(body))
	}

	var result mutationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		// Some endpoints return non-JSON; treat as success if HTTP was 200
		return nil
	}

	if len(result.Validations) > 0 {
		valJSON, _ := json.Marshal(result.Validations)
		return fmt.Errorf("OPNsense validation errors: %s", string(valJSON))
	}

	if result.Result != "saved" && result.Result != "" {
		return fmt.Errorf("OPNsense returned unexpected result: %q (body: %s)", result.Result, string(body))
	}

	return nil
}

// applyChanges tells OPNsense to apply WireGuard configuration changes
func (c *OPNsenseClient) applyChanges() error {
	url := fmt.Sprintf("%s/api/wireguard/service/reconfigure", c.baseURL)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create reconfigure request: %w", err)
	}

	req.SetBasicAuth(c.apiKey, c.apiSecret)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reconfigure: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reconfigure returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RegisterPeerWithOPNsense updates an existing peer's public key, or creates a new one.
// This works even if the peer currently has a placeholder or empty public key (e.g. freshly
// provisioned by Terraform). The peer is looked up by tunnel address, not by key.
func (w *WireGuardService) RegisterPeerWithOPNsense(username string, publicKey string, userIndex int) error {
	if !w.config.AutoRegisterPeers {
		return nil
	}

	if w.opnsenseClient == nil {
		return fmt.Errorf("OPNsense client not configured")
	}

	if userIndex < 0 || userIndex >= len(w.config.ClientAddresses) {
		return fmt.Errorf("invalid user index %d", userIndex)
	}

	client := w.opnsenseClient
	tunnelAddress := w.config.ClientAddresses[userIndex]

	// Search for existing peer by tunnel address (works regardless of current key state)
	existing, err := client.SearchPeerByTunnelAddress(tunnelAddress)
	if err != nil {
		return fmt.Errorf("failed to search for existing peer: %w", err)
	}

	if existing != nil {
		// Update existing peer's public key (rotation or first-time key assignment)
		peerName := existing.Name
		if peerName == "" {
			peerName = username
		}
		if err := client.UpdatePeer(existing.UUID, peerName, publicKey, tunnelAddress, existing.Servers); err != nil {
			return fmt.Errorf("failed to update peer (uuid=%s, tunnel=%s): %w", existing.UUID, tunnelAddress, err)
		}

		// Verify the key was actually persisted by reading it back
		updated, err := client.SearchPeerByTunnelAddress(tunnelAddress)
		if err != nil {
			return fmt.Errorf("failed to verify peer update: %w", err)
		}
		if updated == nil {
			return fmt.Errorf("peer disappeared after update (tunnel=%s)", tunnelAddress)
		}
		if updated.PubKey != publicKey {
			return fmt.Errorf("peer key mismatch after update: OPNsense has %s, expected %s (uuid=%s)", updated.PubKey, publicKey, existing.UUID)
		}

		return nil
	}

	// No existing peer found — create a new one
	return fmt.Errorf("no existing peer found for tunnel address %s — peers must be pre-provisioned by Terraform", tunnelAddress)
}
