package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"github-service/internal/app"
	"github-service/internal/config"
	"github-service/internal/database"
	"github-service/internal/github"
	"github-service/internal/queue"
	"github-service/internal/service"
	"github-service/internal/worker"

	"github.com/rs/zerolog"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// Create logger
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Initialize database connection
	db, err := database.New(cfg.GetDSN())
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Initialize GitHub client
	githubClient := github.NewClient(cfg.GitHub.Token)

	// Create service layer
	svcLogger := logger.With().Str("component", "service").Logger()
	svc := service.New(githubClient, db, &svcLogger)

	// Create job queue
	jobQueue, err := queue.NewPostgresQueue(db.DB())
	if err != nil {
		log.Fatalf("Error creating job queue: %v", err)
	}

	// Create sync worker for repository monitoring
	syncWorker := worker.NewSyncWorker(svc, cfg.GitHub.Interval, 7*24*time.Hour)

	// Create job worker
	workerLogger := logger.With().Str("component", "worker").Logger()
	jobWorker := worker.NewJobWorker(jobQueue, svc, workerLogger)

	// Initialize and start the application
	app, err := app.New(cfg, logger, svc, jobQueue, syncWorker)
	if err != nil {
		log.Fatalf("Error creating application: %v", err)
	}

	// Create context that listens for the interrupt signal
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start job worker in a goroutine
	go func() {
		if err := jobWorker.Start(ctx); err != nil {
			logger.Error().Err(err).Msg("Job worker error")
		}
	}()

	// Start the application
	if err := app.Run(ctx); err != nil {
		logger.Error().Err(err).Msg("Application error")
		os.Exit(1)
	}
}
