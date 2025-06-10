package app

import (
	"encoding/json"
	"fmt"
	"github-service/internal/models"
	"github-service/internal/response"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github-service/internal/queue"

	"github.com/gorilla/mux"
)

// healthCheck handles the health check endpoint
func (a *App) healthCheck(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, response.Success("Service is healthy", map[string]string{"status": "ok"}))
}

// getCommits handles retrieving commits for a repository
func (a *App) getCommits(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	owner, repo := vars["owner"], vars["repo"]
	fullName := fmt.Sprintf("%s/%s", owner, repo)

	a.log.Debug().
		Str("owner", owner).
		Str("repo", repo).
		Msg("Getting commits for repository")

	// Parse pagination parameters
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
	if err != nil || perPage < 1 {
		perPage = 10 // Default page size
	}

	commits, totalItems, err := a.service.GetCommitsByRepository(r.Context(), fullName, page, perPage)
	if err != nil {
		a.log.Error().
			Err(err).
			Str("repository", fullName).
			Int("page", page).
			Int("per_page", perPage).
			Msg("Failed to get commits")
		response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to get commits: %v", err)))
		return
	}

	a.log.Info().
		Str("repository", fullName).
		Int("commit_count", len(commits)).
		Int("page", page).
		Int("per_page", perPage).
		Int("total_items", totalItems).
		Msg("Successfully retrieved commits")

	response.JSON(w, http.StatusOK, response.SuccessPaginated("Commits retrieved successfully", commits, page, perPage, totalItems))
}

// getTopAuthors handles retrieving top commit authors
func (a *App) getTopAuthors(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 10
	}

	// Check if repository is specified
	repoFullName := r.URL.Query().Get("repository")
	var (
		authors []*models.CommitStats
		err     error
	)

	a.log.Debug().
		Int("limit", limit).
		Str("repository", repoFullName).
		Msg("Getting top authors")

	if repoFullName != "" {
		// First check if the repository is being monitored
		if !a.worker.IsRepositoryMonitored(r.Context(), repoFullName) {
			response.JSON(w, http.StatusNotFound, response.Error(fmt.Sprintf("Repository %s is not being monitored", repoFullName)))
			return
		}

		// Get repository-specific authors
		authors, err = a.service.GetTopCommitAuthorsByRepository(r.Context(), repoFullName, limit)
		if err != nil {
			a.log.Error().
				Err(err).
				Int("limit", limit).
				Str("repository", repoFullName).
				Msg("Failed to get top authors")

			// Handle specific error cases
			if strings.Contains(err.Error(), "no commits found") {
				response.JSON(w, http.StatusNotFound, response.Error(fmt.Sprintf("No commits found for repository %s", repoFullName)))
				return
			}

			response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to get top authors: %v", err)))
			return
		}
	} else {
		// Get global top authors
		authors, err = a.service.GetTopCommitAuthors(r.Context(), limit)
		if err != nil {
			a.log.Error().
				Err(err).
				Int("limit", limit).
				Msg("Failed to get top authors")
			response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to get top authors: %v", err)))
			return
		}
	}

	a.log.Info().
		Int("author_count", len(authors)).
		Str("repository", repoFullName).
		Msg("Successfully retrieved top authors")

	response.JSON(w, http.StatusOK, response.Success("Top authors retrieved successfully", map[string]interface{}{
		"authors":    authors,
		"n":          len(authors),
		"repository": repoFullName,
	}))
}

