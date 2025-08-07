.PHONY: test test-unit test-e2e build clean gazelle

# Variables
BINARY_NAME=cloud-update

# Test command - runs both unit and e2e tests
test: test-unit test-e2e

# Run unit tests only with Bazel
test-unit:
	@echo "Running unit tests with Bazel..."
	@bazel test //src/internal/... //src/cmd/... --test_output=errors

# Run init system tests (the new focused E2E tests)
test-init:
	@echo "Running init system tests..."
	@chmod +x scripts/test_init_systems.sh
	@./scripts/test_init_systems.sh

# Run all E2E tests (legacy - for all distributions)
test-e2e-all: test-e2e-alpine test-e2e-ubuntu test-e2e-debian test-e2e-rockylinux test-e2e-fedora test-e2e-arch test-e2e-opensuse
	@echo "✅ All distribution tests completed!"

# Run E2E tests (now points to init system tests)
test-e2e: test-init

# Run E2E test for Alpine
test-e2e-alpine:
	@echo "Running E2E test for Alpine..."
	@chmod +x scripts/test_distro.sh
	@./scripts/test_distro.sh alpine

# Run E2E test for Ubuntu
test-e2e-ubuntu:
	@echo "Running E2E test for Ubuntu..."
	@chmod +x scripts/test_distro.sh
	@./scripts/test_distro.sh ubuntu

# Run E2E test for Debian
test-e2e-debian:
	@echo "Running E2E test for Debian..."
	@chmod +x scripts/test_distro.sh
	@./scripts/test_distro.sh debian

# Run E2E test for Rocky Linux
test-e2e-rockylinux:
	@echo "Running E2E test for Rocky Linux..."
	@chmod +x scripts/test_distro.sh
	@./scripts/test_distro.sh rockylinux

# Run E2E test for Fedora
test-e2e-fedora:
	@echo "Running E2E test for Fedora..."
	@chmod +x scripts/test_distro.sh
	@./scripts/test_distro.sh fedora

# Run E2E test for Arch Linux
test-e2e-arch:
	@echo "Running E2E test for Arch Linux..."
	@chmod +x scripts/test_distro.sh
	@./scripts/test_distro.sh arch

# Run E2E test for openSUSE
test-e2e-opensuse:
	@echo "Running E2E test for openSUSE..."
	@chmod +x scripts/test_distro.sh
	@./scripts/test_distro.sh opensuse

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