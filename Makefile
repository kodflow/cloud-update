.PHONY: help test test-unit test-e2e build clean gazelle

# Variables
BINARY_NAME=cloud-update

# Default target - show help
help:
	@echo "Cloud Update - Available Commands"
	@echo "================================="
	@echo ""
	@echo "Testing:"
	@echo "  make test              - Run all tests (unit + e2e)"
	@echo "  make test-unit         - Run unit tests only"
	@echo "  make test-e2e          - Run all E2E tests (OpenRC, Systemd, SysVInit)"
	@echo "  make test-e2e-alpine   - Test OpenRC (Alpine)"
	@echo "  make test-e2e-ubuntu   - Test Systemd (Ubuntu)"
	@echo "  make test-e2e-debian   - Test SysVInit (Debian)"
	@echo "  make test-e2e-clean    - Clean up E2E test environment"
	@echo ""
	@echo "Building:"
	@echo "  make build             - Build binary for current platform"
	@echo "  make gazelle           - Update BUILD files with Gazelle"
	@echo ""
	@echo "Cleaning:"
	@echo "  make clean             - Clean all build artifacts and containers"
	@echo ""
	@echo "CI/CD:"
	@echo "  make gha               - Simulate GitHub Actions workflow locally"
	@echo ""
	@echo "Help:"
	@echo "  make help              - Show this help message"
	@echo ""

# Test command - runs both unit and e2e tests
test: test-unit test-e2e

# Run unit tests only with Bazel
test-unit:
	@echo "Running unit tests with Bazel..."
	@bazel test //src/internal/... //src/cmd/... --test_output=errors

# Run E2E test for OpenRC (Alpine)
test-e2e-alpine:
	@echo "Running E2E test for OpenRC (Alpine)..."
	@chmod +x src/test/e2e/test_distro.sh
	@./src/test/e2e/test_distro.sh alpine

# Run E2E test for Systemd (Ubuntu)
test-e2e-ubuntu:
	@echo "Running E2E test for Systemd (Ubuntu)..."
	@chmod +x src/test/e2e/test_distro.sh
	@./src/test/e2e/test_distro.sh ubuntu

# Run E2E test for SysVInit (Debian)
test-e2e-debian:
	@echo "Running E2E test for SysVInit (Debian)..."
	@chmod +x src/test/e2e/test_distro.sh
	@./src/test/e2e/test_distro.sh debian

# Run all E2E tests (all init systems)
test-e2e: test-e2e-alpine test-e2e-ubuntu test-e2e-debian
	@echo "✅ All E2E tests completed!"

# Clean up E2E test environment
test-e2e-clean:
	@echo "Cleaning up E2E test environment..."
	@docker compose -f src/test/e2e/docker-compose.yml down --volumes --remove-orphans

# Build binary with Bazel for current platform
build:
	@echo "Building $(BINARY_NAME) with Bazel..."
	@bazel build //src/cmd/cloud-update:cloud-update
	@echo "Binary available at: bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update"

# Clean build artifacts
clean:
	@echo "Cleaning all build artifacts and generated files..."
	@echo "Stopping Docker containers..."
	@docker compose -f src/test/e2e/docker-compose.yml down 2>/dev/null || true
	@echo "Cleaning Bazel artifacts..."
	@bazel clean --expunge 2>/dev/null || true
	@echo "Removing build directories..."
	@rm -rf dist/ build/ release/
	@echo "Removing Bazel symlinks and cache..."
	@rm -rf bazel-* .bazel/
	@echo "Removing Go test cache..."
	@go clean -testcache 2>/dev/null || true
	@echo "Removing temporary files..."
	@find . -type f -name "*.tmp" -o -name "*.temp" -o -name "*.bak" -o -name "*.log" -o -name "*.out" | xargs rm -f 2>/dev/null || true
	@echo "Removing OS specific files..."
	@find . -type f -name ".DS_Store" -o -name "Thumbs.db" -o -name "desktop.ini" | xargs rm -f 2>/dev/null || true
	@echo "Removing IDE files..."
	@rm -rf .idea/ .vscode/ *.iml
	@echo "Removing coverage files..."
	@rm -f coverage.txt coverage.html *.test
	@echo "Removing lock files if they exist..."
	@rm -f MODULE.bazel.lock
	@echo "✅ Clean completed!"

# Update BUILD files with Gazelle
gazelle:
	@echo "Updating BUILD files..."
	@bazel run //:gazelle
	@bazel run //:gazelle -- update-repos -from_file=go.mod

# Simulate GitHub Actions workflow locally
gha:
	@echo "============================================"
	@echo "Simulating GitHub Actions CI workflow"
	@echo "============================================"
	@echo ""
	@echo "=== Job: test ==="
	@echo "Step 1/3: Checkout code ✓"
	@echo "Step 2/3: Setup Bazel ✓"
	@echo "Step 3/3: Run unit tests with Bazel..."
	@bazel test //src/internal/... //src/cmd/... --test_output=errors
	@echo ""
	@echo "✅ Job: test completed successfully"
	@echo ""
	@echo "=== Job: build (depends on test) ==="
	@echo "Step 1/3: Checkout code ✓"
	@echo "Step 2/3: Setup Bazel ✓"
	@echo "Step 3/3: Build with Bazel..."
	@bazel build //src/cmd/cloud-update:cloud-update
	@echo "Step 4/4: Test binary..."
	@bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update --version
	@bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update --help
	@echo ""
	@echo "============================================"
	@echo "✅ GitHub Actions simulation completed!"
	@echo "============================================"