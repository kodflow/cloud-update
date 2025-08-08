# Default target when just running 'make'
.DEFAULT_GOAL := help

# ==================================================================================== #
# VARIABLES
# ==================================================================================== #

BINARY_NAME := cloud-update
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Tools
BAZEL := bazel
GO := go
GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint
DOCKER_COMPOSE := docker compose

# Paths
SRC_PATH := ./src/...
CMD_PATH := //src/cmd/cloud-update:cloud-update
E2E_COMPOSE := src/test/e2e/docker-compose.yml

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo ''
	@echo 'Usage:'
	@echo '  make <target>'
	@echo ''
	@echo 'Development:'
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "run" "run the application locally"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "build" "build the binary for current platform"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "build/all" "build binaries for all platforms"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "deps" "download and verify dependencies"
	@echo ''
	@echo 'Testing:'
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "test" "run all tests with quality checks"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "test/unit" "run unit tests"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "test/e2e" "run end-to-end tests"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "test/quick" "quick test without linting or E2E"
	@echo ''
	@echo 'Quality:'
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/analyze" "run complete code analysis"
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/format" "format all code (Go, YAML, JSON, MD)"
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/lint" "run linters"
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/security" "run security analysis"
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/fix" "automatically fix all fixable issues"
	@printf "  $(BLUE)%-20s$(NC) %s\n" "quality/validate" "validate all quality gates pass"
	@echo ''
	@echo 'Git Hooks:'
	@printf "  $(GREEN)%-20s$(NC) %s\n" "hooks/install" "install Git hooks for automatic checks"
	@printf "  $(GREEN)%-20s$(NC) %s\n" "hooks/uninstall" "remove Git hooks configuration"
	@printf "  $(GREEN)%-20s$(NC) %s\n" "hooks/status" "check Git hooks configuration status"
	@echo ''
	@echo 'Docker:'
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "docker/up" "start E2E test containers"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "docker/down" "stop E2E test containers"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "docker/logs" "show container logs"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "docker/status" "show container status"
	@echo ''
	@echo 'Operations:'
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "install" "install the binary to /usr/local/bin"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "uninstall" "remove the binary from /usr/local/bin"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "clean" "clean build artifacts and test containers"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "version" "show version information"
	@echo ''
	@echo 'Tools:'
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "tools/check" "check if required tools are installed"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "tools/install" "install optional development tools"
	@echo ''
	@echo 'CI/CD:'
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "ci" "run all CI checks (quality, test, build)"
	@printf "  $(YELLOW)%-20s$(NC) %s\n" "ci/validate" "validate CI pipeline locally"
	@echo ''

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run: run the application locally
.PHONY: run
run: build
	@echo "$(YELLOW)▶ Starting $(BINARY_NAME)...$(NC)"
	@./$(BINARY_NAME)

## build: build the binary for current platform
.PHONY: build
build:
	@echo "$(YELLOW)▶ Building $(BINARY_NAME)...$(NC)"
	@GOPROXY=https://proxy.golang.org,direct $(BAZEL) build $(CMD_PATH) \
		--stamp \
		--workspace_status_command="echo BUILD_VERSION $(VERSION)"
	@cp bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update ./$(BINARY_NAME)
	@chmod +x ./$(BINARY_NAME)
	@echo "$(GREEN)✓ Binary built: ./$(BINARY_NAME)$(NC)"

## build/all: build binaries for all platforms
.PHONY: build/all
build/all:
	@echo "$(YELLOW)▶ Building for all platforms...$(NC)"
	@chmod +x scripts/build-all.sh
	@./scripts/build-all.sh
	@echo "$(GREEN)✓ All binaries built$(NC)"

## deps: download and verify dependencies
.PHONY: deps
deps:
	@echo "$(YELLOW)▶ Downloading dependencies...$(NC)"
	@GOPROXY=https://proxy.golang.org,direct $(GO) mod download
	@$(GO) mod verify
	@GOPROXY=https://proxy.golang.org,direct $(BAZEL) run //:gazelle
	@echo "$(GREEN)✓ Dependencies ready$(NC)"

# ==================================================================================== #
# TESTING
# ==================================================================================== #

## test: run all tests with quality checks
.PHONY: test
test: quality/analyze test/unit test/e2e
	@echo "$(GREEN)✓ All tests and quality checks passed$(NC)"

