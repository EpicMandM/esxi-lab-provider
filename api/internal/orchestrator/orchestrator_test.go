package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/logger"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/calendar/v3"
)

// --- mocks ---

type mockVMware struct {
	listFn    func(ctx context.Context) (*models.VMListResponse, error)
	restoreFn func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string)
	closeFn   func(ctx context.Context) error
}

func (m *mockVMware) ListVMSnapshots(ctx context.Context) (*models.VMListResponse, error) {
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return &models.VMListResponse{}, nil
}

func (m *mockVMware) RestoreVMsWithPasswordRotation(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
	if m.restoreFn != nil {
		return m.restoreFn(ctx, vms, users, snap)
	}
	return nil, nil
}

func (m *mockVMware) Close(ctx context.Context) error {
	if m.closeFn != nil {
		return m.closeFn(ctx)
	}
	return nil
}

type mockCalendar struct {
	listFn func(min, max string) ([]*calendar.Event, error)
}

func (m *mockCalendar) ListEvents(min, max string) ([]*calendar.Event, error) {
	if m.listFn != nil {
		return m.listFn(min, max)
	}
	return nil, nil
}

type mockEmail struct {
	calls []emailCall
	errFn func() error
}

type emailCall struct {
	to, vmName, username, password string
	attachment                     *service.EmailAttachment
}

func (m *mockEmail) SendPasswordEmail(to, vmName, username, password string) error {
	return m.SendPasswordEmailWithAttachment(to, vmName, username, password, nil)
}

func (m *mockEmail) SendPasswordEmailWithAttachment(to, vmName, username, password string, att *service.EmailAttachment) error {
	m.calls = append(m.calls, emailCall{to: to, vmName: vmName, username: username, password: password, attachment: att})
	if m.errFn != nil {
		return m.errFn()
	}
	return nil
}

type mockWireGuard struct {
	rotateKeyFn    func(string) (string, string, error)
	genConfigFn    func(string, int) (string, error)
	validateFn     func() error
	registerPeerFn func(string, string, int) error
}

func (m *mockWireGuard) RotateUserKey(u string) (string, string, error) {
	if m.rotateKeyFn != nil {
		return m.rotateKeyFn(u)
	}
	return "priv", "pub", nil
}

func (m *mockWireGuard) GenerateClientConfig(u string, i int) (string, error) {
	if m.genConfigFn != nil {
		return m.genConfigFn(u, i)
	}
	return "[Interface]\nPrivateKey = mock\n", nil
}

func (m *mockWireGuard) ValidateConfig() error {
	if m.validateFn != nil {
		return m.validateFn()
	}
	return nil
}

func (m *mockWireGuard) RegisterPeerWithOPNsense(u, pk string, i int) error {
	if m.registerPeerFn != nil {
		return m.registerPeerFn(u, pk, i)
	}
	return nil
}

// --- helper ---

func newTestOrch() (*Orchestrator, *bytes.Buffer) {
	var buf bytes.Buffer
	return &Orchestrator{
		Logger:   logger.NewWithWriter(&buf),
		VMware:   &mockVMware{},
		Calendar: &mockCalendar{},
		FeatureCfg: &service.FeatureConfig{
			ESXi: service.ESXiConfig{
				UserVMMappings: map[string]string{"alice": "vm-alice", "bob": "vm-bob"},
			},
		},
	}, &buf
}

// --- BuildInventoryMap tests ---

func TestBuildInventoryMap(t *testing.T) {
	vms := []models.VM{{Name: "vm1"}, {Name: "vm2"}}
	m := BuildInventoryMap(vms)
	assert.True(t, m["vm1"])
	assert.True(t, m["vm2"])
	assert.False(t, m["vm3"])
}

func TestBuildInventoryMap_Empty(t *testing.T) {
	m := BuildInventoryMap(nil)
	assert.Empty(t, m)
}

// --- ExtractEmailFromSummary tests ---

