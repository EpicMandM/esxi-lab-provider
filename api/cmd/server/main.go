package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/app"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/config"
)

func main() {
	// Load configuration
	cfg, err := config.LoadWithFile("../../../.env")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create application
	logger := log.New(os.Stdout, "[API] ", log.LstdFlags)
	application := app.New(cfg, logger, os.Stdout)

	ctx := context.Background()

	// Initialize
	if err := application.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Ensure cleanup
	defer func() {
		if err := application.Close(ctx); err != nil {
			log.Printf("Error closing application: %v", err)
		}
	}()

	// Get VM snapshots data
	data, err := application.ListVMSnapshots(ctx)
	if err != nil {
		log.Fatalf("Failed to list VM snapshots: %v", err)
	}

	// Output as JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	fmt.Println(string(jsonData))
}
