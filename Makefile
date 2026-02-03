# Seville - Music Quality Lab
# Build and development commands

.PHONY: all build run dev test clean templ css install deps screenshots help

# Variables
BINARY_NAME=seville
BUILD_DIR=bin
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Default target
all: deps templ css build

# Install dependencies
deps:
	@echo "Installing Go dependencies..."
	go mod download
	go mod tidy
	@echo "Installing templ..."
	go install github.com/a-h/templ/cmd/templ@latest
	@echo "Installing npm dependencies..."
	npm install

# Generate templ templates
templ:
	@echo "Generating templ templates..."
	templ generate

# Build CSS with Tailwind
css:
	@echo "Building CSS..."
	npm run css:build

# Watch CSS changes
css-watch:
	npm run css:watch

# Build the application
build: templ css
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/server

# Run the application
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) -debug

# Development mode with hot reload
dev:
	@echo "Starting development mode..."
	@which air > /dev/null || go install github.com/cosmtrek/air@latest
	air

# Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run E2E tests with playwright-go
test-e2e: build
	@echo "Running E2E tests..."
	go test -v ./tests/e2e/...

# Generate screenshots with playwright-go
screenshots: build
	@echo "Generating screenshots..."
	@mkdir -p screenshots
	go run ./tests/e2e/screenshots.go

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -rf web/static/css/tailwind.css
	rm -rf screenshots
	rm -f *.db *.db-*
	find . -name "*_templ.go" -delete

# Database migration
migrate:
	@echo "Running migrations..."
	./$(BUILD_DIR)/$(BINARY_NAME) -migrate

# Lint
lint:
	@echo "Linting..."
	golangci-lint run ./...
	npm run lint 2>/dev/null || true

# Format code
fmt:
	@echo "Formatting..."
	gofmt -s -w .
	templ fmt .

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t seville:$(VERSION) .

# Docker run
docker-run:
	docker run -p 8080:8080 -v $(PWD)/data:/data seville:$(VERSION)

# Install for production
install: all
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

# Help
help:
	@echo "Seville - Music Quality Lab"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all          Build everything (default)"
	@echo "  deps         Install dependencies"
	@echo "  templ        Generate templ templates"
	@echo "  css          Build Tailwind CSS"
	@echo "  css-watch    Watch CSS changes"
	@echo "  build        Build the binary"
	@echo "  run          Build and run"
	@echo "  dev          Development mode with hot reload"
	@echo "  test         Run unit tests"
	@echo "  test-e2e     Run E2E tests"
	@echo "  screenshots  Generate screenshots"
	@echo "  clean        Clean build artifacts"
	@echo "  lint         Run linters"
	@echo "  fmt          Format code"
	@echo "  docker-build Build Docker image"
	@echo "  docker-run   Run Docker container"
	@echo "  help         Show this help"
