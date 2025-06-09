package service

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github-service/internal/database"
	"github-service/internal/github"
	"github-service/internal/models"
	"github-service/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CommitStats represents statistics about commits for testing
type CommitStats struct {
	AuthorName string
	Count      int
}

func setupTestDB(t *testing.T) *testutil.TestPostgres {
	ctx := context.Background()
	pg, err := testutil.NewTestPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, pg.Close(ctx))
	})
	return pg
}

func TestService_SyncRepository(t *testing.T) {
	// Skip if no GitHub token is provided
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		t.Skip("Skipping test: GITHUB_TOKEN not set")
	}

	pg := setupTestDB(t)
	require.NoError(t, pg.LoadFixtures())

	// Create service
	svc := &Service{
		db:     database.NewFromDB(pg.DB),
		github: github.NewClient(token),
	}

	// Test syncing a repository
	ctx := context.Background()
	since := time.Now().AddDate(0, 0, -7) // Last 7 days

	// Use a small, public repository for testing
	err := svc.SyncRepository(ctx, "golang", "example", since)
	require.NoError(t, err)

	// Test getting commits
	commits, err := svc.GetCommitsByRepository(ctx, "golang/example", 10, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, commits)

	// Test getting top authors
	authors, err := svc.GetTopCommitAuthors(ctx, 5)
	require.NoError(t, err)
	assert.NotEmpty(t, authors)
}

// MockGitHubClient implements the minimal GitHub client interface for testing
type MockGitHubClient struct {
	getRepoErr    error
	getCommitsErr error
}

func (m *MockGitHubClient) GetRepository(ctx context.Context, owner, name string) (*models.Repository, error) {
	if m.getRepoErr != nil {
		return nil, m.getRepoErr
	}
	return &models.Repository{
		GitHubID:        1,
		Name:            name,
		FullName:        owner + "/" + name,
		Description:     "Test repo",
		URL:             "https://github.com/" + owner + "/" + name,
		Language:        "Go",
		ForksCount:      0,
		StarsCount:      0,
		OpenIssuesCount: 0,
		WatchersCount:   0,
		CreatedAt:       time.Now().Add(-24 * time.Hour),
		UpdatedAt:       time.Now(),
	}, nil
}

func (m *MockGitHubClient) GetCommits(ctx context.Context, owner, name string, since time.Time) ([]models.CommitResponse, error) {
	if m.getCommitsErr != nil {
		return nil, m.getCommitsErr
	}

	commit := models.CommitResponse{
		SHA:     "abc123",
		HTMLURL: "https://github.com/test/test/commit/abc123",
	}
	commit.Commit.Message = "Test commit"
	commit.Commit.Author = models.CommitAuthor{
		Name:  "Test Author",
		Email: "test@example.com",
		Date:  time.Now(),
	}
	commit.Commit.Committer = models.CommitAuthor{
		Name:  "Test Committer",
		Email: "test@example.com",
		Date:  time.Now(),
	}

	return []models.CommitResponse{commit}, nil
}

func (m *MockGitHubClient) GetRateLimitInfo() models.RateLimitInfo {
	return models.RateLimitInfo{
		Remaining: 1000,
		Limit:     5000,
		Reset:     time.Now().Add(time.Hour),
	}
}

func TestSyncRepository(t *testing.T) {
	pg := setupTestDB(t)
	require.NoError(t, pg.LoadFixtures())

	tests := []struct {
		name    string
		owner   string
		repo    string
		since   time.Time
		wantErr bool
		setup   func(*testing.T) (*database.DB, *MockGitHubClient)
	}{
		{
			name:    "Valid repository sync",
			owner:   "testowner",
			repo:    "testrepo",
			since:   time.Now().Add(-24 * time.Hour),
			wantErr: false,
			setup: func(t *testing.T) (*database.DB, *MockGitHubClient) {
				return database.NewFromDB(pg.DB), &MockGitHubClient{}
			},
		},
		{
			name:    "Invalid repository",
			owner:   "nonexistent",
			repo:    "nonexistent",
			since:   time.Now().Add(-24 * time.Hour),
			wantErr: true,
			setup: func(t *testing.T) (*database.DB, *MockGitHubClient) {
				return database.NewFromDB(pg.DB), &MockGitHubClient{
					getRepoErr: fmt.Errorf("repository not found"),
				}
			},
		},
		{
			name:    "Rate limited",
			owner:   "testowner",
			repo:    "testrepo",
			since:   time.Now().Add(-24 * time.Hour),
			wantErr: true,
			setup: func(t *testing.T) (*database.DB, *MockGitHubClient) {
				return database.NewFromDB(pg.DB), &MockGitHubClient{
					getRepoErr: fmt.Errorf("API rate limit exceeded"),
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mockClient := tt.setup(t)
			svc := &Service{
				db:     db,
				github: mockClient,
			}

			err := svc.SyncRepository(context.Background(), tt.owner, tt.repo, tt.since)
			if (err != nil) != tt.wantErr {
				t.Errorf("SyncRepository() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetTopCommitAuthors(t *testing.T) {
	pg := setupTestDB(t)
	require.NoError(t, pg.LoadFixtures())

	tests := []struct {
		name    string
		limit   int
		want    []models.CommitStats
		wantErr bool
	}{
		{
			name:  "Get top 3 authors",
			limit: 3,
			want: []models.CommitStats{
				{AuthorName: "author1", Count: 2}, // author1 has 2 commits in fixtures
				{AuthorName: "author2", Count: 1}, // author2 has 1 commit in fixtures
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &Service{
				db: database.NewFromDB(pg.DB),
			}

			got, err := svc.GetTopCommitAuthors(context.Background(), tt.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTopCommitAuthors() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.Equal(t, len(tt.want), len(got), "expected %d authors, got %d", len(tt.want), len(got))
				for i, want := range tt.want {
					assert.Equal(t, want.AuthorName, got[i].AuthorName)
					assert.Equal(t, want.Count, got[i].Count)
				}
			}
		})
	}
}

// Helper function to compare CommitStats slices
func compareCommitStats(a []*models.CommitStats, b []models.CommitStats) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].AuthorName != b[i].AuthorName || a[i].Count != b[i].Count {
			return false
		}
	}
	return true
}
