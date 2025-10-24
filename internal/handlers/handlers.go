package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	waClient       *app.WhatsAppClient
	log            *logger.Logger
	validator      *validation.Validator
	giteaSecret    string
	giteaRecipient string
}

// New creates a new handler instance
func New(waClient *app.WhatsAppClient, log *logger.Logger, giteaSecret, giteaRecipient string) *Handler {
	return &Handler{
		waClient:       waClient,
		log:            log,
		validator:      validation.New(),
		giteaSecret:    giteaSecret,
		giteaRecipient: giteaRecipient,
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

// GiteaWebhook handles Gitea webhook requests
func (h *Handler) GiteaWebhook(w http.ResponseWriter, r *http.Request) {
	// Get signature from header (Gitea uses X-Gitea-Signature)
	headerSignature := r.Header.Get("X-Gitea-Signature")
	if headerSignature == "" {
		h.log.Warn("Gitea webhook received without signature header")
		h.writeAppError(w, errors.New(errors.ErrCodeUnauthorized, "Missing X-Gitea-Signature header"))
		return
	}

	// Read the raw body for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeAppError(w, errors.InvalidRequest("Failed to read request body: "+err.Error()))
		return
	}

	// Verify webhook signature
	if !h.verifyGiteaSignature(body, headerSignature) {
		h.log.Warn("Invalid Gitea webhook signature")
		h.writeAppError(w, errors.New(errors.ErrCodeUnauthorized, "Invalid webhook signature"))
		return
	}

	h.log.Info(string(body))

	// Parse webhook payload
	var payload models.GiteaWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.writeAppError(w, errors.InvalidRequest("Invalid webhook payload: "+err.Error()))
		return
	}

	// Ensure client is connected
	if !h.waClient.IsConnected() {
		ctx := r.Context()
		if err := h.waClient.EnsureConnected(ctx); err != nil {
			h.log.Error("Failed to connect client", err)
			h.writeAppError(w, errors.ConnectionFailed(err))
			return
		}
	}

	// Construct message
	message := h.formatGiteaMessage(payload)

	// Send message to configured recipient
	ctx := r.Context()
	if err := h.waClient.SendText(ctx, h.giteaRecipient, message); err != nil {
		h.log.Error("Failed to send Gitea webhook notification", err)
		h.writeAppError(w, errors.MessageSendFailed(err))
		return
	}

	h.log.Infof("Gitea webhook notification sent to %s", h.giteaRecipient)
	h.writeJSON(w, map[string]string{"status": "notification sent"}, http.StatusOK)
}

// verifyGiteaSignature verifies the HMAC SHA256 signature of the webhook payload
func (h *Handler) verifyGiteaSignature(payload []byte, headerSignature string) bool {
	if h.giteaSecret == "" {
		// If no secret is configured, skip signature verification
		h.log.Warn("Gitea webhook secret not configured, skipping signature verification")
		return true
	}

	// Calculate HMAC SHA256 signature
	mac := hmac.New(sha256.New, []byte(h.giteaSecret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures using constant time comparison to prevent timing attacks
	return hmac.Equal([]byte(headerSignature), []byte(expectedSignature))
}

func (h *Handler) formatGiteaMessage(payload models.GiteaWebhookPayload) string {
	var sb strings.Builder

	// Repository and pusher info
	sb.WriteString(fmt.Sprintf("🔔 *New Push to %s*\n\n", payload.Repository.FullName))
	sb.WriteString(fmt.Sprintf("👤 Pusher: %s\n", payload.Commits[0].Committer.Name))
	sb.WriteString(fmt.Sprintf("🌿 Branch: %s\n", strings.TrimPrefix(payload.Ref, "refs/heads/")))
	sb.WriteString(fmt.Sprintf("📊 Commits: %d\n\n", len(payload.Commits)))

	// List commits
	if len(payload.Commits) > 0 {
		sb.WriteString("*Commits:*\n")
		for i, commit := range payload.Commits {
			// Limit to first 5 commits
			if i >= 5 {
				remaining := len(payload.Commits) - 5
				sb.WriteString(fmt.Sprintf("\n_...and %d more commit(s)_\n", remaining))
				break
			}

			// Get short commit hash (first 7 chars)
			shortHash := commit.ID
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}

			// Get first line of commit message
			message := commit.Message
			if idx := strings.Index(message, "\n"); idx != -1 {
				message = message[:idx]
			}
			// Truncate long messages
			if len(message) > 60 {
				message = message[:57] + "..."
			}

			sb.WriteString(fmt.Sprintf("• %s - %s\n", shortHash, message))
		}
	}

	// Add compare URL if available
	if payload.CompareURL != "" {
		sb.WriteString(fmt.Sprintf("\n🔗 View changes: %s", payload.CompareURL))
	}

	return sb.String()
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
