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

// GiteaWebhookPayload represents the Gitea webhook payload
type GiteaWebhookPayload struct {
	Secret     string          `json:"secret"`
	Ref        string          `json:"ref"`
	Before     string          `json:"before"`
	After      string          `json:"after"`
	CompareURL string          `json:"compare_url"`
	Commits    []GiteaCommit   `json:"commits"`
	Repository GiteaRepository `json:"repository"`
	Pusher     GiteaUser       `json:"pusher"`
	Sender     GiteaUser       `json:"sender"`
}

// GiteaCommit represents a commit in the Gitea webhook
type GiteaCommit struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	URL       string    `json:"url"`
	Author    GiteaUser `json:"author"`
	Committer GiteaUser `json:"committer"`
	Timestamp string    `json:"timestamp"`
}

// GiteaRepository represents a repository in the Gitea webhook
type GiteaRepository struct {
	ID            int       `json:"id"`
	Owner         GiteaUser `json:"owner"`
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Description   string    `json:"description"`
	Private       bool      `json:"private"`
	Fork          bool      `json:"fork"`
	HTMLURL       string    `json:"html_url"`
	SSHURL        string    `json:"ssh_url"`
	CloneURL      string    `json:"clone_url"`
	DefaultBranch string    `json:"default_branch"`
}

// GiteaUser represents a user in the Gitea webhook
type GiteaUser struct {
	ID        int    `json:"id"`
	Login     string `json:"login"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	Username  string `json:"username"`
	Name      string `json:"name"`
}
