package queue

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PostgresQueue implements Queue interface using PostgreSQL
type PostgresQueue struct {
	db *sql.DB
}

// NewPostgresQueue creates a new PostgreSQL-based queue
func NewPostgresQueue(db *sql.DB) (*PostgresQueue, error) {
	if err := initializeQueueSchema(db); err != nil {
		return nil, fmt.Errorf("failed to initialize queue schema: %w", err)
	}
	return &PostgresQueue{db: db}, nil
}

func initializeQueueSchema(db *sql.DB) error {
	// First drop the existing table to recreate with the correct schema
	dropSchema := `DROP TABLE IF EXISTS jobs;`
	if _, err := db.Exec(dropSchema); err != nil {
		return err
	}

	schema := `
		CREATE TABLE jobs (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			status TEXT NOT NULL,
			payload JSONB,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			error TEXT,
			schedule TEXT,
			next_run_at TIMESTAMP WITH TIME ZONE,
			retry_count INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 3,
			last_retry_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
			next_retry_at TIMESTAMP WITH TIME ZONE DEFAULT NULL,
			initial_backoff BIGINT NOT NULL DEFAULT 1000000000 -- 1 second in nanoseconds
		);

		CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
		CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(type);
		CREATE INDEX IF NOT EXISTS idx_jobs_next_run ON jobs(next_run_at) WHERE status = 'pending';
		CREATE INDEX IF NOT EXISTS idx_jobs_next_retry ON jobs(next_retry_at) WHERE status = 'failed';
	`
	_, err := db.Exec(schema)
	return err
}

