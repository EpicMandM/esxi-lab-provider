package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/config"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/service"
)

func main() {
	// Load configuration
	cfg, err := config.LoadWithFile("../../../.env")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	ctx := context.Background()
	// Create application
	logger := log.New(os.Stdout, "[API] ", log.LstdFlags)
	vm, err := service.NewVMwareService(ctx, cfg.VCenterURL, cfg.VCenterUsername, cfg.VCenterPassword, cfg.VCenterInsecure, logger)
	if err != nil {
		log.Fatalf("Failed to create VMware service: %v", err)
	}
	defer func() {
		if err := vm.Close(ctx); err != nil {
			log.Printf("Error closing VMware service: %v", err)
		}
	}()

	// Get VM snapshots data
	data, err := vm.ListVMSnapshots(ctx)
	if err != nil {
		log.Fatalf("Failed to list VM snapshots: %v", err)
	}
	jsonData, err := json.MarshalIndent(data.VMs, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal VM snapshots data: %v", err)
	}
	fmt.Printf("%s\n", data.VCenterName)
	fmt.Printf("%s\n", jsonData)
}
