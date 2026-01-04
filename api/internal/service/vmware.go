package service

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/config"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/logger"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type VMwareService struct {
	client *govmomi.Client
	finder *find.Finder
	logger *logger.Logger
}

func NewVMwareService(ctx context.Context, cfg *config.Config, log *logger.Logger) (*VMwareService, error) {
	if log == nil {
		log = logger.NewWithWriter(io.Discard)
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
		logger: log,
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
			s.logger.Warn("Failed to get VM properties", logger.VM(vm.Name()), logger.Error(err))
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

// RestoreVMsForEvents restores multiple VMs to a snapshot based on event count
func (s *VMwareService) RestoreVMsForEvents(ctx context.Context, vmNames []string, snapshotName string, eventCount int) error {
	if eventCount <= 0 {
		return nil
	}

	// Limit to available VMs
	vmCount := eventCount
	if vmCount > len(vmNames) {
		vmCount = len(vmNames)
	}

	s.logger.Info("Restoring VMs", logger.Count(vmCount), logger.Snapshot(snapshotName))

	for i := 0; i < vmCount; i++ {
		vmName := vmNames[i]
		if err := s.restoreVM(ctx, vmName, snapshotName); err != nil {
			s.logger.Error("VM restore failed", logger.VM(vmName), logger.Error(err))
			continue
		}
		s.logger.Info("VM restored successfully", logger.VM(vmName))
	}

	return nil
}

// RestoreVMsForEventsWithErrorHandling restores VMs and returns errors for each failed VM
func (s *VMwareService) RestoreVMsForEventsWithErrorHandling(ctx context.Context, vmNames []string, snapshotName string) []string {
	var errors []string

	for _, vmName := range vmNames {
		if err := s.restoreVM(ctx, vmName, snapshotName); err != nil {
			errors = append(errors, fmt.Sprintf("failed to restore %s: %v", vmName, err))
			s.logger.Error("VM restore failed", logger.Action("vm_restore"), logger.Status("failed"), logger.VM(vmName), logger.Error(err))
			continue
		}
		s.logger.Info("VM restore successful", logger.Action("vm_restore"), logger.Status("success"), logger.VM(vmName))
	}

	return errors
}

// RestoreVMsWithPasswordRotation restores VMs and rotates ESXi user passwords
func (s *VMwareService) RestoreVMsWithPasswordRotation(ctx context.Context, vmNames []string, userNames []string, snapshotName string) ([]string, map[string]string) {
	var errors []string
	passwords := make(map[string]string)

	// Get ESXi host
	host, err := s.finder.DefaultHostSystem(ctx)
	if err != nil {
		errors = append(errors, fmt.Sprintf("failed to get ESXi host: %v", err))
		return errors, passwords
	}

	for i, vmName := range vmNames {
		// Restore VM snapshot
		if err := s.restoreVM(ctx, vmName, snapshotName); err != nil {
			errors = append(errors, fmt.Sprintf("failed to restore %s: %v", vmName, err))
			s.logger.Error("VM restore failed", logger.Action("vm_restore"), logger.Status("failed"), logger.VM(vmName), logger.Error(err))
			continue
		}
		s.logger.Info("VM restore successful", logger.Action("vm_restore"), logger.Status("success"), logger.VM(vmName))

		// Rotate password for corresponding user if available
		if i < len(userNames) {
			username := userNames[i]
			newPassword, err := s.RotateESXiUserPassword(ctx, host, username)
			if err != nil {
				s.logger.Error("Password rotation failed", logger.Action("password_rotate"), logger.Status("failed"), logger.User(username), logger.Error(err))
				errors = append(errors, fmt.Sprintf("failed to rotate password for user %s: %v", username, err))
			} else {
				passwords[username] = newPassword
				s.logger.Info("Password rotation successful", logger.Action("password_rotate"), logger.Status("success"), logger.User(username), logger.VM(vmName))
			}
		}
	}

	return errors, passwords
}

func (s *VMwareService) restoreVM(ctx context.Context, vmName, snapshotName string) error {
	vm, err := s.finder.VirtualMachine(ctx, vmName)
	if err != nil {
		return fmt.Errorf("VM not found: %w", err)
	}

	var snapshot *types.ManagedObjectReference

	// If snapshot name is empty or "<latest>", use the latest snapshot
	if snapshotName == "" || snapshotName == "<latest>" {
		latest, err := s.findLatestSnapshot(ctx, vm)
		if err != nil {
			return fmt.Errorf("no snapshots found: %w", err)
		}
		snapshot = latest
	} else {
		// Use specific snapshot name
		found, err := s.findSnapshot(ctx, vm, snapshotName)
		if err != nil {
			return fmt.Errorf("snapshot not found: %w", err)
		}
		snapshot = found
	}

	task, err := vm.RevertToSnapshot(ctx, snapshot.Reference().Value, true)
	if err != nil {
		return fmt.Errorf("failed to revert: %w", err)
	}

	if err := task.Wait(ctx); err != nil {
		return fmt.Errorf("revert task failed: %w", err)
	}

	// Power on VM after restore
	task, err = vm.PowerOn(ctx)
	if err != nil {
		return fmt.Errorf("failed to power on: %w", err)
	}

	return task.Wait(ctx)
}

func (s *VMwareService) findSnapshot(ctx context.Context, vm *object.VirtualMachine, name string) (*types.ManagedObjectReference, error) {
	var mvm mo.VirtualMachine

	err := vm.Properties(ctx, vm.Reference(), []string{"snapshot"}, &mvm)
	if err != nil {
		return nil, err
	}
	if mvm.Snapshot == nil || len(mvm.Snapshot.RootSnapshotList) == 0 {
		return nil, fmt.Errorf("no snapshots found")
	}

	ref := findSnapshotInTree(mvm.Snapshot.RootSnapshotList, name)
	if ref == nil {
		return nil, fmt.Errorf("snapshot '%s' not found", name)
	}

	return ref, nil
}

// findLatestSnapshot returns the most recently created snapshot
func (s *VMwareService) findLatestSnapshot(ctx context.Context, vm *object.VirtualMachine) (*types.ManagedObjectReference, error) {
	var mvm mo.VirtualMachine

	err := vm.Properties(ctx, vm.Reference(), []string{"snapshot"}, &mvm)
	if err != nil {
		return nil, err
	}
	if mvm.Snapshot == nil || len(mvm.Snapshot.RootSnapshotList) == 0 {
		return nil, fmt.Errorf("no snapshots found")
	}

	latest := findLatestSnapshotInTree(mvm.Snapshot.RootSnapshotList)
	if latest == nil {
		return nil, fmt.Errorf("no snapshots found")
	}

	return latest, nil
}

func findSnapshotInTree(tree []types.VirtualMachineSnapshotTree, name string) *types.ManagedObjectReference {
	for _, snapshot := range tree {
		if snapshot.Name == name {
			return &snapshot.Snapshot
		}
		if ref := findSnapshotInTree(snapshot.ChildSnapshotList, name); ref != nil {
			return ref
		}
	}
	return nil
}

// findLatestSnapshotInTree finds the most recently created snapshot in the tree
func findLatestSnapshotInTree(tree []types.VirtualMachineSnapshotTree) *types.ManagedObjectReference {
	if len(tree) == 0 {
		return nil
	}

	var latest *types.VirtualMachineSnapshotTree
	for i := range tree {
		if latest == nil || tree[i].CreateTime.After(latest.CreateTime) {
			latest = &tree[i]
		}
	}

	// Check child snapshots recursively
	if latestChild := findLatestSnapshotInTree(latest.ChildSnapshotList); latestChild != nil {
		return latestChild
	}

	return &latest.Snapshot
}

// RotateESXiUserPassword rotates the password for an ESXi local user
func (s *VMwareService) RotateESXiUserPassword(ctx context.Context, host *object.HostSystem, username string) (string, error) {
	newPassword, err := GeneratePassword(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate password: %w", err)
	}

	// Get the HostLocalAccountManager reference from the host
	accountMgr := types.ManagedObjectReference{
		Type:  "HostLocalAccountManager",
		Value: "ha-localacctmgr",
	}

	spec := types.HostAccountSpec{
		Id:          username,
		Password:    newPassword,
		Description: "Password rotated by automation",
	}

	req := types.UpdateUser{
		This: accountMgr,
		User: &spec,
	}

	_, err = methods.UpdateUser(ctx, host.Client(), &req)
	if err != nil {
		return "", fmt.Errorf("failed to update user password: %w", err)
	}

	return newPassword, nil
}
