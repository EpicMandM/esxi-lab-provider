package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	// Base valid environment
	validEnv := map[string]string{
		"VCENTER_URL":      "https://vcenter.example.com",
		"VCENTER_USERNAME": "admin",
		"VCENTER_PASSWORD": "password",
	}

	t.Run("valid config", func(t *testing.T) {
		for k, v := range validEnv {
			t.Setenv(k, v)
		}
		t.Setenv("VCENTER_INSECURE", "true")

		cfg, err := Load()
		require.NoError(t, err)
		assert.Equal(t, "https://vcenter.example.com", cfg.VCenterURL)
		assert.True(t, cfg.VCenterInsecure)
	})

	t.Run("insecure defaults to false when empty", func(t *testing.T) {
		for k, v := range validEnv {
			t.Setenv(k, v)
		}
		t.Setenv("VCENTER_INSECURE", "")
		cfg, err := Load()
		require.NoError(t, err)
		assert.False(t, cfg.VCenterInsecure, "VCENTER_INSECURE should default to false")
	})

	// Table-driven test for missing variables
	missingVarTests := []struct {
		name    string
		unset   string // The env var to leave unset
		wantErr string
	}{
		{
			name:    "missing VCENTER_URL",
			unset:   "VCENTER_URL",
			wantErr: "VCENTER_URL is required",
		},
		{
			name:    "missing VCENTER_USERNAME",
			unset:   "VCENTER_USERNAME",
			wantErr: "VCENTER_USERNAME is required",
		},
		{
			name:    "missing VCENTER_PASSWORD",
			unset:   "VCENTER_PASSWORD",
			wantErr: "VCENTER_PASSWORD is required",
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
