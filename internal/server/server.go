package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nahidhasan98/whatsapp-notifier/internal/config"
	"github.com/nahidhasan98/whatsapp-notifier/internal/handlers"
	"github.com/nahidhasan98/whatsapp-notifier/internal/logger"
	"github.com/nahidhasan98/whatsapp-notifier/internal/middleware"
)

// Server represents the HTTP server
type Server struct {
	httpServer *http.Server
	handler    *handlers.Handler
	middleware *middleware.Middleware
	log        *logger.Logger
}

// New creates a new HTTP server
func New(cfg *config.Config, handler *handlers.Handler, log *logger.Logger) *Server {
	mw := middleware.New(log)
	mw.SetAPIKeys(cfg.Security.APIKeys)

	return &Server{
		handler:    handler,
		middleware: mw,
		log:        log,
	}
}

// Start starts the HTTP server
func (s *Server) Start(cfg *config.Config) error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/health", s.handler.HealthCheck)
	mux.HandleFunc("/contacts", s.handler.GetContacts)
	mux.HandleFunc("/groups", s.handler.GetGroups)
	mux.HandleFunc("/send", s.handler.SendMessage)
	mux.HandleFunc("/webhook/gitea", s.handler.GiteaWebhook)
	mux.HandleFunc("/webhook/github", s.handler.GitHubWebhook)

	// Apply middleware chain
	handler := s.middleware.Recovery(mux)
	handler = s.middleware.Logging(handler)
	handler = s.middleware.Security(handler)
	handler = s.middleware.CORS(handler)
	handler = s.middleware.RateLimit(handler)
	handler = s.middleware.APIKeyAuth(handler) // Add API key authentication

	s.httpServer = &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      handler,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	s.log.Infof("HTTP server listening on %s", cfg.Server.Address())

	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log.Fatal("HTTP server error", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the HTTP server
func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	s.log.Info("HTTP server shutdown complete")
	return nil
}
