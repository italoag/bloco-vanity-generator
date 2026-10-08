# Bloco Vanity Generator Makefile
#
# Works with any Go toolchain (including mise-managed Go): install paths are
# resolved with `go env` instead of relying on a pre-set $GOPATH/$GOBIN
# environment variable. The binary install directory is computed exactly the
# way `go install` computes it: $GOBIN if set, otherwise $GOPATH/bin.

# Project Configuration
NAME            := bloco-vgen
PACKAGE         := bloco-vgen
GO              ?= go
CMD_DIR         := cmd/bloco-vgen
BUILD_DIR       := build
DIST_DIR        := dist
OUTPUT_BIN      ?= ${BUILD_DIR}/${NAME}

VERSION         ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
GIT_REV         ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH      ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Native platform detection (from the active toolchain, e.g. the mise shim)
GOOS            := $(shell $(GO) env GOOS)
GOARCH          := $(shell $(GO) env GOARCH)

# Install directory: GOBIN when set (e.g. mise/asdf-managed Go), else GOPATH/bin
GOBIN_DIR       ?= $(shell $(GO) env GOBIN 2>/dev/null)
ifeq ($(strip $(GOBIN_DIR)),)
GOBIN_DIR       := $(shell $(GO) env GOPATH)/bin
endif

# Container Configuration
IMG_NAME        := ghcr.io/italoag/${NAME}
IMAGE           := ${IMG_NAME}:${VERSION}
BUILD_PLATFORMS ?= linux/amd64,linux/arm64

# CGO policy:
#   - Native darwin/arm64 builds enable CGO so the Metal backend is included.
#   - Cross-compiled builds are pure Go (CGO_ENABLED=0).
ifeq ($(GOOS)-$(GOARCH),darwin-arm64)
NATIVE_CGO      := 1
else
NATIVE_CGO      := 0
endif

# macOS CGO Linker Fix
ifeq ($(shell uname), Darwin)
CGO_LDFLAGS     ?= -Wl,-w
else
CGO_LDFLAGS     ?=
endif

# Date handling for different OS
SOURCE_DATE_EPOCH ?= $(shell date +%s)
ifeq ($(shell uname), Darwin)
DATE            ?= $(shell TZ=UTC date -j -f "%s" ${SOURCE_DATE_EPOCH} +"%Y-%m-%dT%H:%M:%SZ")
else
DATE            ?= $(shell date -u -d @${SOURCE_DATE_EPOCH} +"%Y-%m-%dT%H:%M:%SZ")
endif

# Isolated Go environment for tests: a throwaway HOME keeps keystores, config
# and caches out of the real home directory while preserving the module cache.
RUN_GO_ISOLATED = TEST_GOPATH="$$($(GO) env GOPATH)"; TEST_HOME="$$(mktemp -d)"; trap '[ ! -d "$$TEST_HOME" ] || chmod -R u+w "$$TEST_HOME" 2>/dev/null; rm -rf "$$TEST_HOME"' EXIT; export HOME="$$TEST_HOME" XDG_CONFIG_HOME="$$TEST_HOME/.config" XDG_CACHE_HOME="$$TEST_HOME/.cache" APPDATA="$$TEST_HOME/AppData/Roaming" LOCALAPPDATA="$$TEST_HOME/AppData/Local" USERPROFILE="$$TEST_HOME" GOPATH="$$TEST_GOPATH";

# Build matrix for cross-compilation (pure Go). The Metal-enabled macOS ARM64
# build is appended only on a native darwin/arm64 host (it needs the Metal SDK).
PLATFORMS       := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64
ifeq ($(GOOS)-$(GOARCH),darwin-arm64)
EXTRA_PLATFORMS := darwin/arm64-metal
endif
ALL_PLATFORMS   := $(PLATFORMS) $(EXTRA_PLATFORMS)
BUILD_TOTAL     := $(words $(ALL_PLATFORMS))

# LDFLAGS for version injection
LDFLAGS         := -s -w \
				   -X main.Version=${VERSION} \
				   -X main.GitCommit=${GIT_REV} \
				   -X main.BuildTime=${DATE}

