package service

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// CalendarConfig holds configuration for Google Calendar integration
type CalendarConfig struct {
	CalendarID         string `toml:"calendar_id"`
	ServiceAccountPath string `toml:"service_account_path"`
}

// FeatureConfig holds user-facing feature configurations.
// These are non-sensitive settings that customize application behavior
// and integrations. Users can modify these without redeployment.
// Source: TOML configuration file
type FeatureConfig struct {
	Calendar CalendarConfig `toml:"calendar"`
	VSphere  VSphereConfig  `toml:"vsphere"`
}

type VSphereConfig struct {
	VMs          []string `toml:"vms"`
	Users        []string `toml:"users"`
	SnapshotName *string  `toml:"snapshot_name"` // Optional: if not set, uses latest snapshot
}

// LoadFeatureConfig loads feature configuration from a TOML file
func LoadFeatureConfig(path string) (*FeatureConfig, error) {
	var cfg FeatureConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to load feature config: %w", err)
	}
	return &cfg, nil
}

// LoadServiceAccountToken reads the service account JSON from the configured path
func (c *CalendarConfig) LoadServiceAccountToken() ([]byte, error) {
	if c.ServiceAccountPath == "" {
		return nil, fmt.Errorf("service_account_path is not configured")
	}
	data, err := os.ReadFile(c.ServiceAccountPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account file: %w", err)
	}
	return data, nil
}
