# ShadowMesh Makefile
# Post-Quantum Encrypted Private Network

.PHONY: all build clean test fmt vet lint help install

# Build configuration
VERSION ?= 0.1.0-alpha
BUILD_DIR = build
LDFLAGS = -ldflags="-s -w -X main.version=$(VERSION)"

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOCLEAN = $(GOCMD) clean
GOTEST = $(GOCMD) test
GOGET = $(GOCMD) get
GOFMT = $(GOCMD) fmt
GOVET = $(GOCMD) vet

# Binary names
CLIENT_DAEMON = shadowmesh-daemon
CLIENT_CLI = shadowmesh
RELAY_SERVER = shadowmesh-relay

all: clean build

## build: Build all components
build: build-client-daemon build-client-cli build-relay

## build-client-daemon: Build client daemon
build-client-daemon:
	@echo "Building client daemon..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_DAEMON) ./client/daemon

## build-client-cli: Build client CLI
build-client-cli:
	@echo "Building client CLI..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_CLI) ./client/cli

## build-relay: Build relay server
build-relay:
	@echo "Building relay server..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(RELAY_SERVER) ./relay/server

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

## test: Run all tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -cover -coverprofile=coverage.txt ./...
	$(GOCMD) tool cover -html=coverage.txt -o coverage.html

## fmt: Format Go code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

## lint: Run golangci-lint
lint:
	@echo "Running golangci-lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOCMD) mod download
	$(GOCMD) mod tidy

## install: Install binaries to /usr/local/bin
install: build
	@echo "Installing binaries..."
	@sudo cp $(BUILD_DIR)/$(CLIENT_DAEMON) /usr/local/bin/
	@sudo cp $(BUILD_DIR)/$(CLIENT_CLI) /usr/local/bin/
	@sudo cp $(BUILD_DIR)/$(RELAY_SERVER) /usr/local/bin/
	@echo "Installation complete!"

## dev: Run in development mode
dev:
	@echo "Starting in development mode..."
	$(GOCMD) run ./client/daemon/main.go

## help: Show this help message
help:
	@echo "ShadowMesh Makefile Commands:"
	@echo ""
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /'

# Default target
.DEFAULT_GOAL := help
