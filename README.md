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
   - Add your GitHub token and other required environment variables to or set it directly in your environment:
     e.g

     ````bash
     export GITHUB_SERVICE_GITHUB_TOKEN=your_github_token_here
     ``` `.env`:

     ````

     # Environment Variables

     # GitHub API Token (required)

     GITHUB_SERVICE_GITHUB_TOKEN=your_github_token_here

     # Database Configuration (required)

     DB_HOST=<host>
     DB_PORT=<port>
     DB_USER=<user>
     DB_PASSWORD=<password>
     DB_NAME=<db>
     DB_SSLMODE=require

     # Server Configuration

     SERVER_PORT=8080

     # GitHub Service Configuration (optional)

     GITHUB_SERVICE_MONITOR_INTERVAL=1h
     GITHUB_SERVICE_MONITOR_ENABLED=true
     GITHUB_SERVICE_LOG_LEVEL=info
     GITHUB_SERVICE_LOG_FORMAT=json

     # GitHub API Configuration (optional)

     GITHUB_SERVICE_GITHUB_RATE_LIMIT=1s
     GITHUB_SERVICE_GITHUB_REQUEST_TIMEOUT=30s
     GITHUB_SERVICE_GITHUB_MAX_RETRIES=3
     GITHUB_SERVICE_GITHUB_RETRY_BACKOFF=2s

     # Log Configuration (optional)

     LOG_LEVEL=info
     LOG_FORMAT=json

     ```

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

The service can be configured using environment variables or a configuration file. For security, sensitive information like database credentials and API tokens should be provided through environment variables.

### Required Environment Variables

```bash
# Database Configuration
DB_HOST=localhost        # Database host
DB_PORT=5432            # Database port
DB_USER=postgres        # Database user
DB_PASSWORD=<secret>    # Database password
DB_NAME=github_service  # Database name
DB_SSLMODE=require     # Database SSL mode

# GitHub Configuration
GITHUB_TOKEN=<secret>   # GitHub Personal Access Token

# Optional Environment Variables
MONITOR_INTERVAL=1h     # Repository sync interval
LOG_LEVEL=info         # Logging level (debug, info, warn, error)
LOG_FORMAT=json        # Logging format (json, text)
```

### Configuration File

The service also supports configuration through a YAML file. Create a copy of `config.template.yaml` and modify it according to your needs:

```bash
cp config.template.yaml config.yaml
```

Note: Environment variables take precedence over values in the configuration file.

### Security Best Practices

1. Never commit sensitive information (passwords, tokens) to version control
2. Use environment variables for secrets in production
3. Keep the config.yaml file in .gitignore
4. Use strong, unique passwords for database access
5. Create a dedicated GitHub token with minimal required permissions

## Getting Started

1. Copy the configuration template:

   ```bash
   cp config.template.yaml config.yaml
   ```

2. Set up your environment variables:

   ```bash
   export DB_USER=your_db_user
   export DB_PASSWORD=your_db_password
   export DB_PORT=your_db_port
   export DB_USER=your_db_username
   export DB_NAME=your_db_name
   export DB_SSLMODE=require

   export GITHUB_TOKEN=your_github_token
   ```

3. Start the service:
   ```bash
   go run cmd/github-service/main.go
   ```

## API Documentation

The service provides a RESTful API for managing repository synchronization. See the OpenAPI documentation in `docs/api.yaml` for details.

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
