package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/logger"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/service"
	"google.golang.org/api/calendar/v3"
)

// EventInfo stores information about an active event including participant email.
type EventInfo struct {
	Summary string
	Email   string
}

// Orchestrator coordinates the VM restore workflow.
type Orchestrator struct {
	Logger     *logger.Logger
	Calendar   service.CalendarClient
	VMware     service.VMwareClient
	Email      service.EmailSender
	WireGuard  service.WireGuardManager
	FeatureCfg *service.FeatureConfig
}

// Run executes the full orchestration: fetch inventory → check calendar →
// restore all VMs → rotate passwords + send emails for active bookings.
// Snapshot revert happens on every inventory host every run, regardless
// of whether a booking exists.
// Returns an error if any critical step fails.
func (o *Orchestrator) Run() error {
	vmList, err := o.FetchVMInventory()
	if err != nil {
		return err
	}
	defer func() {
		if cerr := o.VMware.Close(context.Background()); cerr != nil {
			o.Logger.Error("Failed to close VMware service", logger.Error(cerr))
		}
	}()

	activeEvents, err := o.FetchActiveEvents()
	if err != nil {
		return err
	}

	if len(activeEvents) == 0 {
		o.Logger.Info("No active calendar events", logger.Action("calendar"), logger.Status("no_active_events"))
	}

	pairs := o.SelectAllVMs(vmList)
	if len(pairs) == 0 {
		if len(activeEvents) > 0 {
			return fmt.Errorf("no VMs available in inventory")
		}
		return nil
	}

	return o.RestoreVMs(pairs, activeEvents)
}

// FetchVMInventory fetches the VM snapshot inventory from VMware.
func (o *Orchestrator) FetchVMInventory() (*models.VMListResponse, error) {
	o.Logger.Info("Fetching VM inventory", logger.Action("startup"), logger.Status("fetching_vms"))

	vmList, err := o.VMware.ListVMSnapshots(context.Background())
	if err != nil {
		o.Logger.Error("Failed to fetch VMs", logger.Error(err))
		return nil, err
	}

	o.Logger.Info("VM inventory fetched", logger.Action("startup"), logger.Status("vm_inventory"), logger.Count(len(vmList.VMs)))
	o.LogVMInventory(vmList.VMs)

	return vmList, nil
}

// LogVMInventory logs details about each VM and its snapshots.
func (o *Orchestrator) LogVMInventory(vms []models.VM) {
	for _, vm := range vms {
		o.Logger.Info("VM found", logger.VM(vm.Name), logger.F("SNAPSHOT_COUNT", len(vm.Snapshots)))
		for _, snapshot := range vm.Snapshots {
			o.Logger.Info("Snapshot details",
				logger.Snapshot(snapshot.Name),
				logger.F("STATE", snapshot.State),
				logger.F("CREATED", snapshot.Created.Format("2006-01-02 15:04:05")))
		}
	}
}

// FetchActiveEvents queries the calendar for events active within ±5 minutes
// of the provided time.
func (o *Orchestrator) FetchActiveEvents() ([]EventInfo, error) {
	return o.FetchActiveEventsAt(time.Now())
}

// FetchActiveEventsAt queries the calendar for events active within ±5 minutes
// of the provided time. Exposed for testing with a deterministic clock.
func (o *Orchestrator) FetchActiveEventsAt(now time.Time) ([]EventInfo, error) {
	timeMin := now.Add(-5 * time.Minute).Format(time.RFC3339)
	timeMax := now.Add(5 * time.Minute).Format(time.RFC3339)

	o.Logger.Info("Fetching calendar events", logger.Action("calendar"), logger.Status("fetching_events"), logger.TimeWindow("±5min"))

	events, err := o.Calendar.ListEvents(timeMin, timeMax)
	if err != nil {
		o.Logger.Error("Failed to fetch calendar events", logger.Error(err))
		return nil, err
	}

	activeEvents := FilterActiveEvents(events, now)
	return activeEvents, nil
}

