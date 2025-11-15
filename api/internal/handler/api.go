package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/EpicMandM/esxi-lab-provider/api/internal/models"
	"github.com/EpicMandM/esxi-lab-provider/api/internal/service"
	"github.com/google/uuid"
)

type APIHandler struct {
	bookingService *service.BookingService
}

func NewAPIHandler(bookingService *service.BookingService) *APIHandler {
	return &APIHandler{
		bookingService: bookingService,
	}
}

func (h *APIHandler) ListVMs(w http.ResponseWriter, r *http.Request) {
	vms, err := h.bookingService.ListVMs()
	if err != nil {
		http.Error(w, "Failed to list VMs", http.StatusInternalServerError)
		return
	}
	// Serialize and write VMs to response
	jsonData, err := json.Marshal(vms)
	if err != nil {
		http.Error(w, "Failed to serialize VMs", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write(jsonData); err != nil {
		log.Printf("failed to write VM list response: %v", err)
	}
}

// ...existing code...
type bookRequest struct {
	VMName    string    `json:"vm_name"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

func (h *APIHandler) BookVM(w http.ResponseWriter, r *http.Request) {
	var (
		req bookRequest
		err error
	)

	if r.Header.Get("Content-Type") == "application/json" {
		defer func() {
			if cerr := r.Body.Close(); cerr != nil {
				log.Printf("failed to close request body: %v", cerr)
			}
		}()
		if err = json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
	} else {
		if err = r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form", http.StatusBadRequest)
			return
		}
		req = bookRequest{
			VMName: r.Form.Get("vm_name"),
		}
		if req.StartTime, err = time.Parse(time.RFC3339, r.Form.Get("start_time")); err != nil {
			http.Error(w, "Invalid start_time format", http.StatusBadRequest)
			return
		}
		if req.EndTime, err = time.Parse(time.RFC3339, r.Form.Get("end_time")); err != nil {
			http.Error(w, "Invalid end_time format", http.StatusBadRequest)
			return
		}
	}

	if req.VMName == "" {
		http.Error(w, "vm_name parameter is required", http.StatusBadRequest)
		return
	}
	if req.StartTime.IsZero() || req.EndTime.IsZero() {
		http.Error(w, "start_time and end_time are required", http.StatusBadRequest)
		return
	}
	err = h.bookingService.BookVM(&models.Booking{
		ID:        uuid.New().String(),
		VMName:    req.VMName,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	})
	if err != nil {
		http.Error(w, "Failed to book VM", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("VM booked successfully")); err != nil {
		log.Printf("failed to write booking response: %v", err)
	}
}
