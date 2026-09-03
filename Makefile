# Bloco Vanity Generator Makefile
#
# Works with any Go toolchain (including mise-managed Go): install paths are
# resolved with `go env` instead of relying on a pre-set $GOPATH/$GOBIN
# environment variable. The binary install directory is computed exactly the
# way `go install` computes it: $GOBIN if set, otherwise $GOPATH/bin.

# ---------------------------------------------------------------------------
# Project configuration
# ---------------------------------------------------------------------------
BINARY_NAME  := bloco-vgen
PACKAGE_PATH := ./cmd/bloco-vgen
MODULE_NAME  := bloco-vgen

# ---------------------------------------------------------------------------
# Go toolchain
# ---------------------------------------------------------------------------
GOCMD  := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD  := $(GOCMD) mod
GOGET  := $(GOCMD) get

# Native platform detection (from the active toolchain, e.g. the mise shim)
GOOS   := $(shell $(GOCMD) env GOOS)
GOARCH := $(shell $(GOCMD) env GOARCH)

# Binary install directory. `go env GOBIN` is empty unless explicitly set, so
# fall back to the computed GOPATH/bin. This matches `go install` semantics.
GOBIN  := $(shell $(GOCMD) env GOBIN)
GOPATH := $(shell $(GOCMD) env GOPATH)
ifeq ($(strip $(GOBIN)),)
GOBIN  := $(GOPATH)/bin
endif

# ---------------------------------------------------------------------------
# Version information (injected into the binary with -ldflags -X)
# ---------------------------------------------------------------------------
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS = -ldflags="-s -w -X main.Version=$(VERSION) -X main.GitCommit=$(GIT_COMMIT) -X main.BuildTime=$(BUILD_TIME)"

# CGO policy:
#   - Native darwin/arm64 builds enable CGO so the Metal backend is included.
#   - Cross-compiled builds are pure Go (CGO_ENABLED=0).
ifeq ($(GOOS)-$(GOARCH),darwin-arm64)
NATIVE_CGO := 1
else
NATIVE_CGO := 0
endif

RACE_FLAGS := -race

# Default target
.DEFAULT_GOAL := build

# ---------------------------------------------------------------------------
# Help
# ---------------------------------------------------------------------------
.PHONY: help
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Install directory: $(GOBIN)"
	@echo "Native platform:   $(GOOS)/$(GOARCH) (CGO=$(NATIVE_CGO))"

# ---------------------------------------------------------------------------
# Module management
# ---------------------------------------------------------------------------
.PHONY: init
init: ## Initialize Go module and download dependencies
	@if [ ! -f go.mod ]; then $(GOMOD) init $(MODULE_NAME); fi
	$(GOMOD) tidy
	@echo "Dependencies initialized successfully"

.PHONY: deps
deps: ## Download and verify dependencies
	$(GOMOD) download
	$(GOMOD) verify
	@echo "Dependencies downloaded and verified"

.PHONY: tidy
tidy: ## Clean up dependencies
	$(GOMOD) tidy
	@echo "Dependencies tidied"

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------
.PHONY: build
build: ## Build for the current platform (includes Metal on macOS arm64)
	CGO_ENABLED=$(NATIVE_CGO) $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) $(PACKAGE_PATH)
	@echo "Build completed: $(BINARY_NAME) ($(GOOS)/$(GOARCH), version $(VERSION))"

# Build for different platforms
.PHONY: build-linux
build-linux: ## Build for Linux AMD64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 $(PACKAGE_PATH)
	@echo "Linux build completed: $(BINARY_NAME)-linux-amd64"

.PHONY: build-windows
build-windows: ## Build for Windows AMD64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe $(PACKAGE_PATH)
	@echo "Windows build completed: $(BINARY_NAME)-windows-amd64.exe"

.PHONY: build-darwin
build-darwin: ## Build for macOS AMD64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 $(PACKAGE_PATH)
	@echo "macOS build completed: $(BINARY_NAME)-darwin-amd64"

.PHONY: build-darwin-arm64
build-darwin-arm64: ## Build CPU-only for macOS ARM64 (M1/M2)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 $(PACKAGE_PATH)
	@echo "macOS ARM64 CPU-only build completed: $(BINARY_NAME)-darwin-arm64"

.PHONY: build-darwin-arm64-metal
build-darwin-arm64-metal: ## Build Metal-enabled for macOS ARM64 (requires native macOS/Metal toolchain)
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64-metal $(PACKAGE_PATH)
	@echo "macOS ARM64 Metal build completed: $(BINARY_NAME)-darwin-arm64-metal"

# Build for all platforms. The Metal-enabled macOS ARM64 build is included
# only when building natively on macOS ARM64 (it needs the Metal SDK).
ifeq ($(GOOS)-$(GOARCH),darwin-arm64)
BUILD_ALL_TARGETS := build-linux build-windows build-darwin build-darwin-arm64 build-darwin-arm64-metal
else
BUILD_ALL_TARGETS := build-linux build-windows build-darwin build-darwin-arm64
endif

.PHONY: build-all
build-all: $(BUILD_ALL_TARGETS) ## Build for all platforms
	@echo "All platform builds completed"

# ---------------------------------------------------------------------------
# Run
# ---------------------------------------------------------------------------
.PHONY: run
run: build ## Build and run the application with default parameters
	./$(BINARY_NAME) --prefix abc --count 1

