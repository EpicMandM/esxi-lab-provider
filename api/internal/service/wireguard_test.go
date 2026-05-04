package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- GenerateKeyPair tests ---

func TestGenerateKeyPair_ReturnsValidBase64(t *testing.T) {
	priv, pub, err := GenerateKeyPair()
	require.NoError(t, err)

	privBytes, err := base64.StdEncoding.DecodeString(priv)
	require.NoError(t, err)
	assert.Len(t, privBytes, 32)

	pubBytes, err := base64.StdEncoding.DecodeString(pub)
	require.NoError(t, err)
	assert.Len(t, pubBytes, 32)
}

func TestGenerateKeyPair_DifferentEachTime(t *testing.T) {
	_, pub1, err := GenerateKeyPair()
	require.NoError(t, err)
	_, pub2, err := GenerateKeyPair()
	require.NoError(t, err)
	assert.NotEqual(t, pub1, pub2)
}

// --- RotateUserKey tests ---

func TestRotateUserKey_StoresPrivateKey(t *testing.T) {
	svc := NewWireGuardService(&WireGuardConfig{Enabled: true}, nil)

	priv, pub, err := svc.RotateUserKey("alice")
	require.NoError(t, err)
	assert.NotEmpty(t, priv)
	assert.NotEmpty(t, pub)

	// Key should be stored
	assert.Equal(t, priv, svc.privateKeys["alice"])
}

func TestRotateUserKey_ChangesKeyOnSubsequentCall(t *testing.T) {
	svc := NewWireGuardService(&WireGuardConfig{Enabled: true}, nil)

	priv1, _, err := svc.RotateUserKey("alice")
	require.NoError(t, err)
	priv2, _, err := svc.RotateUserKey("alice")
	require.NoError(t, err)
	assert.NotEqual(t, priv1, priv2)
}

// --- GenerateClientConfig tests ---

func TestGenerateClientConfig_Success(t *testing.T) {
	cfg := &WireGuardConfig{
		Enabled:         true,
		ServerPublicKey: "serverpubkey123",
		ServerEndpoint:  "vpn.example.com:51820",
		AllowedIPs:      []string{"10.0.0.0/8"},
		MTU:             1420,
		ClientAddresses: []string{"172.17.18.101/32", "172.17.18.102/32"},
		Keepalive:       25,
	}
	svc := NewWireGuardService(cfg, nil)
	svc.privateKeys["alice"] = "privkey-alice"

	config, err := svc.GenerateClientConfig("alice", 0)
	require.NoError(t, err)

	assert.Contains(t, config, "[Interface]")
	assert.Contains(t, config, "PrivateKey = privkey-alice")
	assert.Contains(t, config, "Address = 172.17.18.101/32")
	assert.Contains(t, config, "MTU = 1420")
	assert.Contains(t, config, "[Peer]")
	assert.Contains(t, config, "PublicKey = serverpubkey123")
	assert.Contains(t, config, "Endpoint = vpn.example.com:51820")
	assert.Contains(t, config, "AllowedIPs = 10.0.0.0/8")
	assert.Contains(t, config, "PersistentKeepalive = 25")
}

