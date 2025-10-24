package handlers

import (
	"net/http"
	"time"

	"github.com/nahidhasan98/whatsapp-notifier/internal/models"
)

// HealthCheck handles health check requests
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	connectionStatus := h.waClient.GetConnectionStatus()

	response := &models.HealthResponse{
		Status:    "ok",
		Connected: connectionStatus["connected"].(bool),
		Timestamp: time.Now().Unix(),
	}

	// Add detailed connection info if requested
	if r.URL.Query().Get("detailed") == "true" {
		// Add connection status details to response
		h.writeJSON(w, map[string]interface{}{
			"status":            response.Status,
			"connected":         response.Connected,
			"timestamp":         response.Timestamp,
			"connection_status": connectionStatus,
		}, http.StatusOK)
		return
	}

	h.writeJSON(w, response, http.StatusOK)
}