.PHONY: run-demo
run-demo: build ## Run demo with custom parameters
	./$(BINARY_NAME) --prefix dead --suffix beef --count 2 --progress

# ---------------------------------------------------------------------------
# Test
# ---------------------------------------------------------------------------
.PHONY: test
test: ## Run tests
	$(GOTEST) -v ./...

.PHONY: test-race
test-race: ## Run tests with race detection
	$(GOTEST) $(RACE_FLAGS) -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: bench
bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

.PHONY: test-unit
test-unit: ## Run unit tests only (skip integration tests)
	$(GOTEST) -short -v ./...

# ---------------------------------------------------------------------------
# Quality
# ---------------------------------------------------------------------------
.PHONY: lint
lint: ## Lint the code (requires golangci-lint)
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2"; \
	fi

.PHONY: fmt
fmt: ## Format the code
	$(GOCMD) fmt ./...
	@echo "Code formatted"

.PHONY: vet
vet: ## Vet the code
	$(GOCMD) vet ./...
	@echo "Code vetted"

# ---------------------------------------------------------------------------
# Install / uninstall
# ---------------------------------------------------------------------------
.PHONY: install
install: ## Install the binary into the Go bin directory ($(GOBIN))
	@mkdir -p "$(GOBIN)"
	GOBIN="$(GOBIN)" $(GOCMD) install $(LDFLAGS) $(PACKAGE_PATH)
	@echo "Installed $(BINARY_NAME) $(VERSION) to:"
	@echo "  $(GOBIN)/$(BINARY_NAME)"
	@echo "This directory is managed by mise and is already on your PATH"
	@echo "when the go toolchain is active (check with: which $(BINARY_NAME))."

.PHONY: uninstall
uninstall: ## Remove the installed binary from $(GOBIN)
	@rm -f "$(GOBIN)/$(BINARY_NAME)"
	@echo "Removed $(GOBIN)/$(BINARY_NAME)"

# ---------------------------------------------------------------------------
# Clean / development / release
# ---------------------------------------------------------------------------
.PHONY: clean
clean: ## Clean build artifacts
	$(GOCMD) clean
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html
	@echo "Clean completed"

.PHONY: dev
dev: fmt vet test build ## Run development workflow (format, vet, test, build)
	@echo "Development workflow completed"

.PHONY: ci
ci: deps fmt vet test-race ## Run CI workflow (deps, format, vet, race tests)
	@echo "CI workflow completed"

.PHONY: check
check: fmt vet test-unit ## Quick validation (format, vet, unit tests)
	@echo "Quick check completed"

.PHONY: release
release: clean ci build-all ## Prepare release (clean, CI, build all platforms)
	@echo "Release preparation completed"
	@echo "Built binaries (version $(VERSION)):"
	@ls -la $(BINARY_NAME)-*

# ---------------------------------------------------------------------------
# Misc
# ---------------------------------------------------------------------------
.PHONY: version
version: ## Show toolchain, install paths, and build version info
	@echo "Go version:"
	$(GOCMD) version
	@echo "GOROOT:    $(shell $(GOCMD) env GOROOT)"
	@echo "GOPATH:    $(GOPATH)"
	@echo "GOBIN:     $(GOBIN)"
	@echo ""
	@echo "Build version info:"
	@echo "  Version:    $(VERSION)"
	@echo "  GitCommit:  $(GIT_COMMIT)"
	@echo "  BuildTime:  $(BUILD_TIME)"
	@echo ""
	@echo "Module info:"
	$(GOMOD) list -m all

.PHONY: security
security: ## Run security checks (requires gosec)
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

.PHONY: docs
docs: build ## Generate documentation
	./$(BINARY_NAME) --help > docs/CLI_USAGE.md
	@echo "Documentation updated"

.PHONY: test-data
test-data: build ## Generate test wallets for validation
	@echo "Generating test wallets..."
	./$(BINARY_NAME) --prefix a --count 1
	./$(BINARY_NAME) --suffix 1 --count 1
	./$(BINARY_NAME) --prefix ab --suffix cd --count 1

.PHONY: perf-test
perf-test: build ## Run performance test with different complexities
	@echo "Running performance tests..."
	@echo "\n1 hex char prefix (expected ~16 attempts):"
	time ./$(BINARY_NAME) --prefix a --count 1
	@echo "\n2 hex char prefix (expected ~256 attempts):"
	time ./$(BINARY_NAME) --prefix ab --count 1
	@echo "\n3 hex char prefix (expected ~4096 attempts):"
	time ./$(BINARY_NAME) --prefix abc --count 1

.PHONY: benchmark-test
benchmark-test: build ## Run benchmark tests
	@echo "Running benchmark tests..."
	./$(BINARY_NAME) benchmark --attempts 5000 --pattern "ff"
	./$(BINARY_NAME) benchmark --attempts 2500 --pattern "abc"

.PHONY: stats-test
stats-test: build ## Test statistics functionality
	@echo "Testing statistics functionality..."
	./$(BINARY_NAME) stats --prefix abc
	./$(BINARY_NAME) stats --prefix dead --suffix beef
	./$(BINARY_NAME) stats --prefix ABC --checksum

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
