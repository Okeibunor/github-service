# Database Operations: Repository Queries

This document outlines how to perform repository-related database operations in the GitHub Service.

## Get Repository by Name

Retrieve a repository's details using its full name (owner/repo format).

### Function Signature

```go
func (db *DB) GetRepositoryByName(ctx context.Context, fullName string) (*models.Repository, error)
```

### Usage Example

```go
// Get repository details
repo, err := db.GetRepositoryByName(ctx, "owner/repo-name")
if err != nil {
    return fmt.Errorf("failed to get repository: %w", err)
}
```

### Implementation Details

The query retrieves all repository fields using the unique full name:

```sql
SELECT * FROM repositories WHERE full_name = $1
```

### Performance Considerations

- Uses the unique index on `full_name` column for efficient lookups
- Returns `nil, nil` if the repository is not found
- The operation is read-only and can be executed concurrently

## Database Schema

The relevant table schema for these operations:

```sql
CREATE TABLE repositories (
    id SERIAL PRIMARY KEY,
    github_id BIGINT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL UNIQUE,
    description TEXT,
    url TEXT NOT NULL,
    language TEXT,
    forks_count INTEGER DEFAULT 0,
    stars_count INTEGER DEFAULT 0,
    open_issues_count INTEGER DEFAULT 0,
    watchers_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    last_commit_check TIMESTAMP WITH TIME ZONE,
    commits_since TIMESTAMP WITH TIME ZONE,
    created_at_local TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at_local TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_repositories_name ON repositories(name, full_name);
```

## API Integration

The repository queries are typically used in conjunction with the GitHub API client. Here's how they work together:

1. The API client fetches repository data from GitHub
2. The data is mapped to our internal `models.Repository` struct
3. The repository is either created or updated in our database
4. Subsequent queries can use the stored data without hitting GitHub's API

### Example Integration Flow

```go
// Fetch from GitHub API
repo, err := githubClient.GetRepository(ctx, owner, name)
if err != nil {
    return fmt.Errorf("failed to fetch from GitHub: %w", err)
}

// Store in our database
dbRepo := &models.Repository{
    GitHubID: repo.ID,
    Name:     repo.Name,
    FullName: repo.FullName,
    // ... other fields
}

// Check if repository exists
existing, err := db.GetRepositoryByName(ctx, repo.FullName)
if err != nil {
    return fmt.Errorf("checking existing repository: %w", err)
}

if existing != nil {
    // Update existing repository
    err = db.UpdateRepository(ctx, dbRepo)
} else {
    // Create new repository
    err = db.CreateRepository(ctx, dbRepo)
}
```