func TestExtractEmailFromSummary(t *testing.T) {
	tests := []struct {
		name    string
		summary string
		want    string
	}{
		{"standard format", "John Doe (john@example.com)", "john@example.com"},
		{"no parens", "John Doe", ""},
		{"empty parens", "John ()", ""},
		{"nested parens", "John (foo (bar@baz.com))", "bar@baz.com"},
		{"only opening", "John (email", ""},
		{"empty string", "", ""},
		{"parens at start", "(email@example.com) John", "email@example.com"},
		{"multiple tokens", "Lab - Student (student@school.edu)", "student@school.edu"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExtractEmailFromSummary(tt.summary))
		})
	}
}

// --- FilterActiveEvents tests ---

func TestFilterActiveEvents_ActiveEvent(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	events := []*calendar.Event{
		{
			Summary: "Lab Session (student@example.com)",
			Start:   &calendar.EventDateTime{DateTime: "2025-06-15T13:00:00Z"},
			End:     &calendar.EventDateTime{DateTime: "2025-06-15T15:00:00Z"},
		},
	}

	result := FilterActiveEvents(events, now)
	require.Len(t, result, 1)
	assert.Equal(t, "Lab Session (student@example.com)", result[0].Summary)
	assert.Equal(t, "student@example.com", result[0].Email)
}

func TestFilterActiveEvents_EventExactStart(t *testing.T) {
	now := time.Date(2025, 6, 15, 13, 0, 0, 0, time.UTC)
	events := []*calendar.Event{
		{
			Summary: "Exact start",
			Start:   &calendar.EventDateTime{DateTime: "2025-06-15T13:00:00Z"},
			End:     &calendar.EventDateTime{DateTime: "2025-06-15T14:00:00Z"},
		},
	}

	result := FilterActiveEvents(events, now)
	assert.Len(t, result, 1)
}

func TestFilterActiveEvents_FutureEvent(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	events := []*calendar.Event{
		{
			Summary: "Future",
			Start:   &calendar.EventDateTime{DateTime: "2025-06-15T13:00:00Z"},
			End:     &calendar.EventDateTime{DateTime: "2025-06-15T14:00:00Z"},
		},
	}

	result := FilterActiveEvents(events, now)
	assert.Empty(t, result)
}

func TestFilterActiveEvents_PastEvent(t *testing.T) {
	now := time.Date(2025, 6, 15, 16, 0, 0, 0, time.UTC)
	events := []*calendar.Event{
		{
			Summary: "Past",
			Start:   &calendar.EventDateTime{DateTime: "2025-06-15T13:00:00Z"},
			End:     &calendar.EventDateTime{DateTime: "2025-06-15T14:00:00Z"},
		},
	}

	result := FilterActiveEvents(events, now)
	assert.Empty(t, result)
}

func TestFilterActiveEvents_SkipsAllDayEvents(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	events := []*calendar.Event{
		{
			Summary: "All day",
			Start:   &calendar.EventDateTime{Date: "2025-06-15"},
			End:     &calendar.EventDateTime{Date: "2025-06-16"},
		},
	}

	result := FilterActiveEvents(events, now)
	assert.Empty(t, result)
}

func TestFilterActiveEvents_SkipsNilStartEnd(t *testing.T) {
	now := time.Now()
	events := []*calendar.Event{
		{Summary: "nil start", Start: nil, End: &calendar.EventDateTime{DateTime: "2025-06-15T14:00:00Z"}},
		{Summary: "nil end", Start: &calendar.EventDateTime{DateTime: "2025-06-15T13:00:00Z"}, End: nil},
	}
	result := FilterActiveEvents(events, now)
	assert.Empty(t, result)
}

func TestFilterActiveEvents_InvalidStartTime(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	events := []*calendar.Event{
		{
			Summary: "bad start",
			Start:   &calendar.EventDateTime{DateTime: "not-a-time"},
			End:     &calendar.EventDateTime{DateTime: "2025-06-15T15:00:00Z"},
		},
	}

	result := FilterActiveEvents(events, now)
	assert.Empty(t, result)
}