// listRepositories handles listing all monitored repositories
func (a *App) listRepositories(w http.ResponseWriter, r *http.Request) {
	a.log.Debug().Msg("Listing repositories")

	// Get monitored repositories
	monitoredRepos, err := a.service.DB().GetMonitoredRepositories(r.Context())
	if err != nil {
		a.log.Error().Err(err).Msg("Failed to list repositories")
		response.JSON(w, http.StatusInternalServerError, response.Error("Failed to list repositories"))
		return
	}

	// Get full repository details for each monitored repository
	var repositories []*models.Repository
	for _, monitoredRepo := range monitoredRepos {
		repo, err := a.service.GetRepositoryByName(r.Context(), monitoredRepo.FullName)
		if err != nil {
			a.log.Error().
				Err(err).
				Str("repository", monitoredRepo.FullName).
				Msg("Failed to get repository details")
			continue
		}
		if repo != nil {
			repositories = append(repositories, repo)
		}
	}

	a.log.Info().
		Int("repository_count", len(repositories)).
		Msg("Successfully listed repositories")

	response.JSON(w, http.StatusOK, response.Success("Repositories retrieved successfully", map[string]interface{}{
		"count":        len(repositories),
		"repositories": repositories,
	}))
}

// addRepository handles adding a new repository to monitor
func (a *App) addRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	owner, repo := vars["owner"], vars["repo"]

	a.log.Debug().
		Str("owner", owner).
		Str("repo", repo).
		Msg("Adding repository")

	// First check if repository exists in GitHub without syncing commits
	exists, err := a.service.RepositoryExists(r.Context(), owner, repo)
	if err != nil {
		a.log.Error().
			Err(err).
			Str("owner", owner).
			Str("repo", repo).
			Msg("Failed to validate repository")

		if strings.Contains(strings.ToLower(err.Error()), "rate limit") {
			response.JSON(w, http.StatusTooManyRequests, response.Error("GitHub rate limit exceeded, please try again later"))
			return
		}

		response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to validate repository: %v", err)))
		return
	}

	if !exists {
		response.JSON(w, http.StatusNotFound, response.Error(fmt.Sprintf("Repository %s/%s not found on GitHub", owner, repo)))
		return
	}

	// Get repository information from GitHub and sync it to our database
	if err := a.service.SyncRepository(r.Context(), owner, repo, time.Now().AddDate(0, 0, -7)); err != nil {
		a.log.Error().
			Err(err).
			Str("owner", owner).
			Str("repo", repo).
			Msg("Failed to sync repository")
		response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to sync repository: %v", err)))
		return
	}

	// Add to monitoring list
	if err := a.worker.AddRepository(r.Context(), owner, repo); err != nil {
		a.log.Error().
			Err(err).
			Str("owner", owner).
			Str("repo", repo).
			Msg("Failed to add repository to monitoring")
		response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to add repository to monitoring: %v", err)))
		return
	}

	// Create a sync job for full history
	payload := queue.SyncPayload{
		Owner: owner,
		Repo:  repo,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		a.log.Error().
			Err(err).
			Msg("Failed to marshal sync payload")
		response.JSON(w, http.StatusInternalServerError, response.Error("Internal server error"))
		return
	}

	job := &queue.Job{
		Type:    queue.JobTypeSync,
		Payload: payloadBytes,
	}

	if err := a.queue.Enqueue(job); err != nil {
		a.log.Error().
			Err(err).
			Str("owner", owner).
			Str("repo", repo).
			Msg("Failed to enqueue sync job")
		response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to schedule repository sync: %v", err)))
		return
	}

	response.JSON(w, http.StatusAccepted, response.Success(
		fmt.Sprintf("Repository %s/%s scheduled for synchronization", owner, repo),
		map[string]interface{}{
			"job_id": job.ID,
			"status": "scheduled",
			"owner":  owner,
			"repo":   repo,
		},
	))
}

