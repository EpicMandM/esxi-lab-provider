package app

import (
	"bytes"
	"context"
	"log"
	"testing"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	cfg := &config.Config{
		VCenterURL:      "https://vcenter.example.com",
		VCenterUsername: "admin",
		VCenterPassword: "password",
		VCenterInsecure: true,
	}

	t.Run("with all parameters", func(t *testing.T) {
		output := &bytes.Buffer{}
		logger := log.New(output, "[TEST] ", 0)

		app := New(cfg, logger, output)

		assert.NotNil(t, app)
		assert.Equal(t, cfg, app.config)
		assert.Equal(t, logger, app.logger)
		assert.Equal(t, output, app.output)
	})

	t.Run("with nil logger", func(t *testing.T) {
		output := &bytes.Buffer{}

		app := New(cfg, nil, output)

		assert.NotNil(t, app)
		assert.NotNil(t, app.logger)
		assert.Equal(t, cfg, app.config)
	})

	t.Run("with nil output", func(t *testing.T) {
		logger := log.New(&bytes.Buffer{}, "", 0)

		app := New(cfg, logger, nil)

		assert.NotNil(t, app)
		assert.NotNil(t, app.output)
		assert.Equal(t, cfg, app.config)
	})
}

func TestListVMSnapshots_NotInitialized(t *testing.T) {
	cfg := &config.Config{
		VCenterURL:      "https://vcenter.example.com",
		VCenterUsername: "admin",
		VCenterPassword: "password",
	}
	app := New(cfg, nil, &bytes.Buffer{})

	result, err := app.ListVMSnapshots(context.Background())

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "service not initialized")
}

func TestClose_NotInitialized(t *testing.T) {
	cfg := &config.Config{
		VCenterURL:      "https://vcenter.example.com",
		VCenterUsername: "admin",
		VCenterPassword: "password",
	}
	app := New(cfg, nil, nil)

	err := app.Close(context.Background())

	assert.NoError(t, err)
}
