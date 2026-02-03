package service

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	// OPNsense API configuration (loaded from environment)
	OPNsenseURL       string `toml:"opnsense_url"`
	OPNsenseAPIKey    string `toml:"opnsense_api_key"`
	OPNsenseAPISecret string `toml:"opnsense_api_secret"`
	AutoRegisterPeers bool   `toml:"auto_register_peers"`
}

// WireGuardService manages WireGuard tunnel configurations
type WireGuardService struct {
	config *WireGuardConfig
	// Store private keys per user (in production, this should be persisted)
	privateKeys map[string]string
}

// NewWireGuardService creates a new WireGuard service
func NewWireGuardService(config *WireGuardConfig) *WireGuardService {
	return &WireGuardService{
		config:      config,
		privateKeys: make(map[string]string),
	}
}

// GenerateKeyPair generates a new WireGuard private/public key pair
func GenerateKeyPair() (privateKey, publicKey string, err error) {
	// Generate random 32-byte private key
	var privKeyBytes [32]byte
	if _, err := rand.Read(privKeyBytes[:]); err != nil {
		return "", "", fmt.Errorf("failed to generate random key: %w", err)
	}

	// Clamp the private key as per Curve25519 requirements
	privKeyBytes[0] &= 248
	privKeyBytes[31] &= 127
	privKeyBytes[31] |= 64

	// Derive public key from private key
	var pubKeyBytes [32]byte
	curve25519.ScalarBaseMult(&pubKeyBytes, &privKeyBytes)

	// Encode to base64
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

	// Store the private key (in production, persist this securely)
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

	// Build the configuration file
	var sb strings.Builder

	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", privateKey))
	sb.WriteString(fmt.Sprintf("Address = %s\n", w.config.ClientAddresses[userIndex]))

	if w.config.MTU > 0 {
		sb.WriteString(fmt.Sprintf("MTU = %d\n", w.config.MTU))
	}

	sb.WriteString("\n[Peer]\n")
	sb.WriteString(fmt.Sprintf("PublicKey = %s\n", w.config.ServerPublicKey))
	sb.WriteString(fmt.Sprintf("Endpoint = %s\n", w.config.ServerEndpoint))

	if len(w.config.AllowedIPs) > 0 {
		sb.WriteString(fmt.Sprintf("AllowedIPs = %s\n", strings.Join(w.config.AllowedIPs, ", ")))
	}

	if w.config.Keepalive > 0 {
		sb.WriteString(fmt.Sprintf("PersistentKeepalive = %d\n", w.config.Keepalive))
	}

	return sb.String(), nil
}

// GetPublicKey retrieves the public key for a username
func (w *WireGuardService) GetPublicKey(username string) (string, error) {
	privateKey, ok := w.privateKeys[username]
	if !ok {
		return "", fmt.Errorf("no private key found for user %s", username)
	}

	// Decode private key
	privKeyBytes, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil || len(privKeyBytes) != 32 {
		return "", fmt.Errorf("invalid private key for user %s", username)
	}

	// Derive public key
	var privKey, pubKey [32]byte
	copy(privKey[:], privKeyBytes)
	curve25519.ScalarBaseMult(&pubKey, &privKey)

	return base64.StdEncoding.EncodeToString(pubKey[:]), nil
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

	// Validate server public key format
	decoded, err := base64.StdEncoding.DecodeString(w.config.ServerPublicKey)
	if err != nil || len(decoded) != 32 {
		return fmt.Errorf("invalid server_public_key format (must be 32-byte base64)")
	}

	return nil
}

// SecureCompare performs constant-time comparison of two strings
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// OPNsenseClient represents a client for OPNsense WireGuard API
type OPNsenseClient struct {
	baseURL   string
	apiKey    string
	apiSecret string
	client    *http.Client
}

// NewOPNsenseClient creates a new OPNsense API client
func NewOPNsenseClient(url, apiKey, apiSecret string) *OPNsenseClient {
	return &OPNsenseClient{
		baseURL:   url,
		apiKey:    apiKey,
		apiSecret: apiSecret,
		client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// RegisterPeer registers a new WireGuard peer with OPNsense
func (c *OPNsenseClient) RegisterPeer(name, publicKey, tunnelAddress string, keepalive int) error {
	// OPNsense WireGuard API endpoint for creating a client
	url := fmt.Sprintf("%s/api/wireguard/client/addClient", c.baseURL)

	payload := map[string]interface{}{
		"client": map[string]interface{}{
			"enabled":       "1",
			"name":          name,
			"pubkey":        publicKey,
			"tunneladdress": tunnelAddress,
			"keepalive":     fmt.Sprintf("%d", keepalive),
		},
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
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("OPNsense API returned status %d", resp.StatusCode)
	}

	// Apply the changes
	return c.applyChanges()
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
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			err = fmt.Errorf("failed to close response body: %w", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("reconfigure returned status %d", resp.StatusCode)
	}

	return nil
}

// RegisterPeerWithOPNsense registers the peer's public key with OPNsense server
func (w *WireGuardService) RegisterPeerWithOPNsense(username string, publicKey string, userIndex int) error {
	if !w.config.AutoRegisterPeers {
		return nil // Auto-registration disabled
	}

	if w.config.OPNsenseURL == "" || w.config.OPNsenseAPIKey == "" {
		return fmt.Errorf("OPNsense API credentials not configured")
	}

	if userIndex < 0 || userIndex >= len(w.config.ClientAddresses) {
		return fmt.Errorf("invalid user index %d", userIndex)
	}

	client := NewOPNsenseClient(w.config.OPNsenseURL, w.config.OPNsenseAPIKey, w.config.OPNsenseAPISecret)

	// Use the configured tunnel address for this user
	tunnelAddress := w.config.ClientAddresses[userIndex]

	err := client.RegisterPeer(username, publicKey, tunnelAddress, w.config.Keepalive)
	if err != nil {
		return fmt.Errorf("failed to register peer with OPNsense: %w", err)
	}

	return nil
}
