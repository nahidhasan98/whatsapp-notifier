package models

import "strings"

// GitHubWebhookPayload represents the GitHub webhook payload
type GitHubWebhookPayload struct {
	Ref        string           `json:"ref"`
	Before     string           `json:"before"`
	After      string           `json:"after"`
	Compare    string           `json:"compare"`
	Commits    []GitHubCommit   `json:"commits"`
	HeadCommit GitHubCommit     `json:"head_commit"`
	Repository GitHubRepository `json:"repository"`
	Pusher     GitHubPusher     `json:"pusher"`
	Sender     GitHubUser       `json:"sender"`
	Created    bool             `json:"created"`
	Deleted    bool             `json:"deleted"`
	Forced     bool             `json:"forced"`
}

// GitHubCommit represents a commit in the GitHub webhook
type GitHubCommit struct {
	ID        string           `json:"id"`
	TreeID    string           `json:"tree_id"`
	Distinct  bool             `json:"distinct"`
	Message   string           `json:"message"`
	Timestamp string           `json:"timestamp"`
	URL       string           `json:"url"`
	Author    GitHubCommitUser `json:"author"`
	Committer GitHubCommitUser `json:"committer"`
	Added     []string         `json:"added"`
	Removed   []string         `json:"removed"`
	Modified  []string         `json:"modified"`
}

// GitHubCommitUser represents a user in a commit
type GitHubCommitUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

// GitHubRepository represents a repository in the GitHub webhook
type GitHubRepository struct {
	ID              int        `json:"id"`
	NodeID          string     `json:"node_id"`
	Name            string     `json:"name"`
	FullName        string     `json:"full_name"`
	Private         bool       `json:"private"`
	Owner           GitHubUser `json:"owner"`
	HTMLURL         string     `json:"html_url"`
	Description     *string    `json:"description"`
	Fork            bool       `json:"fork"`
	URL             string     `json:"url"`
	DefaultBranch   string     `json:"default_branch"`
	CreatedAt       int64      `json:"created_at"`
	UpdatedAt       string     `json:"updated_at"`
	PushedAt        int64      `json:"pushed_at"`
	GitURL          string     `json:"git_url"`
	SSHURL          string     `json:"ssh_url"`
	CloneURL        string     `json:"clone_url"`
	SVNURL          string     `json:"svn_url"`
	Size            int        `json:"size"`
	StargazersCount int        `json:"stargazers_count"`
	Language        string     `json:"language"`
	HasIssues       bool       `json:"has_issues"`
	HasProjects     bool       `json:"has_projects"`
	HasDownloads    bool       `json:"has_downloads"`
	HasWiki         bool       `json:"has_wiki"`
	HasPages        bool       `json:"has_pages"`
	ForksCount      int        `json:"forks_count"`
	Archived        bool       `json:"archived"`
	Disabled        bool       `json:"disabled"`
	OpenIssuesCount int        `json:"open_issues_count"`
	Visibility      string     `json:"visibility"`
}

// GitHubUser represents a user in the GitHub webhook
type GitHubUser struct {
	Login             string `json:"login"`
	ID                int    `json:"id"`
	NodeID            string `json:"node_id"`
	AvatarURL         string `json:"avatar_url"`
	GravatarID        string `json:"gravatar_id"`
	URL               string `json:"url"`
	HTMLURL           string `json:"html_url"`
	FollowersURL      string `json:"followers_url"`
	FollowingURL      string `json:"following_url"`
	GistsURL          string `json:"gists_url"`
	StarredURL        string `json:"starred_url"`
	SubscriptionsURL  string `json:"subscriptions_url"`
	OrganizationsURL  string `json:"organizations_url"`
	ReposURL          string `json:"repos_url"`
	EventsURL         string `json:"events_url"`
	ReceivedEventsURL string `json:"received_events_url"`
	Type              string `json:"type"`
	SiteAdmin         bool   `json:"site_admin"`
	Name              string `json:"name,omitempty"`
	Email             string `json:"email,omitempty"`
}

// GitHubPusher represents the pusher in the GitHub webhook
type GitHubPusher struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetRepositoryName returns the full repository name
func (p GitHubWebhookPayload) GetRepositoryName() string {
	return p.Repository.FullName
}

// GetPusherName returns the pusher's name
func (p GitHubWebhookPayload) GetPusherName() string {
	// Prefer committer name from first commit, fallback to pusher
	if len(p.Commits) > 0 && p.Commits[0].Committer.Name != "" {
		return p.Commits[0].Committer.Name
	}
	return p.Pusher.Name
}

// GetBranch returns the branch name without refs/heads/ prefix
func (p GitHubWebhookPayload) GetBranch() string {
	return strings.TrimPrefix(p.Ref, "refs/heads/")
}

// GetCommitCount returns the number of commits
func (p GitHubWebhookPayload) GetCommitCount() int {
	return len(p.Commits)
}

// GetCommits returns commits in a generic format
func (p GitHubWebhookPayload) GetCommits() []CommitInfo {
	commits := make([]CommitInfo, len(p.Commits))
	for i, c := range p.Commits {
		commits[i] = CommitInfo{
			ID:      c.ID,
			Message: c.Message,
			URL:     c.URL,
		}
	}
	return commits
}

// GetCompareURL returns the compare URL
func (p GitHubWebhookPayload) GetCompareURL() string {
	return p.Compare
}
