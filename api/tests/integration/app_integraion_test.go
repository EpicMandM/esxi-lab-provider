package integration

import (
	"context"
	"os"
	"testing"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/app"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/config"
	"github.com/stretchr/testify/require"
)

// TestApp_FullFlow tests the complete application flow
// This requires a real vCenter connection, so it's skipped by default
func TestApp_FullFlow(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	// Load real config from environment
	cfg, err := config.Load()
	require.NoError(t, err)

	// Create app
	application := app.New(cfg, nil, os.Stdout)

	ctx := context.Background()

	// Initialize
	err = application.Initialize(ctx)
	require.NoError(t, err)

	// List VMs
	data, err := application.ListVMSnapshots(ctx)
	require.NoError(t, err)
	require.NotNil(t, data)

	// Close
	err = application.Close(ctx)
	require.NoError(t, err)
}
