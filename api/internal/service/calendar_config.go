package service

import (
	"fmt"
	"os"
	"sort"

	"github.com/BurntSushi/toml"
)

type CalendarConfig struct {
	CalendarID         string `toml:"calendar_id"`
	ServiceAccountPath string `toml:"service_account_path"`
}

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

type UserVMPair struct {
	User string
	VMs  []string
}

// UserVMPairs returns user-VM pairs sorted by username.
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

// VMs returns all VM prefixes from user-VM mappings.
func (c *ESXiConfig) VMs() []string {
	pairs := c.UserVMPairs()
	var vms []string
	for _, p := range pairs {
		vms = append(vms, p.VMs...)
	}
	return vms
}

// Users returns sorted usernames from user-VM mappings.
func (c *ESXiConfig) Users() []string {
	pairs := c.UserVMPairs()
	users := make([]string, len(pairs))
	for i, p := range pairs {
		users[i] = p.User
	}
	return users
}

func LoadFeatureConfig(path string) (*FeatureConfig, error) {
	var cfg FeatureConfig
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return nil, fmt.Errorf("failed to load feature config: %w", err)
	}
	return &cfg, nil
}

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
