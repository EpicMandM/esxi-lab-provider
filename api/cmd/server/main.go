package main

import (
	"context"
	"os"
	"time"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/config"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/logger"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/service"
	"google.golang.org/api/calendar/v3"
)

type App struct {
	ctx         context.Context
	logger      *logger.Logger
	featureCfg  *service.FeatureConfig
	calendarSvc *service.CalendarService
	vmwareSvc   *service.VMwareService
}

func main() {
	app := &App{
		ctx:    context.Background(),
		logger: logger.New(),
	}

	if err := app.run(); err != nil {
		app.logger.Error("Application error", logger.Error(err))
		os.Exit(1)
	}
}

func (a *App) run() error {
	if err := a.initialize(); err != nil {
		return err
	}
	defer func() {
		if err := a.vmwareSvc.Close(a.ctx); err != nil {
			a.logger.Error("Failed to close VMware service", logger.Error(err))
		}
	}()

	vmList, err := a.fetchVMInventory()
	if err != nil {
		return err
	}

	activeEvents, err := a.fetchActiveEvents()
	if err != nil {
		return err
	}

	if len(activeEvents) == 0 {
		a.logger.Info("No active calendar events", logger.Action("calendar"), logger.Status("no_active_events"))
		return nil
	}

	vmsToRestore := a.selectVMsToRestore(vmList, len(activeEvents))
	if len(vmsToRestore) == 0 {
		a.logger.Error("No VMs available in inventory", logger.Action("validation"), logger.Status("no_vms_available"))
		os.Exit(1)
	}

	return a.restoreVMs(vmsToRestore, len(activeEvents))
}

func (a *App) initialize() error {
	configPath := getEnvOrDefault("CONFIG_PATH", "./data/user_config.toml")
	featureCfg, err := service.LoadFeatureConfig(configPath)
	if err != nil {
		a.logger.Error("Failed to load feature config", logger.Error(err), logger.F("path", configPath))
		return err
	}
	a.featureCfg = featureCfg

	envPath := ".env"
	infraCfg, err := config.LoadWithFile(envPath)
	if err != nil {
		a.logger.Error("Failed to load infrastructure config", logger.Error(err), logger.F("path", envPath))
		return err
	}

	calendarSvc, err := service.NewCalendarService(a.ctx, featureCfg.Calendar)
	if err != nil {
		a.logger.Error("Failed to initialize calendar service", logger.Error(err))
		return err
	}
	a.calendarSvc = calendarSvc

	vmwareSvc, err := service.NewVMwareService(a.ctx, infraCfg, a.logger)
	if err != nil {
		a.logger.Error("Failed to initialize VMware service", logger.Error(err))
		return err
	}
	a.vmwareSvc = vmwareSvc

	return nil
}

func (a *App) fetchVMInventory() (*models.VMListResponse, error) {
	a.logger.Info("Fetching VM inventory", logger.Action("startup"), logger.Status("fetching_vms"))

	vmList, err := a.vmwareSvc.ListVMSnapshots(a.ctx)
	if err != nil {
		a.logger.Error("Failed to fetch VMs", logger.Error(err))
		return nil, err
	}

	a.logger.Info("VM inventory fetched", logger.Action("startup"), logger.Status("vm_inventory"), logger.Count(len(vmList.VMs)))
	a.logVMInventory(vmList.VMs)

	return vmList, nil
}

func (a *App) logVMInventory(vms []models.VM) {
	for _, vm := range vms {
		a.logger.Info("VM found", logger.VM(vm.Name), logger.F("SNAPSHOT_COUNT", len(vm.Snapshots)))
		for _, snapshot := range vm.Snapshots {
			a.logger.Info("Snapshot details",
				logger.Snapshot(snapshot.Name),
				logger.F("STATE", snapshot.State),
				logger.F("CREATED", snapshot.Created.Format("2006-01-02 15:04:05")))
		}
	}
}

func (a *App) fetchActiveEvents() ([]string, error) {
	now := time.Now()
	timeMin := now.Add(-6 * time.Minute).Format(time.RFC3339)
	timeMax := now.Add(6 * time.Minute).Format(time.RFC3339)

	a.logger.Info("Fetching calendar events", logger.Action("calendar"), logger.Status("fetching_events"), logger.TimeWindow("±6min"))

	events, err := a.calendarSvc.ListEvents(timeMin, timeMax)
	if err != nil {
		a.logger.Error("Failed to fetch calendar events", logger.Error(err))
		return nil, err
	}

	activeEvents := a.filterActiveEvents(events, now)
	return activeEvents, nil
}

