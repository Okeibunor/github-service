# Database Operations: Commit Queries

This document outlines how to perform common commit-related database operations in the GitHub Service.

## Get Top N Commit Authors

Retrieve the most active contributors based on their commit counts. This can be done either globally across all repositories or for a specific repository.

### Function Signatures

```go
// Get top authors across all repositories
func (db *DB) GetTopCommitAuthors(ctx context.Context, limit int) ([]*models.CommitStats, error)

// Get top authors for a specific repository
func (db *DB) GetTopCommitAuthorsByRepository(ctx context.Context, repoID int64, limit int) ([]*models.CommitStats, error)
```

### Usage Examples

```go
// Get top 10 commit authors across all repositories
authors, err := db.GetTopCommitAuthors(ctx, 10)
if err != nil {
    return fmt.Errorf("failed to get top authors: %w", err)
}

// Get top 10 commit authors for a specific repository
authors, err := db.GetTopCommitAuthorsByRepository(ctx, repositoryID, 10)
if err != nil {
    return fmt.Errorf("failed to get repository top authors: %w", err)
}
```

### HTTP API Usage

```bash
# Get global top authors
curl -X GET "http://api/top-authors?limit=10"

# Get repository-specific top authors
curl -X GET "http://api/top-authors?repository=owner/repo-name&limit=10"
```

### Implementation Details

The global query uses a GROUP BY clause to aggregate commits by author and orders them by commit count in descending order:

```sql
SELECT author_name, author_email, COUNT(*) as commit_count
FROM commits
GROUP BY author_name, author_email
ORDER BY commit_count DESC
LIMIT $1
```

The repository-specific query adds a WHERE clause to filter by repository:

```sql
SELECT author_name, author_email, COUNT(*) as commit_count
FROM commits
WHERE repository_id = $1
GROUP BY author_name, author_email
ORDER BY commit_count DESC
LIMIT $2
```

### Performance Considerations

- Both queries utilize the `idx_commits_author` index on `(author_name, author_email)`
- The repository-specific query also uses the `repository_id` column which is part of the primary key
- Results are limited by the input parameter to prevent excessive memory usage
- The operations are read-only and can be executed concurrently

## Get Repository Commits

Retrieve commits for a specific repository with pagination support.

### Function Signature

```go
func (db *DB) GetCommitsByRepository(ctx context.Context, repoID int64, limit, offset int) ([]*models.Commit, error)
```

### Usage Example

```go
// Get the first 20 commits for a repository
commits, err := db.GetCommitsByRepository(ctx, repositoryID, 20, 0)
if err != nil {
    return fmt.Errorf("failed to get repository commits: %w", err)
}
```

### Implementation Details

The query retrieves commits for a specific repository, ordered by commit date:

```sql
SELECT * FROM commits
WHERE repository_id = $1
ORDER BY commit_date DESC
LIMIT $2 OFFSET $3
```

### Performance Considerations

- Uses the `idx_commits_repository_date` composite index on `(repository_id, commit_date DESC)`
- Pagination prevents memory issues when dealing with repositories with many commits
- The `repository_id` foreign key ensures data integrity
- Results are ordered by commit date for chronological consistency

## Database Schema

The relevant table schema for these operations:

```sql
CREATE TABLE commits (
    id SERIAL PRIMARY KEY,
    repository_id INTEGER NOT NULL,
    sha TEXT NOT NULL,
    message TEXT NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_date TIMESTAMP WITH TIME ZONE NOT NULL,
    committer_name TEXT NOT NULL,
    committer_email TEXT NOT NULL,
    commit_date TIMESTAMP WITH TIME ZONE NOT NULL,
    url TEXT NOT NULL,
    created_at_local TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
    UNIQUE(repository_id, sha)
);

-- Indexes
CREATE INDEX idx_commits_repository_date ON commits(repository_id, commit_date DESC);
CREATE INDEX idx_commits_author ON commits(author_name, author_email);
```
