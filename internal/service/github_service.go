package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github-service/internal/github"
	"github-service/internal/models"
)

// GitHubService handles the business logic for GitHub data synchronization
type GitHubService struct {
	db     *sql.DB
	client *github.Client
}

// NewGitHubService creates a new GitHub service
func NewGitHubService(db *sql.DB, client *github.Client) *GitHubService {
	return &GitHubService{
		db:     db,
		client: client,
	}
}

// SyncRepository synchronizes repository data and its commits
func (s *GitHubService) SyncRepository(ctx context.Context, owner, name string, since time.Time) error {
	// Fetch repository information
	repo, err := s.client.GetRepository(ctx, owner, name)
	if err != nil {
		return fmt.Errorf("fetching repository: %w", err)
	}

	// Begin transaction
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Upsert repository
	var repoID int64
	err = tx.QueryRowContext(ctx, `
		INSERT INTO repositories (
			name, full_name, description, url, language,
			forks_count, stars_count, watchers_count, open_issues,
			created_at, updated_at, last_sync_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
		ON CONFLICT (full_name) DO UPDATE SET
			description = EXCLUDED.description,
			url = EXCLUDED.url,
			language = EXCLUDED.language,
			forks_count = EXCLUDED.forks_count,
			stars_count = EXCLUDED.stars_count,
			watchers_count = EXCLUDED.watchers_count,
			open_issues = EXCLUDED.open_issues,
			updated_at = EXCLUDED.updated_at,
			last_sync_at = NOW()
		RETURNING id
	`,
		repo.Name, repo.FullName, repo.Description, repo.URL, repo.Language,
		repo.ForksCount, repo.StargazersCount, repo.WatchersCount, repo.OpenIssuesCount,
		repo.CreatedAt, repo.UpdatedAt,
	).Scan(&repoID)
	if err != nil {
		return fmt.Errorf("upserting repository: %w", err)
	}

	// Fetch and store commits
	commits, err := s.client.GetCommits(ctx, owner, name, since)
	if err != nil {
		return fmt.Errorf("fetching commits: %w", err)
	}

	// Prepare commit statement
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO commits (
			repository_id, sha, message,
			author_name, author_email,
			committer_name, committer_email,
			url, committed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (repository_id, sha) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("preparing commit statement: %w", err)
	}
	defer stmt.Close()

	// Insert commits
	for _, commit := range commits {
		_, err = stmt.ExecContext(ctx,
			repoID, commit.SHA, commit.Commit.Message,
			commit.Commit.Author.Name, commit.Commit.Author.Email,
			commit.Commit.Committer.Name, commit.Commit.Committer.Email,
			commit.HTMLURL, commit.Commit.Author.Date,
		)
		if err != nil {
			return fmt.Errorf("inserting commit: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

// GetTopCommitAuthors returns the top N commit authors by commit count
func (s *GitHubService) GetTopCommitAuthors(ctx context.Context, n int) ([]struct {
	AuthorEmail string `db:"author_email"`
	AuthorName  string `db:"author_name"`
	CommitCount int    `db:"commit_count"`
}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT author_email, author_name, COUNT(*) as commit_count
		FROM commits
		GROUP BY author_email, author_name
		ORDER BY commit_count DESC
		LIMIT $1
	`, n)
	if err != nil {
		return nil, fmt.Errorf("querying top authors: %w", err)
	}
	defer rows.Close()

	var results []struct {
		AuthorEmail string `db:"author_email"`
		AuthorName  string `db:"author_name"`
		CommitCount int    `db:"commit_count"`
	}

	for rows.Next() {
		var result struct {
			AuthorEmail string `db:"author_email"`
			AuthorName  string `db:"author_name"`
			CommitCount int    `db:"commit_count"`
		}
		if err := rows.Scan(&result.AuthorEmail, &result.AuthorName, &result.CommitCount); err != nil {
			return nil, fmt.Errorf("scanning row: %w", err)
		}
		results = append(results, result)
	}

	return results, nil
}

// GetRepositoryCommits returns commits for a specific repository
func (s *GitHubService) GetRepositoryCommits(ctx context.Context, repoName string) ([]models.Commit, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT c.* FROM commits c
		JOIN repositories r ON c.repository_id = r.id
		WHERE r.name = $1
		ORDER BY c.commit_date DESC
	`, repoName)
	if err != nil {
		return nil, fmt.Errorf("querying repository commits: %w", err)
	}
	defer rows.Close()

	var commits []models.Commit
	for rows.Next() {
		var commit models.Commit
		if err := rows.Scan(
			&commit.ID,
			&commit.RepositoryID,
			&commit.SHA,
			&commit.Message,
			&commit.AuthorName,
			&commit.AuthorEmail,
			&commit.CommitterName,
			&commit.CommitterEmail,
			&commit.URL,
			&commit.CommitDate,
			&commit.CreatedAtLocal,
		); err != nil {
			return nil, fmt.Errorf("scanning commit: %w", err)
		}
		commits = append(commits, commit)
	}

	return commits, nil
}
