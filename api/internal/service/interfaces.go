package service

import (
	"context"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	"google.golang.org/api/calendar/v3"
)

// VMwareClient abstracts VMware operations for testability.
type VMwareClient interface {
	ListVMSnapshots(ctx context.Context) (*models.VMListResponse, error)
	RestoreVMsWithPasswordRotation(ctx context.Context, vmNames []string, userNames []string, snapshotName string) ([]string, map[string]string)
	Close(ctx context.Context) error
}

// CalendarClient abstracts Google Calendar operations for testability.
type CalendarClient interface {
	ListEvents(timeMin, timeMax string) ([]*calendar.Event, error)
}

// EmailSender abstracts email sending operations for testability.
type EmailSender interface {
	SendPasswordEmail(to, vmName, username, password string) error
	SendPasswordEmailWithAttachment(to, vmName, username, password string, attachment *EmailAttachment) error
}

// WireGuardManager abstracts WireGuard operations for testability.
type WireGuardManager interface {
	RotateUserKey(username string) (privateKey, publicKey string, err error)
	GenerateClientConfig(username string, userIndex int) (string, error)
	ValidateConfig() error
	RegisterPeerWithOPNsense(username, publicKey string, userIndex int) error
}

// OPNsenseAPI abstracts OPNsense WireGuard API operations for testability.
type OPNsenseAPI interface {
	SearchPeerByTunnelAddress(tunnelAddress string) (*PeerRow, error)
	UpdatePeer(uuid, name, publicKey, tunnelAddress, servers string) error
	CreatePeer(name, publicKey, tunnelAddress string, keepalive int) error
}