func TestFilterActiveEvents_InvalidEndTime(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	events := []*calendar.Event{
		{
			Summary: "bad end",
			Start:   &calendar.EventDateTime{DateTime: "2025-06-15T13:00:00Z"},
			End:     &calendar.EventDateTime{DateTime: "not-a-time"},
		},
	}

	result := FilterActiveEvents(events, now)
	assert.Empty(t, result)
}

func TestFilterActiveEvents_AttendeeEmail(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	events := []*calendar.Event{
		{
			Summary: "With attendee",
			Start:   &calendar.EventDateTime{DateTime: "2025-06-15T13:00:00Z"},
			End:     &calendar.EventDateTime{DateTime: "2025-06-15T15:00:00Z"},
			Attendees: []*calendar.EventAttendee{
				{Email: "organizer@example.com", Organizer: true},
				{Email: "student@example.com", Organizer: false},
			},
		},
	}

	result := FilterActiveEvents(events, now)
	require.Len(t, result, 1)
	assert.Equal(t, "student@example.com", result[0].Email)
}

func TestFilterActiveEvents_NoNonOrganizerAttendee(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	events := []*calendar.Event{
		{
			Summary: "Org only (fallback@example.com)",
			Start:   &calendar.EventDateTime{DateTime: "2025-06-15T13:00:00Z"},
			End:     &calendar.EventDateTime{DateTime: "2025-06-15T15:00:00Z"},
			Attendees: []*calendar.EventAttendee{
				{Email: "organizer@example.com", Organizer: true},
			},
		},
	}

	result := FilterActiveEvents(events, now)
	require.Len(t, result, 1)
	assert.Equal(t, "fallback@example.com", result[0].Email)
}

func TestFilterActiveEvents_EmptyList(t *testing.T) {
	result := FilterActiveEvents(nil, time.Now())
	assert.Nil(t, result)
}

func TestFilterActiveEvents_AttendeeWithEmptyEmail(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	events := []*calendar.Event{
		{
			Summary: "Empty email (summary@example.com)",
			Start:   &calendar.EventDateTime{DateTime: "2025-06-15T13:00:00Z"},
			End:     &calendar.EventDateTime{DateTime: "2025-06-15T15:00:00Z"},
			Attendees: []*calendar.EventAttendee{
				{Email: "", Organizer: false},
			},
		},
	}

	result := FilterActiveEvents(events, now)
	require.Len(t, result, 1)
	assert.Equal(t, "summary@example.com", result[0].Email)
}

// --- FetchVMInventory tests ---

func TestFetchVMInventory_Success(t *testing.T) {
	o, _ := newTestOrch()
	o.VMware = &mockVMware{
		listFn: func(ctx context.Context) (*models.VMListResponse, error) {
			return &models.VMListResponse{
				VMs:      []models.VM{{Name: "vm1"}, {Name: "vm2"}},
				TotalVMs: 2,
			}, nil
		},
	}

	result, err := o.FetchVMInventory()
	require.NoError(t, err)
	assert.Len(t, result.VMs, 2)
}

