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

	"github.com/nahidhasan98/whatsapp-notifier/internal/errors"
	"github.com/nahidhasan98/whatsapp-notifier/internal/models"
)

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

// formatGiteaMessage constructs a formatted WhatsApp message from Gitea webhook payload
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
