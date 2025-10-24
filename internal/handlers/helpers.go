package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nahidhasan98/whatsapp-notifier/internal/errors"
	"github.com/nahidhasan98/whatsapp-notifier/internal/models"
)

// writeJSON writes a JSON response with the given status code
func (h *Handler) writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("Failed to encode JSON response", err)
	}
}

// writeAppError writes an application error response
func (h *Handler) writeAppError(w http.ResponseWriter, appErr *errors.AppError) {
	response := &models.ErrorResponse{
		Error:   appErr.Message,
		Code:    string(appErr.Code),
		Details: appErr.Details,
	}

	// Log the error for internal monitoring
	h.log.With("error_code", appErr.Code).
		With("status_code", appErr.StatusCode).
		Error(appErr.Message, appErr.Err)

	h.writeJSON(w, response, appErr.StatusCode)
}
