package github

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGetRepository(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check request method
			if r.Method != "GET" {
				t.Errorf("Expected 'GET' request, got '%s'", r.Method)
			}

			// Check request path
			if r.URL.Path != "/repos/owner/repo" {
				t.Errorf("Expected path '/repos/owner/repo', got '%s'", r.URL.Path)
			}

			// Check headers
			if r.Header.Get("Accept") != "application/vnd.github.v3+json" {
				t.Errorf("Expected Accept header 'application/vnd.github.v3+json', got '%s'", r.Header.Get("Accept"))
			}

			// Return mock response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": 1,
				"name": "repo",
				"full_name": "owner/repo",
				"description": "Test repository",
				"html_url": "https://github.com/owner/repo",
				"language": "Go",
				"forks_count": 10,
				"stargazers_count": 20,
				"watchers_count": 20,
				"open_issues_count": 5,
				"created_at": "2020-01-01T00:00:00Z",
				"updated_at": "2020-01-02T00:00:00Z"
			}`))
		}))
		defer server.Close()

		client := &Client{
			httpClient: server.Client(),
			token:      "test-token",
		}
		baseURL = server.URL

		ctx := context.Background()
		repo, err := client.GetRepository(ctx, "owner", "repo")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Verify response
		if repo.Name != "repo" {
			t.Errorf("Expected name 'repo', got '%s'", repo.Name)
		}
		if repo.FullName != "owner/repo" {
			t.Errorf("Expected full name 'owner/repo', got '%s'", repo.FullName)
		}
		if repo.Description != "Test repository" {
			t.Errorf("Expected description 'Test repository', got '%s'", repo.Description)
		}
		if repo.Language != "Go" {
			t.Errorf("Expected language 'Go', got '%s'", repo.Language)
		}
		if repo.ForksCount != 10 {
			t.Errorf("Expected forks count 10, got %d", repo.ForksCount)
		}
		if repo.StarsCount != 20 {
			t.Errorf("Expected stars count 20, got %d", repo.StarsCount)
		}
	})

	t.Run("rate limit exceeded", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Hour).Unix()))
			w.Header().Set("X-RateLimit-Limit", "60")
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		client := &Client{
			httpClient: server.Client(),
			token:      "test-token",
		}
		baseURL = server.URL

		ctx := context.Background()
		_, err := client.GetRepository(ctx, "owner", "repo")
		if err == nil {
			t.Error("Expected rate limit error, got nil")
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &Client{
			httpClient: server.Client(),
			token:      "test-token",
		}
		baseURL = server.URL

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := client.GetRepository(ctx, "owner", "repo")
		if err == nil {
			t.Error("Expected context deadline exceeded error, got nil")
		}
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`invalid json`))
		}))
		defer server.Close()

		client := &Client{
			httpClient: server.Client(),
			token:      "test-token",
		}
		baseURL = server.URL

		ctx := context.Background()
		_, err := client.GetRepository(ctx, "owner", "repo")
		if err == nil {
			t.Error("Expected JSON decoding error, got nil")
		}
	})
}

func TestGetCommits(t *testing.T) {
	t.Run("successful request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check request method
			if r.Method != "GET" {
				t.Errorf("Expected 'GET' request, got '%s'", r.Method)
			}

			// Check request path
			if r.URL.Path != "/repos/owner/repo/commits" {
				t.Errorf("Expected path '/repos/owner/repo/commits', got '%s'", r.URL.Path)
			}

			// Return mock response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[{
				"sha": "abc123",
				"commit": {
					"author": {
						"name": "Test Author",
						"email": "author@example.com",
						"date": "2020-01-01T00:00:00Z"
					},
					"committer": {
						"name": "Test Committer",
						"email": "committer@example.com",
						"date": "2020-01-01T00:00:00Z"
					},
					"message": "Test commit"
				},
				"html_url": "https://github.com/owner/repo/commit/abc123"
			}]`))
		}))
		defer server.Close()

		client := &Client{
			httpClient: server.Client(),
			token:      "test-token",
		}
		baseURL = server.URL

		ctx := context.Background()
		since := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
		commits, err := client.GetCommits(ctx, "owner", "repo", since)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if len(commits) != 1 {
			t.Fatalf("Expected 1 commit, got %d", len(commits))
		}

		commit := commits[0]
		if commit.SHA != "abc123" {
			t.Errorf("Expected SHA 'abc123', got '%s'", commit.SHA)
		}
		if commit.Commit.Author.Name != "Test Author" {
			t.Errorf("Expected author name 'Test Author', got '%s'", commit.Commit.Author.Name)
		}
		if commit.Commit.Author.Email != "author@example.com" {
			t.Errorf("Expected author email 'author@example.com', got '%s'", commit.Commit.Author.Email)
		}
		if commit.Commit.Message != "Test commit" {
			t.Errorf("Expected message 'Test commit', got '%s'", commit.Commit.Message)
		}
	})

	t.Run("empty commits list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
		}))
		defer server.Close()

		client := &Client{
			httpClient: server.Client(),
			token:      "test-token",
		}
		baseURL = server.URL

		ctx := context.Background()
		since := time.Now().Add(-24 * time.Hour)
		commits, err := client.GetCommits(ctx, "owner", "repo", since)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if len(commits) != 0 {
			t.Errorf("Expected empty commits list, got %d commits", len(commits))
		}
	})
}

func TestRateLimitHandling(t *testing.T) {
	t.Run("rate limit info update", func(t *testing.T) {
		resetTime := time.Now().Add(time.Hour)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-RateLimit-Remaining", "42")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
			w.Header().Set("X-RateLimit-Limit", "60")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 1}`))
		}))
		defer server.Close()

		client := &Client{
			httpClient: server.Client(),
			token:      "test-token",
		}
		baseURL = server.URL

		ctx := context.Background()
		_, err := client.GetRepository(ctx, "owner", "repo")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		info := client.GetRateLimitInfo()
		if info.Remaining != 42 {
			t.Errorf("Expected remaining rate limit 42, got %d", info.Remaining)
		}
		if info.Limit != 60 {
			t.Errorf("Expected rate limit 60, got %d", info.Limit)
		}
	})

	t.Run("rate limit wait", func(t *testing.T) {
		resetTime := time.Now().Add(200 * time.Millisecond)
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount == 1 {
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
				w.Header().Set("X-RateLimit-Limit", "60")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Header().Set("X-RateLimit-Remaining", "59")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id": 1}`))
		}))
		defer server.Close()

		client := &Client{
			httpClient: server.Client(),
			token:      "test-token",
			rateLimit: RateLimitInfo{
				Remaining: 0,
				Reset:     resetTime,
				Limit:     60,
			},
		}
		baseURL = server.URL

		ctx := context.Background()
		start := time.Now()
		_, err := client.GetRepository(ctx, "owner", "repo")
		duration := time.Since(start)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if duration < 200*time.Millisecond {
			t.Errorf("Expected request to wait for rate limit reset, but it completed too quickly")
		}
	})
}
