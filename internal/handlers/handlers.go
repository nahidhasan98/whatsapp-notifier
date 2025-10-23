package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nahidhasan98/whatsapp-notifier/internal/app"
	"github.com/nahidhasan98/whatsapp-notifier/internal/errors"
	"github.com/nahidhasan98/whatsapp-notifier/internal/logger"
	"github.com/nahidhasan98/whatsapp-notifier/internal/models"
	"github.com/nahidhasan98/whatsapp-notifier/internal/validation"
)

// Handler holds dependencies for HTTP handlers
type Handler struct {
	waClient  *app.WhatsAppClient
	log       *logger.Logger
	validator *validation.Validator
}

// New creates a new handler instance
func New(waClient *app.WhatsAppClient, log *logger.Logger) *Handler {
	return &Handler{
		waClient:  waClient,
		log:       log,
		validator: validation.New(),
	}
}

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

// GetContacts handles requests to get all contacts
func (h *Handler) GetContacts(w http.ResponseWriter, r *http.Request) {
	if h.waClient == nil {
		h.writeAppError(w, errors.ClientNotConnected())
		return
	}

	ctx := r.Context()
	contacts, err := h.waClient.GetContacts(ctx)
	if err != nil {
		h.log.Error("Failed to get contacts", err)
		h.writeAppError(w, errors.InternalError(err))
		return
	}

	h.writeJSON(w, contacts, http.StatusOK)
}

// GetGroups handles requests to get all groups
func (h *Handler) GetGroups(w http.ResponseWriter, r *http.Request) {
	if h.waClient == nil {
		h.writeAppError(w, errors.ClientNotConnected())
		return
	}

	ctx := r.Context()
	groups, err := h.waClient.GetJoinedGroups(ctx)
	if err != nil {
		h.log.Error("Failed to get groups", err)
		h.writeAppError(w, errors.InternalError(err))
		return
	}

	h.writeJSON(w, groups, http.StatusOK)
}

// SendMessage handles requests to send a message
func (h *Handler) SendMessage(w http.ResponseWriter, r *http.Request) {
	if h.waClient == nil {
		h.writeAppError(w, errors.ClientNotConnected())
		return
	}

	// Parse request body
	var req models.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeAppError(w, errors.InvalidRequest("Invalid request body: "+err.Error()))
		return
	}

	// Validate request
	if appErr := h.validator.ValidateSendMessageRequest(&req); appErr != nil {
		h.writeAppError(w, appErr)
		return
	}

	// Sanitize message
	req.Message = h.validator.SanitizeMessage(req.Message)

	// Ensure client is connected
	if !h.waClient.IsConnected() {
		ctx := r.Context()
		if err := h.waClient.EnsureConnected(ctx); err != nil {
			h.log.Error("Failed to connect client", err)
			h.writeAppError(w, errors.ConnectionFailed(err))
			return
		}
	}

	// Check if it's a LID - warn user that it might not work
	if strings.HasSuffix(req.To, "@lid") {
		h.log.Infof("LID detected: %s. Attempting to send directly (may fail if not messageable)", req.To)
	}

	// Send message
	ctx := r.Context()
	if err := h.waClient.SendText(ctx, req.To, req.Message); err != nil {
		h.log.Error("Failed to send message", err)

		// Provide helpful error message for LIDs
		if strings.HasSuffix(req.To, "@lid") {
			h.writeAppError(w, errors.MessageSendFailed(
				fmt.Errorf("cannot send message to LID %s. LIDs are internal identifiers and cannot receive messages directly. Please use the corresponding @s.whatsapp.net JID instead. Original error: %w", req.To, err)))
			return
		}

		h.writeAppError(w, errors.MessageSendFailed(err))
		return
	}

	response := &models.SendMessageResponse{
		Status:    "sent",
		To:        req.To,
		Timestamp: time.Now().Unix(),
	}
	h.writeJSON(w, response, http.StatusAccepted)
}

// Helper functions

func (h *Handler) writeJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.log.Error("Failed to encode JSON response", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, message string, status int) {
	h.writeJSON(w, map[string]string{"error": message}, status)
}

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