func TestGenerateClientConfig_Disabled(t *testing.T) {
	cfg := &WireGuardConfig{Enabled: false}
	svc := NewWireGuardService(cfg, nil)

	_, err := svc.GenerateClientConfig("alice", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
}

func TestGenerateClientConfig_InvalidIndex(t *testing.T) {
	cfg := &WireGuardConfig{
		Enabled:         true,
		ClientAddresses: []string{"172.17.18.101/32"},
	}
	svc := NewWireGuardService(cfg, nil)
	svc.privateKeys["alice"] = "key"

	_, err := svc.GenerateClientConfig("alice", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user index")
}

func TestGenerateClientConfig_NegativeIndex(t *testing.T) {
	cfg := &WireGuardConfig{
		Enabled:         true,
		ClientAddresses: []string{"172.17.18.101/32"},
	}
	svc := NewWireGuardService(cfg, nil)
	svc.privateKeys["alice"] = "key"

	_, err := svc.GenerateClientConfig("alice", -1)
	assert.Error(t, err)
}

func TestGenerateClientConfig_NoPrivateKey(t *testing.T) {
	cfg := &WireGuardConfig{
		Enabled:         true,
		ClientAddresses: []string{"172.17.18.101/32"},
	}
	svc := NewWireGuardService(cfg, nil)

	_, err := svc.GenerateClientConfig("unknown", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no private key found")
}

func TestGenerateClientConfig_NoMTU(t *testing.T) {
	cfg := &WireGuardConfig{
		Enabled:         true,
		ServerPublicKey: "key",
		ServerEndpoint:  "ep",
		ClientAddresses: []string{"172.17.18.101/32"},
	}
	svc := NewWireGuardService(cfg, nil)
	svc.privateKeys["alice"] = "pk"

	config, err := svc.GenerateClientConfig("alice", 0)
	require.NoError(t, err)
	assert.NotContains(t, config, "MTU")
}

func TestGenerateClientConfig_NoAllowedIPs(t *testing.T) {
	cfg := &WireGuardConfig{
		Enabled:         true,
		ServerPublicKey: "key",
		ServerEndpoint:  "ep",
		ClientAddresses: []string{"172.17.18.101/32"},
		AllowedIPs:      nil,
	}
	svc := NewWireGuardService(cfg, nil)
	svc.privateKeys["alice"] = "pk"

	config, err := svc.GenerateClientConfig("alice", 0)
	require.NoError(t, err)
	assert.NotContains(t, config, "AllowedIPs")
}

func TestGenerateClientConfig_NoKeepalive(t *testing.T) {
	cfg := &WireGuardConfig{
		Enabled:         true,
		ServerPublicKey: "key",
		ServerEndpoint:  "ep",
		ClientAddresses: []string{"172.17.18.101/32"},
		Keepalive:       0,
	}
	svc := NewWireGuardService(cfg, nil)
	svc.privateKeys["alice"] = "pk"

	config, err := svc.GenerateClientConfig("alice", 0)
	require.NoError(t, err)
	assert.NotContains(t, config, "PersistentKeepalive")
}

// --- ValidateConfig tests ---

func TestValidateConfig_Disabled(t *testing.T) {
	cfg := &WireGuardConfig{Enabled: false}
	svc := NewWireGuardService(cfg, nil)
	assert.NoError(t, svc.ValidateConfig())
}

func TestValidateConfig_MissingServerPublicKey(t *testing.T) {
	cfg := &WireGuardConfig{
		Enabled:        true,
		ServerEndpoint: "ep",
	}
	svc := NewWireGuardService(cfg, nil)
	err := svc.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server_public_key is required")
}

func TestValidateConfig_MissingServerEndpoint(t *testing.T) {
	// Valid 32-byte key encoded in base64
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	cfg := &WireGuardConfig{
		Enabled:        true,
		ServerPublicKey: key,
	}
	svc := NewWireGuardService(cfg, nil)
	err := svc.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "server_endpoint is required")
}

func TestValidateConfig_EmptyClientAddresses(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	cfg := &WireGuardConfig{
		Enabled:         true,
		ServerPublicKey: key,
		ServerEndpoint:  "ep",
		ClientAddresses: []string{},
	}
	svc := NewWireGuardService(cfg, nil)
	err := svc.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client_addresses cannot be empty")
}

func TestValidateConfig_InvalidKeyFormat(t *testing.T) {
	cfg := &WireGuardConfig{
		Enabled:         true,
		ServerPublicKey: "not-valid-base64!!!",
		ServerEndpoint:  "ep",
		ClientAddresses: []string{"addr"},
	}
	svc := NewWireGuardService(cfg, nil)
	err := svc.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid server_public_key format")
}

func TestValidateConfig_KeyWrongLength(t *testing.T) {
	// 16 bytes instead of 32
	key := base64.StdEncoding.EncodeToString(make([]byte, 16))
	cfg := &WireGuardConfig{
		Enabled:         true,
		ServerPublicKey: key,
		ServerEndpoint:  "ep",
		ClientAddresses: []string{"addr"},
	}
	svc := NewWireGuardService(cfg, nil)
	err := svc.ValidateConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid server_public_key format")
}

func TestValidateConfig_Valid(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	cfg := &WireGuardConfig{
		Enabled:         true,
		ServerPublicKey: key,
		ServerEndpoint:  "vpn.example.com:51820",
		ClientAddresses: []string{"172.17.18.101/32"},
	}
	svc := NewWireGuardService(cfg, nil)
	assert.NoError(t, svc.ValidateConfig())
}

// --- normalizeTunnelAddress tests ---

func TestNormalizeTunnelAddress(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"172.17.18.103/32", "172.17.18.103"},
		{"172.17.18.103", "172.17.18.103"},
		{"  172.17.18.103/32  ", "172.17.18.103"},
		{"10.0.0.1/24", "10.0.0.1"},
		{"", ""},
		{"  ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeTunnelAddress(tt.input))
		})
	}
}

