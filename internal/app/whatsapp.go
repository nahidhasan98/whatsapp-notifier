package app

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"github.com/nahidhasan98/whatsapp-notifier/internal/logger"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
)

// WhatsAppClient wraps the whatsmeow client with additional functionality
type WhatsAppClient struct {
	Client    *whatsmeow.Client
	Container *sqlstore.Container
	log       *logger.Logger

	// Reconnection handling
	isConnected     bool
	reconnectMutex  sync.RWMutex
	reconnectConfig ReconnectConfig
	cancelReconnect context.CancelFunc
}

// ReconnectConfig holds configuration for automatic reconnection
type ReconnectConfig struct {
	MaxRetries      int           // Maximum number of reconnection attempts
	InitialInterval time.Duration // Initial retry interval
	MaxInterval     time.Duration // Maximum retry interval
	Multiplier      float64       // Backoff multiplier
}

// NewWhatsAppClient creates and initializes a new WhatsApp client
func NewWhatsAppClient(ctx context.Context, dbDriver, dbDSN, logLevel, deviceName string, log *logger.Logger) (*WhatsAppClient, error) {
	// Create database logger
	dbLog := waLog.Stdout("Database", logLevel, true)

	// Initialize database container
	container, err := sqlstore.New(ctx, dbDriver, dbDSN, dbLog)
	if err != nil {
		return nil, fmt.Errorf("failed to create database container: %w", err)
	}

	// Get the first device from the store
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get device store: %w", err)
	}

	// Create client logger
	clientLog := waLog.Stdout("Client", logLevel, true)

	// Customize the OS name shown in WhatsApp's linked devices
	// This uses the SetOSInfo function to change DeviceProps.Os
	if deviceName == "" {
		deviceName = "macOS"
	}
	store.SetOSInfo(deviceName, [3]uint32{0, 1, 0})

	// Also set the platform field in the device store
	deviceStore.Platform = deviceName

	// Create WhatsApp client
	client := whatsmeow.NewClient(deviceStore, clientLog)

	wac := &WhatsAppClient{
		Client:    client,
		Container: container,
		log:       log,
		reconnectConfig: ReconnectConfig{
			MaxRetries:      10,
			InitialInterval: 5 * time.Second,
			MaxInterval:     5 * time.Minute,
			Multiplier:      1.5,
		},
	}

	// Add internal event handler for connection management
	wac.Client.AddEventHandler(wac.handleConnectionEvents)

	return wac, nil
}

// handleConnectionEvents handles connection-related events for automatic reconnection
func (w *WhatsAppClient) handleConnectionEvents(evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		w.reconnectMutex.Lock()
		w.isConnected = true
		// Cancel any ongoing reconnection attempts since we're now connected
		if w.cancelReconnect != nil {
			w.cancelReconnect()
			w.cancelReconnect = nil
		}
		w.reconnectMutex.Unlock()
		w.log.Info("WhatsApp client connected")

	case *events.Disconnected:
		w.reconnectMutex.Lock()
		w.isConnected = false
		shouldReconnect := w.cancelReconnect == nil // Only start reconnection if not already in progress
		w.reconnectMutex.Unlock()

		w.log.Warn("WhatsApp client disconnected")

		// Start reconnection process if not already in progress and not explicitly disconnected
		if shouldReconnect {
			go w.startReconnection()
		}

	case *events.StreamError:
		w.log.Errorf("WhatsApp stream error: %v", v)
	}
}

// startReconnection starts the automatic reconnection process
func (w *WhatsAppClient) startReconnection() {
	w.reconnectMutex.Lock()

	// Check if already connected or reconnection already in progress
	if w.isConnected || w.cancelReconnect != nil {
		w.reconnectMutex.Unlock()
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	w.cancelReconnect = cancel
	w.reconnectMutex.Unlock()

	defer func() {
		w.reconnectMutex.Lock()
		w.cancelReconnect = nil
		w.reconnectMutex.Unlock()
	}()

	interval := w.reconnectConfig.InitialInterval

	for attempt := 1; attempt <= w.reconnectConfig.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			w.log.Info("Reconnection cancelled")
			return
		case <-time.After(interval):
			// Check if we're already connected before attempting reconnection
			w.reconnectMutex.RLock()
			alreadyConnected := w.isConnected
			w.reconnectMutex.RUnlock()

			if alreadyConnected {
				w.log.Info("Already connected, stopping reconnection attempts")
				return
			}

			w.log.Infof("Reconnection attempt %d/%d", attempt, w.reconnectConfig.MaxRetries)

			// Check if client is already connected at the protocol level
			if w.Client.IsConnected() {
				w.log.Info("Client already connected at protocol level")
				w.reconnectMutex.Lock()
				w.isConnected = true
				w.reconnectMutex.Unlock()
				return
			}

			if err := w.Client.Connect(); err != nil {
				w.log.Errorf("Reconnection attempt %d failed: %v", attempt, err)

				// Calculate next interval with exponential backoff
				interval = time.Duration(float64(interval) * w.reconnectConfig.Multiplier)
				if interval > w.reconnectConfig.MaxInterval {
					interval = w.reconnectConfig.MaxInterval
				}

				continue
			}

			w.log.Info("Successfully reconnected to WhatsApp")
			return
		}
	}

	w.log.Error("All reconnection attempts failed", nil)
}

