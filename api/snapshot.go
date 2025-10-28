package main

import (
	"context"
	"text/tabwriter"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

// Print writes snapshot information to a tabwriter
func printSnapshots(tw *tabwriter.Writer, vmName string, snapshots []types.VirtualMachineSnapshotTree) {
	for _, snapshot := range snapshots {
		if _, err := tw.Write([]byte(vmName + "\t" + snapshot.Name + "\t" + snapshot.Description + "\t" + snapshot.CreateTime.Format("2006-01-02 15:04:05") + "\n")); err != nil {
			// Log error but continue processing other snapshots
			continue
		}

		if len(snapshot.ChildSnapshotList) > 0 {
			printSnapshots(tw, vmName, snapshot.ChildSnapshotList)
		}
	}
}

// Revert reverts a VM to a specific snapshot or current snapshot
func revertSnapshot(ctx context.Context, vm *object.VirtualMachine, snapshotName string, suppressPowerOn bool) error {
	var task *object.Task
	var err error

	if snapshotName != "" {
		task, err = vm.RevertToSnapshot(ctx, snapshotName, suppressPowerOn)
	} else {
		task, err = vm.RevertToCurrentSnapshot(ctx, suppressPowerOn)
	}

	if err != nil {
		return err
	}

	return task.Wait(ctx)
}
