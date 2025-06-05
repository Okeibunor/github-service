package worker

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github-service/internal/service"
)

// SyncWorker handles background synchronization of repositories
type SyncWorker struct {
	service      *service.Service
	syncInterval time.Duration
	defaultAge   time.Duration
	repos        map[string]string // map[owner/repo]lastSyncTime
	mu           sync.RWMutex
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
		repos:        make(map[string]string),
		stop:         make(chan struct{}),
	}
}

// AddRepository adds a repository to be monitored
func (w *SyncWorker) AddRepository(owner, name string) error {
	w.mu.Lock()
	fullName := owner + "/" + name
	w.repos[fullName] = ""
	w.mu.Unlock()

	// Perform initial sync
	since := time.Now().Add(-w.defaultAge)
	err := w.service.SyncRepository(context.Background(), owner, name, since)
	if err != nil {
		// If sync fails, remove from monitoring
		w.mu.Lock()
		delete(w.repos, fullName)
		w.mu.Unlock()
		return fmt.Errorf("initial sync failed: %w", err)
	}

	// Update last sync time
	w.mu.Lock()
	w.repos[fullName] = time.Now().UTC().Format(time.RFC3339)
	w.mu.Unlock()

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
	w.mu.RLock()
	repos := make(map[string]string, len(w.repos))
	for k, v := range w.repos {
		repos[k] = v
	}
	w.mu.RUnlock()

	for fullName := range repos {
		owner, name := splitRepoName(fullName)
		if owner == "" || name == "" {
			log.Printf("Invalid repository name format: %s", fullName)
			continue
		}

		since := w.getSyncTime(fullName)

		// Implement retry logic with exponential backoff
		maxRetries := 3
		for attempt := 1; attempt <= maxRetries; attempt++ {
			err := w.service.SyncRepository(ctx, owner, name, since)
			if err == nil {
				w.mu.Lock()
				w.repos[fullName] = time.Now().UTC().Format(time.RFC3339)
				w.mu.Unlock()
				break
			}

			if attempt == maxRetries {
				log.Printf("Error syncing repository %s after %d attempts: %v", fullName, maxRetries, err)
				continue
			}

			// Exponential backoff
			backoffDuration := time.Duration(attempt*attempt) * time.Second
			log.Printf("Retry attempt %d for repository %s after %v: %v", attempt, fullName, backoffDuration, err)
			select {
			case <-time.After(backoffDuration):
				continue
			case <-ctx.Done():
				return
			}
		}
	}
}

// getSyncTime returns the time since which to sync commits
func (w *SyncWorker) getSyncTime(fullName string) time.Time {
	w.mu.RLock()
	lastSync := w.repos[fullName]
	w.mu.RUnlock()

	if lastSync == "" {
		return time.Now().Add(-w.defaultAge)
	}

	t, err := time.Parse(time.RFC3339, lastSync)
	if err != nil {
		return time.Now().Add(-w.defaultAge)
	}
	return t
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
func (w *SyncWorker) IsRepositoryMonitored(fullName string) bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	_, exists := w.repos[fullName]
	return exists
}

// ResetRepository resets the sync time for a repository
func (w *SyncWorker) ResetRepository(owner, name string, since time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.repos[owner+"/"+name] = since.UTC().Format(time.RFC3339)
}

// RemoveRepository removes a repository from monitoring
func (w *SyncWorker) RemoveRepository(owner, name string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.repos, owner+"/"+name)
}

// ListRepositories returns all monitored repositories
func (w *SyncWorker) ListRepositories() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	repos := make([]string, 0, len(w.repos))
	for repo := range w.repos {
		repos = append(repos, repo)
	}
	return repos
}
