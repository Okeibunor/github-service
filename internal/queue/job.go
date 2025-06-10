package queue

import (
	"encoding/json"
	"time"
)

// JobType represents different types of jobs
type JobType string

const (
	JobTypeSync    JobType = "sync"
	JobTypeResync  JobType = "resync"
	JobTypeCleanup JobType = "cleanup"
)

// JobStatus represents the status of a job
type JobStatus string

const (
	JobStatusPending  JobStatus = "pending"
	JobStatusRunning  JobStatus = "running"
	JobStatusComplete JobStatus = "complete"
	JobStatusFailed   JobStatus = "failed"
	JobStatusStopped  JobStatus = "stopped" // New status for jobs that hit max retries
)

// Default retry configuration
const (
	DefaultMaxRetries     = 3
	DefaultInitialBackoff = 1 * time.Second
	DefaultMaxBackoff     = 1 * time.Hour
	DefaultBackoffFactor  = 2.0
	DefaultJitterFactor   = 0.1
)

// Job represents a background job
type Job struct {
	ID        string          `json:"id"`
	Type      JobType         `json:"type"`
	Status    JobStatus       `json:"status"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Error     string          `json:"error,omitempty"`
	Schedule  string          `json:"schedule,omitempty"` // Cron expression for scheduled jobs

	// Retry configuration
	RetryCount     int           `json:"retry_count"`
	MaxRetries     int           `json:"max_retries"`
	LastRetryAt    time.Time     `json:"last_retry_at,omitempty"`
	NextRetryAt    time.Time     `json:"next_retry_at,omitempty"`
	InitialBackoff time.Duration `json:"initial_backoff"`
}

// SyncPayload represents the payload for sync jobs
type SyncPayload struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
}

// Queue interface defines the methods for job queue operations
type Queue interface {
	Enqueue(job *Job) error
	Dequeue() (*Job, error)
	Complete(jobID string) error
	Fail(jobID string, err error) error
	GetStatus(jobID string) (JobStatus, error)
	GetJobs() ([]*Job, error)
}
