package testutil

import (
	"context"
	"fmt"
	"time"

	"github-service/internal/database"
	"github-service/internal/github"
	"github-service/internal/models"
)

// SeedChromiumData fetches and stores data from the Chromium repository
func SeedChromiumData(ctx context.Context, db *database.DB, githubToken string) error {
	client := github.NewClient(githubToken)

	// Fetch Chromium repository data
	repo, err := client.GetRepository(ctx, "chromium", "chromium")
	if err != nil {
		return fmt.Errorf("failed to fetch Chromium repository: %w", err)
	}

	// Store repository data
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
		CommitsSince:    &time.Time{},
	}

	if err := db.CreateRepository(ctx, dbRepo); err != nil {
		return fmt.Errorf("failed to create repository record: %w", err)
	}

	// Fetch recent commits (last 7 days)
	since := time.Now().AddDate(0, 0, -7)
	commits, err := client.GetCommits(ctx, "chromium", "chromium", since)
	if err != nil {
		return fmt.Errorf("failed to fetch Chromium commits: %w", err)
	}

	// Store commit data
	for _, commit := range commits {
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

		if err := db.CreateCommit(ctx, dbCommit); err != nil {
			return fmt.Errorf("failed to create commit record: %w", err)
		}
	}

	return nil
}

// GetChromiumStats returns statistics about the seeded Chromium data
func GetChromiumStats(ctx context.Context, db *database.DB) (*models.Repository, []*models.CommitStats, error) {
	// Get repository info
	repo, err := db.GetRepositoryByName(ctx, "chromium/chromium")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	// Get top commit authors
	authors, err := db.GetTopCommitAuthors(ctx, 10)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get top authors: %w", err)
	}

	return repo, authors, nil
}