func TestFetchVMInventory_Error(t *testing.T) {
	o, _ := newTestOrch()
	o.VMware = &mockVMware{
		listFn: func(ctx context.Context) (*models.VMListResponse, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	_, err := o.FetchVMInventory()
	assert.Error(t, err)
}

// --- LogVMInventory tests ---

func TestLogVMInventory(t *testing.T) {
	o, buf := newTestOrch()
	vms := []models.VM{
		{
			Name: "vm1",
			Snapshots: []models.VMSnapshot{
				{Name: "snap1", State: "poweredOff", Created: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
			},
		},
	}

	o.LogVMInventory(vms)
	output := buf.String()
	assert.Contains(t, output, "vm1")
	assert.Contains(t, output, "snap1")
}

// --- SelectVMsToRestore tests ---

func TestSelectVMsToRestore_AllConfigured(t *testing.T) {
	o, _ := newTestOrch()
	vmList := &models.VMListResponse{
		VMs: []models.VM{{Name: "vm-alice"}, {Name: "vm-bob"}},
	}

	pairs := o.SelectVMsToRestore(vmList, 2)
	assert.Len(t, pairs, 2)
}

func TestSelectVMsToRestore_WithFallback(t *testing.T) {
	o, _ := newTestOrch()
	o.FeatureCfg.ESXi.UserVMMappings = map[string]string{"alice": "vm-alice"}

	vmList := &models.VMListResponse{
		VMs: []models.VM{{Name: "vm-alice"}, {Name: "vm-extra"}},
	}

	pairs := o.SelectVMsToRestore(vmList, 2)
	assert.Len(t, pairs, 2)
	assert.Equal(t, "vm-alice", pairs[0].VM)
	assert.Equal(t, "vm-extra", pairs[1].VM)
	assert.Equal(t, "", pairs[1].User) // fallback has no user
}

func TestSelectVMsToRestore_NoConfiguredVMs(t *testing.T) {
	o, _ := newTestOrch()
	o.FeatureCfg.ESXi.UserVMMappings = map[string]string{"alice": "vm-nonexistent"}

	vmList := &models.VMListResponse{
		VMs: []models.VM{{Name: "vm-inventory"}},
	}

	pairs := o.SelectVMsToRestore(vmList, 1)
	assert.Len(t, pairs, 1)
	assert.Equal(t, "vm-inventory", pairs[0].VM)
}

func TestSelectVMsToRestore_InsufficientVMs(t *testing.T) {
	o, buf := newTestOrch()
	o.FeatureCfg.ESXi.UserVMMappings = map[string]string{}

	vmList := &models.VMListResponse{
		VMs: []models.VM{{Name: "vm1"}},
	}

	pairs := o.SelectVMsToRestore(vmList, 5)
	assert.Len(t, pairs, 1)
	assert.Contains(t, buf.String(), "Insufficient VMs available")
}

// --- SelectConfiguredVMs tests ---

func TestSelectConfiguredVMs(t *testing.T) {
	o, _ := newTestOrch()
	inv := map[string]bool{"vm-alice": true, "vm-bob": true}

	pairs := o.SelectConfiguredVMs(inv, 1)
	assert.Len(t, pairs, 1)
}

func TestSelectConfiguredVMs_NoneInInventory(t *testing.T) {
	o, _ := newTestOrch()
	inv := map[string]bool{"vm-other": true}

	pairs := o.SelectConfiguredVMs(inv, 2)
	assert.Empty(t, pairs)
}

// --- AddFallbackVMs tests ---

func TestAddFallbackVMs(t *testing.T) {
	o, _ := newTestOrch()
	existing := []service.UserVMPair{{User: "alice", VM: "vm-alice"}}
	inv := []models.VM{{Name: "vm-alice"}, {Name: "vm-extra1"}, {Name: "vm-extra2"}}

	result := o.AddFallbackVMs(existing, inv, 3)
	assert.Len(t, result, 3)
	assert.Equal(t, "vm-extra1", result[1].VM)
	assert.Equal(t, "vm-extra2", result[2].VM)
}

func TestAddFallbackVMs_AlreadyEnough(t *testing.T) {
	o, _ := newTestOrch()
	existing := []service.UserVMPair{{User: "alice", VM: "vm-alice"}}
	inv := []models.VM{{Name: "vm-alice"}, {Name: "vm-extra"}}

	result := o.AddFallbackVMs(existing, inv, 1)
	assert.Len(t, result, 1)
}

// --- FetchActiveEventsAt tests ---

func TestFetchActiveEventsAt_Success(t *testing.T) {
	o, _ := newTestOrch()
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)

	o.Calendar = &mockCalendar{
		listFn: func(min, max string) ([]*calendar.Event, error) {
			return []*calendar.Event{
				{
					Summary: "Active (user@example.com)",
					Start:   &calendar.EventDateTime{DateTime: "2025-06-15T13:00:00Z"},
					End:     &calendar.EventDateTime{DateTime: "2025-06-15T15:00:00Z"},
				},
			}, nil
		},
	}

	events, err := o.FetchActiveEventsAt(now)
	require.NoError(t, err)
	assert.Len(t, events, 1)
	assert.Equal(t, "user@example.com", events[0].Email)
}

func TestFetchActiveEventsAt_CalendarError(t *testing.T) {
	o, _ := newTestOrch()
	o.Calendar = &mockCalendar{
		listFn: func(min, max string) ([]*calendar.Event, error) {
			return nil, fmt.Errorf("api error")
		},
	}

	_, err := o.FetchActiveEventsAt(time.Now())
	assert.Error(t, err)
}

// --- RestoreVMs tests ---

func TestRestoreVMs_SuccessNoWireGuardNoEmail(t *testing.T) {
	o, buf := newTestOrch()
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, map[string]string{"alice": "newpw"}
		},
	}

	pairs := []service.UserVMPair{{User: "alice", VM: "vm-alice"}}
	events := []EventInfo{{Summary: "Session", Email: "a@example.com"}}

	err := o.RestoreVMs(pairs, events)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Restore completed successfully")
}

