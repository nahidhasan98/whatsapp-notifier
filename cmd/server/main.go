package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/nahidhasan98/whatsapp-notifier/internal/app"
	"github.com/nahidhasan98/whatsapp-notifier/internal/config"
	"github.com/nahidhasan98/whatsapp-notifier/internal/handlers"
	"github.com/nahidhasan98/whatsapp-notifier/internal/logger"
	"github.com/nahidhasan98/whatsapp-notifier/internal/server"
)

// Global variables for configuration and services
var (
	cfg      *config.Config
	log      *logger.Logger
	waClient *app.WhatsAppClient
	errChan  = make(chan error, 2)
)

func main() {
	// Create a context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a wait group for graceful shutdown
	var wg sync.WaitGroup

	// Initialize configuration and services
	if err := initialize(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Initialization error: %v\n", err)
		os.Exit(1)
	}

	// Initialize and start WhatsApp client
	startWhatsAppClient(ctx, &wg)

	// Start the web server
	startWebServer(ctx, &wg)

	// Handle shutdown signals
	waitForShutdown(cancel, &wg)
}

func initialize(ctx context.Context) error {
	var err error

	// Load configuration
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	log = logger.New(cfg.Log.Level, cfg.Log.Format)
	log.Info("Starting WhatsApp Notifier Application")

	// Initialize WhatsApp client
	waClient, err = app.NewWhatsAppClient(
		ctx,
		cfg.Database.Driver,
		cfg.Database.DSN,
		cfg.WhatsApp.LogLevel,
		cfg.WhatsApp.DeviceName,
		log,
	)
	if err != nil {
		return fmt.Errorf("failed to create WhatsApp client: %w", err)
	}

	// Add event handler
	waClient.AddEventHandler(app.DefaultEventHandler(log))

	return nil
}

func startWhatsAppClient(ctx context.Context, wg *sync.WaitGroup) {
	wg.Go(func() {
		defer func() {
			waClient.Disconnect()
			log.Info("WhatsApp client shutdown complete")
		}()

		log.Info("Starting WhatsApp client...")
		if err := waClient.Connect(ctx); err != nil {
			errChan <- fmt.Errorf("failed to connect to WhatsApp: %w", err)
			return
		}

		// Keep the WhatsApp client running
		// It will handle reconnections automatically
		<-ctx.Done()
		log.Info("WhatsApp client shutting down...")
	})
}

func startWebServer(ctx context.Context, wg *sync.WaitGroup) {
	wg.Go(func() {
		log.Info("Starting HTTP server...")

		// Initialize HTTP handlers
		httpHandler := handlers.New(
			waClient,
			log,
			cfg.Gitea.WebhookSecret,
			cfg.Gitea.Recipient,
			cfg.GitHub.WebhookSecret,
			cfg.GitHub.Recipient,
		)

		// Initialize and start HTTP server
		httpServer := server.New(cfg, httpHandler, log)
		if err := httpServer.Start(cfg); err != nil {
			errChan <- fmt.Errorf("failed to start HTTP server: %w", err)
			return
		}

		// Keep the server running until shutdown
		<-ctx.Done()
		log.Info("HTTP server shutting down...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer shutdownCancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error("Error during HTTP server shutdown", err)
		}
	})
}

func waitForShutdown(cancel context.CancelFunc, wg *sync.WaitGroup) {
	// Wait for either service to fail or for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Error("Service failed", err)
	case <-sigChan:
		log.Info("Received shutdown signal")
	}

	// Cancel context to signal goroutines to shutdown
	cancel()

	// Wait for all goroutines to finish
	wg.Wait()

	log.Info("Application stopped")
}
