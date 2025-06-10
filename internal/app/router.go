package app

import (
	"github-service/internal/response"
	"net/http"

	"github.com/gorilla/mux"
)

// initializeRouter configures all routes for the application
func (a *App) initializeRouter(router *mux.Router) {
	// Set custom error handlers for 404 and 405 responses
	router.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusNotFound, response.Error("Route not found"))
	})
	router.MethodNotAllowedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response.JSON(w, http.StatusMethodNotAllowed, response.Error("Method not allowed"))
	})

	// Apply common middleware
	router.Use(a.loggingMiddleware)
	router.Use(a.recoveryMiddleware)

	// Health check endpoints
	router.HandleFunc("/", a.healthCheck).Methods(http.MethodGet)
	router.HandleFunc("/health", a.healthCheck).Methods(http.MethodGet)

	// API v1 routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", a.healthCheck).Methods(http.MethodGet)

	// Repository endpoints with their own subrouter
	initRepositoryRoutes(api.PathPrefix("/repositories").Subrouter(), a)

	// Statistics endpoints with their own subrouter
	initStatsRoutes(api.PathPrefix("/stats").Subrouter(), a)

	// Jobs endpoints
	api.HandleFunc("/jobs", a.listJobs).Methods(http.MethodGet)
	api.HandleFunc("/jobs/{job_id}", a.getJobStatus).Methods(http.MethodGet)
}

// initRepositoryRoutes configures all repository-related routes
func initRepositoryRoutes(router *mux.Router, a *App) {
	router.HandleFunc("", a.listRepositories).Methods(http.MethodGet)
	router.HandleFunc("/{owner}/{repo}", a.addRepository).Methods(http.MethodPut)
	router.HandleFunc("/{owner}/{repo}", a.removeRepository).Methods(http.MethodDelete)
	router.HandleFunc("/{owner}/{repo}/commits", a.getCommits).Methods(http.MethodGet)
	router.HandleFunc("/{owner}/{repo}/sync", a.resyncRepository).Methods(http.MethodPost)
}

// initStatsRoutes configures all statistics-related routes
func initStatsRoutes(router *mux.Router, a *App) {
	router.HandleFunc("/top-authors", a.getTopAuthors).Methods(http.MethodGet)
}

// loggingMiddleware logs information about each request
func (a *App) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Str("remote_addr", r.RemoteAddr).
			Msg("Incoming request")

		next.ServeHTTP(w, r)
	})
}

// recoveryMiddleware recovers from panics and returns a 500 error
func (a *App) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				a.log.Error().
					Interface("error", err).
					Str("path", r.URL.Path).
					Msg("Panic recovered in request handler")

				response.JSON(w, http.StatusInternalServerError, response.Error("Internal server error"))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
