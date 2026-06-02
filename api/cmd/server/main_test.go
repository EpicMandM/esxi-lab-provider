package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestResolveEnvFile_ExplicitPath(t *testing.T) {
	t.Setenv("ENV_PATH", "/custom/.env")
	assert.Equal(t, "/custom/.env", resolveEnvFile())
}

func TestResolveEnvFile_FindsLocalEnv(t *testing.T) {
	t.Setenv("ENV_PATH", "")
	dir := t.TempDir()
	path := dir + "/.env"
	require.NoError(t, os.WriteFile(path, []byte("ESXI_URL=x"), 0o644))

	cwd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	assert.Equal(t, ".env", resolveEnvFile())
}
