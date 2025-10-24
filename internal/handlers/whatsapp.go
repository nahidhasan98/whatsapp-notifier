package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nahidhasan98/whatsapp-notifier/internal/errors"
	"github.com/nahidhasan98/whatsapp-notifier/internal/models"
)

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
