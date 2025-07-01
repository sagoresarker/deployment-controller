# Deployment Controller Makefile

# Variables
APP_NAME=deployment-controller
BINARY_NAME=deployment-controller
GO_VERSION=1.23
DOCKER_IMAGE=$(APP_NAME):latest

# Default target
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  build        - Build the application binary"
	@echo "  run          - Run the application"
	@echo "  dev          - Run the application in development mode"
	@echo "  test         - Run Go unit tests"
	@echo "  test-quick   - Run quick integration tests"
	@echo "  test-all     - Run all tests (Go + integration)"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  db-setup     - Setup database schema"
	@echo "  db-migrate   - Run database migrations"
	@echo "  lint         - Run Go linting"
	@echo "  fmt          - Format Go code"

# Build the application
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	go build -o bin/$(BINARY_NAME) cmd/server/main.go

# Run the application
.PHONY: run
run: build
	@echo "Running $(APP_NAME)..."
	./bin/$(BINARY_NAME)

# Run in development mode
.PHONY: dev
dev:
	@echo "Running $(APP_NAME) in development mode..."
	go run cmd/server/main.go

# Run Go tests
.PHONY: test
test:
	@echo "Running Go tests..."
	go test -v ./...

# Run quick integration tests
.PHONY: test-quick
test-quick:
	@echo "Running quick integration tests..."
	./scripts/quick-test.sh

# Run all tests (Go + integration)
.PHONY: test-all
test-all: test test-quick

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning up..."
	rm -rf bin/
	go clean

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .

# Run Docker container
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file config.yaml $(DOCKER_IMAGE)

# Setup database schema
.PHONY: db-setup
db-setup:
	@echo "Setting up database schema..."
	@if [ -z "$(DB_URL)" ]; then \
		echo "Please set DB_URL environment variable"; \
		echo "Example: make db-setup DB_URL=postgres://user:pass@localhost:5432/deployment_controller"; \
		exit 1; \
	fi
	psql $(DB_URL) -f db/schema.sql

# Database migration (same as setup for now)
.PHONY: db-migrate
db-migrate: db-setup

# Run Go linting
.PHONY: lint
lint:
	@echo "Running Go linting..."
	golangci-lint run

# Format Go code
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run application with hot reload (requires air)
.PHONY: watch
watch:
	@echo "Running with hot reload..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Please install air: go install github.com/cosmtrek/air@latest"; \
	fi

# Create release build
.PHONY: release
release:
	@echo "Creating release build..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o bin/$(BINARY_NAME)-linux-amd64 cmd/server/main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o bin/$(BINARY_NAME)-darwin-amd64 cmd/server/main.go
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o bin/$(BINARY_NAME)-windows-amd64.exe cmd/server/main.go