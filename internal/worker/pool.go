package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github-service/internal/queue"
	"github-service/internal/service"
)

// Pool represents a worker pool for processing jobs
type Pool struct {
	queue    queue.Queue
	service  *service.Service
	workers  int
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewPool creates a new worker pool
func NewPool(queue queue.Queue, service *service.Service, workers int) *Pool {
	if workers <= 0 {
		workers = 5 // default number of workers
	}
	return &Pool{
		queue:    queue,
		service:  service,
		workers:  workers,
		stopChan: make(chan struct{}),
	}
}

// Start starts the worker pool
func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
}

// Stop stops the worker pool
func (p *Pool) Stop() {
	close(p.stopChan)
	p.wg.Wait()
}

func (p *Pool) worker(ctx context.Context, id int) {
	defer p.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping due to context cancellation", id)
			return
		case <-p.stopChan:
			log.Printf("Worker %d stopping due to pool shutdown", id)
			return
		default:
			if err := p.processNextJob(ctx); err != nil {
				log.Printf("Worker %d error processing job: %v", id, err)
				// Add a small delay before trying again
				time.Sleep(time.Second)
			}
		}
	}
}

func (p *Pool) processNextJob(ctx context.Context) error {
	// Get next job from queue
	job, err := p.queue.Dequeue()
	if err != nil {
		return fmt.Errorf("error dequeuing job: %w", err)
	}
	if job == nil {
		// No jobs available, wait a bit
		time.Sleep(time.Second)
		return nil
	}

	log.Printf("Processing job %s of type %s", job.ID, job.Type)

	// Process the job based on its type
	var processErr error
	switch job.Type {
	case queue.JobTypeSync:
		processErr = p.processSyncJob(ctx, job)
	case queue.JobTypeResync:
		processErr = p.processResyncJob(ctx, job)
	case queue.JobTypeCleanup:
		processErr = p.processCleanupJob(ctx, job)
	default:
		processErr = fmt.Errorf("unknown job type: %s", job.Type)
	}

	if processErr != nil {
		if err := p.queue.Fail(job.ID, processErr); err != nil {
			log.Printf("Error marking job %s as failed: %v", job.ID, err)
		}
		return processErr
	}

	if err := p.queue.Complete(job.ID); err != nil {
		return fmt.Errorf("error marking job as complete: %w", err)
	}

	return nil
}

func (p *Pool) processSyncJob(ctx context.Context, job *queue.Job) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("error unmarshaling sync job payload: %w", err)
	}

	// Process repository sync with retries
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := p.service.SyncRepository(ctx, payload.Owner, payload.Repo, time.Time{})
		if err == nil {
			return nil
		}

		if attempt == maxRetries {
			return fmt.Errorf("failed to sync repository after %d attempts: %w", maxRetries, err)
		}

		// Exponential backoff
		backoffDuration := time.Duration(attempt*attempt) * time.Second
		log.Printf("Retry attempt %d for repository %s/%s after %v: %v",
			attempt, payload.Owner, payload.Repo, backoffDuration, err)

		select {
		case <-time.After(backoffDuration):
			continue
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

func (p *Pool) processResyncJob(ctx context.Context, job *queue.Job) error {
	var payload queue.SyncPayload
	if err := json.Unmarshal(job.Payload, &payload); err != nil {
		return fmt.Errorf("error unmarshaling resync job payload: %w", err)
	}

	since := time.Now().AddDate(0, 0, -7) // Last 7 days
	return p.service.SyncRepository(ctx, payload.Owner, payload.Repo, since)
}

func (p *Pool) processCleanupJob(ctx context.Context, job *queue.Job) error {
	// TODO: Implement cleanup logic
	return nil
}