func TestRestoreVMs_WithEmailAndWireGuard(t *testing.T) {
	email := &mockEmail{}
	wg := &mockWireGuard{}
	o, _ := newTestOrch()
	o.Email = email
	o.WireGuard = wg
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, map[string]string{"alice": "pw123"}
		},
	}

	pairs := []service.UserVMPair{{User: "alice", VM: "vm-alice"}}
	events := []EventInfo{{Summary: "Session", Email: "alice@example.com"}}

	err := o.RestoreVMs(pairs, events)
	assert.NoError(t, err)
	require.Len(t, email.calls, 1)
	assert.Equal(t, "alice@example.com", email.calls[0].to)
	assert.Equal(t, "vm-alice", email.calls[0].vmName)
	assert.Equal(t, "pw123", email.calls[0].password)
	assert.NotNil(t, email.calls[0].attachment) // WireGuard config attached
}

func TestRestoreVMs_PartialFailure(t *testing.T) {
	o, _ := newTestOrch()
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return []string{"vm-alice failed"}, map[string]string{"bob": "pw"}
		},
	}

	pairs := []service.UserVMPair{{User: "alice", VM: "vm-alice"}, {User: "bob", VM: "vm-bob"}}
	events := []EventInfo{{Summary: "S1"}, {Summary: "S2"}}

	err := o.RestoreVMs(pairs, events)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "restore partially failed")
}

func TestRestoreVMs_NoPasswordsRotated(t *testing.T) {
	o, buf := newTestOrch()
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, nil
		},
	}

	pairs := []service.UserVMPair{{User: "alice", VM: "vm-alice"}}
	events := []EventInfo{{Summary: "S1", Email: "a@ex.com"}}

	err := o.RestoreVMs(pairs, events)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Restore completed successfully")
}

func TestRestoreVMs_SnapshotNameFromConfig(t *testing.T) {
	o, buf := newTestOrch()
	snapName := "clean-state"
	o.FeatureCfg.ESXi.SnapshotName = &snapName
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			assert.Equal(t, "clean-state", snap)
			return nil, nil
		},
	}

	err := o.RestoreVMs([]service.UserVMPair{{User: "a", VM: "v"}}, []EventInfo{{Summary: "S"}})
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "clean-state")
}

func TestRestoreVMs_WireGuardKeyRotationError(t *testing.T) {
	o, buf := newTestOrch()
	o.WireGuard = &mockWireGuard{
		rotateKeyFn: func(u string) (string, string, error) {
			return "", "", fmt.Errorf("key gen failed")
		},
	}
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, map[string]string{"alice": "pw"}
		},
	}

	err := o.RestoreVMs([]service.UserVMPair{{User: "alice", VM: "vm"}}, []EventInfo{{Summary: "S", Email: "a@ex.com"}})
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Failed to rotate WireGuard key")
}

