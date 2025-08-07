.PHONY: test test-unit test-e2e build clean gazelle

# Variables
BINARY_NAME=cloud-update

# Test command - runs both unit and e2e tests
test: test-unit test-e2e

# Run unit tests only with Bazel
test-unit:
	@echo "Running unit tests with Bazel..."
	@bazel test //src/internal/... //src/cmd/... --test_output=errors

# Run E2E tests (still using Docker for now)
test-e2e:
	@echo "Running E2E tests..."
	@echo "Building Linux binary with Bazel..."
	@bazel build //src/cmd/cloud-update:cloud-update --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64
	@mkdir -p dist/cloud-update_linux_amd64_v1
	@cp bazel-bin/src/cmd/cloud-update/cloud-update_/cloud-update dist/cloud-update_linux_amd64_v1/
	@echo "Starting E2E test environment..."
	@docker compose -f src/test/e2e/docker-compose.yml down 2>/dev/null || true
	@docker compose -f src/test/e2e/docker-compose.yml up -d --build alpine
	@echo "Waiting for services to start..."
	@sleep 10
	@echo "Running E2E tests..."
	@E2E_BASE_URL=http://localhost:9991 E2E_SECRET=test-secret-key-for-e2e go test -v ./src/test/e2e/...
	@echo "Cleaning up E2E environment..."
	@docker compose -f src/test/e2e/docker-compose.yml down

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