// Connect connects the WhatsApp client
func (w *WhatsAppClient) Connect(ctx context.Context) error {
	w.reconnectMutex.Lock()
	defer w.reconnectMutex.Unlock()

	if w.Client.Store.ID == nil {
		// No ID stored, new login required
		w.log.Info("No existing session found, starting QR authentication...")

		// Start QR authentication in background to not block
		go w.authenticateWithQR(ctx)

		// Return immediately so HTTP server can start
		return nil
	} else {
		// Already logged in, just connect
		w.log.Info("Existing session found. Connecting...")
		if err := w.Client.Connect(); err != nil {
			return fmt.Errorf("failed to connect client: %w", err)
		}
	}

	// Wait a moment and verify we're actually connected and authenticated
	time.Sleep(2 * time.Second)

	if !w.Client.IsConnected() {
		w.log.Error("Failed to establish WhatsApp connection after authentication", nil)
		return fmt.Errorf("connection not established after authentication")
	}

	// Double-check that we have a valid session
	if w.Client.Store.ID == nil {
		w.log.Error("Authentication completed but no session ID found", nil)
		return fmt.Errorf("no valid session after authentication")
	}

	w.isConnected = true
	w.log.Info("Successfully connected to WhatsApp")
	w.log.Infof("Device ID: %s", w.Client.Store.ID.String())
	return nil
}

// authenticateWithQR handles QR code authentication in a loop
func (w *WhatsAppClient) authenticateWithQR(ctx context.Context) {
	maxAttempts := 5
	attemptDelay := 5 * time.Second

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// Check if context was cancelled before starting new attempt
		select {
		case <-ctx.Done():
			return
		default:
		}

		if attempt > 1 {
			w.log.Infof("Generating new QR code (attempt %d/%d)...", attempt, maxAttempts)
			time.Sleep(attemptDelay)
		}

		// Create a timeout context for this QR code attempt
		qrCtx, qrCancel := context.WithTimeout(ctx, 60*time.Second)

		qrChan, err := w.Client.GetQRChannel(qrCtx)
		if err != nil {
			qrCancel()
			w.log.Errorf("Failed to get QR channel: %v", err)
			continue
		}

		// Connect the client if not already connected
		if !w.Client.IsConnected() {
			if err := w.Client.Connect(); err != nil {
				qrCancel()
				w.log.Errorf("Failed to connect client: %v", err)
				continue
			}
		}

		// Handle QR code events
		authSuccess, cancelled := w.handleQREvents(ctx, qrCtx, qrChan)
		qrCancel()

		// Check if parent context was cancelled (shutdown signal)
		if cancelled {
			w.log.Info("QR authentication cancelled")
			return
		}

		if authSuccess {
			w.reconnectMutex.Lock()
			w.isConnected = true
			w.reconnectMutex.Unlock()

			w.log.Info("WhatsApp authentication successful")
			if w.Client.Store.ID != nil {
				w.log.Infof("Device ID: %s", w.Client.Store.ID.String())
			}
			return
		}

		w.log.Warn("QR code authentication failed, will retry with new QR code...")
	}

	w.log.Error("Failed to authenticate after multiple attempts", nil)
}

