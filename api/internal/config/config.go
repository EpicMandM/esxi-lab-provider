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

// Load loads configuration from environment variables only (no .env file)
func Load() (*Config, error) {
	return LoadWithFile("")
}

// LoadWithFile loads configuration from environment variables and optional .env file
func LoadWithFile(envFile string) (*Config, error) {
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			return nil, fmt.Errorf("failed to load env file: %w", err)
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

func parseInsecure(s string) bool {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return b
}
