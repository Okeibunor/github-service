package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github-service/internal/database"
	"github-service/internal/github"
	"github-service/internal/models"
)

// Package service provides the core business logic for the GitHub repository synchronization service

// Service coordinates between the GitHub client and database
type Service struct {
	db     *database.DB
	github github.GitHubClient
}

// Config holds the service configuration
type Config struct {
	GitHubToken string
	DB          *database.DB
}

// New creates a new service instance
func New(cfg Config) (*Service, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("database instance is required")
	}

	githubClient := github.NewClient(cfg.GitHubToken)

	return &Service{
		db:     cfg.DB,
		github: githubClient,
	}, nil
}

// Close closes the service and its resources
func (s *Service) Close() error {
	return s.db.Close()
}

// SyncRepository syncs repository information and commits from GitHub
func (s *Service) SyncRepository(ctx context.Context, owner, name string, since time.Time) error {
	repo, err := s.github.GetRepository(ctx, owner, name)
	if err != nil {
		return fmt.Errorf("error fetching repository: %w", err)
	}

	description := repo.Description
	language := repo.Language
	dbRepo := &models.Repository{
		GitHubID:        repo.ID,
		Name:            repo.Name,
		FullName:        repo.FullName,
		Description:     &description,
		URL:             repo.URL,
		Language:        &language,
		ForksCount:      repo.ForksCount,
		StarsCount:      repo.StargazersCount,
		OpenIssuesCount: repo.OpenIssuesCount,
		WatchersCount:   repo.WatchersCount,
		CreatedAt:       repo.CreatedAt,
		UpdatedAt:       repo.UpdatedAt,
		CommitsSince:    &since,
	}

	// Try to get existing repository first
	existingRepo, err := s.db.GetRepositoryByName(ctx, repo.FullName)
	if err != nil {
		return fmt.Errorf("error checking existing repository: %w", err)
	}

	// If repository exists, update it
	if existingRepo != nil {
		dbRepo.ID = existingRepo.ID
		if err := s.db.UpdateRepository(ctx, dbRepo); err != nil {
			return fmt.Errorf("error updating repository: %w", err)
		}
	} else {
		// If repository doesn't exist, create it
		if err := s.db.CreateRepository(ctx, dbRepo); err != nil {
			return fmt.Errorf("error creating repository: %w", err)
		}
	}

	// Get the repository ID (either from update or create)
	dbRepo, err = s.db.GetRepositoryByName(ctx, repo.FullName)
	if err != nil {
		return fmt.Errorf("error fetching repository from database: %w", err)
	}

	commits, err := s.github.GetCommits(ctx, owner, name, since)
	if err != nil {
		return fmt.Errorf("error fetching commits: %w", err)
	}

	for _, commit := range commits {
		existing, err := s.db.GetCommitsBySHA(ctx, dbRepo.ID, commit.SHA)
		if err != nil {
			return fmt.Errorf("error checking existing commit: %w", err)
		}
		if existing != nil {
			continue
		}

		dbCommit := &models.Commit{
			RepositoryID:   dbRepo.ID,
			SHA:            commit.SHA,
			Message:        commit.Commit.Message,
			AuthorName:     commit.Commit.Author.Name,
			AuthorEmail:    commit.Commit.Author.Email,
			AuthorDate:     commit.Commit.Author.Date,
			CommitterName:  commit.Commit.Committer.Name,
			CommitterEmail: commit.Commit.Committer.Email,
			CommitDate:     commit.Commit.Committer.Date,
			URL:            commit.HTMLURL,
		}

		if err := s.db.CreateCommit(ctx, dbCommit); err != nil {
			return fmt.Errorf("error storing commit: %w", err)
		}
	}

	if err := s.db.UpdateLastCommitCheck(ctx, dbRepo.ID, time.Now()); err != nil {
		return fmt.Errorf("error updating last commit check: %w", err)
	}

	return nil
}

// GetTopCommitAuthors returns the top N commit authors
func (s *Service) GetTopCommitAuthors(ctx context.Context, limit int) ([]*models.CommitStats, error) {
	return s.db.GetTopCommitAuthors(ctx, limit)
}

// GetTopCommitAuthorsByRepository returns the top N commit authors for a specific repository
func (s *Service) GetTopCommitAuthorsByRepository(ctx context.Context, fullName string, limit int) ([]*models.CommitStats, error) {
	// First check if the repository exists in the database
	repo, err := s.db.GetRepositoryByName(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("error fetching repository: %w", err)
	}
	if repo == nil {
		return nil, fmt.Errorf("repository not found: %s", fullName)
	}

	// Get the commits for this repository
	commits, err := s.db.GetCommitsByRepository(ctx, repo.ID, 1, 0)
	if err != nil {
		return nil, fmt.Errorf("error checking repository commits: %w", err)
	}
	if len(commits) == 0 {
		return nil, fmt.Errorf("no commits found for repository: %s", fullName)
	}

	return s.db.GetTopCommitAuthorsByRepository(ctx, repo.ID, limit)
}

// GetCommitsByRepository returns commits for a repository with pagination
func (s *Service) GetCommitsByRepository(ctx context.Context, fullName string, limit, offset int) ([]*models.Commit, error) {
	repo, err := s.db.GetRepositoryByName(ctx, fullName)
	if err != nil {
		return nil, fmt.Errorf("error fetching repository: %w", err)
	}
	if repo == nil {
		return nil, fmt.Errorf("repository not found: %s", fullName)
	}

	return s.db.GetCommitsByRepository(ctx, repo.ID, limit, offset)
}

// GetRepositoryByName retrieves a repository by its full name (owner/repo)
func (s *Service) GetRepositoryByName(ctx context.Context, fullName string) (*models.Repository, error) {
	return s.db.GetRepositoryByName(ctx, fullName)
}

// DeleteRepository deletes a repository and its associated commits from the database
func (s *Service) DeleteRepository(ctx context.Context, fullName string) error {
	repo, err := s.db.GetRepositoryByName(ctx, fullName)
	if err != nil {
		return fmt.Errorf("error finding repository: %w", err)
	}
	if repo == nil {
		return fmt.Errorf("repository not found: %s", fullName)
	}

	return s.db.DeleteRepository(ctx, repo.ID)
}

// RepositoryExists checks if a repository exists in GitHub without syncing it
func (s *Service) RepositoryExists(ctx context.Context, owner, name string) (bool, error) {
	_, err := s.github.GetRepository(ctx, owner, name)
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
