package app

import (
	"fmt"
	"github-service/internal/models"
	"github-service/internal/response"
	"net/http"
	"strconv"
	"strings"
	"time"

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

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	if limit <= 0 {
		limit = 10
	}
	if offset < 0 {
		offset = 0
	}

	commits, err := a.service.GetCommitsByRepository(r.Context(), fullName, limit, offset)
	if err != nil {
		a.log.Error().
			Err(err).
			Str("repository", fullName).
			Int("limit", limit).
			Int("offset", offset).
			Msg("Failed to get commits")
		response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to get commits: %v", err)))
		return
	}

	a.log.Info().
		Str("repository", fullName).
		Int("commit_count", len(commits)).
		Msg("Successfully retrieved commits")

	response.JSON(w, http.StatusOK, response.Success("Commits retrieved successfully", commits))
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

	repos, err := a.worker.ListRepositories(r.Context())
	if err != nil {
		a.log.Error().Err(err).Msg("Failed to list repositories")
		response.JSON(w, http.StatusInternalServerError, response.Error("Failed to list repositories"))
		return
	}

	a.log.Info().
		Int("repository_count", len(repos)).
		Msg("Successfully listed repositories")

	response.JSON(w, http.StatusOK, response.Success("Repositories retrieved successfully", map[string]interface{}{
		"repositories": repos,
		"count":        len(repos),
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
		response.JSON(w, http.StatusBadRequest, response.Error(fmt.Sprintf("Unable to add repository: %s/%s. Please verify the repository exists and you have access to it", owner, repo)))
		return
	}

	if !exists {
		response.JSON(w, http.StatusNotFound, response.Error(fmt.Sprintf("Repository %s/%s not found", owner, repo)))
		return
	}

	// Add to worker which will handle the initial sync
	if err := a.worker.AddRepository(r.Context(), owner, repo); err != nil {
		a.log.Error().
			Err(err).
			Str("owner", owner).
			Str("repo", repo).
			Msg("Failed to add repository")
		response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to add repository: %v", err)))
		return
	}

	a.log.Info().
		Str("owner", owner).
		Str("repo", repo).
		Msg("Repository added successfully")

	response.JSON(w, http.StatusCreated, response.Success(
		fmt.Sprintf("Repository %s/%s added successfully", owner, repo),
		map[string]string{
			"owner": owner,
			"repo":  repo,
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
		Msg("Starting repository resync")

	since := time.Now().AddDate(0, 0, -7) // Default to last 7 days
	if sinceStr := r.URL.Query().Get("since"); sinceStr != "" {
		var err error
		since, err = time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			a.log.Error().
				Err(err).
				Str("since", sinceStr).
				Msg("Invalid since parameter")
			response.JSON(w, http.StatusBadRequest, response.Error("Invalid since parameter. Use RFC3339 format"))
			return
		}
	}

	// First verify the repository is being monitored
	if !a.worker.IsRepositoryMonitored(r.Context(), owner+"/"+repo) {
		response.JSON(w, http.StatusNotFound, response.Error(fmt.Sprintf("Repository %s is not being monitored", fullName)))
		return
	}

	// Update sync time and trigger sync
	if err := a.service.SyncRepository(r.Context(), owner, repo, since); err != nil {
		a.log.Error().
			Err(err).
			Str("owner", owner).
			Str("repo", repo).
			Time("since", since).
			Msg("Failed to sync repository")
		response.JSON(w, http.StatusInternalServerError, response.Error(fmt.Sprintf("Failed to sync repository: %v", err)))
		return
	}

	// Update worker's sync time after successful sync
	a.worker.ResetRepository(r.Context(), owner, repo, since)

	a.log.Info().
		Str("owner", owner).
		Str("repo", repo).
		Time("since", since).
		Msg("Repository resync completed")

	response.JSON(w, http.StatusOK, response.Success(
		fmt.Sprintf("Repository %s/%s resync completed", owner, repo),
		map[string]interface{}{
			"owner": owner,
			"repo":  repo,
			"since": since,
		},
	))
}
