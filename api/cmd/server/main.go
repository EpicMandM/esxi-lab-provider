package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/config"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/handler"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/service"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/store"
)

func main() {
	// 1. Load Config
	envPath := findEnvFile()
	cfg, err := config.LoadWithFile(envPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	logger := log.New(os.Stdout, "[API] ", log.LstdFlags)

	// 2. Initialize Database (Store)
	dbStore, err := store.NewSQLiteStore("./data")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := dbStore.Close(); err != nil {
			logger.Printf("Failed to close database: %v", err)
		}
	}()

	ctx := context.Background()
	// 3. Initialize VMware Service
	vmwareService, err := service.NewVMwareService(ctx, cfg, logger)
	if err != nil {
		log.Fatalf("Failed to initialize VMware service: %v", err)
	}
	defer func() {
		if err := vmwareService.Close(ctx); err != nil {
			logger.Printf("Failed to close VMware service: %v", err)
		}
	}()
	// 4. Initialize Business Logic Service (e.g., BookingService)
	// This service gets the interfaces it needs to do its job.
	bookingService := service.NewBookingService(ctx, logger, dbStore, vmwareService)

	// 5. Initialize API Handler
	// The handler only knows about the booking service, not the DB or VMware.
	apiHandler := handler.NewAPIHandler(bookingService)

	// 6. Setup Router and Start Server
	router := http.NewServeMux()
	router.HandleFunc("/vms", apiHandler.ListVMs)
	router.HandleFunc("/book", apiHandler.BookVM)
	// ... other routes

	logger.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func findEnvFile() string {
	wd, err := os.Getwd()
	if err != nil {
		return ".env"
	}
	dir := wd
	for {
		candidate := filepath.Join(dir, ".env")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ".env"
}
