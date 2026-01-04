package service

import (
	"context"
	"log"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/store"
)

type BookingService struct {
	logger        *log.Logger
	store         *store.Store
	vmwareService *VMwareService
}

func NewBookingService(ctx context.Context, logger *log.Logger, store store.Store, vmwareService *VMwareService) *BookingService {
	// Get VMs from VMWare and sync with store
	vms, err := vmwareService.ListAllVMs(ctx)
	if err != nil {
		logger.Printf("Error fetching VMs from VMware: %v", err)
	}
	for _, vm := range vms {
		err := store.SaveVM(vm)
		if err != nil {
			logger.Printf("Error saving VM %s to store: %v", vm.Name, err)
		}
	}
	return &BookingService{
		logger:        logger,
		store:         &store,
		vmwareService: vmwareService,
	}
}

func (s *BookingService) ListVMs() ([]*models.VM, error) {
	return (*s.store).ListVMs()
}

func (s *BookingService) BookVM(booking *models.Booking) error {
	return (*s.store).CreateBooking(booking)
}
