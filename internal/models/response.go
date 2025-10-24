package models

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Connected bool   `json:"connected"`
	Timestamp int64  `json:"timestamp"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// StatusResponse represents a generic status response
type StatusResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
