package handlers

import (
	"github.com/nahidhasan98/whatsapp-notifier/internal/app"
	"github.com/nahidhasan98/whatsapp-notifier/internal/logger"
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