func TestRestoreVMs_WireGuardConfigGenError(t *testing.T) {
	o, buf := newTestOrch()
	o.WireGuard = &mockWireGuard{
		genConfigFn: func(u string, i int) (string, error) {
			return "", fmt.Errorf("config gen failed")
		},
	}
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, map[string]string{"alice": "pw"}
		},
	}

	err := o.RestoreVMs([]service.UserVMPair{{User: "alice", VM: "vm"}}, []EventInfo{{Summary: "S", Email: "a@ex.com"}})
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Failed to generate WireGuard config")
}

func TestRestoreVMs_RegisterPeerError(t *testing.T) {
	o, buf := newTestOrch()
	o.WireGuard = &mockWireGuard{
		registerPeerFn: func(u, pk string, i int) error {
			return fmt.Errorf("register failed")
		},
	}
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, map[string]string{"alice": "pw"}
		},
	}

	err := o.RestoreVMs([]service.UserVMPair{{User: "alice", VM: "vm"}}, []EventInfo{{Summary: "S", Email: "a@ex.com"}})
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Failed to register peer with OPNsense")
}

func TestRestoreVMs_EmailError(t *testing.T) {
	o, buf := newTestOrch()
	o.Email = &mockEmail{errFn: func() error { return fmt.Errorf("smtp error") }}
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, map[string]string{"alice": "pw"}
		},
	}

	pairs := []service.UserVMPair{{User: "alice", VM: "vm-alice"}}
	events := []EventInfo{{Summary: "S", Email: "a@ex.com"}}

	err := o.RestoreVMs(pairs, events)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Failed to send password email")
}

func TestRestoreVMs_NoEmailForEvent(t *testing.T) {
	email := &mockEmail{}
	o, _ := newTestOrch()
	o.Email = email
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, map[string]string{"alice": "pw"}
		},
	}

	pairs := []service.UserVMPair{{User: "alice", VM: "vm-alice"}}
	events := []EventInfo{{Summary: "S", Email: ""}} // no email

	err := o.RestoreVMs(pairs, events)
	assert.NoError(t, err)
	assert.Empty(t, email.calls)
}

func TestRestoreVMs_PasswordNotInMap(t *testing.T) {
	email := &mockEmail{}
	o, _ := newTestOrch()
	o.Email = email
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, map[string]string{"bob": "pw"} // alice not in map
		},
	}

	pairs := []service.UserVMPair{{User: "alice", VM: "vm-alice"}}
	events := []EventInfo{{Summary: "S", Email: "alice@ex.com"}}

	err := o.RestoreVMs(pairs, events)
	assert.NoError(t, err)
	assert.Empty(t, email.calls)
}

func TestRestoreVMs_EmailWithoutWireGuard(t *testing.T) {
	email := &mockEmail{}
	o, _ := newTestOrch()
	o.Email = email
	// No WireGuard service
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, map[string]string{"alice": "pw"}
		},
	}

	pairs := []service.UserVMPair{{User: "alice", VM: "vm-alice"}}
	events := []EventInfo{{Summary: "S", Email: "alice@ex.com"}}

	err := o.RestoreVMs(pairs, events)
	assert.NoError(t, err)
	require.Len(t, email.calls, 1)
	assert.Nil(t, email.calls[0].attachment) // No WireGuard attachment
}

// --- Run tests ---

func TestRun_NoActiveEvents(t *testing.T) {
	o, buf := newTestOrch()
	o.VMware = &mockVMware{
		listFn: func(ctx context.Context) (*models.VMListResponse, error) {
			return &models.VMListResponse{VMs: []models.VM{{Name: "vm1"}}}, nil
		},
	}
	o.Calendar = &mockCalendar{
		listFn: func(min, max string) ([]*calendar.Event, error) {
			return nil, nil
		},
	}

	err := o.Run()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No active calendar events")
}

