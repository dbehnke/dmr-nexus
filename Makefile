.PHONY: all build clean test test-coverage lint fmt vet deps dev frontend prepare-frontend-embed build-embed docker compose-build docker-compose-build docker-compose-up docker-compose-down docker-push run help

# Variables
BINARY_NAME=dmr-nexus
YSF2DMR_BINARY=ysf2dmr
BUILD_DIR=bin
FRONTEND_DIR=frontend
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME) -s -w"

# Colors for output
BLUE=\033[0;34m
NC=\033[0m # No Color

all: build

## help: Display this help message
help:
	@echo "$(BLUE)DMR-Nexus Makefile$(NC)"
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  $(BLUE)%-15s$(NC) %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## build: Build all application binaries (dmr-nexus and ysf2dmr)
build: build-dmr-nexus build-ysf2dmr

## build-dmr-nexus: Build the dmr-nexus binary
build-dmr-nexus: deps
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dmr-nexus

## build-ysf2dmr: Build the ysf2dmr binary
build-ysf2dmr: deps
	@echo "$(BLUE)Building $(YSF2DMR_BINARY)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(YSF2DMR_BINARY) ./cmd/ysf2dmr

## build-old: Build just the dmr-nexus binary (deprecated, use build-dmr-nexus)
build-old: deps
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dmr-nexus



## prepare-frontend-embed: Copy frontend build into package path so go:embed can find it
# Note: this target only copies files. When building locally prefer `make build-embed`
# which depends on the `frontend` target and will build the SPA first.
prepare-frontend-embed:
	@echo "$(BLUE)Preparing frontend assets for go:embed...$(NC)"
	@rm -rf pkg/web/frontend/dist || true
	@mkdir -p pkg/web/frontend
	@cp -a $(FRONTEND_DIR)/dist pkg/web/frontend/


## build-embed: Build the application with embedded frontend assets (uses -tags=embed)
## This target builds the frontend locally first then copies and builds the binary.
build-embed: deps frontend prepare-frontend-embed
	@echo "$(BLUE)Building $(BINARY_NAME) with embedded frontend assets...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build $(LDFLAGS) -tags=embed -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dmr-nexus

## clean: Remove build artifacts
clean:
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.txt coverage.html
	@rm -rf $(FRONTEND_DIR)/dist
	@rm -rf $(FRONTEND_DIR)/node_modules
	@rm -rf pkg/web/frontend/dist || true

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

## docker: Build Docker image with version info
docker:
	@echo "$(BLUE)Building Docker image...$(NC)"
	@echo "Version: $(VERSION)"
	@echo "Commit:  $(GIT_COMMIT)"
	@echo "Build:   $(BUILD_TIME)"
	@docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(BINARY_NAME):$(VERSION) \
		-t $(BINARY_NAME):latest \
		.

## docker-compose-up: Start services with docker-compose
docker-compose-up:
	@echo "$(BLUE)Starting docker-compose services...$(NC)"
	@VERSION=$(VERSION) GIT_COMMIT=$(GIT_COMMIT) BUILD_TIME=$(BUILD_TIME) docker-compose up -d

## compose-build: Build docker compose images using git-derived version info
compose-build:
	@echo "$(BLUE)Building docker compose images with VERSION=$(VERSION)...$(NC)"
	@VERSION=$(VERSION) GIT_COMMIT=$(GIT_COMMIT) BUILD_TIME=$(BUILD_TIME) docker compose build

## docker-compose-build: compatibility alias for compose-build
docker-compose-build: compose-build

## docker-compose-down: Stop docker-compose services
docker-compose-down:
	@echo "$(BLUE)Stopping docker-compose services...$(NC)"
	@docker-compose down

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
