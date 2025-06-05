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

	// Log DSN for debugging
	log.Printf("Using DSN: %s", cfg.GetDSN())

	// Set up database connection
	db, err := database.New(cfg.GetDSN())
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Create service
	svc, err := service.New(service.Config{
		GitHubToken: cfg.GitHub.Token,
		DB:          db,
	})
	if err != nil {
		log.Fatalf("Error creating service: %v", err)
	}

	// Create and start sync worker
	syncWorker := worker.NewSyncWorker(
		svc,
		cfg.GitHub.Interval,
		7*24*time.Hour, // Default sync age of 7 days
	)

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the worker
	go syncWorker.Start(ctx)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Initialize and start the application
	app, err := app.New(cfg, logger, svc, syncWorker)
	if err != nil {
		log.Fatalf("Error creating application: %v", err)
	}

	// Start the application
	if err := app.Run(ctx); err != nil {
		log.Fatalf("Error running application: %v", err)
	}

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutting down...")

	// Stop the worker
	syncWorker.Stop()

	// Create a context with timeout for shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Wait for context to be done
	<-shutdownCtx.Done()
	log.Println("Shutdown complete")
}
