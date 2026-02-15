package models

import "time"

// VMSnapshot represents a snapshot of a virtual machine.
type VMSnapshot struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	State       string    `json:"state"`
	Quiesced    bool      `json:"quiesced"`
}

type VM struct {
	Name      string       `json:"name"`
	Snapshots []VMSnapshot `json:"snapshots"`
}

type VMListResponse struct {
	ESXiName string `json:"esxi_name"`
	TotalVMs int    `json:"total_vms"`
	VMs      []VM   `json:"vms"`
}
