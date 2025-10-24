package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nahidhasan98/whatsapp-notifier/internal/logger"
)

// Middleware represents the middleware dependencies
type Middleware struct {
	log         *logger.Logger
	rateLimiter *RateLimiter
	apiKeys     map[string]bool // Valid API keys
}

// RateLimiter implements a simple rate limiter using token bucket algorithm
type RateLimiter struct {
	clients map[string]*ClientBucket
	mutex   sync.RWMutex

	// Rate limiting configuration
	requestsPerMinute int
	windowSize        time.Duration
}

// ClientBucket represents a rate limit bucket for a specific client
type ClientBucket struct {
	tokens     int
	lastRefill time.Time
	mutex      sync.Mutex
}

// New creates a new middleware instance
func New(log *logger.Logger) *Middleware {
	return &Middleware{
		log: log,
		rateLimiter: &RateLimiter{
			clients:           make(map[string]*ClientBucket),
			requestsPerMinute: 60, // Default: 60 requests per minute
			windowSize:        time.Minute,
		},
		apiKeys: make(map[string]bool),
	}
}

// SetAPIKeys sets the valid API keys for authentication
func (m *Middleware) SetAPIKeys(keys []string) {
	m.apiKeys = make(map[string]bool)
	for _, key := range keys {
		m.apiKeys[key] = true
	}
}

// Logging logs HTTP requests with detailed information
func (m *Middleware) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom response writer to capture the status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		m.log.With("method", r.Method).
			With("path", r.URL.Path).
			With("status", rw.statusCode).
			With("duration", duration.String()).
			With("remote_addr", r.RemoteAddr).
			With("user_agent", r.UserAgent()).
			Infof("HTTP request completed")
	})
}

// CORS adds CORS headers for cross-origin requests
func (m *Middleware) CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Recovery handles panics and returns a 500 error
func (m *Middleware) Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				m.log.Errorf("Panic in HTTP handler: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// RateLimit applies rate limiting based on client IP address
func (m *Middleware) RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)

		if !m.rateLimiter.Allow(clientIP) {
			m.log.Warnf("Rate limit exceeded for client: %s", clientIP)
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded. Please try again later.", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Allow checks if a request is allowed based on rate limiting
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	bucket, exists := rl.clients[clientIP]
	if !exists {
		bucket = &ClientBucket{
			tokens:     rl.requestsPerMinute,
			lastRefill: time.Now(),
		}
		rl.clients[clientIP] = bucket
	}

	bucket.mutex.Lock()
	defer bucket.mutex.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)

	if elapsed >= rl.windowSize {
		bucket.tokens = rl.requestsPerMinute
		bucket.lastRefill = now
	}

	// Check if tokens are available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (for proxies)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// Take the first IP in the comma-separated list
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}

// ContentType sets the Content-Type header to application/json
func (m *Middleware) ContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// APIKeyAuth validates API key authentication
func (m *Middleware) APIKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health endpoint and webhook endpoints
		if r.URL.Path == "/health" || r.URL.Path == "/webhook/gitea" || r.URL.Path == "/webhook/github" {
			next.ServeHTTP(w, r)
			return
		}

		// Get API key from header or query parameter
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			apiKey = r.URL.Query().Get("api_key")
		}

		if apiKey == "" {
			m.log.Warnf("Missing API key from %s", getClientIP(r))
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Missing API key","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}

		// Validate API key using constant-time comparison
		if !m.isValidAPIKey(apiKey) {
			m.log.Warnf("Invalid API key from %s", getClientIP(r))
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Invalid API key","code":"UNAUTHORIZED"}`, http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isValidAPIKey validates API key using constant-time comparison
func (m *Middleware) isValidAPIKey(providedKey string) bool {
	for validKey := range m.apiKeys {
		if subtle.ConstantTimeCompare([]byte(providedKey), []byte(validKey)) == 1 {
			return true
		}
	}
	return false
}

// Security adds basic security headers
func (m *Middleware) Security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// XSS protection
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Disable caching for sensitive endpoints
		if r.URL.Path != "/health" {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}

		next.ServeHTTP(w, r)
	})
}

// responseWriter is a wrapper for http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	rw.statusCode = statusCode
	rw.ResponseWriter.WriteHeader(statusCode)
}
