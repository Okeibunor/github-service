package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github-service/internal/models"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// DB represents the database operations
type DB struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS repositories (
	id SERIAL PRIMARY KEY,
	github_id BIGINT UNIQUE NOT NULL,
	name TEXT NOT NULL,
	full_name TEXT NOT NULL UNIQUE,
	description TEXT,
	url TEXT NOT NULL,
	language TEXT,
	forks_count INTEGER DEFAULT 0,
	stars_count INTEGER DEFAULT 0,
	open_issues_count INTEGER DEFAULT 0,
	watchers_count INTEGER DEFAULT 0,
	created_at TIMESTAMP WITH TIME ZONE NOT NULL,
	updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
	last_commit_check TIMESTAMP WITH TIME ZONE,
	commits_since TIMESTAMP WITH TIME ZONE,
	created_at_local TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	updated_at_local TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS commits (
	id SERIAL PRIMARY KEY,
	repository_id INTEGER NOT NULL REFERENCES repositories(id) ON DELETE CASCADE,
	sha TEXT NOT NULL,
	message TEXT NOT NULL,
	author_name TEXT NOT NULL,
	author_email TEXT NOT NULL,
	author_date TIMESTAMP WITH TIME ZONE NOT NULL,
	committer_name TEXT NOT NULL,
	committer_email TEXT NOT NULL,
	commit_date TIMESTAMP WITH TIME ZONE NOT NULL,
	url TEXT NOT NULL,
	created_at_local TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	UNIQUE(repository_id, sha)
);

CREATE TABLE IF NOT EXISTS monitored_repositories (
	id SERIAL PRIMARY KEY,
	full_name TEXT NOT NULL UNIQUE,
	last_sync_time TIMESTAMP WITH TIME ZONE,
	sync_interval TEXT NOT NULL,
	is_active BOOLEAN DEFAULT true,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_commits_repository_date ON commits(repository_id, commit_date DESC);
CREATE INDEX IF NOT EXISTS idx_commits_author ON commits(author_name, author_email);
CREATE INDEX IF NOT EXISTS idx_monitored_repositories_active ON monitored_repositories(is_active);
`

// New creates a new database connection
func New(dsn string) (*DB, error) {
	fmt.Printf("Connecting to database with DSN: %s\n", dsn)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("error connecting to database: %w", err)
	}
	fmt.Println("Successfully connected to database")

	if err := initializeDB(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("error initializing database: %w", err)
	}
	fmt.Println("Successfully initialized database schema")

	return &DB{db: db}, nil
}

func initializeDB(db *sql.DB) error {
	_, err := db.Exec(schema)
	return err
}

// Close closes the database connection
func (d *DB) Close() error {
	return d.db.Close()
}

// CreateRepository creates a new repository record
func (d *DB) CreateRepository(ctx context.Context, repo *models.Repository) error {
	fmt.Printf("Creating repository: %s (GitHub ID: %d)\n", repo.FullName, repo.GitHubID)
	query := `
		INSERT INTO repositories (
			github_id, name, full_name, description, url, language,
			forks_count, stars_count, open_issues_count, watchers_count,
			created_at, updated_at, commits_since
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id`

	err := d.db.QueryRowContext(ctx, query,
		repo.GitHubID, repo.Name, repo.FullName, repo.Description, repo.URL,
		repo.Language, repo.ForksCount, repo.StarsCount, repo.OpenIssuesCount,
		repo.WatchersCount, repo.CreatedAt, repo.UpdatedAt, repo.CommitsSince,
	).Scan(&repo.ID)

	if err != nil {
		fmt.Printf("Error creating repository %s: %v\n", repo.FullName, err)
		return err
	}
	fmt.Printf("Successfully created repository %s with ID %d\n", repo.FullName, repo.ID)

	return nil
}

// UpdateRepository updates an existing repository record
func (d *DB) UpdateRepository(ctx context.Context, repo *models.Repository) error {
	query := `
		UPDATE repositories SET
			name = $1, description = $2, url = $3, language = $4,
			forks_count = $5, stars_count = $6, open_issues_count = $7,
			watchers_count = $8, updated_at = $9, updated_at_local = CURRENT_TIMESTAMP
		WHERE github_id = $10`

	result, err := d.db.ExecContext(ctx, query,
		repo.Name, repo.Description, repo.URL, repo.Language,
		repo.ForksCount, repo.StarsCount, repo.OpenIssuesCount,
		repo.WatchersCount, repo.UpdatedAt, repo.GitHubID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("repository not found: %d", repo.GitHubID)
	}

	return nil
}

// GetRepositoryByName retrieves a repository by its full name
func (d *DB) GetRepositoryByName(ctx context.Context, fullName string) (*models.Repository, error) {
	query := `SELECT * FROM repositories WHERE full_name = $1`

	repo := &models.Repository{}
	err := d.db.QueryRowContext(ctx, query, fullName).Scan(
		&repo.ID, &repo.GitHubID, &repo.Name, &repo.FullName,
		&repo.Description, &repo.URL, &repo.Language, &repo.ForksCount,
		&repo.StarsCount, &repo.OpenIssuesCount, &repo.WatchersCount,
		&repo.CreatedAt, &repo.UpdatedAt, &repo.LastCommitCheck,
		&repo.CommitsSince, &repo.CreatedAtLocal, &repo.UpdatedAtLocal,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return repo, err
}

// UpdateLastCommitCheck updates the last commit check timestamp
func (d *DB) UpdateLastCommitCheck(ctx context.Context, repoID int64, lastCheck time.Time) error {
	query := `UPDATE repositories SET last_commit_check = $1, updated_at_local = CURRENT_TIMESTAMP WHERE id = $2`
	result, err := d.db.ExecContext(ctx, query, &lastCheck, repoID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("repository not found: %d", repoID)
	}
	return nil
}

// SetCommitsSince sets the commits_since timestamp
func (d *DB) SetCommitsSince(ctx context.Context, repoID int64, since time.Time) error {
	query := `UPDATE repositories SET commits_since = $1, updated_at_local = CURRENT_TIMESTAMP WHERE id = $2`
	result, err := d.db.ExecContext(ctx, query, &since, repoID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("repository not found: %d", repoID)
	}
	return nil
}

// CreateCommit creates a new commit record
func (d *DB) CreateCommit(ctx context.Context, commit *models.Commit) error {
	query := `
		INSERT INTO commits (
			repository_id, sha, message, author_name, author_email,
			author_date, committer_name, committer_email, commit_date, url
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`

	err := d.db.QueryRowContext(ctx, query,
		commit.RepositoryID, commit.SHA, commit.Message,
		commit.AuthorName, commit.AuthorEmail, commit.AuthorDate,
		commit.CommitterName, commit.CommitterEmail, commit.CommitDate,
		commit.URL,
	).Scan(&commit.ID)

	return err
}

// GetCommitsBySHA retrieves a commit by its SHA
func (d *DB) GetCommitsBySHA(ctx context.Context, repoID int64, sha string) (*models.Commit, error) {
	query := `SELECT * FROM commits WHERE repository_id = $1 AND sha = $2`

	commit := &models.Commit{}
	err := d.db.QueryRowContext(ctx, query, repoID, sha).Scan(
		&commit.ID, &commit.RepositoryID, &commit.SHA, &commit.Message,
		&commit.AuthorName, &commit.AuthorEmail, &commit.AuthorDate,
		&commit.CommitterName, &commit.CommitterEmail, &commit.CommitDate,
		&commit.URL, &commit.CreatedAtLocal,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return commit, err
}

// GetCommitsByRepository retrieves commits for a repository with pagination
func (d *DB) GetCommitsByRepository(ctx context.Context, repoID int64, page, perPage int) ([]*models.Commit, error) {
	offset := (page - 1) * perPage
	query := `
		SELECT * FROM commits 
		WHERE repository_id = $1 
		ORDER BY commit_date DESC 
		LIMIT $2 OFFSET $3`

	rows, err := d.db.QueryContext(ctx, query, repoID, perPage, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var commits []*models.Commit
	for rows.Next() {
		commit := &models.Commit{}
		err := rows.Scan(
			&commit.ID, &commit.RepositoryID, &commit.SHA, &commit.Message,
			&commit.AuthorName, &commit.AuthorEmail, &commit.AuthorDate,
			&commit.CommitterName, &commit.CommitterEmail, &commit.CommitDate,
			&commit.URL, &commit.CreatedAtLocal,
		)
		if err != nil {
			return nil, err
		}
		commits = append(commits, commit)
	}
	return commits, rows.Err()
}

// GetCommitCountByRepository returns the total number of commits for a repository
func (d *DB) GetCommitCountByRepository(ctx context.Context, repoID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM commits WHERE repository_id = $1`
	err := d.db.QueryRowContext(ctx, query, repoID).Scan(&count)
	return count, err
}

// GetTopCommitAuthors retrieves the top N commit authors by commit count
func (d *DB) GetTopCommitAuthors(ctx context.Context, limit int) ([]*models.CommitStats, error) {
	query := `
		SELECT author_name, author_email, COUNT(*) as commit_count
		FROM commits
		GROUP BY author_name, author_email
		ORDER BY commit_count DESC
		LIMIT $1`

	rows, err := d.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*models.CommitStats
	for rows.Next() {
		stat := &models.CommitStats{}
		err := rows.Scan(&stat.AuthorName, &stat.AuthorEmail, &stat.Count)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

// GetTopCommitAuthorsByRepository retrieves the top N commit authors for a specific repository
func (d *DB) GetTopCommitAuthorsByRepository(ctx context.Context, repoID int64, limit int) ([]*models.CommitStats, error) {
	query := `
		SELECT author_name, author_email, COUNT(*) as commit_count
		FROM commits
		WHERE repository_id = $1
		GROUP BY author_name, author_email
		ORDER BY commit_count DESC
		LIMIT $2`

	rows, err := d.db.QueryContext(ctx, query, repoID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*models.CommitStats
	for rows.Next() {
		stat := &models.CommitStats{}
		err := rows.Scan(&stat.AuthorName, &stat.AuthorEmail, &stat.Count)
		if err != nil {
			return nil, err
		}
		stats = append(stats, stat)
	}
	return stats, rows.Err()
}

// DeleteRepository deletes a repository and its associated commits from the database
func (d *DB) DeleteRepository(ctx context.Context, repoID int64) error {
	// The commits will be automatically deleted due to ON DELETE CASCADE
	query := `DELETE FROM repositories WHERE id = $1`
	result, err := d.db.ExecContext(ctx, query, repoID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("repository not found: %d", repoID)
	}

	return nil
}

// NewFromDB creates a new DB instance from an existing *sql.DB
func NewFromDB(db *sql.DB) *DB {
	return &DB{db: db}
}

// MonitoredRepository represents a repository being monitored
type MonitoredRepository struct {
	ID           int64
	FullName     string
	LastSyncTime time.Time
	SyncInterval time.Duration
	IsActive     bool
}

// AddMonitoredRepository adds a repository to the monitoring list
func (d *DB) AddMonitoredRepository(ctx context.Context, fullName string, syncInterval time.Duration) error {
	query := `
		INSERT INTO monitored_repositories (full_name, last_sync_time, sync_interval, is_active)
		VALUES ($1, $2, $3, true)
		ON CONFLICT (full_name) 
		DO UPDATE SET sync_interval = $3, is_active = true, updated_at = CURRENT_TIMESTAMP
	`
	_, err := d.db.ExecContext(ctx, query, fullName, time.Now().UTC(), syncInterval.String())
	return err
}

// GetMonitoredRepositories returns all actively monitored repositories
func (d *DB) GetMonitoredRepositories(ctx context.Context) ([]models.MonitoredRepository, error) {
	query := `
		SELECT id, full_name, last_sync_time, sync_interval, is_active
		FROM monitored_repositories
		WHERE is_active = true
	`
	rows, err := d.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []models.MonitoredRepository
	for rows.Next() {
		var repo models.MonitoredRepository
		var intervalStr string
		err := rows.Scan(&repo.ID, &repo.FullName, &repo.LastSyncTime, &intervalStr, &repo.IsActive)
		if err != nil {
			return nil, err
		}
		repo.SyncInterval, err = time.ParseDuration(intervalStr)
		if err != nil {
			return nil, fmt.Errorf("invalid sync interval for %s: %w", repo.FullName, err)
		}
		repos = append(repos, repo)
	}
	return repos, rows.Err()
}

// UpdateMonitoredRepositorySync updates the last sync time for a monitored repository
func (d *DB) UpdateMonitoredRepositorySync(ctx context.Context, fullName string, lastSyncTime time.Time) error {
	query := `
		UPDATE monitored_repositories
		SET last_sync_time = $2, updated_at = CURRENT_TIMESTAMP
		WHERE full_name = $1
	`
	result, err := d.db.ExecContext(ctx, query, fullName, lastSyncTime)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("monitored repository not found: %s", fullName)
	}
	return nil
}

// RemoveMonitoredRepository marks a repository as inactive
func (d *DB) RemoveMonitoredRepository(ctx context.Context, fullName string) error {
	query := `
		UPDATE monitored_repositories
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE full_name = $1
	`
	result, err := d.db.ExecContext(ctx, query, fullName)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("monitored repository not found: %s", fullName)
	}
	return nil
}

// DB returns the underlying sql.DB instance
func (d *DB) DB() *sql.DB {
	return d.db
}