// --- checkMutationResponse tests ---

func TestCheckMutationResponse_Success(t *testing.T) {
	body := `{"result":"saved","uuid":"abc-123"}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	assert.NoError(t, checkMutationResponse(resp))
}

func TestCheckMutationResponse_EmptyResult(t *testing.T) {
	body := `{"result":""}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	assert.NoError(t, checkMutationResponse(resp))
}

func TestCheckMutationResponse_ValidationErrors(t *testing.T) {
	body := `{"result":"","validations":{"client.pubkey":"invalid key format"}}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	err := checkMutationResponse(resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "validation errors")
}

func TestCheckMutationResponse_UnexpectedResult(t *testing.T) {
	body := `{"result":"error"}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	err := checkMutationResponse(resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected result")
}

func TestCheckMutationResponse_HTTPError(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader("server error")),
	}
	err := checkMutationResponse(resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestCheckMutationResponse_NonJSON(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("OK")),
	}
	// Non-JSON with 200 is treated as success
	assert.NoError(t, checkMutationResponse(resp))
}

func TestCheckMutationResponse_HTTP201(t *testing.T) {
	body := `{"result":"saved","uuid":"new-id"}`
	resp := &http.Response{
		StatusCode: http.StatusCreated,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	assert.NoError(t, checkMutationResponse(resp))
}

// --- RegisterPeerWithOPNsense tests (with mock OPNsenseAPI) ---

func TestRegisterPeer_AutoRegisterDisabled(t *testing.T) {
	svc := NewWireGuardService(&WireGuardConfig{AutoRegisterPeers: false}, nil)
	err := svc.RegisterPeerWithOPNsense("alice", "pubkey", 0)
	assert.NoError(t, err)
}

func TestRegisterPeer_NoClient(t *testing.T) {
	svc := NewWireGuardService(&WireGuardConfig{AutoRegisterPeers: true, ClientAddresses: []string{"addr"}}, nil)
	err := svc.RegisterPeerWithOPNsense("alice", "pubkey", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "OPNsense client not configured")
}

func TestRegisterPeer_InvalidIndex(t *testing.T) {
	mock := &mockOPNsenseAPI{}
	svc := NewWireGuardService(&WireGuardConfig{
		AutoRegisterPeers: true,
		ClientAddresses:   []string{"addr"},
	}, mock)
	err := svc.RegisterPeerWithOPNsense("alice", "pubkey", 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid user index")
}

func TestRegisterPeer_NegativeIndex(t *testing.T) {
	mock := &mockOPNsenseAPI{}
	svc := NewWireGuardService(&WireGuardConfig{
		AutoRegisterPeers: true,
		ClientAddresses:   []string{"addr"},
	}, mock)
	err := svc.RegisterPeerWithOPNsense("alice", "pubkey", -1)
	assert.Error(t, err)
}

func TestRegisterPeer_UpdateExisting(t *testing.T) {
	publicKey := "new-pub-key"
	mock := &mockOPNsenseAPI{
		searchFn: func(addr string) (*PeerRow, error) {
			return &PeerRow{UUID: "uuid-1", Name: "alice", TunnelAddress: addr, PubKey: "old-key", Servers: "srv1"}, nil
		},
		updateFn: func(uuid, name, pubKey, tunnelAddr, servers string) error {
			return nil
		},
	}
	// After update, search returns the new key
	callCount := 0
	mock.searchFn = func(addr string) (*PeerRow, error) {
		callCount++
		if callCount == 1 {
			return &PeerRow{UUID: "uuid-1", Name: "alice", TunnelAddress: addr, PubKey: "old-key", Servers: "srv1"}, nil
		}
		return &PeerRow{UUID: "uuid-1", Name: "alice", TunnelAddress: addr, PubKey: publicKey, Servers: "srv1"}, nil
	}

	svc := NewWireGuardService(&WireGuardConfig{
		AutoRegisterPeers: true,
		ClientAddresses:   []string{"172.17.18.101/32"},
	}, mock)

	err := svc.RegisterPeerWithOPNsense("alice", publicKey, 0)
	assert.NoError(t, err)
}

func TestRegisterPeer_UpdateExisting_EmptyName(t *testing.T) {
	publicKey := "new-pub-key"
	callCount := 0
	mock := &mockOPNsenseAPI{
		searchFn: func(addr string) (*PeerRow, error) {
			callCount++
			if callCount == 1 {
				return &PeerRow{UUID: "uuid-1", Name: "", TunnelAddress: addr, PubKey: "old-key", Servers: "srv1"}, nil
			}
			return &PeerRow{UUID: "uuid-1", Name: "alice", TunnelAddress: addr, PubKey: publicKey, Servers: "srv1"}, nil
		},
		updateFn: func(uuid, name, pubKey, tunnelAddr, servers string) error {
			// When original name is empty, should use username
			assert.Equal(t, "alice", name)
			return nil
		},
	}

	svc := NewWireGuardService(&WireGuardConfig{
		AutoRegisterPeers: true,
		ClientAddresses:   []string{"172.17.18.101/32"},
	}, mock)

	err := svc.RegisterPeerWithOPNsense("alice", publicKey, 0)
	assert.NoError(t, err)
}

func TestRegisterPeer_NoPeerFound(t *testing.T) {
	mock := &mockOPNsenseAPI{
		searchFn: func(addr string) (*PeerRow, error) {
			return nil, nil
		},
	}

	svc := NewWireGuardService(&WireGuardConfig{
		AutoRegisterPeers: true,
		ClientAddresses:   []string{"172.17.18.101/32"},
	}, mock)

	err := svc.RegisterPeerWithOPNsense("alice", "pubkey", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no existing peer found")
}

func TestRegisterPeer_SearchError(t *testing.T) {
	mock := &mockOPNsenseAPI{
		searchFn: func(addr string) (*PeerRow, error) {
			return nil, fmt.Errorf("network error")
		},
	}

	svc := NewWireGuardService(&WireGuardConfig{
		AutoRegisterPeers: true,
		ClientAddresses:   []string{"172.17.18.101/32"},
	}, mock)

	err := svc.RegisterPeerWithOPNsense("alice", "pubkey", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to search")
}

func TestRegisterPeer_UpdateError(t *testing.T) {
	mock := &mockOPNsenseAPI{
		searchFn: func(addr string) (*PeerRow, error) {
			return &PeerRow{UUID: "uuid-1", Name: "alice", TunnelAddress: addr, Servers: "srv"}, nil
		},
		updateFn: func(uuid, name, pubKey, tunnelAddr, servers string) error {
			return fmt.Errorf("update failed")
		},
	}

	svc := NewWireGuardService(&WireGuardConfig{
		AutoRegisterPeers: true,
		ClientAddresses:   []string{"172.17.18.101/32"},
	}, mock)

	err := svc.RegisterPeerWithOPNsense("alice", "pubkey", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update peer")
}

func TestRegisterPeer_VerifyError(t *testing.T) {
	callCount := 0
	mock := &mockOPNsenseAPI{
		searchFn: func(addr string) (*PeerRow, error) {
			callCount++
			if callCount == 1 {
				return &PeerRow{UUID: "uuid-1", Name: "alice", TunnelAddress: addr, Servers: "srv"}, nil
			}
			return nil, fmt.Errorf("verify failed")
		},
		updateFn: func(uuid, name, pubKey, tunnelAddr, servers string) error {
			return nil
		},
	}

	svc := NewWireGuardService(&WireGuardConfig{
		AutoRegisterPeers: true,
		ClientAddresses:   []string{"172.17.18.101/32"},
	}, mock)

	err := svc.RegisterPeerWithOPNsense("alice", "pubkey", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to verify")
}

func TestRegisterPeer_PeerDisappearedAfterUpdate(t *testing.T) {
	callCount := 0
	mock := &mockOPNsenseAPI{
		searchFn: func(addr string) (*PeerRow, error) {
			callCount++
			if callCount == 1 {
				return &PeerRow{UUID: "uuid-1", Name: "alice", TunnelAddress: addr, Servers: "srv"}, nil
			}
			return nil, nil
		},
		updateFn: func(uuid, name, pubKey, tunnelAddr, servers string) error {
			return nil
		},
	}

	svc := NewWireGuardService(&WireGuardConfig{
		AutoRegisterPeers: true,
		ClientAddresses:   []string{"172.17.18.101/32"},
	}, mock)

	err := svc.RegisterPeerWithOPNsense("alice", "pubkey", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "peer disappeared")
}

func TestRegisterPeer_KeyMismatchAfterUpdate(t *testing.T) {
	callCount := 0
	mock := &mockOPNsenseAPI{
		searchFn: func(addr string) (*PeerRow, error) {
			callCount++
			if callCount == 1 {
				return &PeerRow{UUID: "uuid-1", Name: "alice", TunnelAddress: addr, PubKey: "old", Servers: "srv"}, nil
			}
			return &PeerRow{UUID: "uuid-1", Name: "alice", TunnelAddress: addr, PubKey: "wrong-key", Servers: "srv"}, nil
		},
		updateFn: func(uuid, name, pubKey, tunnelAddr, servers string) error {
			return nil
		},
	}

	svc := NewWireGuardService(&WireGuardConfig{
		AutoRegisterPeers: true,
		ClientAddresses:   []string{"172.17.18.101/32"},
	}, mock)

	err := svc.RegisterPeerWithOPNsense("alice", "expected-key", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "peer key mismatch")
}

// --- OPNsenseClient tests (with httptest) ---

func TestOPNsenseClient_SearchPeerByTunnelAddress_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/wireguard/client/search_client", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(searchResponse{
			Rows: []PeerRow{
				{UUID: "id1", Name: "alice", TunnelAddress: "172.17.18.101/32", PubKey: "pk1"},
				{UUID: "id2", Name: "bob", TunnelAddress: "172.17.18.102/32", PubKey: "pk2"},
			},
		})
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	peer, err := client.SearchPeerByTunnelAddress("172.17.18.101")
	require.NoError(t, err)
	require.NotNil(t, peer)
	assert.Equal(t, "id1", peer.UUID)
	assert.Equal(t, "alice", peer.Name)
}

func TestOPNsenseClient_SearchPeerByTunnelAddress_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(searchResponse{Rows: []PeerRow{}})
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	peer, err := client.SearchPeerByTunnelAddress("172.17.18.199")
	require.NoError(t, err)
	assert.Nil(t, peer)
}

func TestOPNsenseClient_SearchPeerByTunnelAddress_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	_, err := client.SearchPeerByTunnelAddress("172.17.18.101")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestOPNsenseClient_SearchPeerByTunnelAddress_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	_, err := client.SearchPeerByTunnelAddress("172.17.18.101")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode")
}

func TestOPNsenseClient_UpdatePeer_Success(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// set_client
			assert.Contains(t, r.URL.Path, "/api/wireguard/client/set_client/uuid-1")
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			_ = json.Unmarshal(body, &payload)
			client := payload["client"].(map[string]interface{})
			assert.Equal(t, "new-pubkey", client["pubkey"])
			_, _ = w.Write([]byte(`{"result":"saved"}`))
		} else {
			// reconfigure
			assert.Equal(t, "/api/wireguard/service/reconfigure", r.URL.Path)
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	err := client.UpdatePeer("uuid-1", "alice", "new-pubkey", "172.17.18.101/32", "srv1")
	assert.NoError(t, err)
}

func TestOPNsenseClient_UpdatePeer_ApplyChangesError(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			_, _ = w.Write([]byte(`{"result":"saved"}`))
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("apply failed"))
		}
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	err := client.UpdatePeer("uuid-1", "alice", "pk", "addr", "srv")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reconfigure")
}

