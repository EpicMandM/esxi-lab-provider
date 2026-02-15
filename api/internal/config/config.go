package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ESXiURL      string
	ESXiUsername string
	ESXiPassword string
	ESXiInsecure bool
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
		ESXiURL:      os.Getenv("ESXI_URL"),
		ESXiUsername: os.Getenv("ESXI_USERNAME"),
		ESXiPassword: os.Getenv("ESXI_PASSWORD"),
		ESXiInsecure: parseInsecure(os.Getenv("ESXI_INSECURE")),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if all required fields are set.
func (c *Config) Validate() error {
	if c.ESXiURL == "" {
		return fmt.Errorf("ESXI_URL is required")
	}
	if c.ESXiUsername == "" {
		return fmt.Errorf("ESXI_USERNAME is required")
	}
	if c.ESXiPassword == "" {
		return fmt.Errorf("ESXI_PASSWORD is required")
	}
	return nil
}

// parseInsecure converts a string to a boolean, defaulting to false.
func parseInsecure(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}
