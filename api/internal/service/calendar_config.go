package service

import (
	"fmt"
	"os"
	"sort"

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
	Calendar  CalendarConfig  `toml:"calendar"`
	ESXi      ESXiConfig      `toml:"esxi"`
	WireGuard WireGuardConfig `toml:"wireguard"`
}

type ESXiConfig struct {
	URL            string              `toml:"url"`
	UserVMMappings map[string][]string `toml:"user_vm_mappings"`
	SnapshotName   *string             `toml:"snapshot_name"`
}

// UserVMPair represents a user and all their assigned VMs from the mapping.
type UserVMPair struct {
	User string
	VMs  []string
}

// UserVMPairs returns user-VM pairs in sorted order by username for deterministic iteration.
func (c *ESXiConfig) UserVMPairs() []UserVMPair {
	pairs := make([]UserVMPair, 0, len(c.UserVMMappings))
	for user, vms := range c.UserVMMappings {
		pairs = append(pairs, UserVMPair{User: user, VMs: vms})
	}
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].User < pairs[j].User
	})
	return pairs
}

// VMs returns all VM names from user-VM mappings in sorted user order.
func (c *ESXiConfig) VMs() []string {
	pairs := c.UserVMPairs()
	var vms []string
	for _, p := range pairs {
		vms = append(vms, p.VMs...)
	}
	return vms
}

// Users returns the list of user names from user-VM mappings in sorted order.
func (c *ESXiConfig) Users() []string {
	pairs := c.UserVMPairs()
	users := make([]string, len(pairs))
	for i, p := range pairs {
		users[i] = p.User
	}
	return users
}

// LoadFeatureConfig loads feature configuration from a TOML file
func LoadFeatureConfig(path string) (*FeatureConfig, error) {
	var cfg FeatureConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to load feature config: %w", err)
	}
	return &cfg, nil
}

// LoadServiceAccountToken reads the service account JSON from the configured path.
// The SERVICE_ACCOUNT_PATH environment variable, if set, overrides the TOML value.
func (c *CalendarConfig) LoadServiceAccountToken() ([]byte, error) {
	path := c.ServiceAccountPath
	if envPath := os.Getenv("SERVICE_ACCOUNT_PATH"); envPath != "" {
		path = envPath
	}
	if path == "" {
		return nil, fmt.Errorf("service_account_path is not configured")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account file: %w", err)
	}
	return data, nil
}
