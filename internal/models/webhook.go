package models

// CommitInfo holds common commit information across different webhook providers
type CommitInfo struct {
	ID      string
	Message string
	URL     string
}
