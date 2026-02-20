package service

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestESXiConfig_UserVMPairs(t *testing.T) {
	tests := []struct {
		name     string
		mappings map[string]string
		want     []UserVMPair
	}{
		{
			name:     "sorted by username",
			mappings: map[string]string{"bob": "vm-bob", "alice": "vm-alice"},
			want:     []UserVMPair{{User: "alice", VM: "vm-alice"}, {User: "bob", VM: "vm-bob"}},
		},
		{
			name:     "single mapping",
			mappings: map[string]string{"user1": "vm1"},
			want:     []UserVMPair{{User: "user1", VM: "vm1"}},
		},
		{
			name:     "empty mappings",
			mappings: map[string]string{},
			want:     []UserVMPair{},
		},
		{
			name:     "nil mappings",
			mappings: nil,
			want:     []UserVMPair{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ESXiConfig{UserVMMappings: tt.mappings}
			got := cfg.UserVMPairs()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestESXiConfig_VMs(t *testing.T) {
	cfg := &ESXiConfig{
		UserVMMappings: map[string]string{"bob": "vm-bob", "alice": "vm-alice"},
	}
	vms := cfg.VMs()
	assert.Equal(t, []string{"vm-alice", "vm-bob"}, vms)
}

func TestESXiConfig_VMs_Empty(t *testing.T) {
	cfg := &ESXiConfig{UserVMMappings: map[string]string{}}
	assert.Empty(t, cfg.VMs())
}

func TestESXiConfig_Users(t *testing.T) {
	cfg := &ESXiConfig{
		UserVMMappings: map[string]string{"bob": "vm-bob", "alice": "vm-alice"},
	}
	users := cfg.Users()
	assert.Equal(t, []string{"alice", "bob"}, users)
}

func TestESXiConfig_Users_Empty(t *testing.T) {
	cfg := &ESXiConfig{UserVMMappings: map[string]string{}}
	assert.Empty(t, cfg.Users())
}

func TestLoadFeatureConfig_ValidTOML(t *testing.T) {
	content := `
[calendar]
calendar_id = "test@group.calendar.google.com"
service_account_path = "/tmp/sa.json"

[esxi]
url = "https://esxi.local"

[esxi.user_vm_mappings]
alice = "vm-alice"
bob = "vm-bob"

[wireguard]
enabled = false
`
	tmpFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	cfg, err := LoadFeatureConfig(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "test@group.calendar.google.com", cfg.Calendar.CalendarID)
	assert.Len(t, cfg.ESXi.UserVMMappings, 2)
	assert.False(t, cfg.WireGuard.Enabled)
}

func TestLoadFeatureConfig_InvalidTOML(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "bad.toml")
	require.NoError(t, os.WriteFile(tmpFile, []byte("{{invalid"), 0o644))

	_, err := LoadFeatureConfig(tmpFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load feature config")
}

func TestLoadFeatureConfig_FileNotFound(t *testing.T) {
	_, err := LoadFeatureConfig("/nonexistent/config.toml")
	assert.Error(t, err)
}

func TestLoadServiceAccountToken_FromConfigPath(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "sa.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(`{"type":"service_account"}`), 0o644))

	cfg := &CalendarConfig{ServiceAccountPath: tmpFile}
	data, err := cfg.LoadServiceAccountToken()
	require.NoError(t, err)
	assert.JSONEq(t, `{"type":"service_account"}`, string(data))
}

func TestLoadServiceAccountToken_EnvOverride(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "sa-env.json")
	require.NoError(t, os.WriteFile(tmpFile, []byte(`{"from":"env"}`), 0o644))

	t.Setenv("SERVICE_ACCOUNT_PATH", tmpFile)

	cfg := &CalendarConfig{ServiceAccountPath: "/nonexistent"}
	data, err := cfg.LoadServiceAccountToken()
	require.NoError(t, err)
	assert.JSONEq(t, `{"from":"env"}`, string(data))
}

func TestLoadServiceAccountToken_NoPath(t *testing.T) {
	t.Setenv("SERVICE_ACCOUNT_PATH", "")
	cfg := &CalendarConfig{ServiceAccountPath: ""}
	_, err := cfg.LoadServiceAccountToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "service_account_path is not configured")
}

func TestLoadServiceAccountToken_FileNotFound(t *testing.T) {
	t.Setenv("SERVICE_ACCOUNT_PATH", "")
	cfg := &CalendarConfig{ServiceAccountPath: "/nonexistent/sa.json"}
	_, err := cfg.LoadServiceAccountToken()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read service account file")
}

func TestLoadFeatureConfig_WithSnapshotName(t *testing.T) {
	content := `
[calendar]
calendar_id = "cal@group.calendar.google.com"

[esxi]
url = "https://esxi.local"
snapshot_name = "clean-state"

[wireguard]
enabled = false
`
	tmpFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	cfg, err := LoadFeatureConfig(tmpFile)
	require.NoError(t, err)
	require.NotNil(t, cfg.ESXi.SnapshotName)
	assert.Equal(t, "clean-state", *cfg.ESXi.SnapshotName)
}

func TestLoadFeatureConfig_WireGuardFull(t *testing.T) {
	content := `
[calendar]
calendar_id = "cal@group.calendar.google.com"

[esxi]
url = "https://esxi.local"

[wireguard]
enabled = true
server_public_key = "abc123"
server_endpoint = "vpn.example.com:51820"
server_tunnel_network = "172.17.18.0/24"
allowed_ips = ["10.0.0.0/8", "172.16.0.0/12"]
mtu = 1420
client_addresses = ["172.17.18.101/32", "172.17.18.102/32"]
keepalive = 25
opnsense_url = "https://opnsense.local"
auto_register_peers = true
`
	tmpFile := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(content), 0o644))

	cfg, err := LoadFeatureConfig(tmpFile)
	require.NoError(t, err)
	assert.True(t, cfg.WireGuard.Enabled)
	assert.Equal(t, "abc123", cfg.WireGuard.ServerPublicKey)
	assert.Equal(t, "vpn.example.com:51820", cfg.WireGuard.ServerEndpoint)
	assert.Equal(t, 1420, cfg.WireGuard.MTU)
	assert.Len(t, cfg.WireGuard.AllowedIPs, 2)
	assert.Len(t, cfg.WireGuard.ClientAddresses, 2)
	assert.Equal(t, 25, cfg.WireGuard.Keepalive)
	assert.True(t, cfg.WireGuard.AutoRegisterPeers)
}
