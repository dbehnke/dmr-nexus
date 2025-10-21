.PHONY: all build clean test test-coverage lint fmt vet deps dev frontend docker run help

# Variables
BINARY_NAME=dmr-nexus
BUILD_DIR=bin
FRONTEND_DIR=frontend
VERSION?=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Colors for output
BLUE=\033[0;34m
NC=\033[0m # No Color

all: build

## help: Display this help message
help:
	@echo "$(BLUE)DMR-Nexus Makefile$(NC)"
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(BLUE)%-15s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## build: Build the application binary
build: deps
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dmr-nexus

## clean: Remove build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.txt coverage.html
	@rm -rf $(FRONTEND_DIR)/dist
	@rm -rf $(FRONTEND_DIR)/node_modules

## test: Run tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	@go test -v -race ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	@go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

## test-integration: Run integration tests
test-integration:
	@echo "$(BLUE)Running integration tests...$(NC)"
	@go test -v -race -tags=integration ./...

## lint: Run linters
lint:
	@echo "$(BLUE)Running linters...$(NC)"
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	@golangci-lint run ./...

## fmt: Format code
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@gofmt -s -w .

## vet: Run go vet
vet:
	@echo "$(BLUE)Running go vet...$(NC)"
	@go vet ./...

## deps: Download dependencies
deps:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	@go mod download
	@go mod tidy

## dev: Run with live reload (requires air)
dev:
	@which air > /dev/null || (echo "air not installed. Run: go install github.com/air-verse/air@latest" && exit 1)
	@air

## frontend: Build frontend assets
frontend:
	@echo "$(BLUE)Building frontend...$(NC)"
	@if [ ! -f $(FRONTEND_DIR)/package.json ]; then \
		echo "No frontend/package.json found. If you have a frontend, add package.json in $(FRONTEND_DIR) or run 'make frontend' from the frontend repo."; \
		exit 1; \
	fi
	@cd $(FRONTEND_DIR) && \
	if [ -f package-lock.json ]; then \
		echo "package-lock.json found, attempting npm ci"; \
		npm ci || (echo "npm ci failed, falling back to npm install" && npm install); \
	else \
		echo "no package-lock.json, running npm install"; \
		npm install; \
	fi && npm run build

## docker: Build Docker image
docker:
	@echo "$(BLUE)Building Docker image...$(NC)"
	@docker build -t $(BINARY_NAME):$(VERSION) .
	@docker tag $(BINARY_NAME):$(VERSION) $(BINARY_NAME):latest

## docker-push: Push Docker image to registry
docker-push: docker
	@echo "$(BLUE)Pushing Docker image...$(NC)"
	@docker push $(BINARY_NAME):$(VERSION)
	@docker push $(BINARY_NAME):latest

## run: Run the application
run: build
	@echo "$(BLUE)Running $(BINARY_NAME)...$(NC)"
	@$(BUILD_DIR)/$(BINARY_NAME)

## install: Install the application
install: build
	@echo "$(BLUE)Installing $(BINARY_NAME)...$(NC)"
	@cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

## uninstall: Uninstall the application
uninstall:
	@echo "$(BLUE)Uninstalling $(BINARY_NAME)...$(NC)"
	@rm -f /usr/local/bin/$(BINARY_NAME)
