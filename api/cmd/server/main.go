package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/config"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/logger"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/service"
	"google.golang.org/api/calendar/v3"
)

type App struct {
	ctx          context.Context
	logger       *logger.Logger
	featureCfg   *service.FeatureConfig
	calendarSvc  *service.CalendarService
	vmwareSvc    *service.VMwareService
	emailSvc     *service.EmailService
	wireguardSvc *service.WireGuardService
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

	pairs := a.selectVMsToRestore(vmList, len(activeEvents))
	if len(pairs) == 0 {
		a.logger.Error("No VMs available in inventory", logger.Action("validation"), logger.Status("no_vms_available"))
		os.Exit(1)
	}

	return a.restoreVMs(pairs, activeEvents)
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

	// Initialize Email service if configured
	smtpHost := getEnvOrDefault("SMTP_HOST", "smtp.gmail.com")
	smtpPort := getEnvOrDefault("SMTP_PORT", "587")
	smtpUsername := os.Getenv("SMTP_USERNAME")

	// Read SMTP password from file if SMTP_PASSWORD_FILE is set (Docker secrets)
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	if smtpPasswordFile := os.Getenv("SMTP_PASSWORD_FILE"); smtpPasswordFile != "" {
		passwordBytes, err := os.ReadFile(smtpPasswordFile)
		if err != nil {
			a.logger.Warn("Failed to read SMTP password file", logger.Error(err), logger.F("file", smtpPasswordFile))
		} else {
			smtpPassword = strings.TrimSpace(string(passwordBytes))
			a.logger.Info("SMTP password loaded from file", logger.F("file", smtpPasswordFile))
		}
	}

	smtpFrom := getEnvOrDefault("SMTP_FROM", smtpUsername)
	testEmailOnly := os.Getenv("TEST_EMAIL_ONLY")

	if smtpUsername != "" && smtpPassword != "" {
		emailSvc, err := service.NewEmailService(smtpHost, smtpPort, smtpUsername, smtpPassword, smtpFrom, testEmailOnly)
		if err != nil {
			a.logger.Warn("Email service not available, emails will not be sent", logger.Error(err))
			a.emailSvc = nil
		} else {
			a.emailSvc = emailSvc
			if testEmailOnly != "" {
				a.logger.Info("Email service initialized (TEST MODE)", logger.Status("ready"), logger.F("TEST_EMAIL", testEmailOnly))
			} else {
				a.logger.Info("Email service initialized", logger.Status("ready"))
			}
		}
	} else {
		a.logger.Info("Email service not configured (SMTP_USERNAME/SMTP_PASSWORD missing)")
		a.emailSvc = nil
	}

	// Initialize WireGuard service if configured
	if featureCfg.WireGuard.Enabled {
		// Load OPNsense API credentials from environment
		if apiKey := os.Getenv("OPNSENSE_API_KEY"); apiKey != "" {
			featureCfg.WireGuard.OPNsenseAPIKey = apiKey
		}
		if apiSecret := os.Getenv("OPNSENSE_API_SECRET"); apiSecret != "" {
			featureCfg.WireGuard.OPNsenseAPISecret = apiSecret
		}

		wireguardSvc := service.NewWireGuardService(&featureCfg.WireGuard)
		if err := wireguardSvc.ValidateConfig(); err != nil {
			a.logger.Warn("WireGuard service configuration invalid", logger.Error(err))
			a.wireguardSvc = nil
		} else {
			a.wireguardSvc = wireguardSvc
			a.logger.Info("WireGuard service initialized", logger.Status("ready"))
		}
	} else {
		a.logger.Info("WireGuard service not enabled in configuration")
		a.wireguardSvc = nil
	}

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

// EventInfo stores information about an active event including participant email
type EventInfo struct {
	Summary string
	Email   string
}

func (a *App) fetchActiveEvents() ([]EventInfo, error) {
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

func (a *App) filterActiveEvents(events []*calendar.Event, now time.Time) []EventInfo {
	var activeEvents []EventInfo

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
			// Extract participant email - try to find first attendee or parse from summary
			email := ""
			if len(event.Attendees) > 0 {
				// Get first attendee's email
				for _, attendee := range event.Attendees {
					if attendee.Email != "" && !attendee.Organizer {
						email = attendee.Email
						break
					}
				}
			}

			// If no attendee found, try to extract email from summary (format: "Name (email@domain.com)")
			if email == "" && event.Summary != "" {
				email = extractEmailFromSummary(event.Summary)
			}

			activeEvents = append(activeEvents, EventInfo{
				Summary: event.Summary,
				Email:   email,
			})

			a.logger.Info("Active event found",
				logger.F("EVENT", event.Summary),
				logger.F("EMAIL", email),
				logger.F("START", startTime.Format(time.RFC3339)),
				logger.F("END", endTime.Format(time.RFC3339)))
		}
	}

	return activeEvents
}

func (a *App) selectVMsToRestore(vmList *models.VMListResponse, eventCount int) []service.UserVMPair {
	inventoryVMs := buildInventoryMap(vmList.VMs)
	pairs := a.selectConfiguredVMs(inventoryVMs, eventCount)

	if len(pairs) < eventCount {
		pairs = a.addFallbackVMs(pairs, vmList.VMs, eventCount)
	}

	if len(pairs) < eventCount {
		a.logger.Warn("Insufficient VMs available",
			logger.Events(eventCount),
			logger.F("AVAILABLE_VMS", len(pairs)),
			logger.F("MESSAGE", "will_restore_all_available"))
	}

	return pairs
}

