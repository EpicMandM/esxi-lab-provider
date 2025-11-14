package store

import (
	"time"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
)

// Store defines the interface for database operations.
type Store interface {
	// VM related methods
	GetVMByName(name string) (*models.VM, error)
	SaveVM(vm *models.VM) error
	UpdateVMStatus(name string, status string) error
	ListVMs() ([]*models.VM, error)

	// Booking related methods
	CreateBooking(booking *models.Booking) error
	GetActiveBookingForVM(vmName string, at time.Time) (*models.Booking, error)
	DeleteBooking(id string) error

	Close() error
}
