# Bloco Wallet Generator Makefile

# Variables
BINARY_NAME=bloco-vgen
SOURCE_FILE=./cmd/bloco-vgen/main.go
TEST_FILE=main_test.go

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
BUILD_FLAGS=-ldflags="-s -w"
RACE_FLAGS=-race

# Default target
.DEFAULT_GOAL := build

# Help target
.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# Initialize module and download dependencies
.PHONY: init
init: ## Initialize Go module and download dependencies
	$(GOMOD) init bloco-wallet
	$(GOMOD) tidy
	@echo "Dependencies initialized successfully"

# Download dependencies
.PHONY: deps
deps: ## Download and verify dependencies
	$(GOMOD) download
	$(GOMOD) verify
	@echo "Dependencies downloaded and verified"

# Tidy dependencies
.PHONY: tidy
tidy: ## Clean up dependencies
	$(GOMOD) tidy
	@echo "Dependencies tidied"

# Build the application
.PHONY: build
build: ## Build the application
	$(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME) $(SOURCE_FILE)
	@echo "Build completed: $(BINARY_NAME)"

# Build for different platforms
.PHONY: build-linux
build-linux: ## Build for Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME)-linux-amd64 $(SOURCE_FILE)
	@echo "Linux build completed: $(BINARY_NAME)-linux-amd64"

.PHONY: build-windows
build-windows: ## Build for Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(SOURCE_FILE)
	@echo "Windows build completed: $(BINARY_NAME)-windows-amd64.exe"

.PHONY: build-darwin
build-darwin: ## Build for macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME)-darwin-amd64 $(SOURCE_FILE)
	@echo "macOS build completed: $(BINARY_NAME)-darwin-amd64"

.PHONY: build-darwin-arm64
build-darwin-arm64: ## Build for macOS ARM64 (M1/M2)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) -o $(BINARY_NAME)-darwin-arm64 $(SOURCE_FILE)
	@echo "macOS ARM64 build completed: $(BINARY_NAME)-darwin-arm64"

# Build for all platforms
.PHONY: build-all
build-all: build-linux build-windows build-darwin build-darwin-arm64 ## Build for all platforms
	@echo "All platform builds completed"

# Run the application
.PHONY: run
run: build ## Build and run the application with default parameters
	./$(BINARY_NAME) --prefix abc --count 1

# Run with custom parameters
.PHONY: run-demo
run-demo: build ## Run demo with custom parameters
	./$(BINARY_NAME) --prefix dead --suffix beef --count 2 --progress

# Run tests
.PHONY: test
test: ## Run tests
	$(GOTEST) -v ./...

# Run tests with race detection
.PHONY: test-race
test-race: ## Run tests with race detection
	$(GOTEST) $(RACE_FLAGS) -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
.PHONY: bench
bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

# Run tests excluding integration tests
.PHONY: test-unit
test-unit: ## Run unit tests only (skip integration tests)
	$(GOTEST) -short -v ./...

# Lint the code
.PHONY: lint
lint: ## Lint the code (requires golangci-lint)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2"; \
	fi

# Format the code
.PHONY: fmt
fmt: ## Format the code
	$(GOCMD) fmt ./...
	@echo "Code formatted"

# Vet the code
.PHONY: vet
vet: ## Vet the code
	$(GOCMD) vet ./...
	@echo "Code vetted"

# Clean build artifacts
.PHONY: clean
clean: ## Clean build artifacts
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	@echo "Clean completed"

# Install the application
.PHONY: install
install: ## Install the application to GOPATH/bin
	$(GOCMD) install $(BUILD_FLAGS) $(SOURCE_FILE)
	@echo "Application installed to GOPATH/bin"

# Development workflow
.PHONY: dev
dev: fmt vet test build ## Run development workflow (format, vet, test, build)
	@echo "Development workflow completed"

# CI workflow
.PHONY: ci
ci: deps fmt vet test-race ## Run CI workflow (deps, format, vet, race tests)
	@echo "CI workflow completed"

# Quick validation
.PHONY: check
check: fmt vet test-unit ## Quick validation (format, vet, unit tests)
	@echo "Quick check completed"

# Generate mock data for testing
.PHONY: test-data
test-data: build ## Generate test wallets for validation
	@echo "Generating test wallets..."
	./$(BINARY_NAME) --prefix a --count 1
	./$(BINARY_NAME) --suffix 1 --count 1
	./$(BINARY_NAME) --prefix ab --suffix cd --count 1

# Performance test
.PHONY: perf-test
perf-test: build ## Run performance test with different complexities
	@echo "Running performance tests..."
	@echo "\n1 hex char prefix (expected ~16 attempts):"
	time ./$(BINARY_NAME) --prefix a --count 1
	@echo "\n2 hex char prefix (expected ~256 attempts):"
	time ./$(BINARY_NAME) --prefix ab --count 1
	@echo "\n3 hex char prefix (expected ~4096 attempts):"
	time ./$(BINARY_NAME) --prefix abc --count 1

# Benchmark tests
.PHONY: benchmark-test
benchmark-test: build ## Run benchmark tests
	@echo "Running benchmark tests..."
	./$(BINARY_NAME) benchmark --attempts 5000 --pattern "ff"
	./$(BINARY_NAME) benchmark --attempts 2500 --pattern "abc"

# Statistics tests
.PHONY: stats-test
stats-test: build ## Test statistics functionality
	@echo "Testing statistics functionality..."
	./$(BINARY_NAME) stats --prefix abc
	./$(BINARY_NAME) stats --prefix dead --suffix beef
	./$(BINARY_NAME) stats --prefix ABC --checksum

# Demo run with all features
.PHONY: demo
demo: build ## Run comprehensive demo
	@echo "Bloco Wallet Demo"
	@echo "==================="
	@echo "\n1. Simple wallet generation:"
	./$(BINARY_NAME) --prefix cafe --count 1
	@echo "\n2. Statistics analysis:"
	./$(BINARY_NAME) stats --prefix cafe
	@echo "\n3. Benchmark test:"
	./$(BINARY_NAME) benchmark --attempts 3000 --pattern "ff"
	@echo "\n4. Progress demo:"
	./$(BINARY_NAME) --prefix ab --progress --count 1

# Security check
.PHONY: security
security: ## Run security checks (requires gosec)
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Documentation
.PHONY: docs
docs: build ## Generate documentation
	./$(BINARY_NAME) --help > docs/CLI_USAGE.md
	@echo "Documentation updated"

# Release preparation
.PHONY: release
release: clean ci build-all ## Prepare release (clean, CI, build all platforms)
	@echo "Release preparation completed"
	@echo "Built binaries:"
	@ls -la $(BINARY_NAME)-*

# Show version info
.PHONY: version
version: ## Show Go version and module info
	@echo "Go version:"
	$(GOCMD) version
	@echo "\nModule info:"
	$(GOMOD) list -m all

# Run examples
.PHONY: examples
examples: build ## Run various examples
	@echo "Example 1: Simple prefix"
	./$(BINARY_NAME) --prefix cafe
	@echo "\nExample 2: Prefix + Suffix"
	./$(BINARY_NAME) --prefix dead --suffix beef
	@echo "\nExample 3: Multiple wallets"
	./$(BINARY_NAME) --prefix ab --count 3
	@echo "\nExample 4: With checksum"
	./$(BINARY_NAME) --prefix AbC --checksum
