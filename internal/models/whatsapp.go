package models

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
