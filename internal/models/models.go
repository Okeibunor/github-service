package models

import "time"

// Repository represents a GitHub repository
type Repository struct {
	ID              int64     `json:"id"`
	GitHubID        int64     `json:"github_id"`
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Description     string    `json:"description"`
	URL             string    `json:"url"`
	Language        string    `json:"language"`
	ForksCount      int       `json:"forks_count"`
	StarsCount      int       `json:"stargazers_count"`
	OpenIssuesCount int       `json:"open_issues_count"`
	WatchersCount   int       `json:"watchers_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastCommitCheck time.Time `json:"last_commit_check"`
	CommitsSince    time.Time `json:"commits_since"`
	CreatedAtLocal  time.Time `json:"created_at_local"`
	UpdatedAtLocal  time.Time `json:"updated_at_local"`
}

// Commit represents a Git commit in our database
type Commit struct {
	ID             int64     `json:"id" db:"id"`
	RepositoryID   int64     `json:"repository_id" db:"repository_id"`
	SHA            string    `json:"sha" db:"sha"`
	Message        string    `json:"message" db:"message"`
	AuthorName     string    `json:"author_name" db:"author_name"`
	AuthorEmail    string    `json:"author_email" db:"author_email"`
	AuthorDate     time.Time `json:"author_date" db:"author_date"`
	CommitterName  string    `json:"committer_name" db:"committer_name"`
	CommitterEmail string    `json:"committer_email" db:"committer_email"`
	CommitDate     time.Time `json:"commit_date" db:"commit_date"`
	URL            string    `json:"url" db:"url"`
	CreatedAtLocal time.Time `json:"created_at_local" db:"created_at_local"`
}

// CommitStats represents statistics about commits
type CommitStats struct {
	AuthorName  string `json:"author_name" db:"author_name"`
	AuthorEmail string `json:"author_email" db:"author_email"`
	Count       int    `json:"commit_count" db:"commit_count"`
}

// CommitAuthor represents a commit author or committer
type CommitAuthor struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// CommitResponse represents the GitHub commit API response
type CommitResponse struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author    CommitAuthor `json:"author"`
		Committer CommitAuthor `json:"committer"`
		Message   string       `json:"message"`
	} `json:"commit"`
	HTMLURL string `json:"html_url"`
}

// RateLimitInfo stores GitHub API rate limit information
type RateLimitInfo struct {
	Remaining int
	Reset     time.Time
	Limit     int
}

// MonitoredRepository represents a repository being monitored
type MonitoredRepository struct {
	ID           int64
	FullName     string
	LastSyncTime time.Time
	SyncInterval time.Duration
	IsActive     bool
}