func (a *App) selectConfiguredVMs(inventoryVMs map[string]bool, eventCount int) []service.UserVMPair {
	var pairs []service.UserVMPair

	for _, p := range a.featureCfg.ESXi.UserVMPairs() {
		if inventoryVMs[p.VM] {
			pairs = append(pairs, p)
			if len(pairs) >= eventCount {
				break
			}
		}
	}

	return pairs
}

func (a *App) addFallbackVMs(existing []service.UserVMPair, inventory []models.VM, eventCount int) []service.UserVMPair {
	pairs := existing
	existingSet := make(map[string]bool)

	for _, p := range existing {
		existingSet[p.VM] = true
	}

	for _, vm := range inventory {
		if !existingSet[vm.Name] {
			pairs = append(pairs, service.UserVMPair{User: "", VM: vm.Name})
			a.logger.Info("Using fallback VM from inventory", logger.VM(vm.Name), logger.Reason("config_vm_not_found"))

			if len(pairs) >= eventCount {
				break
			}
		}
	}

	return pairs
}

func (a *App) restoreVMs(pairs []service.UserVMPair, activeEvents []EventInfo) error {
	eventCount := len(activeEvents)
	snapshotName := "<latest>"
	if a.featureCfg.ESXi.SnapshotName != nil {
		snapshotName = *a.featureCfg.ESXi.SnapshotName
	}

	vmsToRestore := make([]string, len(pairs))
	usersToRotate := make([]string, len(pairs))
	for i, p := range pairs {
		vmsToRestore[i] = p.VM
		usersToRotate[i] = p.User
	}

	a.logger.Info("Starting VM restore",
		logger.Action("restore"),
		logger.Status("starting"),
		logger.Events(eventCount),
		logger.F("VMS_TO_RESTORE", len(vmsToRestore)),
		logger.Snapshot(snapshotName))

	restoreErrors, passwords := a.vmwareSvc.RestoreVMsWithPasswordRotation(a.ctx, vmsToRestore, usersToRotate, snapshotName)

	if len(passwords) > 0 {
		a.logger.Info("Password rotation completed", logger.Action("password_rotation"), logger.Status("completed"))

		// Generate WireGuard configs if service is available
		wireguardConfigs := make(map[string]string)
		if a.wireguardSvc != nil {
			for i, username := range usersToRotate {
				// Rotate WireGuard key along with password
				_, pubKey, err := a.wireguardSvc.RotateUserKey(username)
				if err != nil {
					a.logger.Error("Failed to rotate WireGuard key", logger.User(username), logger.Error(err))
					continue
				}

				// Register peer with OPNsense if auto-registration is enabled
				if err := a.wireguardSvc.RegisterPeerWithOPNsense(username, pubKey, i); err != nil {
					a.logger.Error("Failed to register peer with OPNsense", logger.User(username), logger.Error(err))
					// Continue anyway - user can manually register if needed
				} else {
					a.logger.Info("Peer registered with OPNsense", logger.User(username), logger.F("PUBLIC_KEY", pubKey))
				}

				// Generate config file
				config, err := a.wireguardSvc.GenerateClientConfig(username, i)
				if err != nil {
					a.logger.Error("Failed to generate WireGuard config", logger.User(username), logger.Error(err))
					continue
				}

				wireguardConfigs[username] = config
				a.logger.Info("WireGuard config generated", logger.User(username), logger.F("PUBLIC_KEY", pubKey))
			}
		}

		// Send emails with passwords if email service is available
		for i, username := range usersToRotate {
			if password, ok := passwords[username]; ok {
				a.logger.Info("User password rotated", logger.User(username), logger.Password(password))

				// Try to send email if we have the email service and event email
				if a.emailSvc != nil && i < len(activeEvents) && activeEvents[i].Email != "" {
					vmName := ""
					if i < len(vmsToRestore) {
						vmName = vmsToRestore[i]
					}

					// Prepare WireGuard attachment if available
					var attachment *service.EmailAttachment
					if wgConfig, ok := wireguardConfigs[username]; ok {
						attachment = &service.EmailAttachment{
							Filename: fmt.Sprintf("%s-wireguard.conf", username),
							Content:  []byte(wgConfig),
							MimeType: "application/x-wireguard-profile",
						}
					}

					err := a.emailSvc.SendPasswordEmailWithAttachment(activeEvents[i].Email, vmName, username, password, attachment)
					if err != nil {
						a.logger.Error("Failed to send password email",
							logger.F("EMAIL", activeEvents[i].Email),
							logger.User(username),
							logger.Error(err))
					} else {
						logMsg := "Password email sent"
						if attachment != nil {
							logMsg += " with WireGuard config"
						}
						a.logger.Info(logMsg,
							logger.F("EMAIL", activeEvents[i].Email),
							logger.User(username),
							logger.VM(vmName))
					}
				}
			}
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

func buildInventoryMap(vms []models.VM) map[string]bool {
	inventoryVMs := make(map[string]bool)
	for _, vm := range vms {
		inventoryVMs[vm.Name] = true
	}
	return inventoryVMs
}

// extractEmailFromSummary extracts email from format "Name (email@domain.com)"
func extractEmailFromSummary(summary string) string {
	start := -1
	end := -1

	for i, char := range summary {
		if char == '(' {
			start = i + 1
		} else if char == ')' && start > 0 {
			end = i
			break
		}
	}

	if start > 0 && end > start {
		return summary[start:end]
	}

	return ""
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
