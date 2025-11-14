package models

import "time"

// Booking represents a reservation for a virtual machine.
type Booking struct {
	ID        string    `json:"id"`
	VMName    string    `json:"vm_name"`
	User      string    `json:"user"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}
