package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	VCenterURL      string
	VCenterUsername string
	VCenterPassword string
	VCenterInsecure bool
}

// Load loads configuration from environment variables only.
func Load() (*Config, error) {
	return LoadWithFile("")
}

// LoadWithFile loads configuration from an optional .env file and environment variables.
func LoadWithFile(envFile string) (*Config, error) {
	// Attempt to load .env file if provided, but don't fail if it doesn't exist.
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("error loading .env file: %w", err)
		}
	}

	cfg := &Config{
		VCenterURL:      os.Getenv("VCENTER_URL"),
		VCenterUsername: os.Getenv("VCENTER_USERNAME"),
		VCenterPassword: os.Getenv("VCENTER_PASSWORD"),
		VCenterInsecure: parseInsecure(os.Getenv("VCENTER_INSECURE")),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if all required fields are set.
func (c *Config) Validate() error {
	if c.VCenterURL == "" {
		return fmt.Errorf("VCENTER_URL is required")
	}
	if c.VCenterUsername == "" {
		return fmt.Errorf("VCENTER_USERNAME is required")
	}
	if c.VCenterPassword == "" {
		return fmt.Errorf("VCENTER_PASSWORD is required")
	}
	return nil
}

// parseInsecure converts a string to a boolean, defaulting to false.
func parseInsecure(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}