func (q *PostgresQueue) Enqueue(job *Job) error {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = time.Now()
	}
	job.UpdatedAt = time.Now()
	job.Status = JobStatusPending
	job.RetryCount = 0

	// Set default retry configuration
	if job.MaxRetries <= 0 {
		job.MaxRetries = DefaultMaxRetries
	}
	if job.InitialBackoff <= 0 {
		job.InitialBackoff = DefaultInitialBackoff
	}

	query := `
		INSERT INTO jobs (
			id, type, status, payload, created_at, updated_at, error,
			retry_count, max_retries, initial_backoff
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := q.db.Exec(
		query,
		job.ID, job.Type, job.Status, job.Payload, job.CreatedAt, job.UpdatedAt, job.Error,
		job.RetryCount, job.MaxRetries, int64(job.InitialBackoff),
	)
	return err
}

func (q *PostgresQueue) Dequeue() (*Job, error) {
	tx, err := q.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	query := `
		UPDATE jobs
		SET status = $1, updated_at = $2
		WHERE id = (
			SELECT id
			FROM jobs
			WHERE status = $3
			ORDER BY created_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, type, status, payload, created_at, updated_at, error, schedule,
			retry_count, max_retries, last_retry_at, next_retry_at, initial_backoff
	`

	job := &Job{
		MaxRetries:     DefaultMaxRetries,
		InitialBackoff: DefaultInitialBackoff,
	}

	var errMsg sql.NullString
	var schedule sql.NullString
	var payload []byte
	var lastRetryAt, nextRetryAt sql.NullTime
	var initialBackoff sql.NullInt64

	row := tx.QueryRow(query, JobStatusRunning, time.Now(), JobStatusPending)
	err = row.Scan(
		&job.ID,
		&job.Type,
		&job.Status,
		&payload,
		&job.CreatedAt,
		&job.UpdatedAt,
		&errMsg,
		&schedule,
		&job.RetryCount,
		&job.MaxRetries,
		&lastRetryAt,
		&nextRetryAt,
		&initialBackoff,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if len(payload) > 0 {
		job.Payload = json.RawMessage(payload)
	}
	if errMsg.Valid {
		job.Error = errMsg.String
	}
	if schedule.Valid {
		job.Schedule = schedule.String
	}
	if lastRetryAt.Valid {
		job.LastRetryAt = lastRetryAt.Time
	}
	if nextRetryAt.Valid {
		job.NextRetryAt = nextRetryAt.Time
	}
	if initialBackoff.Valid {
		job.InitialBackoff = time.Duration(initialBackoff.Int64)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return job, nil
}

func (q *PostgresQueue) Complete(jobID string) error {
	query := `
		UPDATE jobs
		SET 
			status = $1,
			updated_at = $2
		WHERE id = $3
	`
	_, err := q.db.Exec(query, JobStatusComplete, time.Now(), jobID)
	return err
}

func (q *PostgresQueue) Fail(jobID string, err error) error {
	query := `
		UPDATE jobs
		SET 
			status = $1,
			updated_at = $2,
			error = $3,
			retry_count = COALESCE(retry_count, 0) + 1,
			last_retry_at = $4,
			next_retry_at = $5
		WHERE id = $6
		RETURNING retry_count
	`
	now := time.Now()
	var retryCount int
	row := q.db.QueryRow(query, JobStatusFailed, now, err.Error(), now, now.Add(DefaultInitialBackoff), jobID)
	if scanErr := row.Scan(&retryCount); scanErr != nil {
		return fmt.Errorf("failed to update job status: %w", scanErr)
	}

	// If this was the first retry, update the initial backoff
	if retryCount == 1 {
		_, updateErr := q.db.Exec(`
			UPDATE jobs 
			SET initial_backoff = $1 
			WHERE id = $2 AND retry_count = 1
		`, int64(DefaultInitialBackoff), jobID)
		if updateErr != nil {
			return fmt.Errorf("failed to update initial backoff: %w", updateErr)
		}
	}

	return nil
}

func (q *PostgresQueue) GetStatus(jobID string) (JobStatus, error) {
	query := `
		SELECT status, error 
		FROM jobs 
		WHERE id = $1
	`

	var status JobStatus
	var errMsg sql.NullString

	err := q.db.QueryRow(query, jobID).Scan(&status, &errMsg)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("job not found")
	}
	if err != nil {
		return "", err
	}

	return status, nil
}

// GetJobs retrieves all jobs from the queue
func (q *PostgresQueue) GetJobs() ([]*Job, error) {
	query := `
		SELECT 
			id, type, status, payload, created_at, updated_at, error, schedule,
			retry_count, max_retries, last_retry_at, next_retry_at, initial_backoff
		FROM jobs
		ORDER BY created_at DESC
	`

	rows, err := q.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*Job
	for rows.Next() {
		job := &Job{
			MaxRetries:     DefaultMaxRetries,
			InitialBackoff: DefaultInitialBackoff,
		}

		var errMsg sql.NullString
		var schedule sql.NullString
		var payload []byte
		var lastRetryAt, nextRetryAt sql.NullTime
		var initialBackoff sql.NullInt64

		if err := rows.Scan(
			&job.ID,
			&job.Type,
			&job.Status,
			&payload,
			&job.CreatedAt,
			&job.UpdatedAt,
			&errMsg,
			&schedule,
			&job.RetryCount,
			&job.MaxRetries,
			&lastRetryAt,
			&nextRetryAt,
			&initialBackoff,
		); err != nil {
			return nil, fmt.Errorf("error scanning job: %w", err)
		}

		// Handle nullable fields
		if len(payload) > 0 {
			job.Payload = json.RawMessage(payload)
		}
		if errMsg.Valid {
			job.Error = errMsg.String
		}
		if schedule.Valid {
			job.Schedule = schedule.String
		}
		if lastRetryAt.Valid {
			job.LastRetryAt = lastRetryAt.Time
		}
		if nextRetryAt.Valid {
			job.NextRetryAt = nextRetryAt.Time
		}
		if initialBackoff.Valid {
			job.InitialBackoff = time.Duration(initialBackoff.Int64)
		}

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating jobs: %w", err)
	}

	return jobs, nil
}
