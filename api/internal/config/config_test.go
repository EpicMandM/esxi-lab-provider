package config

import (
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
