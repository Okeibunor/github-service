package app

import (
	"context"
	"fmt"
	"github-service/internal/config"
	"github-service/internal/queue"
	"github-service/internal/service"
	"github-service/internal/worker"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
)

// @title GitHub Service API
// @version 1.0
// @description A service that monitors GitHub repositories, fetches commit information, and provides statistics.
// @host localhost:8080
// @BasePath /api/v1

type App struct {
	cfg     *config.Config
	log     zerolog.Logger
	service *service.Service
	server  *http.Server
	monitor *time.Ticker
	queue   queue.Queue
	worker  *worker.SyncWorker
}

func New(cfg *config.Config, log zerolog.Logger, svc *service.Service, queue queue.Queue, worker *worker.SyncWorker) (*App, error) {
	app := &App{
		cfg:     cfg,
		log:     log,
		service: svc,
		queue:   queue,
		worker:  worker,
	}

	router := mux.NewRouter()
	app.initializeRouter(router)

	app.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	if a.cfg.GitHub.Interval > 0 {
		a.monitor = time.NewTicker(a.cfg.GitHub.Interval)
		go a.runMonitor(ctx)
	}

	go func() {
		<-ctx.Done()
		if a.monitor != nil {
			a.monitor.Stop()
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.server.Shutdown(shutdownCtx); err != nil {
			a.log.Error().Err(err).Msg("Failed to shutdown server gracefully")
		}
	}()

	a.log.Info().Msgf("Starting server on port %d", a.cfg.Server.Port)
	if err := a.server.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

func (a *App) runMonitor(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-a.monitor.C:
			since := a.cfg.GitHub.Since
			if since.IsZero() {
				since = time.Now().AddDate(0, 0, -7)
			}

			if a.cfg.GitHub.Repo != "" {
				parts := strings.Split(a.cfg.GitHub.Repo, "/")
				if len(parts) == 2 {
					err := a.service.SyncRepository(ctx, parts[0], parts[1], since)
					if err != nil {
						a.log.Error().
							Err(err).
							Str("repo", a.cfg.GitHub.Repo).
							Msg("Failed to sync repository")
						continue
					}

					a.log.Info().
						Str("repo", a.cfg.GitHub.Repo).
						Msg("Successfully synced repository")
				}
			}
		}
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	return a.server.Shutdown(ctx)
}

func (a *App) Close() error {
	return a.service.Close()
}