func TestRun_NoVMsAvailable(t *testing.T) {
	o, _ := newTestOrch()
	o.FeatureCfg.ESXi.UserVMMappings = map[string]string{}
	o.VMware = &mockVMware{
		listFn: func(ctx context.Context) (*models.VMListResponse, error) {
			return &models.VMListResponse{VMs: nil}, nil
		},
	}
	o.Calendar = &mockCalendar{
		listFn: func(min, max string) ([]*calendar.Event, error) {
			now := time.Now()
			return []*calendar.Event{
				{
					Summary: "Active",
					Start:   &calendar.EventDateTime{DateTime: now.Add(-1 * time.Hour).Format(time.RFC3339)},
					End:     &calendar.EventDateTime{DateTime: now.Add(1 * time.Hour).Format(time.RFC3339)},
				},
			}, nil
		},
	}

	err := o.Run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no VMs available")
}

func TestRun_FetchVMError(t *testing.T) {
	o, _ := newTestOrch()
	o.VMware = &mockVMware{
		listFn: func(ctx context.Context) (*models.VMListResponse, error) {
			return nil, fmt.Errorf("vmware down")
		},
	}

	err := o.Run()
	assert.Error(t, err)
}

func TestRun_CalendarError(t *testing.T) {
	o, _ := newTestOrch()
	o.VMware = &mockVMware{
		listFn: func(ctx context.Context) (*models.VMListResponse, error) {
			return &models.VMListResponse{VMs: []models.VM{{Name: "vm1"}}}, nil
		},
	}
	o.Calendar = &mockCalendar{
		listFn: func(min, max string) ([]*calendar.Event, error) {
			return nil, fmt.Errorf("calendar error")
		},
	}

	err := o.Run()
	assert.Error(t, err)
}

func TestRun_HappyPath(t *testing.T) {
	o, buf := newTestOrch()
	email := &mockEmail{}
	o.Email = email
	o.VMware = &mockVMware{
		listFn: func(ctx context.Context) (*models.VMListResponse, error) {
			return &models.VMListResponse{
				VMs: []models.VM{{Name: "vm-alice"}, {Name: "vm-bob"}},
			}, nil
		},
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, map[string]string{"alice": "pw-alice"}
		},
	}
	now := time.Now()
	o.Calendar = &mockCalendar{
		listFn: func(min, max string) ([]*calendar.Event, error) {
			return []*calendar.Event{
				{
					Summary: "Lab (alice@ex.com)",
					Start:   &calendar.EventDateTime{DateTime: now.Add(-30 * time.Minute).Format(time.RFC3339)},
					End:     &calendar.EventDateTime{DateTime: now.Add(30 * time.Minute).Format(time.RFC3339)},
				},
			}, nil
		},
	}

	err := o.Run()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Restore completed successfully")
	require.Len(t, email.calls, 1)
	assert.Equal(t, "alice@ex.com", email.calls[0].to)
}

func TestRun_CloseError(t *testing.T) {
	o, buf := newTestOrch()
	o.VMware = &mockVMware{
		listFn: func(ctx context.Context) (*models.VMListResponse, error) {
			return &models.VMListResponse{VMs: []models.VM{{Name: "vm1"}}}, nil
		},
		closeFn: func(ctx context.Context) error {
			return fmt.Errorf("close error")
		},
	}
	o.Calendar = &mockCalendar{
		listFn: func(min, max string) ([]*calendar.Event, error) {
			return nil, nil // no events
		},
	}

	err := o.Run()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Failed to close VMware service")
}