// FilterActiveEvents filters calendar events to only those currently active
// at the given time, and extracts participant email addresses.
func FilterActiveEvents(events []*calendar.Event, now time.Time) []EventInfo {
	var activeEvents []EventInfo

	for _, event := range events {
		if event.Start == nil || event.End == nil {
			continue
		}
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
			email := ""
			if len(event.Attendees) > 0 {
				for _, attendee := range event.Attendees {
					if attendee.Email != "" && !attendee.Organizer {
						email = attendee.Email
						break
					}
				}
			}

			if email == "" && event.Summary != "" {
				email = ExtractEmailFromSummary(event.Summary)
			}

			activeEvents = append(activeEvents, EventInfo{
				Summary: event.Summary,
				Email:   email,
			})
		}
	}

	return activeEvents
}

// SelectVMsToRestore selects which VMs to restore based on configured pairs.
// Only VMs explicitly listed in user_vm_mappings are considered.
func (o *Orchestrator) SelectVMsToRestore(vmList *models.VMListResponse, eventCount int) []service.UserVMPair {
	inventoryVMs := BuildInventoryMap(vmList.VMs)
	pairs := o.SelectConfiguredVMs(inventoryVMs, eventCount)

	if len(pairs) < eventCount {
		o.Logger.Warn("Insufficient VMs available",
			logger.Events(eventCount),
			logger.F("AVAILABLE_VMS", len(pairs)),
			logger.F("MESSAGE", "will_restore_all_available"))
	}

	return pairs
}

// SelectConfiguredVMs picks VMs from the configured user-VM mappings that
// exist in the current inventory.
func (o *Orchestrator) SelectConfiguredVMs(inventoryVMs map[string]bool, eventCount int) []service.UserVMPair {
	var pairs []service.UserVMPair

	for _, p := range o.FeatureCfg.ESXi.UserVMPairs() {
		var validVMs []string
		for _, vm := range p.VMs {
			if inventoryVMs[vm] {
				validVMs = append(validVMs, vm)
			}
		}
		if len(validVMs) > 0 {
			pairs = append(pairs, service.UserVMPair{User: p.User, VMs: validVMs})
			if len(pairs) >= eventCount {
				break
			}
		}
	}

	return pairs
}