// handleQREvents processes QR code events and returns (success, cancelled)
// success: true if authentication succeeded
// cancelled: true if parent context was cancelled (shutdown)
func (w *WhatsAppClient) handleQREvents(parentCtx, qrCtx context.Context, qrChan <-chan whatsmeow.QRChannelItem) (bool, bool) {
	qrCodeDisplayed := false

	for {
		select {
		case <-parentCtx.Done():
			// Parent context cancelled (shutdown signal) - don't log warnings
			return false, true

		case <-qrCtx.Done():
			// QR context timed out (60 seconds expired)
			if qrCodeDisplayed {
				w.log.Warn("QR code timed out without being scanned")
			}
			return false, false

		case evt, ok := <-qrChan:
			if !ok {
				// Channel closed - check if parent context was cancelled
				select {
				case <-parentCtx.Done():
					return false, true // Cancelled
				default:
					return false, false // Normal closure
				}
			}

			switch evt.Event {
			case "code":
				qrCodeDisplayed = true

				// Render QR code in terminal
				fmt.Println("\n" + strings.Repeat("=", 64))
				fmt.Println("📱 SCAN QR CODE WITH WHATSAPP MOBILE APP")
				fmt.Println(strings.Repeat("=", 64))

				cfg := qrterminal.Config{
					Level:      qrterminal.M,
					Writer:     os.Stdout,
					HalfBlocks: true,
					QuietZone:  1,
				}
				qrterminal.GenerateWithConfig(evt.Code, cfg)

				fmt.Println(strings.Repeat("=", 64))
				fmt.Println("⏰ You have 60 seconds to scan the QR code")
				fmt.Println("📱 Open WhatsApp > Settings > Linked Devices > Link a Device")
				fmt.Println(strings.Repeat("=", 64) + "\n")

			case "success":
				w.log.Info("QR code scanned successfully! Completing authentication...")
				return true, false

			case "timeout":
				w.log.Warn("QR code expired, generating new one...")
				return false, false

			default:
				w.log.Infof("Authentication event: %s", evt.Event)
			}
		}
	}
}

// Disconnect disconnects the WhatsApp client
func (w *WhatsAppClient) Disconnect() {
	w.reconnectMutex.Lock()
	defer w.reconnectMutex.Unlock()

	// Cancel any ongoing reconnection attempts
	if w.cancelReconnect != nil {
		w.cancelReconnect()
		w.cancelReconnect = nil
	}

	w.Client.Disconnect()
	w.isConnected = false
	w.log.Info("Disconnected from WhatsApp")
}

// AddEventHandler adds an event handler to the client
func (w *WhatsAppClient) AddEventHandler(handler func(interface{})) {
	w.Client.AddEventHandler(handler)
}

// SendText sends a text message to the specified JID
func (w *WhatsAppClient) SendText(ctx context.Context, toJID string, text string) error {
	jid, err := types.ParseJID(toJID)
	if err != nil {
		return fmt.Errorf("invalid JID %s: %w", toJID, err)
	}

	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	_, err = w.Client.SendMessage(ctx, jid, msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	w.log.Infof("Message sent to %s", toJID)
	return nil
}

// GetContacts retrieves all contacts from the store
func (w *WhatsAppClient) GetContacts(ctx context.Context) (map[types.JID]types.ContactInfo, error) {
	contacts, err := w.Client.Store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %w", err)
	}
	return contacts, nil
}

// GetJoinedGroups retrieves all groups the account is a member of
func (w *WhatsAppClient) GetJoinedGroups(ctx context.Context) ([]*types.GroupInfo, error) {
	groups, err := w.Client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get joined groups: %w", err)
	}
	return groups, nil
}

// IsConnected checks if the client is connected
func (w *WhatsAppClient) IsConnected() bool {
	w.reconnectMutex.RLock()
	defer w.reconnectMutex.RUnlock()

	// Check both our internal state, the actual client state, and valid session
	hasValidSession := w.Client.Store.ID != nil
	return w.isConnected && w.Client.IsConnected() && hasValidSession
}

// EnsureConnected ensures the client is connected, attempting to reconnect if necessary
func (w *WhatsAppClient) EnsureConnected(ctx context.Context) error {
	if w.IsConnected() {
		return nil
	}

	w.log.Info("Client not connected, attempting to connect...")
	return w.Connect(ctx)
}

// GetConnectionStatus returns detailed connection status information
func (w *WhatsAppClient) GetConnectionStatus() map[string]interface{} {
	w.reconnectMutex.RLock()
	defer w.reconnectMutex.RUnlock()

	hasValidSession := w.Client.Store.ID != nil
	clientConnected := w.Client.IsConnected()
	actuallyConnected := w.isConnected && clientConnected && hasValidSession

	return map[string]interface{}{
		"connected":      actuallyConnected,
		"internal_state": w.isConnected,
		"client_state":   clientConnected,
		"has_session":    hasValidSession,
		"session_id": func() string {
			if hasValidSession {
				return w.Client.Store.ID.String()
			}
			return "none"
		}(),
		"reconnection_active": w.cancelReconnect != nil,
	}
}

// DefaultEventHandler is a default event handler that logs received messages
func DefaultEventHandler(log *logger.Logger) func(interface{}) {
	return func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			log.Infof("Received message from %s: %s", v.Info.Sender.String(), v.Message.GetConversation())
		case *events.Receipt:
			log.Debugf("Received receipt for message %s", v.MessageIDs)
		case *events.Presence:
			log.Debugf("Presence update from %s: unavailable=%v", v.From.String(), v.Unavailable)
		case *events.HistorySync:
			log.Debugf("Received history sync")
		case *events.Connected:
			log.Info("Client connected")
		case *events.Disconnected:
			log.Info("Client disconnected")
		}
	}
}