func TestRestoreVMs_MoreEventsThanPairs(t *testing.T) {
	email := &mockEmail{}
	o, _ := newTestOrch()
	o.Email = email
	o.VMware = &mockVMware{
		restoreFn: func(ctx context.Context, vms, users []string, snap string) ([]string, map[string]string) {
			return nil, map[string]string{"alice": "pw"}
		},
	}

	// 1 pair but 2 events
	pairs := []service.UserVMPair{{User: "alice", VM: "vm-alice"}}
	events := []EventInfo{
		{Summary: "S1", Email: "a@ex.com"},
		{Summary: "S2", Email: "b@ex.com"},
	}

	err := o.RestoreVMs(pairs, events)
	assert.NoError(t, err)
	require.Len(t, email.calls, 1)
	assert.Equal(t, "a@ex.com", email.calls[0].to)
}

// --- SelectAllVMs tests ---

func TestSelectAllVMs_ConfiguredAndUnconfigured(t *testing.T) {
	o, _ := newTestOrch()
	// UserVMMappings: alice→vm-alice, bob→vm-bob
	vmList := &models.VMListResponse{
		VMs: []models.VM{{Name: "vm-alice"}, {Name: "vm-bob"}, {Name: "vm-extra"}},
	}

	pairs := o.SelectAllVMs(vmList)
	require.Len(t, pairs, 3)
	// Configured pairs first (sorted by user)
	assert.Equal(t, "alice", pairs[0].User)
	assert.Equal(t, "vm-alice", pairs[0].VM)
	assert.Equal(t, "bob", pairs[1].User)
	assert.Equal(t, "vm-bob", pairs[1].VM)
	// Unconfigured VM last with empty user
	assert.Equal(t, "", pairs[2].User)
	assert.Equal(t, "vm-extra", pairs[2].VM)
}

func TestSelectAllVMs_OnlyConfigured(t *testing.T) {
	o, _ := newTestOrch()
	vmList := &models.VMListResponse{
		VMs: []models.VM{{Name: "vm-alice"}, {Name: "vm-bob"}},
	}

	pairs := o.SelectAllVMs(vmList)
	require.Len(t, pairs, 2)
	assert.Equal(t, "alice", pairs[0].User)
	assert.Equal(t, "bob", pairs[1].User)
}

func TestSelectAllVMs_OnlyUnconfigured(t *testing.T) {
	o, _ := newTestOrch()
	o.FeatureCfg.ESXi.UserVMMappings = map[string]string{}
	vmList := &models.VMListResponse{
		VMs: []models.VM{{Name: "vm-x"}, {Name: "vm-y"}},
	}

	pairs := o.SelectAllVMs(vmList)
	require.Len(t, pairs, 2)
	assert.Equal(t, "", pairs[0].User)
	assert.Equal(t, "", pairs[1].User)
}

func TestSelectAllVMs_NilVMList(t *testing.T) {
	o, _ := newTestOrch()
	pairs := o.SelectAllVMs(nil)
	assert.Nil(t, pairs)
}

func TestSelectAllVMs_EmptyVMs(t *testing.T) {
	o, _ := newTestOrch()
	vmList := &models.VMListResponse{VMs: nil}
	pairs := o.SelectAllVMs(vmList)
	assert.Nil(t, pairs)
}

func TestSelectAllVMs_ConfiguredVMNotInInventory(t *testing.T) {
	o, _ := newTestOrch()
	// Configured VMs are vm-alice, vm-bob but neither is in inventory
	vmList := &models.VMListResponse{
		VMs: []models.VM{{Name: "vm-other"}},
	}

	pairs := o.SelectAllVMs(vmList)
	require.Len(t, pairs, 1)
	assert.Equal(t, "", pairs[0].User)
	assert.Equal(t, "vm-other", pairs[0].VM)
}

func TestRun_NoVMsAndNoEvents(t *testing.T) {
	o, buf := newTestOrch()
	o.FeatureCfg.ESXi.UserVMMappings = map[string]string{}
	o.VMware = &mockVMware{
		listFn: func(ctx context.Context) (*models.VMListResponse, error) {
			return &models.VMListResponse{VMs: nil}, nil
		},
	}
	o.Calendar = &mockCalendar{
		listFn: func(min, max string) ([]*calendar.Event, error) {
			return nil, nil
		},
	}

	err := o.Run()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No active calendar events")
}
