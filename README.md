# GitHub API Data Fetching and Service

A Go service that monitors GitHub repositories, fetches commit information, and stores it in a PostgreSQL database. The service continuously syncs repository data and provides easy access to commit statistics through a RESTful API.

## Features

- Fetches and stores repository metadata
- Continuously monitors repositories for new commits
- Stores commit information in a PostgreSQL database
- Configurable date range for commit fetching
- RESTful API for managing repositories and retrieving statistics
- Detailed commit analytics and author information
- Swagger/OpenAPI documentation
- Comprehensive database queries for analytics

## Requirements

- Go 1.21 or later
- GitHub Personal Access Token
- PostgreSQL 13 or later

## Documentation

The `/docs` folder contains comprehensive documentation for the service:

- `api.yaml` - OpenAPI/Swagger specification for the REST API endpoints
- `commit_queries.md` - Documentation for commit-related database operations and analytics
- `repository_queries.md` - Documentation for repository management database operations
- `chromium_test.md` - Example implementation using the Chromium repository

## Installation and Setup

1. Clone the repository:

```bash
git clone https://github.com/okeibunor/github-service.git
cd github-service
```

2. Install dependencies:

```bash
go mod download
```

3. Set up environment:

```bash
make setup
```

4. Configure your GitHub token:

   - Copy `.env.example` to `.env`
   - Add your GitHub token to `.env`:
     ```
     GITHUB_SERVICE_GITHUB_TOKEN=your_github_token_here
     ```
   - Or set it directly in your environment:
     ```bash
     export GITHUB_SERVICE_GITHUB_TOKEN=your_github_token_here
     ```

5. Run the service:

```bash
make run
```

## Security Notes

- Never commit your GitHub token to version control
- Use environment variables or `.env` file to manage sensitive credentials
- The `.env` file is automatically ignored by git
- Regularly rotate your GitHub token for better security
- Use the minimum required permissions for your GitHub token

## Configuration

### GitHub Token Setup

1. Create a GitHub Personal Access Token with the following permissions:
   - `repo` (for private repositories)
   - `public_repo` (for public repositories)
2. Token can be created at: https://github.com/settings/tokens
3. Store the token securely and never share it

### Configuration Methods

1. Environment Variables:

   - `GITHUB_SERVICE_GITHUB_TOKEN`: Your GitHub Personal Access Token
   - `DB_HOST`: PostgreSQL host (default: localhost)
   - `DB_PORT`: PostgreSQL port (default: 5432)
   - `DB_USER`: Database user (default: postgres)
   - `DB_PASSWORD`: Database password
   - `DB_NAME`: Database name (default: github_service)

2. Command Line Arguments:
   - `-token`: GitHub API token (can also be set via GITHUB_SERVICE_GITHUB_TOKEN environment variable)
   - `-db`: PostgreSQL connection string
   - `-repo`: Repository to monitor (format: owner/name)
   - `-since`: Start date for commits (format: YYYY-MM-DD)
   - `-interval`: Sync interval (default: 1h)

## API Usage

The service provides a RESTful API for managing repositories and retrieving data. Full API documentation is available in `/docs/api.yaml`.

### Key Endpoints

- `GET /api/v1/repositories` - List all tracked repositories
- `PUT /api/v1/repositories/{owner}/{repo}` - Add a repository to track
- `DELETE /api/v1/repositories/{owner}/{repo}` - Remove a repository
- `GET /api/v1/repositories/{owner}/{repo}/commits` - Get repository commits
- `POST /api/v1/repositories/{owner}/{repo}/sync` - Trigger manual sync

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
