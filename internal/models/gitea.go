package models

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
