package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/nahidhasan98/whatsapp-notifier/internal/errors"
	"github.com/nahidhasan98/whatsapp-notifier/internal/models"
)

// WebhookProvider represents different webhook providers
type WebhookProvider string

const (
	ProviderGitea  WebhookProvider = "Gitea"
	ProviderGitHub WebhookProvider = "GitHub"
)

// WebhookConfig holds configuration for webhook processing
type WebhookConfig struct {
	Provider        WebhookProvider
	SignatureHeader string
	Secret          string
	Recipient       string
	SignaturePrefix string // e.g., "sha256=" for GitHub
}

// WebhookPayload is a generic interface for webhook payloads
type WebhookPayload interface {
	GetRepositoryName() string
	GetPusherName() string
	GetBranch() string
	GetCommitCount() int
	GetCommits() []models.CommitInfo
	GetCompareURL() string
	GetFileChangeSummary() models.FileChangeSummary
}

// handleWebhook is a generic webhook handler that processes both Gitea and GitHub webhooks
func (h *Handler) handleWebhook(w http.ResponseWriter, r *http.Request, config WebhookConfig, parsePayload func([]byte) (WebhookPayload, error)) {
	// Get signature from header
	headerSignature := r.Header.Get(config.SignatureHeader)
	if headerSignature == "" {
		h.log.Warnf("%s webhook received without signature header", config.Provider)
		h.writeAppError(w, errors.New(errors.ErrCodeUnauthorized, fmt.Sprintf("Missing %s header", config.SignatureHeader)))
		return
	}

	// Read the raw body for signature verification
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeAppError(w, errors.InvalidRequest("Failed to read request body: "+err.Error()))
		return
	}

	// Verify webhook signature
	if !h.verifyWebhookSignature(body, headerSignature, config) {
		h.log.Warnf("Invalid %s webhook signature", config.Provider)
		h.writeAppError(w, errors.New(errors.ErrCodeUnauthorized, "Invalid webhook signature"))
		return
	}

	h.log.Infof("%s webhook received", config.Provider)

	// Parse webhook payload using provider-specific parser
	payload, err := parsePayload(body)
	if err != nil {
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
	message := h.formatWebhookMessage(payload, config.Provider)
	if message == "" {
		h.writeAppError(w, errors.InvalidRequest("Webhook payload has no commits to notify"))
		return
	}

	// Send message to configured recipient
	ctx := r.Context()
	if err := h.waClient.SendText(ctx, config.Recipient, message); err != nil {
		h.log.Errorf("Failed to send %s webhook notification: %v", config.Provider, err)
		h.writeAppError(w, errors.MessageSendFailed(err))
		return
	}

	h.log.Infof("%s webhook notification sent to %s", config.Provider, config.Recipient)
	h.writeJSON(w, map[string]string{"status": "notification sent"}, http.StatusOK)
}

// verifyWebhookSignature verifies the HMAC SHA256 signature of the webhook payload
func (h *Handler) verifyWebhookSignature(payload []byte, headerSignature string, config WebhookConfig) bool {
	if config.Secret == "" {
		// If no secret is configured, skip signature verification
		h.log.Warnf("%s webhook secret not configured, skipping signature verification", config.Provider)
		return true
	}

	// Handle signature prefix (e.g., "sha256=" for GitHub)
	providedSignature := headerSignature
	if config.SignaturePrefix != "" {
		if !strings.HasPrefix(headerSignature, config.SignaturePrefix) {
			return false
		}
		providedSignature = strings.TrimPrefix(headerSignature, config.SignaturePrefix)
	}

	// Calculate HMAC SHA256 signature
	mac := hmac.New(sha256.New, []byte(config.Secret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures using constant time comparison to prevent timing attacks
	return hmac.Equal([]byte(providedSignature), []byte(expectedSignature))
}

// formatWebhookMessage constructs a formatted WhatsApp message from webhook payload
func (h *Handler) formatWebhookMessage(payload WebhookPayload, provider WebhookProvider) string {
	var sb strings.Builder

	// Repository and pusher info
	sb.WriteString(fmt.Sprintf("🔔 New Push to *%s*\n", payload.GetRepositoryName()))
	sb.WriteString("\n```")
	sb.WriteString(fmt.Sprintf("👤 Pusher : %s\n", payload.GetPusherName()))
	sb.WriteString(fmt.Sprintf("🌿 Branch : %s\n", payload.GetBranch()))
	sb.WriteString(fmt.Sprintf("📊 Commits: %d\n", payload.GetCommitCount()))
	sb.WriteString("```\n")

	// List commits
	commits := payload.GetCommits()
	if len(commits) == 0 {
		// write a log message and return empty string
		h.log.Warnf("%s webhook payload has zero commits. Skipping notification", provider)
		return sb.String()
	}

	// Add commit details
	sb.WriteString("*Commits:*\n")
	for i, commit := range commits {
		// Limit to first 5 commits
		if i >= 5 {
			remaining := len(commits) - 5
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

		sb.WriteString(fmt.Sprintf("• `%s` - %s\n", shortHash, message))
	}

	// Add file change summary (only for GitHub)
	if provider == ProviderGitHub {
		// Add compare URL if available
		if compareURL := payload.GetCompareURL(); compareURL != "" {
			sb.WriteString(fmt.Sprintf("\n🔗 View changes: %s", compareURL))
		}

		fileChanges := payload.GetFileChangeSummary()
		totalChanges := fileChanges.TotalAdded + fileChanges.TotalModified + fileChanges.TotalRemoved
		if totalChanges > 0 {
			sb.WriteString("\n\n*File Changes:*\n")

			if fileChanges.TotalAdded > 0 {
				sb.WriteString(fmt.Sprintf("✅ Added: %d\n", fileChanges.TotalAdded))
				for i, file := range fileChanges.AddedFiles {
					if i >= 20 {
						remaining := len(fileChanges.AddedFiles) - 20
						sb.WriteString(fmt.Sprintf("   _...and %d more_\n", remaining))
						break
					}
					sb.WriteString(fmt.Sprintf("   • %s\n", file))
				}
			}

			if fileChanges.TotalModified > 0 {
				sb.WriteString(fmt.Sprintf("\n📝 Modified: %d\n", fileChanges.TotalModified))
				for i, file := range fileChanges.ModifiedFiles {
					if i >= 20 {
						remaining := len(fileChanges.ModifiedFiles) - 20
						sb.WriteString(fmt.Sprintf("   _...and %d more_\n", remaining))
						break
					}
					sb.WriteString(fmt.Sprintf("   • %s\n", file))
				}
			}

			if fileChanges.TotalRemoved > 0 {
				sb.WriteString(fmt.Sprintf("\n❌ Removed: %d\n", fileChanges.TotalRemoved))
				for i, file := range fileChanges.RemovedFiles {
					if i >= 20 {
						remaining := len(fileChanges.RemovedFiles) - 20
						sb.WriteString(fmt.Sprintf("   _...and %d more_\n", remaining))
						break
					}
					sb.WriteString(fmt.Sprintf("   • %s\n", file))
				}
			}
		}
	}

	return sb.String()
}
