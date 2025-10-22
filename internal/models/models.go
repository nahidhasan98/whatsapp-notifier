package models

// SendMessageRequest represents the request payload for sending messages
type SendMessageRequest struct {
	To      string `json:"to" validate:"required"`
	Message string `json:"message" validate:"required,min=1"`
}

// SendMessageResponse represents the response after sending a message
type SendMessageResponse struct {
	Status    string `json:"status"`
	To        string `json:"to"`
	MessageID string `json:"message_id,omitempty"`
	Timestamp int64  `json:"timestamp"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string `json:"status"`
	Connected bool   `json:"connected"`
	Timestamp int64  `json:"timestamp"`
}

// ContactInfo represents a WhatsApp contact
type ContactInfo struct {
	JID          string `json:"jid"`
	PushName     string `json:"push_name,omitempty"`
	BusinessName string `json:"business_name,omitempty"`
	FirstName    string `json:"first_name,omitempty"`
	FullName     string `json:"full_name,omitempty"`
}

// GroupInfo represents a WhatsApp group
type GroupInfo struct {
	JID         string `json:"jid"`
	Name        string `json:"name"`
	Topic       string `json:"topic,omitempty"`
	IsAnnounce  bool   `json:"is_announce"`
	IsLocked    bool   `json:"is_locked"`
	IsEphemeral bool   `json:"is_ephemeral"`
	CreatedAt   int64  `json:"created_at,omitempty"`
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
