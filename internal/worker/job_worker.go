package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github-service/internal/queue"
	"github-service/internal/service"

	"github.com/rs/zerolog"
)

// JobWorker processes jobs from the queue
type JobWorker struct {
	queue   queue.Queue
	service *service.Service
	log     zerolog.Logger
	stop    chan struct{}
}

// NewJobWorker creates a new job worker
func NewJobWorker(queue queue.Queue, service *service.Service, log zerolog.Logger) *JobWorker {
	return &JobWorker{
		queue:   queue,
		service: service,
		log:     log,
		stop:    make(chan struct{}),
	}
}

// calculateBackoff calculates the next retry backoff duration with jitter
func (w *JobWorker) calculateBackoff(job *queue.Job) time.Duration {
	if job.InitialBackoff == 0 {
		job.InitialBackoff = queue.DefaultInitialBackoff
	}

	backoff := float64(job.InitialBackoff) * math.Pow(queue.DefaultBackoffFactor, float64(job.RetryCount))

	// Add jitter
	jitter := rand.Float64() * queue.DefaultJitterFactor * backoff
	backoff = backoff + jitter

	// Cap at max backoff
	if backoff > float64(queue.DefaultMaxBackoff) {
		backoff = float64(queue.DefaultMaxBackoff)
	}

	return time.Duration(backoff)
}

// Start starts the job worker
func (w *JobWorker) Start(ctx context.Context) error {
	w.log.Info().Msg("Starting job worker")

	for {
		select {
		case <-ctx.Done():
			w.log.Info().Msg("Job worker stopped")
			return nil
		case <-w.stop:
			w.log.Info().Msg("Job worker stopped")
			return nil
		default:
			if err := w.processNextJob(ctx); err != nil {
				w.log.Error().Err(err).Msg("Failed to process job")
			}
			// Small delay to prevent tight loop
			time.Sleep(time.Second)
		}
	}
}

// Stop stops the job worker
func (w *JobWorker) Stop() {
	close(w.stop)
}

// processNextJob processes the next job in the queue
func (w *JobWorker) processNextJob(ctx context.Context) error {
	job, err := w.queue.Dequeue()
	if err != nil {
		return fmt.Errorf("failed to dequeue job: %w", err)
	}
	if job == nil {
		return nil // No jobs available
	}

	w.log.Info().
		Str("job_id", job.ID).
		Str("type", string(job.Type)).
		Int("retry_count", job.RetryCount).
		Msg("Processing job")

	var processErr error
	switch job.Type {
	case queue.JobTypeSync:
		processErr = w.handleSyncJob(ctx, job)
	case queue.JobTypeResync:
		processErr = w.handleResyncJob(ctx, job)
	default:
		processErr = fmt.Errorf("unknown job type: %s", job.Type)
	}

	if processErr != nil {
		w.log.Error().
			Err(processErr).
			Str("job_id", job.ID).
			Str("type", string(job.Type)).
			Int("retry_count", job.RetryCount).
			Msg("Job failed")

		// Check if we should retry
		if job.RetryCount >= job.MaxRetries {
			w.log.Warn().
				Str("job_id", job.ID).
				Int("max_retries", job.MaxRetries).
				Msg("Job reached maximum retries, marking as stopped")

			// Update job status to stopped
			job.Status = queue.JobStatusStopped
			return w.queue.Fail(job.ID, fmt.Errorf("max retries reached: %w", processErr))
		}

		// Calculate next retry time with exponential backoff
		job.RetryCount++
		job.LastRetryAt = time.Now()
		backoff := w.calculateBackoff(job)
		job.NextRetryAt = job.LastRetryAt.Add(backoff)

		w.log.Info().
			Str("job_id", job.ID).
			Int("retry_count", job.RetryCount).
			Dur("backoff", backoff).
			Time("next_retry", job.NextRetryAt).
			Msg("Scheduling job retry")

		return w.queue.Fail(job.ID, processErr)
	}

	w.log.Info().
		Str("job_id", job.ID).
		Str("type", string(job.Type)).
		Msg("Job completed")
	return w.queue.Complete(job.ID)
}

func (w *JobWorker) handleSyncJob(ctx context.Context, job *queue.Job) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal sync payload: %w", err)
	}

	return w.service.SyncRepository(ctx, payload.Owner, payload.Repo, time.Time{})
}

func (w *JobWorker) handleResyncJob(ctx context.Context, job *queue.Job) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal resync payload: %w", err)
	}

	since := time.Now().AddDate(0, 0, -7) // Last 7 days
	return w.service.SyncRepository(ctx, payload.Owner, payload.Repo, since)
}