func (a *App) filterActiveEvents(events []*calendar.Event, now time.Time) []string {
	var activeEvents []string

	for _, event := range events {
		if event.Start.DateTime == "" || event.End.DateTime == "" {
			continue
		}

		startTime, err := time.Parse(time.RFC3339, event.Start.DateTime)
		if err != nil {
			continue
		}

		endTime, err := time.Parse(time.RFC3339, event.End.DateTime)
		if err != nil {
			continue
		}

		if (startTime.Before(now) || startTime.Equal(now)) && endTime.After(now) {
			activeEvents = append(activeEvents, event.Summary)
			a.logger.Info("Active event found",
				logger.F("EVENT", event.Summary),
				logger.F("START", startTime.Format(time.RFC3339)),
				logger.F("END", endTime.Format(time.RFC3339)))
		}
	}

	return activeEvents
}

func (a *App) selectVMsToRestore(vmList *models.VMListResponse, eventCount int) []string {
	inventoryVMs := buildInventoryMap(vmList.VMs)
	vmsToRestore := a.selectConfiguredVMs(inventoryVMs, eventCount)

	if len(vmsToRestore) < eventCount {
		vmsToRestore = a.addFallbackVMs(vmsToRestore, vmList.VMs, eventCount)
	}

	if len(vmsToRestore) < eventCount {
		a.logger.Warn("Insufficient VMs available",
			logger.Events(eventCount),
			logger.F("AVAILABLE_VMS", len(vmsToRestore)),
			logger.F("MESSAGE", "will_restore_all_available"))
	}

	return vmsToRestore
}

func (a *App) selectConfiguredVMs(inventoryVMs map[string]bool, eventCount int) []string {
	var vmsToRestore []string

	for _, configVM := range a.featureCfg.VSphere.VMs {
		if inventoryVMs[configVM] {
			vmsToRestore = append(vmsToRestore, configVM)
			if len(vmsToRestore) >= eventCount {
				break
			}
		}
	}

	return vmsToRestore
}

func (a *App) addFallbackVMs(existing []string, inventory []models.VM, eventCount int) []string {
	vmsToRestore := existing
	existingSet := make(map[string]bool)

	for _, vm := range existing {
		existingSet[vm] = true
	}

	for _, vm := range inventory {
		if !existingSet[vm.Name] {
			vmsToRestore = append(vmsToRestore, vm.Name)
			a.logger.Info("Using fallback VM from inventory", logger.VM(vm.Name), logger.Reason("config_vm_not_found"))

			if len(vmsToRestore) >= eventCount {
				break
			}
		}
	}

	return vmsToRestore
}

func (a *App) restoreVMs(vmsToRestore []string, eventCount int) error {
	snapshotName := "<latest>"
	if a.featureCfg.VSphere.SnapshotName != nil {
		snapshotName = *a.featureCfg.VSphere.SnapshotName
	}

	a.logger.Info("Starting VM restore",
		logger.Action("restore"),
		logger.Status("starting"),
		logger.Events(eventCount),
		logger.F("VMS_TO_RESTORE", len(vmsToRestore)),
		logger.Snapshot(snapshotName))

	usersToRotate := a.getUsersForVMs(len(vmsToRestore))
	restoreErrors, passwords := a.vmwareSvc.RestoreVMsWithPasswordRotation(a.ctx, vmsToRestore, usersToRotate, snapshotName)

	if len(passwords) > 0 {
		a.logger.Info("Password rotation completed", logger.Action("password_rotation"), logger.Status("completed"))
		for username, password := range passwords {
			a.logger.Info("User password rotated", logger.User(username), logger.Password(password))
		}
	}

	if len(restoreErrors) > 0 {
		for i, errMsg := range restoreErrors {
			a.logger.Error("VM restore failed", logger.VMIndex(i), logger.F("MESSAGE", errMsg))
		}
		a.logger.Error("Restore partially failed",
			logger.Action("restore"),
			logger.Status("partial_failure"),
			logger.Restored(len(vmsToRestore)-len(restoreErrors)),
			logger.Failed(len(restoreErrors)))
		os.Exit(1)
	}

	a.logger.Info("Restore completed successfully",
		logger.Action("restore"),
		logger.Status("success"),
		logger.Events(eventCount),
		logger.F("VMS_RESTORED", len(vmsToRestore)),
		logger.F("PASSWORDS_ROTATED", len(passwords)))
	return nil
}

func (a *App) getUsersForVMs(vmCount int) []string {
	users := make([]string, 0, vmCount)
	for i := 0; i < vmCount && i < len(a.featureCfg.VSphere.Users); i++ {
		users = append(users, a.featureCfg.VSphere.Users[i])
	}
	return users
}

func buildInventoryMap(vms []models.VM) map[string]bool {
	inventoryVMs := make(map[string]bool)
	for _, vm := range vms {
		inventoryVMs[vm.Name] = true
	}
	return inventoryVMs
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
