package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/config"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type VMwareService struct {
	client *govmomi.Client
	finder *find.Finder
	logger *log.Logger
}

func NewVMwareService(ctx context.Context, cfg *config.Config, logger *log.Logger) (*VMwareService, error) {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	u, err := soap.ParseURL(cfg.VCenterURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	u.User = url.UserPassword(cfg.VCenterUsername, cfg.VCenterPassword)

	client, err := govmomi.NewClient(ctx, u, cfg.VCenterInsecure)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	finder := find.NewFinder(client.Client, true)
	dc, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get datacenter: %w", err)
	}
	finder.SetDatacenter(dc)

	return &VMwareService{
		client: client,
		finder: finder,
		logger: logger,
	}, nil
}
func (s *VMwareService) GetFinder() *find.Finder {
	return s.finder
}
func (s *VMwareService) GetClient() *govmomi.Client {
	return s.client
}
func (s *VMwareService) Close(ctx context.Context) error {
	if s.client == nil {
		return s.client.Logout(ctx)
	}
	return nil
}

func (s *VMwareService) ListAllVMs(ctx context.Context) ([]*models.VM, error) {
	if s == nil {
		return nil, fmt.Errorf("service not initialized")
	}
	finder := s.GetFinder()
	vms, err := finder.VirtualMachineList(ctx, "*")
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual machines: %w", err)
	}
	var vmList []*models.VM
	for _, vm := range vms {
		vmList = append(vmList, &models.VM{
			Name: vm.Name(),
		})
	}
	return vmList, nil
}
func (s *VMwareService) ListVMSnapshots(ctx context.Context) (*models.VMListResponse, error) {
	if s == nil {
		return nil, fmt.Errorf("service not initialized")
	}
	client := s.GetClient()
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
			s.logger.Printf("Warning: Failed to get properties for %s: %v", vm.Name(), err)
			continue
		}
		vmData := models.VM{
			Name:      vm.Name(),
			Snapshots: []models.VMSnapshot{},
		}
		if mvm.Snapshot != nil {
			vmData.Snapshots = s.extractSnapshots(mvm.Snapshot.RootSnapshotList)
		}

		response.VMs = append(response.VMs, vmData)
	}

	response.TotalVMs = len(response.VMs)
	return response, nil
}

func (s *VMwareService) extractSnapshots(snapshots []types.VirtualMachineSnapshotTree) []models.VMSnapshot {
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
			result = append(result, s.extractSnapshots(snapshot.ChildSnapshotList)...)
		}
	}
	return result
}
