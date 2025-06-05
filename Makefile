# Include environment variables from .env file if it exists
ifneq (,$(wildcard .env))
    include .env
    export
endif

.PHONY: build test clean run dev setup

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
BINARY_NAME=github-service
BINARY_UNIX=$(BINARY_NAME)_unix

all: test build

# Build the application
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/github-service

# Run tests
test:
	$(GOTEST) -v ./...

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f *.db
	rm -f *.db-journal

# Run the application
run:
	@if [ -z "$(GITHUB_SERVICE_GITHUB_TOKEN)" ]; then \
		echo "Error: GITHUB_SERVICE_GITHUB_TOKEN is not set. Please set it first:"; \
		echo "export GITHUB_SERVICE_GITHUB_TOKEN=your_token"; \
		exit 1; \
	fi
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/github-service
	./$(BINARY_NAME)

# Format code
fmt:
	$(GOCMD) fmt ./...

# Run linter
lint:
	golangci-lint run

# Run with air for development (hot reload)
dev:
	@if [ -z "$(GITHUB_SERVICE_GITHUB_TOKEN)" ]; then \
		echo "Error: GITHUB_SERVICE_GITHUB_TOKEN is not set. Please set it first:"; \
		echo "export GITHUB_SERVICE_GITHUB_TOKEN=your_token"; \
		exit 1; \
	fi
	air

# Setup development environment
setup:
	@echo "Setting up development environment..."
	@if [ ! -f .env.example ]; then \
		echo "Creating .env.example file..."; \
		echo "GITHUB_SERVICE_GITHUB_TOKEN=your_github_token_here" > .env.example; \
		echo ".env.example file created. Copy it to .env and add your token"; \
	fi
	@if [ ! -d bin ]; then \
		mkdir -p bin; \
	fi
	@if [ ! -f .gitignore ]; then \
		echo "Creating .gitignore..."; \
		echo ".env\ntmp/\nbin/\n*.db\n*.db-journal" > .gitignore; \
	fi
	@go install github.com/air-verse/air@latest
	@echo "Development environment setup complete!"
	@echo "Important:"
	@echo "1. Copy .env.example to .env"
	@echo "2. Add your GitHub token to .env"
	@echo "3. Run 'make run' to start the service"

# Help target
help:
	@echo "Available targets:"
	@echo "  build              - Build the application"
	@echo "  test               - Run tests"
	@echo "  clean              - Clean build files"
	@echo "  run                - Build and run the application"
	@echo "  fmt                - Format code"
	@echo "  lint               - Run linter"
	@echo "  dev                - Run development environment"
	@echo "  setup              - Setup development environment"