## test/unit: run unit tests
.PHONY: test/unit
test/unit:
	@echo "$(YELLOW)▶ Running unit tests...$(NC)"
	@GOPROXY=https://proxy.golang.org,direct $(BAZEL) test //src/internal/... //src/cmd/... \
		--test_output=errors \
		--cache_test_results=no
	@echo "$(GREEN)✓ Unit tests passed$(NC)"

## test/e2e: run end-to-end tests
.PHONY: test/e2e
test/e2e: test/e2e/prepare
	@echo "$(YELLOW)▶ Running E2E tests...$(NC)"
	@chmod +x src/test/e2e/run_parallel.sh
	@./src/test/e2e/run_parallel.sh
	@echo "$(GREEN)✓ E2E tests completed$(NC)"

## test/e2e/prepare: prepare E2E test environment
.PHONY: test/e2e/prepare
test/e2e/prepare:
	@echo "$(YELLOW)▶ Preparing E2E tests...$(NC)"
	@chmod +x scripts/prepare-e2e.sh
	@./scripts/prepare-e2e.sh

## test/e2e/single: run E2E test for specific distro (usage: make test/e2e/single DISTRO=alpine)
.PHONY: test/e2e/single
test/e2e/single:
	@if [ -z "$(DISTRO)" ]; then \
		echo "$(RED)✗ Please specify DISTRO (alpine, ubuntu, or debian)$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)▶ Running E2E test for $(DISTRO)...$(NC)"
	@chmod +x src/test/e2e/test_distro.sh
	@./src/test/e2e/test_distro.sh $(DISTRO)

## test/quick: quick test without linting or E2E
.PHONY: test/quick
test/quick: test/unit build
	@echo "$(GREEN)✓ Quick check passed$(NC)"

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## quality/analyze: run complete code analysis [QUALITY]
.PHONY: quality/analyze
quality/analyze: quality/format quality/lint quality/security quality/secrets
	@echo "$(GREEN)✓ All quality checks passed!$(NC)"

## quality/format: format all code (Go, YAML, JSON, MD) [QUALITY]
.PHONY: quality/format
quality/format:
	@echo "$(YELLOW)▶ Formatting Go code...$(NC)"
	@gofmt -s -w .
	@if command -v goimports > /dev/null 2>&1; then \
		goimports -w -local github.com/kodflow/cloud-update .; \
	fi
	@echo "$(YELLOW)▶ Formatting YAML/JSON/MD files...$(NC)"
	@if command -v prettier > /dev/null 2>&1; then \
		prettier --write "**/*.{yml,yaml,json,md}" --ignore-path .prettierignore 2>/dev/null || echo "$(YELLOW)Some files skipped due to syntax issues$(NC)"; \
	fi
	@echo "$(GREEN)✓ All code formatted$(NC)"

## quality/lint: run linters [QUALITY]
.PHONY: quality/lint
quality/lint:
	@echo "$(YELLOW)▶ Running linters...$(NC)"
	@$(GOLANGCI_LINT) run --timeout=5m $(SRC_PATH)
	@echo "$(GREEN)✓ Linting passed$(NC)"

## quality/security: run security analysis [QUALITY]
.PHONY: quality/security
quality/security:
	@echo "$(YELLOW)▶ Running security scan...$(NC)"
	@if command -v gosec > /dev/null 2>&1; then \
		gosec -conf .gosec.json $(SRC_PATH) 2>/dev/null || \
		echo "$(YELLOW)⚠ Some security warnings found (non-blocking)$(NC)"; \
	else \
		echo "$(YELLOW)⚠ gosec not installed, skipping security scan$(NC)"; \
	fi
	@echo "$(GREEN)✓ Security scan complete$(NC)"

## quality/secrets: scan for secrets with gitleaks [QUALITY]
.PHONY: quality/secrets
quality/secrets:
	@echo "$(YELLOW)▶ Scanning for secrets...$(NC)"
	@if command -v gitleaks > /dev/null 2>&1; then \
		gitleaks detect --no-banner --exit-code 0 || \
		(echo "$(RED)✗ Secrets detected! Run 'gitleaks detect --verbose' for details$(NC)" && exit 1); \
		echo "$(GREEN)✓ No secrets found$(NC)"; \
	else \
		echo "$(YELLOW)⚠ gitleaks not installed$(NC)"; \
		echo "$(YELLOW)  Install with: brew install gitleaks$(NC)"; \
	fi

