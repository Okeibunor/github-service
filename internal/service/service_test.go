package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github-service/internal/database"
	"github-service/internal/models"
	"github-service/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *testutil.TestPostgres {
	ctx := context.Background()
	pg, err := testutil.NewTestPostgres(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, pg.Close(ctx))
	})
	return pg
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
				{AuthorName: "author1", Count: 2},
				{AuthorName: "author2", Count: 1},
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
