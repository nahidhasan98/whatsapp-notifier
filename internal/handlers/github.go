package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nahidhasan98/whatsapp-notifier/internal/models"
)

// GitHubWebhook handles GitHub webhook requests
func (h *Handler) GitHubWebhook(w http.ResponseWriter, r *http.Request) {
	config := WebhookConfig{
		Provider:        ProviderGitHub,
		SignatureHeader: "X-Hub-Signature-256",
		Secret:          h.githubSecret,
		Recipient:       h.githubRecipient,
		SignaturePrefix: "sha256=", // GitHub uses "sha256=" prefix
	}

	parsePayload := func(body []byte) (WebhookPayload, error) {
		var payload models.GitHubWebhookPayload
		err := json.Unmarshal(body, &payload)
		return payload, err
	}

	h.handleWebhook(w, r, config, parsePayload)
}
