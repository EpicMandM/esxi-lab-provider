package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetEnvOrDefault_UsesEnvVar(t *testing.T) {
	t.Setenv("TEST_KEY_XYZ", "from_env")
	assert.Equal(t, "from_env", getEnvOrDefault("TEST_KEY_XYZ", "fallback"))
}

func TestGetEnvOrDefault_UsesDefault(t *testing.T) {
	_ = os.Unsetenv("TEST_KEY_XYZ")
	assert.Equal(t, "fallback", getEnvOrDefault("TEST_KEY_XYZ", "fallback"))
}

func TestGetEnvOrDefault_EmptyEnvUsesDefault(t *testing.T) {
	t.Setenv("TEST_KEY_XYZ", "")
	assert.Equal(t, "fallback", getEnvOrDefault("TEST_KEY_XYZ", "fallback"))
}