// RestoreVMs restores VMs, rotates passwords, generates WireGuard configs,
// and sends notification emails.
func (o *Orchestrator) RestoreVMs(pairs []service.UserVMPair, activeEvents []EventInfo) error {
	eventCount := len(activeEvents)
	snapshotName := "<latest>"
	if o.FeatureCfg.ESXi.SnapshotName != nil {
		snapshotName = *o.FeatureCfg.ESXi.SnapshotName
	}

	// Build flat VM/user lists for the VMware service.
	// The first VM per pair is paired with the user for password rotation;
	// additional VMs per pair use an empty user (snapshot revert only).
	var vmsToRestore []string
	var usersForRestore []string
	for _, p := range pairs {
		for j, vm := range p.VMs {
			vmsToRestore = append(vmsToRestore, vm)
			if j == 0 {
				usersForRestore = append(usersForRestore, p.User)
			} else {
				usersForRestore = append(usersForRestore, "")
			}
		}
	}

	o.Logger.Info("Starting VM restore",
		logger.Action("restore"),
		logger.Status("starting"),
		logger.Events(eventCount),
		logger.F("VMS_TO_RESTORE", len(vmsToRestore)),
		logger.Snapshot(snapshotName))

	restoreErrors, passwords := o.VMware.RestoreVMsWithPasswordRotation(context.Background(), vmsToRestore, usersForRestore, snapshotName)

	if len(passwords) > 0 {
		o.Logger.Info("Password rotation completed", logger.Action("password_rotation"), logger.Status("completed"))

		wireguardConfigs := make(map[string]string)
		if o.WireGuard != nil {
			for i, p := range pairs {
				username := p.User
				_, pubKey, err := o.WireGuard.RotateUserKey(username)
				if err != nil {
					o.Logger.Error("Failed to rotate WireGuard key", logger.User(username), logger.Error(err))
					continue
				}

				if err := o.WireGuard.RegisterPeerWithOPNsense(username, pubKey, i); err != nil {
					o.Logger.Error("Failed to register peer with OPNsense", logger.User(username), logger.Error(err))
				} else {
					o.Logger.Info("Peer registered with OPNsense", logger.User(username), logger.F("PUBLIC_KEY", pubKey))
				}

				config, err := o.WireGuard.GenerateClientConfig(username, i)
				if err != nil {
					o.Logger.Error("Failed to generate WireGuard config", logger.User(username), logger.Error(err))
					continue
				}

				wireguardConfigs[username] = config
				o.Logger.Info("WireGuard config generated", logger.User(username), logger.F("PUBLIC_KEY", pubKey))
			}
		}

		for i, p := range pairs {
			username := p.User
			if password, ok := passwords[username]; ok {
				o.Logger.Info("User password rotated", logger.User(username), logger.Password(password))

				if o.Email != nil && i < len(activeEvents) && activeEvents[i].Email != "" {
					vmName := ""
					if len(p.VMs) > 0 {
						vmName = p.VMs[0]
					}

					var attachment *service.EmailAttachment
					if wgConfig, ok := wireguardConfigs[username]; ok {
						attachment = &service.EmailAttachment{
							Filename: fmt.Sprintf("%s-wireguard.conf", username),
							Content:  []byte(wgConfig),
							MimeType: "application/x-wireguard-profile",
						}
					}

					err := o.Email.SendPasswordEmailWithAttachment(activeEvents[i].Email, vmName, username, password, attachment)
					if err != nil {
						o.Logger.Error("Failed to send password email",
							logger.F("EMAIL", activeEvents[i].Email),
							logger.User(username),
							logger.Error(err))
					} else {
						logMsg := "Password email sent"
						if attachment != nil {
							logMsg += " with WireGuard config"
						}
						o.Logger.Info(logMsg,
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
			o.Logger.Error("VM restore failed", logger.VMIndex(i), logger.F("MESSAGE", errMsg))
		}
		o.Logger.Error("Restore partially failed",
			logger.Action("restore"),
			logger.Status("partial_failure"),
			logger.Restored(len(vmsToRestore)-len(restoreErrors)),
			logger.Failed(len(restoreErrors)))
		return fmt.Errorf("restore partially failed: %d of %d VMs had errors", len(restoreErrors), len(vmsToRestore))
	}

	o.Logger.Info("Restore completed successfully",
		logger.Action("restore"),
		logger.Status("success"),
		logger.Events(eventCount),
		logger.F("VMS_RESTORED", len(vmsToRestore)),
		logger.F("PASSWORDS_ROTATED", len(passwords)))
	return nil
}

// SelectAllVMs returns UserVMPairs for configured VMs that exist in the inventory.
// Only VMs explicitly listed in user_vm_mappings are included.
func (o *Orchestrator) SelectAllVMs(vmList *models.VMListResponse) []service.UserVMPair {
	if vmList == nil || len(vmList.VMs) == 0 {
		return nil
	}

	inventoryVMs := BuildInventoryMap(vmList.VMs)

	var pairs []service.UserVMPair
	for _, p := range o.FeatureCfg.ESXi.UserVMPairs() {
		var validVMs []string
		for _, vm := range p.VMs {
			if inventoryVMs[vm] {
				validVMs = append(validVMs, vm)
			} else {
				o.Logger.Warn("Configured VM not found in inventory", logger.VM(vm), logger.User(p.User))
			}
		}
		if len(validVMs) > 0 {
			pairs = append(pairs, service.UserVMPair{User: p.User, VMs: validVMs})
		}
	}

	return pairs
}

// BuildInventoryMap creates a set of VM names from the inventory for fast lookup.
func BuildInventoryMap(vms []models.VM) map[string]bool {
	inventoryVMs := make(map[string]bool)
	for _, vm := range vms {
		inventoryVMs[vm.Name] = true
	}
	return inventoryVMs
}

// ExtractEmailFromSummary extracts an email from the format "Name (email@domain.com)".
func ExtractEmailFromSummary(summary string) string {
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
