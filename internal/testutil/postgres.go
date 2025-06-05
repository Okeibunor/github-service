package testutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestPostgres struct {
	Container *postgres.PostgresContainer
	DB        *sql.DB
	DSN       string
	Fixtures  *testfixtures.Loader
}

// NewTestPostgres creates a new PostgreSQL container for testing
func NewTestPostgres(ctx context.Context) (*TestPostgres, error) {
	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.WithInitScripts(), // Empty to skip default init scripts
		testcontainers.WithEnv(map[string]string{
			"POSTGRES_HOST_AUTH_METHOD": "trust",
			"POSTGRES_DB":               "testdb",
			"POSTGRES_USER":             "test",
			"POSTGRES_PASSWORD":         "test",
		}),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start postgres container: %w", err)
	}

	// Get the container's connection details
	dsn, err := pgContainer.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	// Add sslmode=disable to the connection string
	dsn += " sslmode=disable"

	// Connect to the database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Initialize schema
	_, filename, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(filename), "..", "database", "schema.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	if _, err := db.ExecContext(ctx, string(schema)); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Initialize fixtures loader
	fixturesPath := filepath.Join(filepath.Dir(filename), "fixtures")
	fixtures, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect("postgres"),
		testfixtures.Directory(fixturesPath),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize fixtures: %w", err)
	}

	return &TestPostgres{
		Container: pgContainer,
		DB:        db,
		DSN:       dsn,
		Fixtures:  fixtures,
	}, nil
}

// Close cleans up the test database resources
func (tp *TestPostgres) Close(ctx context.Context) error {
	if tp.DB != nil {
		if err := tp.DB.Close(); err != nil {
			return fmt.Errorf("failed to close database connection: %w", err)
		}
	}

	if tp.Container != nil {
		if err := tp.Container.Terminate(ctx); err != nil {
			return fmt.Errorf("failed to terminate container: %w", err)
		}
	}

	return nil
}

// LoadFixtures loads all fixtures into the database
func (tp *TestPostgres) LoadFixtures() error {
	return tp.Fixtures.Load()
}

// RunWithinTransaction runs the given function within a transaction and rolls back afterward
func (tp *TestPostgres) RunWithinTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := tp.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	return nil
}
