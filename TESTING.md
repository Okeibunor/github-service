# GitHub Service Testing Documentation

This document provides an overview of the testing infrastructure. For specific test case documentation, see the `docs` directory.

## Test Infrastructure

### Database Testing

The project uses PostgreSQL for both production and testing environments, managed through `testcontainers-go`. This ensures test consistency with the production environment.

#### Test Database Setup

- Uses `postgres:16-alpine` container
- Automatically manages container lifecycle
- Provides isolated test environment
- Handles schema initialization
- Supports transaction-based test isolation

### Test Data Management

#### Fixtures

The project uses YAML fixtures for basic test data, located in `internal/testutil/fixtures/`:

- `repositories.yml`: Repository metadata
- `commits.yml`: Commit information

## Running Tests

### Prerequisites

1. Docker installed and running
2. Go 1.21 or later
3. GitHub API token with repo access

### Environment Setup

```bash
# Set your GitHub token
export GITHUB_TOKEN=your_token_here
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run specific test suite
go test -v ./internal/service -run TestName
```

## Test Categories

### 1. Unit Tests

Basic tests using fixtures for predictable data.

### 2. Integration Tests

Tests that interact with real GitHub repositories. See specific documentation:

- [Chromium Repository Test Case](docs/chromium_test.md)

## Best Practices

1. **Test Isolation**

   - Each test runs in its own database container
   - Uses transactions for data isolation
   - Automatic cleanup after tests

2. **Real Data Testing**
   - Use real repositories for integration tests
   - Capture actual GitHub API behavior
   - Test with realistic data volumes

## Troubleshooting

### Common Issues

1. **SSL Connection Errors**

   - Solution: Add `sslmode=disable` to PostgreSQL connection string
   - Location: `internal/testutil/postgres.go`

2. **Docker Connectivity**

   - Ensure Docker daemon is running
   - Check Docker socket permissions

3. **GitHub API Rate Limits**
   - Use authenticated requests
   - Consider implementing rate limiting in tests
   - Monitor API quota in CI/CD pipelines

## Contributing

When adding new tests:

1. Follow existing patterns for database setup
2. Use transactions for data modifications
3. Clean up resources in test cleanup
4. Document new test helpers or fixtures
5. Consider both unit and integration test coverage
6. Add detailed documentation for significant test cases in the `docs` directory
