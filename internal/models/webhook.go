package models

// CommitInfo holds common commit information across different webhook providers
type CommitInfo struct {
	ID       string
	Message  string
	URL      string
	Added    []string // Files added in this commit
	Modified []string // Files modified in this commit
	Removed  []string // Files removed in this commit
}

// FileChangeSummary holds aggregated file change statistics
type FileChangeSummary struct {
	TotalAdded    int
	TotalModified int
	TotalRemoved  int
	AddedFiles    []string
	ModifiedFiles []string
	RemovedFiles  []string
}
