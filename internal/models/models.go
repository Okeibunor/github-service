package models

import "time"

// Repository represents a GitHub repository in our database
type Repository struct {
	ID              int64      `json:"id" db:"id"`
	GitHubID        int64      `json:"github_id" db:"github_id"`
	Name            string     `json:"name" db:"name"`
	FullName        string     `json:"full_name" db:"full_name"`
	Description     *string    `json:"description" db:"description"`
	URL             string     `json:"url" db:"url"`
	Language        *string    `json:"language" db:"language"`
	ForksCount      int        `json:"forks_count" db:"forks_count"`
	StarsCount      int        `json:"stars_count" db:"stars_count"`
	OpenIssuesCount int        `json:"open_issues_count" db:"open_issues_count"`
	WatchersCount   int        `json:"watchers_count" db:"watchers_count"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	LastCommitCheck *time.Time `json:"last_commit_check" db:"last_commit_check"`
	CommitsSince    *time.Time `json:"commits_since" db:"commits_since"`
	CreatedAtLocal  time.Time  `json:"created_at_local" db:"created_at_local"`
	UpdatedAtLocal  time.Time  `json:"updated_at_local" db:"updated_at_local"`
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