func TestOPNsenseClient_CreatePeer_Success(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			assert.Equal(t, "/api/wireguard/client/add_client", r.URL.Path)
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			_ = json.Unmarshal(body, &payload)
			client := payload["client"].(map[string]interface{})
			assert.Equal(t, "new-peer", client["name"])
			assert.Equal(t, "pubkey", client["pubkey"])
			_, _ = w.Write([]byte(`{"result":"saved","uuid":"new-uuid"}`))
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	err := client.CreatePeer("new-peer", "pubkey", "172.17.18.101/32", 25)
	assert.NoError(t, err)
}

func TestOPNsenseClient_CreatePeer_WithoutKeepalive(t *testing.T) {
	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			body, _ := io.ReadAll(r.Body)
			var payload map[string]interface{}
			_ = json.Unmarshal(body, &payload)
			client := payload["client"].(map[string]interface{})
			_, hasKeepalive := client["keepalive"]
			assert.False(t, hasKeepalive)
			_, _ = w.Write([]byte(`{"result":"saved"}`))
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	err := client.CreatePeer("peer", "pk", "addr", 0)
	assert.NoError(t, err)
}

func TestOPNsenseClient_CreatePeer_ValidationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":"","validations":{"client.pubkey":"bad key"}}`))
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	err := client.CreatePeer("peer", "bad-pk", "addr", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create peer")
}

