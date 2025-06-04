# GitHub API Data Fetching and Service

A Go service that monitors GitHub repositories, fetches commit information, and stores it in a SQLite database. The service continuously syncs repository data and provides easy access to commit statistics.

## Features

- Fetches and stores repository metadata
- Continuously monitors repositories for new commits
- Stores commit information in a SQLite database
- Configurable date range for commit fetching
- Provides commit statistics and author information
- Efficient querying of stored data

## Requirements

- Go 1.21 or later
- GitHub Personal Access Token
- SQLite3

## Installation

1. Clone the repository:

```bash
git clone https://github.com/okeibunor/github-service.git
cd github-service
```

2. Install dependencies:

```bash
go mod download
```

3. Build the service:

```bash
go build -o github-service ./cmd/github-service
```

## Configuration

The service requires a GitHub Personal Access Token with the following permissions:

- `repo` (for private repositories)
- `public_repo` (for public repositories)

You can create a token at: https://github.com/settings/tokens

## Usage

### Running the Service

```bash
# Set your GitHub token
export GITHUB_TOKEN=your_github_token

# Run the service (example with chromium repository)
./github-service -repo chromium/chromium -since 2024-01-01 -interval 1h
```

### Command Line Arguments

- `-token`: GitHub API token (can also be set via GITHUB_TOKEN environment variable)
- `-db`: Path to SQLite database (default: "github.db")
- `-repo`: Repository to monitor (format: owner/name)
- `-since`: Start date for commits (format: YYYY-MM-DD)
- `-interval`: Sync interval (default: 1h)

### Example Queries

The service stores data in a SQLite database, which you can query directly:

1. Get top 10 commit authors:

```sql
SELECT author_name, author_email, COUNT(*) as commit_count
FROM commits
GROUP BY author_name, author_email
ORDER BY commit_count DESC
LIMIT 10;
```

2. Get commits for a specific repository:

```sql
SELECT c.*
FROM commits c
JOIN repositories r ON c.repository_id = r.id
WHERE r.full_name = 'owner/repo'
ORDER BY c.commit_date DESC
LIMIT 100;
```

## Database Schema

### Repositories Table

```sql
CREATE TABLE repositories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    github_id INTEGER UNIQUE NOT NULL,
    name TEXT NOT NULL,
    full_name TEXT NOT NULL UNIQUE,
    description TEXT,
    url TEXT NOT NULL,
    language TEXT,
    forks_count INTEGER DEFAULT 0,
    stars_count INTEGER DEFAULT 0,
    open_issues_count INTEGER DEFAULT 0,
    watchers_count INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    last_commit_check TIMESTAMP,
    commits_since TIMESTAMP,
    created_at_local TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at_local TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Commits Table

```sql
CREATE TABLE commits (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repository_id INTEGER NOT NULL,
    sha TEXT NOT NULL,
    message TEXT NOT NULL,
    author_name TEXT NOT NULL,
    author_email TEXT NOT NULL,
    author_date TIMESTAMP NOT NULL,
    committer_name TEXT NOT NULL,
    committer_email TEXT NOT NULL,
    commit_date TIMESTAMP NOT NULL,
    url TEXT NOT NULL,
    created_at_local TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (repository_id) REFERENCES repositories(id) ON DELETE CASCADE,
    UNIQUE(repository_id, sha)
);
```

## Testing

Run the tests:

```bash
go test -v ./...
```

Note: Some tests require a GitHub token to be set in the environment.

## Error Handling

The service implements comprehensive error handling:

- Graceful handling of API rate limits
- Automatic retry on transient errors
- Duplicate commit detection
- Database constraint violation handling

## Performance Considerations

- Uses efficient database indexes
- Implements pagination for large result sets
- Caches repository metadata
- Avoids duplicate commit fetching
- Uses batch operations where possible

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
