package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Base valid environment
	validEnv := map[string]string{
		"ESXI_URL":      "https://esxi.example.com",
		"ESXI_USERNAME": "admin",
		"ESXI_PASSWORD": "password",
	}

	t.Run("valid config", func(t *testing.T) {
		for k, v := range validEnv {
			t.Setenv(k, v)
		}
		t.Setenv("ESXI_INSECURE", "true")

		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "https://esxi.example.com", cfg.ESXiURL)
		assert.True(t, cfg.ESXiInsecure)
	})

	t.Run("insecure defaults to false when empty", func(t *testing.T) {
		for k, v := range validEnv {
			t.Setenv(k, v)
		}
		t.Setenv("ESXI_INSECURE", "")
		cfg, err := Load()
		require.NoError(t, err)
		assert.False(t, cfg.ESXiInsecure, "ESXI_INSECURE should default to false")
	})

	// Table-driven test for missing variables
	missingVarTests := []struct {
		name    string
		unset   string // The env var to leave unset
		wantErr string
	}{
		{
			name:    "missing ESXI_URL",
			unset:   "ESXI_URL",
			wantErr: "ESXI_URL is required",
		},
		{
			name:    "missing ESXI_USERNAME",
			unset:   "ESXI_USERNAME",
			wantErr: "ESXI_USERNAME is required",
		},
		{
			name:    "missing ESXI_PASSWORD",
			unset:   "ESXI_PASSWORD",
			wantErr: "ESXI_PASSWORD is required",
		},
	}

	for _, tt := range missingVarTests {
		t.Run(tt.name, func(t *testing.T) {
			// Set all valid envs first
			for k, v := range validEnv {
				t.Setenv(k, v)
			}
			// Then unset the one for this test case
			t.Setenv(tt.unset, "")

			_, err := Load()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestParseInsecure(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"true string", "true", true},
		{"false string", "false", false},
		{"empty string", "", false},
		{"invalid string", "abc", false},
		{"number 1", "1", true},
		{"number 0", "0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseInsecure(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadWithFile_RealEnvFile(t *testing.T) {
	dir := t.TempDir()
	envFile := dir + "/.env"
	content := "ESXI_URL=https://envfile.example.com\nESXI_USERNAME=envuser\nESXI_PASSWORD=envpass\nESXI_INSECURE=true\n"
	require.NoError(t, os.WriteFile(envFile, []byte(content), 0o644))

	// godotenv.Load does NOT overwrite existing env vars, so we must unset them.
	// t.Setenv saves the original for restore-on-cleanup; os.Unsetenv actually clears them.
	for _, key := range []string{"ESXI_URL", "ESXI_USERNAME", "ESXI_PASSWORD", "ESXI_INSECURE"} {
		t.Setenv(key, "") // save original for cleanup
		_ = os.Unsetenv(key)  // truly remove so godotenv can populate
	}

	cfg, err := LoadWithFile(envFile)
	require.NoError(t, err)
	assert.Equal(t, "https://envfile.example.com", cfg.ESXiURL)
	assert.Equal(t, "envuser", cfg.ESXiUsername)
	assert.True(t, cfg.ESXiInsecure)
}

func TestLoadWithFile_NonExistentFile(t *testing.T) {
	// Should not fail - just proceeds with env vars
	t.Setenv("ESXI_URL", "https://esxi.example.com")
	t.Setenv("ESXI_USERNAME", "admin")
	t.Setenv("ESXI_PASSWORD", "password")

	cfg, err := LoadWithFile("/nonexistent/.env")
	require.NoError(t, err)
	assert.Equal(t, "https://esxi.example.com", cfg.ESXiURL)
}

func TestLoadWithFile_EmptyPath(t *testing.T) {
	t.Setenv("ESXI_URL", "https://esxi.example.com")
	t.Setenv("ESXI_USERNAME", "admin")
	t.Setenv("ESXI_PASSWORD", "password")

	cfg, err := LoadWithFile("")
	require.NoError(t, err)
	assert.Equal(t, "https://esxi.example.com", cfg.ESXiURL)
}

func TestLoadWithFile_GodotenvError(t *testing.T) {
	// A directory path causes godotenv to return a non-IsNotExist error
	dir := t.TempDir()
	_, err := LoadWithFile(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "error loading .env file")
}

func TestValidate_AllFieldsSet(t *testing.T) {
	cfg := &Config{
		ESXiURL:      "https://esxi.example.com",
		ESXiUsername: "admin",
		ESXiPassword: "pass",
	}
	assert.NoError(t, cfg.Validate())
}

func TestValidate_MissingURL(t *testing.T) {
	cfg := &Config{ESXiUsername: "admin", ESXiPassword: "pass"}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESXI_URL is required")
}

func TestValidate_MissingUsername(t *testing.T) {
	cfg := &Config{ESXiURL: "url", ESXiPassword: "pass"}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESXI_USERNAME is required")
}

func TestValidate_MissingPassword(t *testing.T) {
	cfg := &Config{ESXiURL: "url", ESXiUsername: "admin"}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESXI_PASSWORD is required")
}