func TestOPNsenseClient_ApplyChanges_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/wireguard/service/reconfigure", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	err := client.applyChanges()
	assert.NoError(t, err)
}

func TestOPNsenseClient_ApplyChanges_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("service down"))
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	err := client.applyChanges()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reconfigure returned status 503")
}

func TestNewOPNsenseClient_DefaultHTTPClient(t *testing.T) {
	client := NewOPNsenseClient("https://example.com", "key", "secret", nil, true)
	assert.NotNil(t, client.client)
	assert.Equal(t, "https://example.com", client.baseURL)
}

func TestNewOPNsenseClient_CustomHTTPClient(t *testing.T) {
	custom := &http.Client{}
	client := NewOPNsenseClient("https://example.com", "key", "secret", custom, false)
	assert.Equal(t, custom, client.client)
}

// --- OPNsenseClient network error tests ---

func TestOPNsenseClient_SearchPeerByTunnelAddress_NetworkError(t *testing.T) {
	client := NewOPNsenseClient("http://127.0.0.1:1", "key", "secret", &http.Client{}, false)
	_, err := client.SearchPeerByTunnelAddress("172.17.18.101")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to search peers")
}

func TestOPNsenseClient_UpdatePeer_NetworkError(t *testing.T) {
	client := NewOPNsenseClient("http://127.0.0.1:1", "key", "secret", &http.Client{}, false)
	err := client.UpdatePeer("uuid", "name", "pk", "addr", "srv")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update peer")
}

