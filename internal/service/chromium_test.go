package service

import (
	"context"
	"os"
	"testing"

	"github-service/internal/database"
	"github-service/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChromiumRepositoryAnalysis(t *testing.T) {
	// Skip if no GitHub token is provided
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Skipping test: GITHUB_TOKEN not set")
	}

	// Set up test database
	ctx := context.Background()
	pg := setupTestDB(t)

	// Create database wrapper
	db := database.NewFromDB(pg.DB)

	// Seed Chromium data
	err := testutil.SeedChromiumData(ctx, db, token)
	require.NoError(t, err, "Failed to seed Chromium data")

	// Create service instance
	svc := &Service{
		db: db,
	}

	t.Run("GetCommitsByRepository", func(t *testing.T) {
		// Test fetching commits with pagination
		commits, err := svc.GetCommitsByRepository(ctx, "chromium/chromium", 10, 0)
		require.NoError(t, err)
		assert.NotEmpty(t, commits)

		// Verify commit data
		for _, commit := range commits {
			assert.NotEmpty(t, commit.SHA)
			assert.NotEmpty(t, commit.Message)
			assert.NotEmpty(t, commit.AuthorName)
			assert.NotEmpty(t, commit.AuthorEmail)
			assert.False(t, commit.AuthorDate.IsZero())
			assert.NotEmpty(t, commit.CommitterName)
			assert.NotEmpty(t, commit.CommitterEmail)
			assert.False(t, commit.CommitDate.IsZero())
			assert.NotEmpty(t, commit.URL)
		}

		// Test pagination
		nextCommits, err := svc.GetCommitsByRepository(ctx, "chromium/chromium", 10, 10)
		require.NoError(t, err)
		assert.NotEmpty(t, nextCommits)
		assert.NotEqual(t, commits[0].SHA, nextCommits[0].SHA)
	})

	t.Run("GetTopCommitAuthors", func(t *testing.T) {
		// Get top 5 contributors
		authors, err := svc.GetTopCommitAuthors(ctx, 5)
		require.NoError(t, err)
		assert.NotEmpty(t, authors)

		// Verify author stats
		for _, author := range authors {
			assert.NotEmpty(t, author.AuthorName)
			assert.NotEmpty(t, author.AuthorEmail)
			assert.Greater(t, author.Count, 0)
		}

		// Verify ordering
		for i := 1; i < len(authors); i++ {
			assert.GreaterOrEqual(t, authors[i-1].Count, authors[i].Count)
		}
	})

	t.Run("RepositoryStats", func(t *testing.T) {
		// Get repository info and stats
		repo, authors, err := testutil.GetChromiumStats(ctx, db)
		require.NoError(t, err)

		// Verify repository info
		assert.Equal(t, "chromium/chromium", repo.FullName)
		assert.NotEmpty(t, repo.Description)
		assert.NotEmpty(t, repo.URL)
		assert.NotZero(t, repo.ForksCount)
		assert.NotZero(t, repo.StarsCount)
		assert.NotZero(t, repo.WatchersCount)

		// Print repository statistics
		t.Logf("Repository: %s", repo.FullName)
		t.Logf("Description: %s", repo.Description)
		t.Logf("Stars: %d, Forks: %d, Watchers: %d",
			repo.StarsCount, repo.ForksCount, repo.WatchersCount)
		t.Logf("\nTop contributors in the last 7 days:")
		for _, author := range authors {
			t.Logf("- %s <%s>: %d commits",
				author.AuthorName, author.AuthorEmail, author.Count)
		}
	})
}
