package service

import (
	"context"
	"time"

	"github-service/internal/models"
)

// GitHubClient defines the interface for GitHub operations
type GitHubClient interface {
	GetRepository(ctx context.Context, owner, repo string) (*models.Repository, error)
	GetCommits(ctx context.Context, owner, repo string, since time.Time) ([]models.CommitResponse, error)
	GetRateLimitInfo() models.RateLimitInfo
}

// Database defines the interface for database operations
type Database interface {
	CreateRepository(ctx context.Context, repo *models.Repository) error
	UpdateRepository(ctx context.Context, repo *models.Repository) error
	GetRepositoryByName(ctx context.Context, fullName string) (*models.Repository, error)
	UpdateLastCommitCheck(ctx context.Context, repoID int64, lastCheck time.Time) error
	SetCommitsSince(ctx context.Context, repoID int64, since time.Time) error
	CreateCommit(ctx context.Context, commit *models.Commit) error
	GetCommitsBySHA(ctx context.Context, repoID int64, sha string) (*models.Commit, error)
	GetCommitsByRepository(ctx context.Context, repoID int64, page, perPage int) ([]*models.Commit, error)
	GetCommitCountByRepository(ctx context.Context, repoID int64) (int, error)
	GetTopCommitAuthors(ctx context.Context, limit int) ([]*models.CommitStats, error)
	GetTopCommitAuthorsByRepository(ctx context.Context, repoID int64, limit int) ([]*models.CommitStats, error)
	DeleteRepository(ctx context.Context, repoID int64) error

	// Monitored repositories
	AddMonitoredRepository(ctx context.Context, fullName string, syncInterval time.Duration) error
	GetMonitoredRepositories(ctx context.Context) ([]models.MonitoredRepository, error)
	UpdateMonitoredRepositorySync(ctx context.Context, fullName string, lastSyncTime time.Time) error
	RemoveMonitoredRepository(ctx context.Context, fullName string) error

	// Migration
	MigrateDB(migrationsPath string) error
	MigrateDBDown() error

	// Connection management
	Close() error
}