// removeRepository handles removing a repository from monitoring
func (a *App) removeRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	owner, repo := vars["owner"], vars["repo"]
	fullName := fmt.Sprintf("%s/%s", owner, repo)

	a.log.Debug().
		Str("owner", owner).
		Str("repo", repo).
		Msg("Removing repository")

	// First remove from worker's monitoring list
	a.worker.RemoveRepository(r.Context(), owner, repo)

	// Then remove from database
	dbRepo, err := a.service.GetRepositoryByName(r.Context(), fullName)
	if err != nil {
		a.log.Error().
			Err(err).
			Str("repository", fullName).
			Msg("Failed to find repository in database")
		// Continue anyway as we want to ensure it's removed from monitoring
	} else if dbRepo != nil {
		if err := a.service.DeleteRepository(r.Context(), fullName); err != nil {
			a.log.Error().
				Err(err).
				Str("repository", fullName).
				Msg("Failed to delete repository from database")
			response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to delete repository %s: %v", fullName, err)))
			return
		}
	}

	a.log.Info().
		Str("owner", owner).
		Str("repo", repo).
		Msg("Repository removed successfully")

	response.JSON(w, http.StatusOK, response.Success(
		fmt.Sprintf("Repository %s/%s removed successfully", owner, repo),
		map[string]string{
			"owner": owner,
			"repo":  repo,
		},
	))
}

// resyncRepository handles repository resynchronization with a specific time
func (a *App) resyncRepository(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	owner, repo := vars["owner"], vars["repo"]
	fullName := fmt.Sprintf("%s/%s", owner, repo)

	a.log.Debug().
		Str("owner", owner).
		Str("repo", repo).
		Msg("Resyncing repository")

	// Check if repository is being monitored
	if !a.worker.IsRepositoryMonitored(r.Context(), fullName) {
		response.JSON(w, http.StatusNotFound, response.Error(fmt.Sprintf("Repository %s is not being monitored", fullName)))
		return
	}

	// Create a resync job
	payload := queue.SyncPayload{
		Owner: owner,
		Repo:  repo,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		a.log.Error().
			Err(err).
			Msg("Failed to marshal resync payload")
		response.JSON(w, http.StatusInternalServerError, response.Error("Internal server error"))
		return
	}

	job := &queue.Job{
		Type:    queue.JobTypeResync,
		Payload: payloadBytes,
	}

	if err := a.queue.Enqueue(job); err != nil {
		a.log.Error().
			Err(err).
			Str("owner", owner).
			Str("repo", repo).
			Msg("Failed to enqueue resync job")
		response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to schedule repository resync: %v", err)))
		return
	}

	response.JSON(w, http.StatusAccepted, response.Success(
		fmt.Sprintf("Repository %s/%s scheduled for resynchronization", owner, repo),
		map[string]interface{}{
			"job_id": job.ID,
			"status": "scheduled",
			"owner":  owner,
			"repo":   repo,
		},
	))
}

func (a *App) getJobStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	jobID := vars["job_id"]

	a.log.Debug().
		Str("job_id", jobID).
		Msg("Getting job status")

	status, err := a.queue.GetStatus(jobID)
	if err != nil {
		a.log.Error().
			Err(err).
			Str("job_id", jobID).
			Msg("Failed to get job status")

		if strings.Contains(err.Error(), "job not found") {
			response.JSON(w, http.StatusNotFound, response.Error(fmt.Sprintf("Job %s not found", jobID)))
			return
		}

		response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to get job status: %v", err)))
		return
	}

	a.log.Info().
		Str("job_id", jobID).
		Str("status", string(status)).
		Msg("Successfully retrieved job status")

	response.JSON(w, http.StatusOK, response.Success("Job status retrieved successfully", map[string]interface{}{
		"job_id": jobID,
		"status": status,
	}))
}

// listJobs handles retrieving all jobs
func (a *App) listJobs(w http.ResponseWriter, r *http.Request) {
	a.log.Debug().Msg("Listing all jobs")

	jobs, err := a.queue.GetJobs()
	if err != nil {
		a.log.Error().
			Err(err).
			Msg("Failed to get jobs")
		response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to get jobs: %v", err)))
		return
	}

	a.log.Info().
		Int("job_count", len(jobs)).
		Msg("Successfully retrieved jobs")

	response.JSON(w, http.StatusOK, response.Success("Jobs retrieved successfully", map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	}))
}
