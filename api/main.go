package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/vmware/govmomi/examples"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/mo"
)

func main() {
	ctx := context.Background()
	client, err := examples.NewClient(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Connected to vCenter Server: %s\n", client.ServiceContent.About.FullName)

	finder := find.NewFinder(client)
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		panic(err)
	}
	finder.SetDatacenter(dc)

	vms, err := finder.VirtualMachineList(ctx, "*")
	if err != nil {
		panic(err)
	}

	tw := tabwriter.NewWriter(os.Stdout, 2, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "VM\tSnapshot\tDescription\tCreated"); err != nil {
		panic(err)
	}

	for _, vm := range vms {
		var mvm mo.VirtualMachine
		err := vm.Properties(ctx, vm.Reference(), []string{"snapshot"}, &mvm)
		if err != nil {
			panic(err)
		}

		if mvm.Snapshot != nil {
			printSnapshots(tw, vm.Name(), mvm.Snapshot.RootSnapshotList)

			if len(mvm.Snapshot.RootSnapshotList) > 0 {
				snapshotName := mvm.Snapshot.RootSnapshotList[0].Name
				fmt.Printf("\nReverting %s to snapshot: %s\n", vm.Name(), snapshotName)

				if err := revertSnapshot(ctx, vm, snapshotName, false); err != nil {
					panic(err)
				}

				fmt.Printf("Successfully reverted %s to snapshot: %s\n", vm.Name(), snapshotName)
			}
		}
	}

	if err := tw.Flush(); err != nil {
		panic(err)
	}
}
