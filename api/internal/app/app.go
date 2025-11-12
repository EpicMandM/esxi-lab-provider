package app

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/config"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/service"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

type App struct {
	config  *config.Config
	service *service.VMwareService
	logger  *log.Logger
	output  io.Writer
}

func New(cfg *config.Config, logger *log.Logger, output io.Writer) *App {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	if output == nil {
		output = io.Discard
	}
	return &App{
		config: cfg,
		logger: logger,
		output: output,
	}
}

func (a *App) Initialize(ctx context.Context) error {
	vmwareService, err := service.NewVMwareService(ctx, a.config.VCenterURL, a.config.VCenterUsername, a.config.VCenterPassword, a.config.VCenterInsecure)
	if err != nil {
		return fmt.Errorf("failed to connect to vCenter: %w", err)
	}
	a.service = vmwareService
	a.logger.Printf("Connected to vCenter: %s", a.config.VCenterURL)
	return nil
}

func (a *App) ListVMSnapshots(ctx context.Context) (*models.VMListResponse, error) {
	if a.service == nil {
		return nil, fmt.Errorf("service not initialized")
	}
	client := a.service.GetClient()

	finder := find.NewFinder(client.Client)
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get default datacenter: %w", err)
	}
	finder.SetDatacenter(dc)

	vms, err := finder.VirtualMachineList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual machines: %w", err)
	}

	response := &models.VMListResponse{
		VCenterName: client.ServiceContent.About.FullName,
		VMs:         make([]models.VM, 0, len(vms)),
	}

	for _, vm := range vms {
		var mvm mo.VirtualMachine
		if err := vm.Properties(ctx, vm.Reference(), []string{"snapshot"}, &mvm); err != nil {
			a.logger.Printf("Warning: Failed to get properties for %s: %v", vm.Name(), err)
			continue
		}
		vmData := models.VM{
			Name:      vm.Name(),
			Snapshots: []models.VMSnapshot{},
		}
		if mvm.Snapshot != nil {
			vmData.Snapshots = a.extractSnapshots(mvm.Snapshot.RootSnapshotList)
		}

		response.VMs = append(response.VMs, vmData)
	}

	response.TotalVMs = len(response.VMs)
	return response, nil
}

func (a *App) extractSnapshots(snapshots []types.VirtualMachineSnapshotTree) []models.VMSnapshot {
	var result []models.VMSnapshot
	for _, snapshot := range snapshots {
		result = append(result, models.VMSnapshot{
			Name:        snapshot.Name,
			Description: snapshot.Description,
			Created:     snapshot.CreateTime,
			State:       string(snapshot.State),
			Quiesced:    snapshot.Quiesced,
		})
		if len(snapshot.ChildSnapshotList) > 0 {
			result = append(result, a.extractSnapshots(snapshot.ChildSnapshotList)...)
		}
	}
	return result
}

func (a *App) Close(ctx context.Context) error {
	if a.service == nil {
		return nil
	}
	if err := a.service.Close(ctx); err != nil {
		return fmt.Errorf("failed to close VMware service: %w", err)
	}
	a.logger.Println("Disconnected from vCenter")
	return nil
}
