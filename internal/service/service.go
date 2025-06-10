package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github-service/internal/errors"
	"github-service/internal/models"

	"github.com/rs/zerolog"
)

// Package service provides the core business logic for the GitHub repository synchronization service

// Service handles the core business logic
type Service struct {
	github GitHubClient
	db     Database
	logger *zerolog.Logger
}

// Config holds the service configuration
type Config struct {
	GitHubToken string
	DB          Database
}

// New creates a new service instance
func New(githubClient GitHubClient, db Database, logger *zerolog.Logger) *Service {
	return &Service{
		github: githubClient,
		db:     db,
		logger: logger,
	}
}

// DB returns the database instance
func (s *Service) DB() Database {
	return s.db
}

// Close closes the service and its resources
func (s *Service) Close() error {
	return s.db.Close()
}

// SyncRepository synchronizes a repository's information and commits
func (s *Service) SyncRepository(ctx context.Context, owner, name string, since time.Time) error {
	// Get repository information from GitHub
	repo, err := s.github.GetRepository(ctx, owner, name)
	if err != nil {
		return errors.NewGitHubError("GetRepository", fmt.Sprintf("%s/%s", owner, name), err)
	}

	// Check if repository exists in database
	existingRepo, err := s.db.GetRepositoryByName(ctx, repo.FullName)
	if err != nil {
		return errors.NewDatabaseError("GetRepositoryByName", err)
	}

	if existingRepo == nil {
		// Create new repository
		if err := s.db.CreateRepository(ctx, repo); err != nil {
			return errors.NewRepositoryError(owner, name, "CreateRepository", err)
		}
	} else {
		// Update existing repository
		repo.ID = existingRepo.ID
		if err := s.db.UpdateRepository(ctx, repo); err != nil {
			return errors.NewRepositoryError(owner, name, "UpdateRepository", err)
		}
	}

	// Get commits since the specified time
	commits, err := s.github.GetCommits(ctx, owner, name, since)
	if err != nil {
		return errors.NewGitHubError("GetCommits", fmt.Sprintf("%s/%s", owner, name), err)
	}

	// Process each commit
	for _, c := range commits {
		commit := &models.Commit{
			RepositoryID:   repo.ID,
			SHA:            c.SHA,
			Message:        c.Commit.Message,
			AuthorName:     c.Commit.Author.Name,
			AuthorEmail:    c.Commit.Author.Email,
			AuthorDate:     c.Commit.Author.Date,
			CommitterName:  c.Commit.Committer.Name,
			CommitterEmail: c.Commit.Committer.Email,
			CommitDate:     c.Commit.Committer.Date,
			URL:            c.HTMLURL,
		}

		// Check if commit exists
		existingCommit, err := s.db.GetCommitsBySHA(ctx, repo.ID, commit.SHA)
		if err != nil {
			return errors.NewCommitError(repo.ID, commit.SHA, "GetCommitsBySHA", err)
		}

		if existingCommit == nil {
			if err := s.db.CreateCommit(ctx, commit); err != nil {
				return errors.NewCommitError(repo.ID, commit.SHA, "CreateCommit", err)
			}
		}
	}

	// Update last commit check time
	if err := s.db.UpdateLastCommitCheck(ctx, repo.ID, time.Now()); err != nil {
		return errors.NewRepositoryError(owner, name, "UpdateLastCommitCheck", err)
	}

	// Update commits since time
	if err := s.db.SetCommitsSince(ctx, repo.ID, since); err != nil {
		return errors.NewRepositoryError(owner, name, "SetCommitsSince", err)
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
func (s *Service) GetCommitsByRepository(ctx context.Context, fullName string, page, perPage int) ([]*models.Commit, int, error) {
	repo, err := s.db.GetRepositoryByName(ctx, fullName)
	if err != nil {
		return nil, 0, fmt.Errorf("error fetching repository: %w", err)
	}
	if repo == nil {
		return nil, 0, fmt.Errorf("repository not found: %s", fullName)
	}

	// Get total count
	totalCount, err := s.db.GetCommitCountByRepository(ctx, repo.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting commit count: %w", err)
	}

	commits, err := s.db.GetCommitsByRepository(ctx, repo.ID, page, perPage)
	if err != nil {
		return nil, 0, fmt.Errorf("error fetching commits: %w", err)
	}

	return commits, totalCount, nil
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