## quality/fix: automatically fix all fixable issues [QUALITY]
.PHONY: quality/fix
quality/fix:
	@echo "$(YELLOW)▶ Auto-fixing issues...$(NC)"
	@gofmt -s -w .
	@if command -v goimports > /dev/null 2>&1; then \
		goimports -w -local github.com/kodflow/cloud-update .; \
	fi
	@$(GOLANGCI_LINT) run --fix $(SRC_PATH) 2>/dev/null || true
	@if command -v prettier > /dev/null 2>&1; then \
		prettier --write "**/*.{yml,yaml,json,md}" --ignore-path .prettierignore 2>/dev/null || true; \
	fi
	@$(GO) mod tidy
	@echo "$(GREEN)✓ Auto-fix complete$(NC)"

## quality/validate: validate all quality gates pass [QUALITY]
.PHONY: quality/validate
quality/validate:
	@echo "$(YELLOW)▶ Validating quality gates...$(NC)"
	@./scripts/validate-quality.sh
	@echo "$(GREEN)✓ Quality validation passed$(NC)"

# ==================================================================================== #
# GIT HOOKS
# ==================================================================================== #

## hooks/install: install Git hooks for automatic quality checks
.PHONY: hooks/install
hooks/install:
	@echo "$(YELLOW)▶ Installing Git hooks...$(NC)"
	@chmod +x .githooks/*.sh 2>/dev/null || true
	@chmod +x .githooks/pre-* 2>/dev/null || true
	@./.githooks/install.sh
	@echo "$(GREEN)✓ Git hooks installed$(NC)"

## hooks/uninstall: remove Git hooks configuration
.PHONY: hooks/uninstall
hooks/uninstall:
	@echo "$(YELLOW)▶ Removing Git hooks...$(NC)"
	@chmod +x .githooks/uninstall.sh 2>/dev/null || true
	@./.githooks/uninstall.sh
	@echo "$(GREEN)✓ Git hooks removed$(NC)"

## hooks/status: check Git hooks configuration status
.PHONY: hooks/status
hooks/status:
	@echo "$(YELLOW)▶ Git hooks status:$(NC)"
	@if [ "$$(git config --get core.hooksPath)" = ".githooks" ]; then \
		echo "$(GREEN)  ✓ Hooks are installed and active$(NC)"; \
		echo "$(BLUE)  Path: .githooks$(NC)"; \
		echo "$(BLUE)  Active hooks:$(NC)"; \
		for hook in .githooks/pre-* .githooks/post-* .githooks/commit-msg; do \
			if [ -f "$$hook" ] && [ -x "$$hook" ]; then \
				echo "$(GREEN)    ✓ $$(basename $$hook)$(NC)"; \
			fi; \
		done; \
	else \
		echo "$(YELLOW)  ⚠ Hooks are not installed$(NC)"; \
		echo "$(BLUE)  Run 'make hooks/install' to install$(NC)"; \
	fi

# ==================================================================================== #
# DOCKER
# ==================================================================================== #

## docker/up: start E2E test containers
.PHONY: docker/up
docker/up:
	@echo "$(YELLOW)▶ Starting test containers...$(NC)"
	@$(DOCKER_COMPOSE) -f $(E2E_COMPOSE) up -d
	@echo "$(GREEN)✓ Containers started$(NC)"

## docker/down: stop E2E test containers
.PHONY: docker/down
docker/down:
	@echo "$(YELLOW)▶ Stopping test containers...$(NC)"
	@$(DOCKER_COMPOSE) -f $(E2E_COMPOSE) down --volumes --remove-orphans
	@echo "$(GREEN)✓ Containers stopped$(NC)"

## docker/logs: show container logs
.PHONY: docker/logs
docker/logs:
	@$(DOCKER_COMPOSE) -f $(E2E_COMPOSE) logs -f

## docker/status: show container status
.PHONY: docker/status
docker/status:
	@$(DOCKER_COMPOSE) -f $(E2E_COMPOSE) ps

# ==================================================================================== #
# OPERATIONS
# ==================================================================================== #

## install: install the binary to /usr/local/bin
.PHONY: install
install: confirm build
	@echo "$(YELLOW)▶ Installing $(BINARY_NAME)...$(NC)"
	@sudo cp ./$(BINARY_NAME) /usr/local/bin/
	@echo "$(GREEN)✓ Installed to /usr/local/bin/$(BINARY_NAME)$(NC)"

## uninstall: remove the binary from /usr/local/bin
.PHONY: uninstall
uninstall: confirm
	@echo "$(YELLOW)▶ Uninstalling $(BINARY_NAME)...$(NC)"
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(GREEN)✓ Uninstalled$(NC)"

## clean: clean build artifacts and test containers
.PHONY: clean
clean:
	@echo "$(YELLOW)▶ Cleaning...$(NC)"
	@$(BAZEL) clean
	@rm -f ./$(BINARY_NAME)
	@rm -rf dist/
	@$(DOCKER_COMPOSE) -f $(E2E_COMPOSE) down --volumes --remove-orphans 2>/dev/null || true
	@echo "$(GREEN)✓ Cleaned$(NC)"

## version: show version information
.PHONY: version
version:
	@echo "$(BINARY_NAME) version: $(VERSION)"

# ==================================================================================== #
# CI/CD
# ==================================================================================== #

## ci: run all CI checks (quality, test, build)
.PHONY: ci
ci: quality/analyze test/unit build test/e2e
	@echo "$(GREEN)✓ All CI checks passed!$(NC)"

## ci/validate: validate CI pipeline locally
.PHONY: ci/validate
ci/validate:
	@echo "$(YELLOW)▶ Validating CI pipeline...$(NC)"
	@./scripts/validate-build.sh
	@echo "$(GREEN)✓ CI validation passed$(NC)"

# ==================================================================================== #
# UTILITIES
# ==================================================================================== #

## tools/check: check if required tools are installed
.PHONY: tools/check
tools/check:
	@echo "$(YELLOW)▶ Checking required tools...$(NC)"
	@echo ""
	@echo "Required tools:"
	@command -v go >/dev/null 2>&1 && echo "  $(GREEN)✓$(NC) go $(shell go version | cut -d' ' -f3)" || echo "  $(RED)✗$(NC) go"
	@command -v bazel >/dev/null 2>&1 && echo "  $(GREEN)✓$(NC) bazel $(shell bazel version | grep 'Build label' | cut -d' ' -f3)" || echo "  $(RED)✗$(NC) bazel"
	@echo ""
	@echo "Optional tools:"
	@command -v golangci-lint >/dev/null 2>&1 && echo "  $(GREEN)✓$(NC) golangci-lint $(shell golangci-lint version | head -1 | cut -d' ' -f4)" || echo "  $(YELLOW)○$(NC) golangci-lint"
	@command -v gosec >/dev/null 2>&1 && echo "  $(GREEN)✓$(NC) gosec" || echo "  $(YELLOW)○$(NC) gosec"
	@command -v prettier >/dev/null 2>&1 && echo "  $(GREEN)✓$(NC) prettier $(shell prettier --version)" || echo "  $(YELLOW)○$(NC) prettier"
	@command -v goimports >/dev/null 2>&1 && echo "  $(GREEN)✓$(NC) goimports" || echo "  $(YELLOW)○$(NC) goimports"
	@echo ""

## tools/install: install optional development tools
.PHONY: tools/install
tools/install:
	@echo "$(YELLOW)▶ Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install github.com/securego/gosec/v2/cmd/gosec@latest
	@go install golang.org/x/tools/cmd/goimports@latest
	@echo "$(YELLOW)▶ Installing prettier (requires npm)...$(NC)"
	@npm install -g prettier 2>/dev/null || echo "$(YELLOW)⚠ npm not found, skipping prettier$(NC)"
	@echo "$(GREEN)✓ Tools installed$(NC)"

# ==================================================================================== #
# SHORTCUTS (Legacy compatibility)
# ==================================================================================== #

.PHONY: all
all: test ## Run all tests and build

.PHONY: fmt
fmt: quality/format ## Format code (alias for quality/format)

.PHONY: lint
lint: quality/lint ## Run linters (alias for quality/lint)

.PHONY: test-unit
test-unit: test/unit ## Run unit tests (alias)

.PHONY: test-e2e
test-e2e: test/e2e ## Run E2E tests (alias)

.PHONY: analyze
analyze: quality/analyze ## Run analysis (alias)

.PHONY: fix
fix: quality/fix ## Auto-fix issues (alias)

.PHONY: security
security: quality/security ## Run security scan (alias)

.PHONY: quality
quality: quality/analyze ## Run quality checks (alias)

.PHONY: quick
quick: test/quick ## Quick test (alias)

.PHONY: check-tools
check-tools: tools/check ## Check tools (alias)