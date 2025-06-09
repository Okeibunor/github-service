package worker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github-service/internal/service"
)

// SyncWorker handles background synchronization of repositories
type SyncWorker struct {
	service      *service.Service
	syncInterval time.Duration
	defaultAge   time.Duration
	stop         chan struct{}
}

// NewSyncWorker creates a new sync worker
func NewSyncWorker(service *service.Service, syncInterval, defaultAge time.Duration) *SyncWorker {
	if syncInterval <= 0 {
		syncInterval = time.Hour // default to 1 hour if not set or invalid
	}
	return &SyncWorker{
		service:      service,
		syncInterval: syncInterval,
		defaultAge:   defaultAge,
		stop:         make(chan struct{}),
	}
}

// AddRepository adds a repository to be monitored
func (w *SyncWorker) AddRepository(ctx context.Context, owner, name string) error {
	fullName := owner + "/" + name

	// Add to database first
	if err := w.service.DB().AddMonitoredRepository(ctx, fullName, w.syncInterval); err != nil {
		return fmt.Errorf("failed to add repository to monitoring: %w", err)
	}

	// Perform initial sync
	since := time.Now().Add(-w.defaultAge)
	if err := w.service.SyncRepository(ctx, owner, name, since); err != nil {
		// If sync fails, mark repository as inactive
		if removeErr := w.service.DB().RemoveMonitoredRepository(ctx, fullName); removeErr != nil {
			log.Printf("Failed to remove repository after sync failure: %v", removeErr)
		}
		return fmt.Errorf("initial sync failed: %w", err)
	}

	// Update last sync time
	if err := w.service.DB().UpdateMonitoredRepositorySync(ctx, fullName, time.Now().UTC()); err != nil {
		log.Printf("Failed to update last sync time: %v", err)
	}

	return nil
}

// Start begins the background sync process
func (w *SyncWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.syncInterval)
	defer ticker.Stop()

	// Initial sync
	w.syncAll(ctx)

	for {
		select {
		case <-ticker.C:
			w.syncAll(ctx)
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		}
	}
}

// Stop stops the background sync process
func (w *SyncWorker) Stop() {
	close(w.stop)
}

// syncAll synchronizes all monitored repositories
func (w *SyncWorker) syncAll(ctx context.Context) {
	repos, err := w.service.DB().GetMonitoredRepositories(ctx)
	if err != nil {
		log.Printf("Error fetching monitored repositories: %v", err)
		return
	}

	for _, repo := range repos {
		owner, name := splitRepoName(repo.FullName)
		if owner == "" || name == "" {
			log.Printf("Invalid repository name format: %s", repo.FullName)
			continue
		}

		// Implement retry logic with exponential backoff
		maxRetries := 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := w.service.SyncRepository(ctx, owner, name, repo.LastSyncTime)
			if err == nil {
				if updateErr := w.service.DB().UpdateMonitoredRepositorySync(ctx, repo.FullName, time.Now().UTC()); updateErr != nil {
					log.Printf("Failed to update last sync time for %s: %v", repo.FullName, updateErr)
				}
				break
			}

			if attempt == maxRetries {
				log.Printf("Error syncing repository %s after %d attempts: %v", repo.FullName, maxRetries, err)
				continue
			}

			// Exponential backoff
			backoffDuration := time.Duration(attempt*attempt) * time.Second
			log.Printf("Retry attempt %d for repository %s after %v: %v", attempt, repo.FullName, backoffDuration, err)
			select {
			case <-time.After(backoffDuration):
				continue
			case <-ctx.Done():
				return
			}
		}
	}
}

// splitRepoName splits a full repository name into owner and repository parts
func splitRepoName(fullName string) (owner, name string) {
	parts := strings.Split(fullName, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

// IsRepositoryMonitored checks if a repository is being monitored
func (w *SyncWorker) IsRepositoryMonitored(ctx context.Context, fullName string) bool {
	repos, err := w.service.DB().GetMonitoredRepositories(ctx)
	if err != nil {
		log.Printf("Error checking monitored status: %v", err)
		return false
	}
	for _, repo := range repos {
		if repo.FullName == fullName {
			return true
		}
	}
	return false
}

// ResetRepository resets the sync time for a repository
func (w *SyncWorker) ResetRepository(ctx context.Context, owner, name string, since time.Time) error {
	fullName := owner + "/" + name
	return w.service.DB().UpdateMonitoredRepositorySync(ctx, fullName, since)
}

// RemoveRepository removes a repository from monitoring
func (w *SyncWorker) RemoveRepository(ctx context.Context, owner, name string) error {
	fullName := owner + "/" + name
	return w.service.DB().RemoveMonitoredRepository(ctx, fullName)
}

// ListRepositories returns all monitored repositories
func (w *SyncWorker) ListRepositories(ctx context.Context) ([]string, error) {
	repos, err := w.service.DB().GetMonitoredRepositories(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(repos))
	for i, repo := range repos {
		names[i] = repo.FullName
	}
	return names, nil
}
