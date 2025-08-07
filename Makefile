.PHONY: all build test lint e2e clean help

# Variables
BINARY_NAME := cloud-update
BAZEL := bazel
GO := go
GOLANGCI_LINT := golangci-lint
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Colors
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[1;33m
NC := \033[0m # No Color

## help: Display this help message
help:
	@echo "Available targets:"
	@grep -E '^##' Makefile | sed 's/## //'

## all: Build and test everything
all: lint test build

## build: Build the binary
build:
	@echo "$(YELLOW)🔨 Building $(BINARY_NAME)...$(NC)"
	@$(BAZEL) build //src/cmd/cloud-update:cloud-update \
		--stamp \
		--workspace_status_command="echo BUILD_VERSION $(VERSION)"
	@cp bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update ./$(BINARY_NAME)
	@chmod +x ./$(BINARY_NAME)
	@echo "$(GREEN)✅ Binary built: ./$(BINARY_NAME)$(NC)"

## test: Run unit tests
test:
	@echo "$(YELLOW)🧪 Running unit tests...$(NC)"
	@$(BAZEL) test //src/internal/... //src/cmd/... --test_output=errors --cache_test_results=no
	@echo "$(GREEN)✅ Unit tests passed$(NC)"

## lint: Run golangci-lint
lint:
	@echo "$(YELLOW)🔍 Running linter...$(NC)"
	@$(GOLANGCI_LINT) run --timeout=5m ./src/...
	@echo "$(GREEN)✅ Linting passed$(NC)"

## e2e: Run E2E tests in parallel
e2e:
	@echo "$(YELLOW)🚀 Running E2E tests...$(NC)"
	@chmod +x src/test/e2e/run_parallel.sh
	@./src/test/e2e/run_parallel.sh

## e2e-single: Run E2E test for a specific distro (usage: make e2e-single DISTRO=alpine)
e2e-single:
	@if [ -z "$(DISTRO)" ]; then \
		echo "$(RED)❌ Please specify DISTRO (alpine, ubuntu, or debian)$(NC)"; \
		exit 1; \
	fi
	@echo "$(YELLOW)🧪 Running E2E test for $(DISTRO)...$(NC)"
	@chmod +x src/test/e2e/test_distro.sh
	@./src/test/e2e/test_distro.sh $(DISTRO)

## run: Run the binary locally
run: build
	@echo "$(YELLOW)🚀 Starting $(BINARY_NAME)...$(NC)"
	@./$(BINARY_NAME)

## install: Install the binary to /usr/local/bin
install: build
	@echo "$(YELLOW)📦 Installing $(BINARY_NAME)...$(NC)"
	@sudo cp ./$(BINARY_NAME) /usr/local/bin/
	@echo "$(GREEN)✅ Installed to /usr/local/bin/$(BINARY_NAME)$(NC)"

## clean: Clean build artifacts
clean:
	@echo "$(YELLOW)🧹 Cleaning...$(NC)"
	@$(BAZEL) clean
	@rm -f ./$(BINARY_NAME)
	@docker compose -f src/test/e2e/docker-compose.yml down --volumes --remove-orphans 2>/dev/null || true
	@echo "$(GREEN)✅ Cleaned$(NC)"

## deps: Download and verify dependencies
deps:
	@echo "$(YELLOW)📦 Downloading dependencies...$(NC)"
	@$(GO) mod download
	@$(GO) mod verify
	@$(BAZEL) run //:gazelle
	@echo "$(GREEN)✅ Dependencies ready$(NC)"

## fmt: Format Go code
fmt:
	@echo "$(YELLOW)✨ Formatting code...$(NC)"
	@$(GO) fmt ./...
	@echo "$(GREEN)✅ Code formatted$(NC)"

## ci: Run the same checks as CI (lint, test, build, e2e)
ci: lint test build e2e
	@echo "$(GREEN)✅ All CI checks passed!$(NC)"

## quick: Quick build and test (no linting or E2E)
quick: test build
	@echo "$(GREEN)✅ Quick check passed$(NC)"

## docker-up: Start E2E test containers
docker-up:
	@echo "$(YELLOW)🐳 Starting test containers...$(NC)"
	@docker compose -f src/test/e2e/docker-compose.yml up -d
	@echo "$(GREEN)✅ Containers started$(NC)"

## docker-down: Stop E2E test containers
docker-down:
	@echo "$(YELLOW)🐳 Stopping test containers...$(NC)"
	@docker compose -f src/test/e2e/docker-compose.yml down --volumes --remove-orphans
	@echo "$(GREEN)✅ Containers stopped$(NC)"

## version: Show version information
version:
	@echo "$(BINARY_NAME) version: $(VERSION)"