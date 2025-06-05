# Chromium Repository Test Case

This document provides detailed information about the Chromium repository integration test case implemented in `internal/service/chromium_test.go`.

## Overview

The Chromium repository test case serves as a real-world integration test for our GitHub service. It uses the actual Chromium repository data to verify our service's ability to handle large, active repositories with multiple contributors.

## Test Components

### 1. Data Seeding (`internal/testutil/seeder.go`)

```go
func SeedChromiumData(ctx context.Context, db *database.DB, githubToken string) error
```

This function:

- Fetches Chromium repository metadata
- Retrieves recent commits (last 7 days)
- Stores data in the test database
- Handles data transformation between GitHub API and our models

### 2. Test Cases (`internal/service/chromium_test.go`)

The test suite includes three main test cases:

#### a. Repository Data Retrieval

```go
t.Run("GetCommitsByRepository", func(t *testing.T) {
    commits, err := svc.GetCommitsByRepository(ctx, "chromium/chromium", 10, 0)
    // ...
})
```

- Tests pagination functionality
- Verifies commit data completeness
- Checks data consistency across pages

#### b. Author Statistics

```go
t.Run("GetTopCommitAuthors", func(t *testing.T) {
    authors, err := svc.GetTopCommitAuthors(ctx, 5)
    // ...
})
```

- Analyzes contributor statistics
- Verifies author ordering by commit count
- Validates author information completeness

#### c. Repository Statistics

```go
t.Run("RepositoryStats", func(t *testing.T) {
    repo, authors, err := testutil.GetChromiumStats(ctx, db)
    // ...
})
```

- Checks repository metadata accuracy
- Displays comprehensive statistics
- Provides insights into repository activity

## Database Schema

The test uses our standard schema with PostgreSQL-specific optimizations:

```sql
CREATE TABLE repositories (
    id SERIAL PRIMARY KEY,
    github_id BIGINT UNIQUE NOT NULL,
    -- ...
);

CREATE TABLE commits (
    id SERIAL PRIMARY KEY,
    repository_id INTEGER NOT NULL,
    -- ...
);
```

## Running the Test

### Prerequisites

- Docker running
- GitHub API token with public repo access
- Go 1.21+

### Execution

```bash
# Set GitHub token
export GITHUB_TOKEN=your_token_here

# Run only Chromium tests
go test -v ./internal/service -run TestChromiumRepositoryAnalysis

# Run with increased verbosity
go test -v -count=1 ./internal/service -run TestChromiumRepositoryAnalysis
```

## Test Data Analysis

### Repository Metrics

Based on recent test runs:

- Stars: ~20,800
- Forks: ~7,600
- Active contributors: 10+ per week
- Commit frequency: Multiple commits per day

### Interesting Findings

1. Automated Contributions

   - `chromium-autoroll` bot is often among top contributors
   - Indicates automated processes in the workflow

2. Contribution Patterns

   - Mix of Google and non-Google contributors
   - Regular activity from core team members
   - Diverse geographical distribution of contributors

3. Commit Characteristics
   - Detailed commit messages
   - Multiple authors and committers
   - Consistent commit frequency

## Maintenance and Updates

### Regular Maintenance

1. Update expected statistics ranges periodically
2. Monitor GitHub API usage and rate limits
3. Review and update test assertions based on repository changes

### Known Limitations

1. API rate limiting may affect test reliability
2. Test duration depends on API response times
3. Results vary based on recent repository activity

### Future Improvements

1. Cache API responses for faster test execution
2. Add more detailed commit content analysis
3. Implement parallel test execution
4. Add historical data comparison

## Troubleshooting

### Common Issues

1. **API Rate Limiting**

   ```
   Error: API rate limit exceeded
   ```

   Solution: Wait for rate limit reset or use authenticated requests

2. **Data Inconsistency**

   ```
   Error: unexpected number of commits
   ```

   Solution: Check the time window for commit fetching

3. **Database Connection**
   ```
   Error: failed to connect to database
   ```
   Solution: Verify Docker is running and ports are available

## References

1. [Chromium Repository](https://github.com/chromium/chromium)
2. [GitHub API Documentation](https://docs.github.com/en/rest)
3. [testcontainers-go Documentation](https://golang.testcontainers.org)
