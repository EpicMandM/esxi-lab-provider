package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVMSnapshot_JSONRoundTrip(t *testing.T) {
	ts := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	snap := VMSnapshot{
		Name:        "clean-state",
		Description: "A clean snapshot",
		Created:     ts,
		State:       "poweredOff",
		Quiesced:    true,
	}

	data, err := json.Marshal(snap)
	require.NoError(t, err)

	var got VMSnapshot
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, snap, got)
}

func TestVM_JSONRoundTrip(t *testing.T) {
	vm := VM{
		Name: "test-vm",
		Snapshots: []VMSnapshot{
			{Name: "s1", State: "poweredOn"},
		},
	}

	data, err := json.Marshal(vm)
	require.NoError(t, err)

	var got VM
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, vm.Name, got.Name)
	assert.Len(t, got.Snapshots, 1)
	assert.Equal(t, "s1", got.Snapshots[0].Name)
}

func TestVMListResponse_JSONRoundTrip(t *testing.T) {
	resp := VMListResponse{
		ESXiName: "esxi-host.local",
		TotalVMs: 2,
		VMs: []VM{
			{Name: "vm1"},
			{Name: "vm2"},
		},
	}

	data, err := json.Marshal(resp)
	require.NoError(t, err)

	var got VMListResponse
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, resp.ESXiName, got.ESXiName)
	assert.Equal(t, resp.TotalVMs, got.TotalVMs)
	assert.Len(t, got.VMs, 2)
}

func TestVM_EmptySnapshots(t *testing.T) {
	vm := VM{Name: "empty-vm"}
	assert.Nil(t, vm.Snapshots)

	data, err := json.Marshal(vm)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"name":"empty-vm"`)
}