func TestOPNsenseClient_CreatePeer_NetworkError(t *testing.T) {
	client := NewOPNsenseClient("http://127.0.0.1:1", "key", "secret", &http.Client{}, false)
	err := client.CreatePeer("name", "pk", "addr", 0)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send request")
}

func TestOPNsenseClient_ApplyChanges_NetworkError(t *testing.T) {
	client := NewOPNsenseClient("http://127.0.0.1:1", "key", "secret", &http.Client{}, false)
	err := client.applyChanges()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reconfigure")
}

func TestOPNsenseClient_UpdatePeer_MutationError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":"","validations":{"client.pubkey":"invalid"}}`))
	}))
	defer srv.Close()

	client := NewOPNsenseClient(srv.URL, "key", "secret", srv.Client(), false)
	err := client.UpdatePeer("uuid-1", "alice", "bad-pk", "addr", "srv")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update peer uuid-1")
}

// --- helper mock for OPNsenseAPI in wireguard unit tests ---

type mockOPNsenseAPI struct {
	searchFn func(addr string) (*PeerRow, error)
	updateFn func(uuid, name, publicKey, tunnelAddress, servers string) error
	createFn func(name, publicKey, tunnelAddress string, keepalive int) error
}

func (m *mockOPNsenseAPI) SearchPeerByTunnelAddress(addr string) (*PeerRow, error) {
	if m.searchFn != nil {
		return m.searchFn(addr)
	}
	return nil, nil
}

func (m *mockOPNsenseAPI) UpdatePeer(uuid, name, publicKey, tunnelAddress, servers string) error {
	if m.updateFn != nil {
		return m.updateFn(uuid, name, publicKey, tunnelAddress, servers)
	}
	return nil
}

func (m *mockOPNsenseAPI) CreatePeer(name, publicKey, tunnelAddress string, keepalive int) error {
	if m.createFn != nil {
		return m.createFn(name, publicKey, tunnelAddress, keepalive)
	}
	return nil
}
