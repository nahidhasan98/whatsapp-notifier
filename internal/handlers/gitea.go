package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/nahidhasan98/whatsapp-notifier/internal/models"
)

// GiteaWebhook handles Gitea webhook requests
func (h *Handler) GiteaWebhook(w http.ResponseWriter, r *http.Request) {
	config := WebhookConfig{
		Provider:        ProviderGitea,
		SignatureHeader: "X-Gitea-Signature",
		Secret:          h.giteaSecret,
		Recipient:       h.giteaRecipient,
		SignaturePrefix: "", // Gitea doesn't use a prefix
	}

	parsePayload := func(body []byte) (WebhookPayload, error) {
		var payload models.GiteaWebhookPayload
		err := json.Unmarshal(body, &payload)
		return payload, err
	}

	h.handleWebhook(w, r, config, parsePayload)
}
