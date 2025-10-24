package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds the application configuration
type Config struct {
	// Server configuration
	Server ServerConfig

	// Database configuration
	Database DatabaseConfig

	// WhatsApp configuration
	WhatsApp WhatsAppConfig

	// Logging configuration
	Log LogConfig

	// Security configuration
	Security SecurityConfig

	// Gitea configuration
	Gitea GiteaConfig
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds database-specific configuration
type DatabaseConfig struct {
	Driver string
	DSN    string
}

// WhatsAppConfig holds WhatsApp-specific configuration
type WhatsAppConfig struct {
	LogLevel   string
	DeviceName string // Custom device name that appears in WhatsApp linked devices
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string
	Format string // "json" or "text"
}

// SecurityConfig holds security-specific configuration
type SecurityConfig struct {
	// API Keys - sent by clients for authentication
	APIKeys []string
}

// GiteaConfig holds Gitea webhook configuration
type GiteaConfig struct {
	WebhookSecret string // Secret for webhook validation
	Recipient     string // WhatsApp JID to send notifications to
}

// Load loads configuration from environment variables with sensible defaults
func Load() (*Config, error) {
	// Try to load .env file (ignore errors - it's optional)
	_ = godotenv.Load(".env")

	cfg := &Config{
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", ""),
			Port:            getEnvAsInt("SERVER_PORT", 8080),
			ReadTimeout:     getEnvAsDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getEnvAsDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			ShutdownTimeout: getEnvAsDuration("SERVER_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		Database: DatabaseConfig{
			Driver: getEnv("DB_DRIVER", "sqlite3"),
			DSN:    getEnv("DB_DSN", "file:mywhatsapp.db?_foreign_keys=on"),
		},
		WhatsApp: WhatsAppConfig{
			LogLevel:   getEnv("WHATSAPP_LOG_LEVEL", "INFO"),
			DeviceName: getEnv("WHATSAPP_DEVICE_NAME", "macOS"),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "text"),
		},
		Security: SecurityConfig{
			// API Keys that clients use to authenticate
			APIKeys: getEnvAsSlice("API_KEYS", []string{}),
		},
		Gitea: GiteaConfig{
			WebhookSecret: getEnv("GITEA_WEBHOOK_SECRET", ""),
			Recipient:     getEnv("GITEA_RECIPIENT", ""),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}

	if c.Database.Driver == "" {
		return fmt.Errorf("database driver is required")
	}

	if c.Database.DSN == "" {
		return fmt.Errorf("database DSN is required")
	}

	// Security validation
	if len(c.Security.APIKeys) == 0 {
		return fmt.Errorf("at least one API key is required")
	}

	// Check for default/insecure API keys
	for _, key := range c.Security.APIKeys {
		if key == "default-api-key" || key == "api-key-123" || len(key) < 8 {
			return fmt.Errorf("insecure or default API key detected: '%s'. Please set secure API keys in environment variables", key)
		}
	}

	return nil
}

// Address returns the server address in the format host:port
func (s *ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// Helper functions to get environment variables

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value, err := time.ParseDuration(valueStr)
	if err != nil {
		return defaultValue
	}

	return value
}

func getEnvAsSlice(key string, defaultValue []string) []string {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	// Split by comma and trim spaces
	values := make([]string, 0)
	for _, v := range splitAndTrim(valueStr, ",") {
		if v != "" {
			values = append(values, v)
		}
	}

	return values
}

func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, part := range splitString(s, sep) {
		trimmed := trimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}

	parts := make([]string, 0)
	start := 0

	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
			i = start - 1
		}
	}
	parts = append(parts, s[start:])

	return parts
}

func trimSpace(s string) string {
	start := 0
	end := len(s)

	// Trim leading spaces
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}

	// Trim trailing spaces
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}
