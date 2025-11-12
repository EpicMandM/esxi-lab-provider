package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/app"
)

type APIHandler struct {
	app    *app.App
	logger *log.Logger
}

func NewAPIHandler(application *app.App, logger *log.Logger) *APIHandler {
	return &APIHandler{
		app:    application,
		logger: logger,
	}
}

// ListVMSnapshots handles GET /api/vms/snapshots
func (h *APIHandler) ListVMSnapshots(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data, err := h.app.ListVMSnapshots(ctx)
	if err != nil {
		h.logger.Printf("Error listing VM snapshots: %v", err)
		http.Error(w, "Failed to list VM snapshots", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Printf("Error encoding JSON: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}
