package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var baseURL = "https://api.github.com"

// RateLimitInfo stores GitHub API rate limit information
type RateLimitInfo struct {
	Remaining int
	Reset     time.Time
	Limit     int
}

// GitHubClient defines the interface for GitHub operations
type GitHubClient interface {
	GetRepository(ctx context.Context, owner, repo string) (*Repository, error)
	GetCommits(ctx context.Context, owner, repo string, since time.Time) ([]CommitResponse, error)
	GetRateLimitInfo() RateLimitInfo
}

// Client handles interactions with the GitHub API
type Client struct {
	httpClient *http.Client
	token      string

	// Rate limiting
	rateLimitMu sync.RWMutex
	rateLimit   RateLimitInfo
}

// NewClient creates a new GitHub API client
func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: time.Second * 30,
		},
		token: token,
		rateLimit: RateLimitInfo{
			Remaining: 60, // Default GitHub API limit
			Reset:     time.Now().Add(time.Hour),
			Limit:     60,
		},
	}
}

// Repository represents the GitHub repository response
type Repository struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	FullName        string    `json:"full_name"`
	Description     string    `json:"description"`
	URL             string    `json:"html_url"`
	Language        string    `json:"language"`
	ForksCount      int       `json:"forks_count"`
	StargazersCount int       `json:"stargazers_count"`
	WatchersCount   int       `json:"watchers_count"`
	OpenIssuesCount int       `json:"open_issues_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CommitResponse represents the GitHub commit response
type CommitResponse struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
		Committer struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"committer"`
		Message string `json:"message"`
	} `json:"commit"`
	HTMLURL string `json:"html_url"`
}

// GetRateLimitInfo returns current rate limit information
func (c *Client) GetRateLimitInfo() RateLimitInfo {
	c.rateLimitMu.RLock()
	defer c.rateLimitMu.RUnlock()
	return c.rateLimit
}

// updateRateLimit updates rate limit information from response headers
func (c *Client) updateRateLimit(resp *http.Response) {
	c.rateLimitMu.Lock()
	defer c.rateLimitMu.Unlock()

	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			c.rateLimit.Remaining = val
		}
	}

	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if val, err := strconv.ParseInt(reset, 10, 64); err == nil {
			c.rateLimit.Reset = time.Unix(val, 0)
		}
	}

	if limit := resp.Header.Get("X-RateLimit-Limit"); limit != "" {
		if val, err := strconv.Atoi(limit); err == nil {
			c.rateLimit.Limit = val
		}
	}
}

// checkRateLimit checks if we should wait due to rate limiting
func (c *Client) checkRateLimit(ctx context.Context) error {
	c.rateLimitMu.RLock()
	defer c.rateLimitMu.RUnlock()

	if c.rateLimit.Remaining == 0 {
		waitTime := time.Until(c.rateLimit.Reset)
		if waitTime > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
				return nil
			}
		}
	}
	return nil
}

// doRequest performs an HTTP request with rate limit handling
func (c *Client) doRequest(req *http.Request) (*http.Response, error) {
	if err := c.checkRateLimit(req.Context()); err != nil {
		return nil, fmt.Errorf("rate limit check: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	c.updateRateLimit(resp)

	if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0" {
		return nil, fmt.Errorf("rate limit exceeded, resets at %v", c.rateLimit.Reset)
	}

	return resp, nil
}

// GetRepository fetches repository information from GitHub
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	url := fmt.Sprintf("%s/repos/%s/%s", baseURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)
	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var repository Repository
	if err := json.NewDecoder(resp.Body).Decode(&repository); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &repository, nil
}

// GetCommits fetches commits from GitHub since a specific time
func (c *Client) GetCommits(ctx context.Context, owner, repo string, since time.Time) ([]CommitResponse, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/commits?since=%s", baseURL, owner, repo, since.Format(time.RFC3339))
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.setHeaders(req)
	resp, err := c.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var commits []CommitResponse
	if err := json.NewDecoder(resp.Body).Decode(&commits); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return commits, nil
}

// setHeaders sets the required headers for GitHub API requests
func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if c.token != "" {
		req.Header.Set("Authorization", "token "+c.token)
	}
}
