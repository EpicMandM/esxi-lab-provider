package service

import (
	"testing"
	"time"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware/govmomi/vim25/types"
)

// --- findSnapshotInTree tests ---

func TestFindSnapshotInTree_Found(t *testing.T) {
	ref := types.ManagedObjectReference{Type: "Snapshot", Value: "snap-1"}
	tree := []types.VirtualMachineSnapshotTree{
		{Name: "snap-a", Snapshot: ref},
	}

	got := findSnapshotInTree(tree, "snap-a")
	require.NotNil(t, got)
	assert.Equal(t, "snap-1", got.Value)
}

func TestFindSnapshotInTree_NotFound(t *testing.T) {
	tree := []types.VirtualMachineSnapshotTree{
		{Name: "snap-a", Snapshot: types.ManagedObjectReference{Value: "snap-1"}},
	}

	got := findSnapshotInTree(tree, "nonexistent")
	assert.Nil(t, got)
}

func TestFindSnapshotInTree_NestedChild(t *testing.T) {
	childRef := types.ManagedObjectReference{Type: "Snapshot", Value: "snap-child"}
	tree := []types.VirtualMachineSnapshotTree{
		{
			Name:     "parent",
			Snapshot: types.ManagedObjectReference{Value: "snap-parent"},
			ChildSnapshotList: []types.VirtualMachineSnapshotTree{
				{Name: "child", Snapshot: childRef},
			},
		},
	}

	got := findSnapshotInTree(tree, "child")
	require.NotNil(t, got)
	assert.Equal(t, "snap-child", got.Value)
}

func TestFindSnapshotInTree_EmptyTree(t *testing.T) {
	got := findSnapshotInTree(nil, "anything")
	assert.Nil(t, got)
}

func TestFindSnapshotInTree_DeepNesting(t *testing.T) {
	deepRef := types.ManagedObjectReference{Value: "snap-deep"}
	tree := []types.VirtualMachineSnapshotTree{
		{
			Name:     "level1",
			Snapshot: types.ManagedObjectReference{Value: "snap-l1"},
			ChildSnapshotList: []types.VirtualMachineSnapshotTree{
				{
					Name:     "level2",
					Snapshot: types.ManagedObjectReference{Value: "snap-l2"},
					ChildSnapshotList: []types.VirtualMachineSnapshotTree{
						{Name: "level3", Snapshot: deepRef},
					},
				},
			},
		},
	}

	got := findSnapshotInTree(tree, "level3")
	require.NotNil(t, got)
	assert.Equal(t, "snap-deep", got.Value)
}

// --- findLatestSnapshotInTree tests ---

func TestFindLatestSnapshotInTree_SingleSnapshot(t *testing.T) {
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ref := types.ManagedObjectReference{Value: "snap-1"}
	tree := []types.VirtualMachineSnapshotTree{
		{Name: "only", Snapshot: ref, CreateTime: ts},
	}

	got := findLatestSnapshotInTree(tree)
	require.NotNil(t, got)
	assert.Equal(t, "snap-1", got.Value)
}

func TestFindLatestSnapshotInTree_MultipleSnapshots(t *testing.T) {
	old := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	tree := []types.VirtualMachineSnapshotTree{
		{Name: "old", Snapshot: types.ManagedObjectReference{Value: "snap-old"}, CreateTime: old},
		{Name: "newer", Snapshot: types.ManagedObjectReference{Value: "snap-newer"}, CreateTime: newer},
	}

	got := findLatestSnapshotInTree(tree)
	require.NotNil(t, got)
	assert.Equal(t, "snap-newer", got.Value)
}

func TestFindLatestSnapshotInTree_ChildNewer(t *testing.T) {
	parentTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	childTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	tree := []types.VirtualMachineSnapshotTree{
		{
			Name:       "parent",
			Snapshot:   types.ManagedObjectReference{Value: "snap-parent"},
			CreateTime: parentTime,
			ChildSnapshotList: []types.VirtualMachineSnapshotTree{
				{Name: "child", Snapshot: types.ManagedObjectReference{Value: "snap-child"}, CreateTime: childTime},
			},
		},
	}

	got := findLatestSnapshotInTree(tree)
	require.NotNil(t, got)
	assert.Equal(t, "snap-child", got.Value)
}

func TestFindLatestSnapshotInTree_Empty(t *testing.T) {
	got := findLatestSnapshotInTree(nil)
	assert.Nil(t, got)
}

func TestFindLatestSnapshotInTree_ParentNewerThanChild(t *testing.T) {
	parentTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	childTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	tree := []types.VirtualMachineSnapshotTree{
		{
			Name:       "parent",
			Snapshot:   types.ManagedObjectReference{Value: "snap-parent"},
			CreateTime: parentTime,
			ChildSnapshotList: []types.VirtualMachineSnapshotTree{
				{Name: "child", Snapshot: types.ManagedObjectReference{Value: "snap-child"}, CreateTime: childTime},
			},
		},
	}

	// The function recurses into the latest root's children.
	// The child snapshot is returned because the function always recurses into
	// the latest root's child list, even if the child is older.
	got := findLatestSnapshotInTree(tree)
	require.NotNil(t, got)
	assert.Equal(t, "snap-child", got.Value)
}

// --- extractSnapshots tests ---

func TestExtractSnapshots_Flat(t *testing.T) {
	ts := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	tree := []types.VirtualMachineSnapshotTree{
		{Name: "snap1", Description: "first", CreateTime: ts, State: "poweredOff", Quiesced: false},
		{Name: "snap2", Description: "second", CreateTime: ts, State: "poweredOn", Quiesced: true},
	}

	svc := &VMwareService{}
	result := svc.extractSnapshots(tree)

	assert.Len(t, result, 2)
	assert.Equal(t, "snap1", result[0].Name)
	assert.Equal(t, "snap2", result[1].Name)
	assert.True(t, result[1].Quiesced)
}

func TestExtractSnapshots_WithChildren(t *testing.T) {
	ts := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	tree := []types.VirtualMachineSnapshotTree{
		{
			Name:        "parent",
			Description: "parent snap",
			CreateTime:  ts,
			State:       "poweredOn",
			ChildSnapshotList: []types.VirtualMachineSnapshotTree{
				{Name: "child", Description: "child snap", CreateTime: ts, State: "poweredOff"},
			},
		},
	}

	svc := &VMwareService{}
	result := svc.extractSnapshots(tree)

	assert.Len(t, result, 2)
	assert.Equal(t, "parent", result[0].Name)
	assert.Equal(t, "child", result[1].Name)
}

func TestExtractSnapshots_Empty(t *testing.T) {
	svc := &VMwareService{}
	result := svc.extractSnapshots(nil)
	assert.Nil(t, result)
}

func TestExtractSnapshots_FieldMapping(t *testing.T) {
	ts := time.Date(2025, 3, 15, 12, 30, 0, 0, time.UTC)
	tree := []types.VirtualMachineSnapshotTree{
		{Name: "test", Description: "desc", CreateTime: ts, State: "poweredOff", Quiesced: true},
	}

	svc := &VMwareService{}
	result := svc.extractSnapshots(tree)
	require.Len(t, result, 1)

	expected := models.VMSnapshot{
		Name:        "test",
		Description: "desc",
		Created:     ts,
		State:       "poweredOff",
		Quiesced:    true,
	}
	assert.Equal(t, expected, result[0])
}