# Colors for output (disabled when NO_COLOR is set)
# Fixed 256-color palette so hues render identically in any terminal theme
# (basic ANSI 30-37 colors get remapped by the theme: cyan → green, etc.)
ifeq ($(NO_COLOR),)
RED     := \033[38;5;203m
GREEN   := \033[38;5;84m
YELLOW  := \033[38;5;228m
BLUE    := \033[38;5;111m
PURPLE  := \033[38;5;141m
CYAN    := \033[38;5;87m
WHITE   := \033[38;5;255m
BOLD    := \033[1m
RESET   := \033[0m
endif

.PHONY: help
default: help

# Renders one progress-bar line. Usage inside a recipe:
#   $(call bar,<step>,<total>,<label>)
define bar
@i=$(1); total=$(2); width=24; \
	filled=$$((i * width / total)); \
	printf "  $(PURPLE)["; j=0; \
	while [ $$j -lt $$filled ]; do printf "█"; j=$$((j+1)); done; \
	while [ $$j -lt $$width ]; do printf "░"; j=$$((j+1)); done; \
	printf "]$(RESET) $(WHITE)%3d%%$(RESET) $(CYAN)(%d/%d)$(RESET) %s\n" \
		$$((i * 100 / total)) $$i $$total "$(3)"
endef

##@ Build Targets

.PHONY: build
build: ## Build for the current platform (includes Metal on native darwin/arm64)
	@echo "$(CYAN)Building ${NAME} for $(GOOS)/$(GOARCH) (CGO=$(NATIVE_CGO))...$(RESET)"
	@mkdir -p ${BUILD_DIR}
	@CGO_ENABLED=$(NATIVE_CGO) CGO_LDFLAGS="$(CGO_LDFLAGS)" $(GO) build \
		-ldflags "${LDFLAGS}" \
		-o ${OUTPUT_BIN} \
		./${CMD_DIR}
	@echo "$(GREEN)✓ Build complete: ${OUTPUT_BIN} (version ${VERSION})$(RESET)"

.PHONY: build-all
build-all: clean ## Build binaries and archives for all supported platforms
	@echo "$(CYAN)Building ${NAME} v${VERSION} for $(BUILD_TOTAL) platforms...$(RESET)"
	@mkdir -p ${DIST_DIR}
	@$(eval _done :=)$(foreach platform,$(ALL_PLATFORMS),$(call build_platform,$(platform),$(BUILD_TOTAL)))
	@echo "$(GREEN)✓ All platform builds complete ($(BUILD_TOTAL) artifacts in ${DIST_DIR})$(RESET)"

.PHONY: build-linux
build-linux: ## Build for Linux platforms (amd64, arm64)
	@echo "$(CYAN)Building ${NAME} for Linux platforms...$(RESET)"
	@mkdir -p ${DIST_DIR}
	@$(eval _done :=)$(call build_platform,linux/amd64,2)$(call build_platform,linux/arm64,2)
	@echo "$(GREEN)✓ Linux builds complete$(RESET)"

.PHONY: build-darwin
build-darwin: ## Build for macOS platforms (amd64, arm64)
	@echo "$(CYAN)Building ${NAME} for macOS platforms...$(RESET)"
	@mkdir -p ${DIST_DIR}
	@$(eval _done :=)$(call build_platform,darwin/amd64,2)$(call build_platform,darwin/arm64,2)
	@echo "$(GREEN)✓ macOS builds complete$(RESET)"

.PHONY: build-windows
build-windows: ## Build for Windows platform (amd64)
	@echo "$(CYAN)Building ${NAME} for Windows...$(RESET)"
	@mkdir -p ${DIST_DIR}
	@$(eval _done :=)$(call build_platform,windows/amd64,1)
	@echo "$(GREEN)✓ Windows build complete$(RESET)"

.PHONY: build-darwin-arm64-metal
build-darwin-arm64-metal: ## Build Metal-enabled for macOS ARM64 (requires native macOS host)
	@if [ "$(GOOS)-$(GOARCH)" != "darwin-arm64" ]; then \
		echo "$(RED)✗ Metal builds require a native macOS ARM64 host$(RESET)"; exit 1; \
	fi
	@echo "$(CYAN)Building ${NAME} for darwin/arm64 (Metal)...$(RESET)"
	@mkdir -p ${DIST_DIR}
	@$(eval _done :=)$(call build_platform,darwin/arm64-metal,1)
	@echo "$(GREEN)✓ Metal build complete$(RESET)"

##@ Run Targets

.PHONY: run
run: build ## Build and run the application with default parameters
	@${OUTPUT_BIN} --prefix abc --count 1

.PHONY: run-demo
run-demo: build ## Run demo with custom parameters
	@${OUTPUT_BIN} --prefix dead --suffix beef --count 2 --progress

##@ Testing Targets

.PHONY: test
test: ## Run all tests with race detector
	@echo "$(CYAN)Running tests...$(RESET)"
	@$(GO) clean --testcache
	@$(RUN_GO_ISOLATED) CGO_LDFLAGS="$(CGO_LDFLAGS)" $(GO) test ./... -v -race -count=1 -shuffle=on
	@echo "$(GREEN)✓ Tests complete$(RESET)"

.PHONY: test-fast
test-fast: ## Run tests with fastest parameters (development)
	@echo "$(CYAN)Running fast tests for development...$(RESET)"
	@$(GO) clean --testcache
	@$(RUN_GO_ISOLATED) CGO_LDFLAGS="$(CGO_LDFLAGS)" $(GO) test ./... -short -v -count=1 -shuffle=on
	@echo "$(GREEN)✓ Fast tests complete$(RESET)"

.PHONY: t
t: test-fast ## Alias for test-fast (quick development testing)

.PHONY: test-unit
test-unit: test-fast ## Alias for test-fast (unit tests only)

.PHONY: test-race
test-race: test ## Alias for test (race detector included)

.PHONY: test-metal
test-metal: ## Run engine/Metal backend tests
	@echo "$(CYAN)Running engine tests (Metal backend on darwin/arm64)...$(RESET)"
	@$(RUN_GO_ISOLATED) CGO_LDFLAGS="$(CGO_LDFLAGS)" CGO_ENABLED=$(NATIVE_CGO) $(GO) test ./internal/engine -v -count=1 -shuffle=on
	@echo "$(GREEN)✓ Engine tests complete$(RESET)"

.PHONY: test-backup
test-backup: ## Run fnox/age backup integration tests (requires fnox + age-keygen)
	@echo "$(CYAN)Running backup integration tests...$(RESET)"
	@$(RUN_GO_ISOLATED) BLOCO_FNOX_INTEGRATION=1 $(GO) test -race ./internal/backup ./internal/cli -run Fnox -count=1 -timeout=5m
	@echo "$(GREEN)✓ Backup integration tests complete$(RESET)"

.PHONY: cover
cover: ## Generate test coverage report (coverage.html)
	@echo "$(CYAN)Generating test coverage...$(RESET)"
	@$(RUN_GO_ISOLATED) CGO_LDFLAGS="$(CGO_LDFLAGS)" $(GO) test ./... -count=1 -shuffle=on -coverprofile=coverage.out
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(RESET)"

.PHONY: test-coverage
test-coverage: cover ## Alias for cover

.PHONY: bench
bench: ## Run benchmarks
	@echo "$(CYAN)Running benchmarks...$(RESET)"
	@$(RUN_GO_ISOLATED) CGO_LDFLAGS="$(CGO_LDFLAGS)" $(GO) test ./... -bench=. -benchmem
	@echo "$(GREEN)✓ Benchmarks complete$(RESET)"

##@ Code Quality Targets

.PHONY: lint
lint: ## Run linting checks
	@echo "$(CYAN)Running linter...$(RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "$(YELLOW)⚠ golangci-lint not installed, running go vet instead$(RESET)"; \
		$(GO) vet ./...; \
	fi
	@echo "$(GREEN)✓ Linting complete$(RESET)"

.PHONY: fmt
fmt: ## Format code
	@echo "$(CYAN)Formatting code...$(RESET)"
	@$(GO) fmt ./...
	@echo "$(GREEN)✓ Code formatted$(RESET)"

.PHONY: vet
vet: ## Run go vet
	@echo "$(CYAN)Running go vet...$(RESET)"
	@$(GO) vet ./...
	@echo "$(GREEN)✓ Vet complete$(RESET)"

.PHONY: mod
mod: ## Tidy and download modules
	@echo "$(CYAN)Tidying modules...$(RESET)"
	@$(GO) mod tidy
	@$(GO) mod download
	@echo "$(GREEN)✓ Modules updated$(RESET)"

.PHONY: tidy
tidy: mod ## Alias for mod

.PHONY: security
security: ## Run security checks (requires gosec)
	@echo "$(CYAN)Running security scan...$(RESET)"
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "$(YELLOW)⚠ gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest$(RESET)"; \
	fi
	@echo "$(GREEN)✓ Security scan complete$(RESET)"

##@ Container Targets

.PHONY: docker-build
docker-build: ## Build Docker image for the current platform
	@echo "$(CYAN)Building Docker image ${IMAGE}...$(RESET)"
	@docker buildx build \
		--build-arg VERSION=${VERSION} \
		--build-arg GIT_REV=${GIT_REV} \
		--build-arg BUILD_DATE=${DATE} \
		--rm -t ${IMAGE} \
		--load .
	@echo "$(GREEN)✓ Docker image built: ${IMAGE}$(RESET)"

.PHONY: docker-push
docker-push: ## Build and push multi-platform Docker images
	@echo "$(CYAN)Pushing Docker images for platforms: ${BUILD_PLATFORMS}$(RESET)"
	@docker buildx build \
		--platform ${BUILD_PLATFORMS} \
		--build-arg VERSION=${VERSION} \
		--build-arg GIT_REV=${GIT_REV} \
		--build-arg BUILD_DATE=${DATE} \
		--rm -t ${IMAGE} \
		--push .
	@echo "$(GREEN)✓ Docker images pushed$(RESET)"

.PHONY: docker-run
docker-run: ## Run the container locally
	@echo "$(CYAN)Running container locally...$(RESET)"
	@docker run --rm -it ${IMAGE}

##@ Release Targets

.PHONY: release-prep
release-prep: clean build-all checksums ## Prepare release artifacts
	@echo "$(GREEN)✓ Release artifacts prepared in ${DIST_DIR}$(RESET)"

.PHONY: release-local
release-local: release-prep ## Create local release package
	@echo "$(CYAN)Creating local release package...$(RESET)"
	@mkdir -p ${DIST_DIR}/release
	@cd ${DIST_DIR} && tar -czf release/${NAME}-${VERSION}-release.tar.gz *.tar.gz *.zip checksums.txt
	@echo "$(GREEN)✓ Local release package created: ${DIST_DIR}/release/${NAME}-${VERSION}-release.tar.gz$(RESET)"

.PHONY: checksums
checksums: ## Generate checksums for release artifacts
	@echo "$(CYAN)Generating checksums...$(RESET)"
	@cd ${DIST_DIR} && \
		find . -name "*.tar.gz" -o -name "*.zip" | \
		xargs shasum -a 256 > checksums.txt
	@echo "$(GREEN)✓ Checksums generated: ${DIST_DIR}/checksums.txt$(RESET)"

.PHONY: release-smoke
release-smoke: build ## Smoke-test the built artifact and validate the version banner
	@echo "$(CYAN)Smoke-testing ${OUTPUT_BIN}...$(RESET)"
	@${OUTPUT_BIN} --version | grep -qF "${VERSION}" || (echo "$(RED)✗ version banner does not match ${VERSION}$(RESET)"; exit 1)
	@echo "$(GREEN)✓ Artifact runs and reports version $(VERSION)$(RESET)"
	@if ls ${DIST_DIR}/*.tar.gz ${DIST_DIR}/*.zip >/dev/null 2>&1; then \
		echo "$(CYAN)Smoke-testing promoted artifacts...$(RESET)"; \
		for archive in ${DIST_DIR}/*.tar.gz ${DIST_DIR}/*.zip; do \
			shasum -a 256 "$$archive" | grep -q "^" || exit 1; \
		done; \
		echo "$(GREEN)✓ Promoted artifacts present and checksummed$(RESET)"; \
	fi

.PHONY: sbom
sbom: ## Generate a CycloneDX-style SBOM from the module graph
	@echo "$(CYAN)Generating SBOM...$(RESET)"
	@mkdir -p ${DIST_DIR}
	@$(GO) list -m all | python3 -c 'import sys,time,json; modules=[(p[0],p[1]) for l in sys.stdin if l.strip() for p in [l.split()] if len(p) >= 2 and p[1] != "=>"]; doc={"bomFormat":"CycloneDX","specVersion":"1.5","version":1,"metadata":{"timestamp":time.strftime("%Y-%m-%dT%H:%M:%SZ",time.gmtime()),"component":{"type":"application","name":"${NAME}","version":"${VERSION}"}},"components":[{"type":"library","name":m[0],"version":m[1]} for m in sorted(modules)]}; open("${DIST_DIR}/sbom.json","w").write(json.dumps(doc,indent=2))'
	@echo "$(GREEN)✓ SBOM generated: ${DIST_DIR}/sbom.json$(RESET)"

.PHONY: release-check
release-check: test lint release-smoke sbom ## Run all release gates before promotion
	@echo "$(GREEN)✓ Release gates passed$(RESET)"

##@ Workflow Targets

.PHONY: check
check: ## Quick validation (fmt → vet → test-fast)
	$(call bar,1,3,fmt)
	@$(MAKE) --no-print-directory fmt
	$(call bar,2,3,vet)
	@$(MAKE) --no-print-directory vet
	$(call bar,3,3,test-fast)
	@$(MAKE) --no-print-directory test-fast
	@echo "$(GREEN)✓ Quick check completed$(RESET)"

.PHONY: dev
dev: ## Development workflow (fmt → vet → test-fast → build)
	$(call bar,1,4,fmt)
	@$(MAKE) --no-print-directory fmt
	$(call bar,2,4,vet)
	@$(MAKE) --no-print-directory vet
	$(call bar,3,4,test-fast)
	@$(MAKE) --no-print-directory test-fast
	$(call bar,4,4,build)
	@$(MAKE) --no-print-directory build
	@echo "$(GREEN)✓ Development workflow completed$(RESET)"

.PHONY: ci
ci: ## CI workflow (deps → fmt → vet → test)
	$(call bar,1,4,deps)
	@$(MAKE) --no-print-directory deps
	$(call bar,2,4,fmt)
	@$(MAKE) --no-print-directory fmt
	$(call bar,3,4,vet)
	@$(MAKE) --no-print-directory vet
	$(call bar,4,4,test)
	@$(MAKE) --no-print-directory test
	@echo "$(GREEN)✓ CI workflow completed$(RESET)"

.PHONY: release
release: ## Prepare release (clean → ci → build-all → checksums)
	$(call bar,1,4,clean)
	@$(MAKE) --no-print-directory clean
	$(call bar,2,4,ci)
	@$(MAKE) --no-print-directory ci
	$(call bar,3,4,build-all)
	@$(MAKE) --no-print-directory build-all
	$(call bar,4,4,checksums)
	@$(MAKE) --no-print-directory checksums
	@echo "$(GREEN)✓ Release preparation completed$(RESET)"
	@echo "$(CYAN)Built artifacts (version $(VERSION)):$(RESET)"
	@ls -lh ${DIST_DIR}

##@ Demo & Examples

.PHONY: test-data
test-data: build ## Generate test wallets for validation
	@echo "$(CYAN)Generating test wallets...$(RESET)"
	@${OUTPUT_BIN} --prefix a --count 1
	@${OUTPUT_BIN} --suffix 1 --count 1
	@${OUTPUT_BIN} --prefix ab --suffix cd --count 1
	@echo "$(GREEN)✓ Test wallets generated$(RESET)"

.PHONY: perf-test
perf-test: build ## Run performance test with different complexities
	@echo "$(CYAN)Running performance tests...$(RESET)"
	@echo "$(YELLOW)1 hex char prefix (expected ~16 attempts):$(RESET)"
	@time ${OUTPUT_BIN} --prefix a --count 1
	@echo "$(YELLOW)2 hex char prefix (expected ~256 attempts):$(RESET)"
	@time ${OUTPUT_BIN} --prefix ab --count 1
	@echo "$(YELLOW)3 hex char prefix (expected ~4096 attempts):$(RESET)"
	@time ${OUTPUT_BIN} --prefix abc --count 1
	@echo "$(GREEN)✓ Performance tests complete$(RESET)"

.PHONY: benchmark-test
benchmark-test: build ## Run benchmark tests
	@echo "$(CYAN)Running benchmark tests...$(RESET)"
	@${OUTPUT_BIN} benchmark --attempts 5000 --pattern "ff"
	@${OUTPUT_BIN} benchmark --attempts 2500 --pattern "abc"
	@echo "$(GREEN)✓ Benchmark tests complete$(RESET)"

.PHONY: stats-test
stats-test: build ## Test statistics functionality
	@echo "$(CYAN)Testing statistics functionality...$(RESET)"
	@${OUTPUT_BIN} stats --prefix abc
	@${OUTPUT_BIN} stats --prefix dead --suffix beef
	@${OUTPUT_BIN} stats --prefix ABC --checksum
	@echo "$(GREEN)✓ Statistics tests complete$(RESET)"

.PHONY: demo
demo: build ## Run comprehensive demo
	@echo "$(CYAN)Bloco Vanity Generator Demo$(RESET)"
	@echo "$(CYAN)===========================$(RESET)"
	@echo "$(YELLOW)1. Simple wallet generation:$(RESET)"
	@${OUTPUT_BIN} --prefix cafe --count 1
	@echo "$(YELLOW)2. Statistics analysis:$(RESET)"
	@${OUTPUT_BIN} stats --prefix cafe
	@echo "$(YELLOW)3. Benchmark test:$(RESET)"
	@${OUTPUT_BIN} benchmark --attempts 3000 --pattern "ff"
	@echo "$(YELLOW)4. Progress demo:$(RESET)"
	@${OUTPUT_BIN} --prefix ab --progress --count 1
	@echo "$(GREEN)✓ Demo complete$(RESET)"

.PHONY: examples
examples: build ## Run various examples
	@echo "$(CYAN)Example 1: Simple prefix$(RESET)"
	@${OUTPUT_BIN} --prefix cafe
	@echo "$(CYAN)Example 2: Prefix + Suffix$(RESET)"
	@${OUTPUT_BIN} --prefix dead --suffix beef
	@echo "$(CYAN)Example 3: Multiple wallets$(RESET)"
	@${OUTPUT_BIN} --prefix ab --count 3
	@echo "$(CYAN)Example 4: With checksum$(RESET)"
	@${OUTPUT_BIN} --prefix AbC --checksum
	@echo "$(GREEN)✓ Examples complete$(RESET)"

##@ Utility Targets

.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(CYAN)Cleaning build artifacts...$(RESET)"
	@$(GO) clean
	@rm -rf ${BUILD_DIR} ${DIST_DIR}
	@rm -f ${NAME} ${NAME}-* coverage.out coverage.html
	@echo "$(GREEN)✓ Clean complete$(RESET)"

.PHONY: deps
deps: ## Install build dependencies
	@echo "$(CYAN)Installing dependencies...$(RESET)"
	@$(GO) mod download
	@$(GO) mod verify
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing golangci-lint...$(RESET)"; \
		$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2; \
	fi
	@echo "$(GREEN)✓ Dependencies installed$(RESET)"

.PHONY: install
install: ## Install the binary into the Go bin directory ($(GOBIN_DIR))
	@echo "$(CYAN)Installing ${NAME} to ${GOBIN_DIR}...$(RESET)"
	@mkdir -p "${GOBIN_DIR}"
	@GOBIN="${GOBIN_DIR}" CGO_ENABLED=$(NATIVE_CGO) CGO_LDFLAGS="$(CGO_LDFLAGS)" $(GO) install -ldflags "${LDFLAGS}" ./${CMD_DIR}
	@echo "$(GREEN)✓ ${NAME} ${VERSION} installed to ${GOBIN_DIR}/${NAME}$(RESET)"

.PHONY: uninstall
uninstall: ## Remove the installed binary from $(GOBIN_DIR)
	@echo "$(CYAN)Uninstalling ${NAME}...$(RESET)"
	@rm -f "${GOBIN_DIR}/${NAME}"
	@echo "$(GREEN)✓ ${NAME} uninstalled$(RESET)"

.PHONY: docs
docs: build ## Generate documentation
	@echo "$(CYAN)Generating documentation...$(RESET)"
	@${OUTPUT_BIN} --help > docs/CLI_USAGE.md
	@echo "$(GREEN)✓ Documentation updated: docs/CLI_USAGE.md$(RESET)"

.PHONY: version
version: ## Show version information
	@echo "Name:        ${NAME}"
	@echo "Version:     ${VERSION}"
	@echo "Git Commit:  ${GIT_REV}"
	@echo "Git Branch:  ${GIT_BRANCH}"
	@echo "Build Date:  ${DATE}"
	@echo "Go Version:  $$($(GO) version)"

.PHONY: info
info: ## Show build environment information
	@echo "$(CYAN)Build Environment Information:$(RESET)"
	@echo "Name:           ${NAME}"
	@echo "Version:        ${VERSION}"
	@echo "Package:        ${PACKAGE}"
	@echo "Git Commit:     ${GIT_REV}"
	@echo "Git Branch:     ${GIT_BRANCH}"
	@echo "Build Date:     ${DATE}"
	@echo "Go Version:     $$($(GO) version)"
	@echo "Go Env GOOS:    $(GOOS)"
	@echo "Go Env GOARCH:  $(GOARCH)"
	@echo "Native CGO:     $(NATIVE_CGO)"
	@echo "CGO LDFLAGS:    ${CGO_LDFLAGS}"
	@echo "GOBIN dir:      ${GOBIN_DIR}"
	@echo "Platforms:      $(ALL_PLATFORMS)"
	@echo "Docker image:   ${IMAGE}"

##@ Help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\n$(PURPLE)Usage:$(RESET)\n  make $(CYAN)<target>$(RESET)\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  $(CYAN)%-24s$(RESET) %s\n", $$1, $$2 } /^##@/ { printf "\n$(PURPLE)%s$(RESET)\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(WHITE)Install dir:$(RESET) $(GOBIN_DIR)"
	@echo "$(WHITE)Platform:$(RESET)    $(GOOS)/$(GOARCH) (CGO=$(NATIVE_CGO))"

# Cross-compilation function. $(1) = os/arch[-variant], $(2) = total steps.
# The "metal" arch variant flips CGO on (native darwin/arm64 only).
define build_platform
	$(eval GOOS_P := $(word 1,$(subst /, ,$(1))))
	$(eval ARCH_RAW := $(word 2,$(subst /, ,$(1))))
	$(eval GOARCH_P := $(firstword $(subst -, ,$(ARCH_RAW))))
	$(eval VARIANT := $(if $(filter %metal,$(ARCH_RAW)),-metal,))
	$(eval P_CGO := $(if $(VARIANT),1,0))
	$(eval EXT := $(if $(filter windows,$(GOOS_P)),.exe,))
	$(eval OUT_BIN := ${DIST_DIR}/${NAME}-${VERSION}-$(GOOS_P)-$(ARCH_RAW)$(EXT))
	$(eval ARCHIVE := ${DIST_DIR}/${NAME}-${VERSION}-$(GOOS_P)-$(ARCH_RAW)$(if $(filter windows,$(GOOS_P)),.zip,.tar.gz))
	$(eval _done := $(_done) x)
	$(call bar,$(words $(_done)),$(2),$(GOOS_P)/$(ARCH_RAW))
	@CGO_ENABLED=$(P_CGO) GOOS=$(GOOS_P) GOARCH=$(GOARCH_P) CGO_LDFLAGS="$(CGO_LDFLAGS)" \
		$(GO) build -ldflags "${LDFLAGS}" -o $(OUT_BIN) ./${CMD_DIR}
	@if [ "$(GOOS_P)" = "windows" ]; then \
		cd ${DIST_DIR} && zip -q $(notdir $(ARCHIVE)) $(notdir $(OUT_BIN)); \
	else \
		cd ${DIST_DIR} && tar -czf $(notdir $(ARCHIVE)) $(notdir $(OUT_BIN)); \
	fi
	@rm -f $(OUT_BIN)
	@echo "$(GREEN)    ✓ $(GOOS_P)/$(GOARCH_P)$(VARIANT) → $(notdir $(ARCHIVE))$(RESET)"
endef
