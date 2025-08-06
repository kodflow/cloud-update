.PHONY: test test-unit test-e2e build clean

# Variables
BINARY_NAME=cloud-update

# Test command - runs both unit and e2e tests
test: test-unit test-e2e

# Run unit tests only
test-unit:
	@echo "Running unit tests..."
	@CLOUD_UPDATE_SECRET=test go test -v -race $$(go list ./src/... | grep -v /e2e)

# Run E2E tests
test-e2e:
	@echo "Running E2E tests..."
	@echo "Building Linux binary for E2E tests..."
	@GOOS=linux GOARCH=amd64 go install github.com/goreleaser/goreleaser/v2@latest 2>/dev/null || true
	@GOOS=linux GOARCH=amd64 GITHUB_REPOSITORY_OWNER=local goreleaser build --snapshot --clean --single-target
	@echo "Starting E2E test environment..."
	@docker compose -f src/test/e2e/docker-compose.yml down 2>/dev/null || true
	@docker compose -f src/test/e2e/docker-compose.yml up -d --build alpine
	@echo "Waiting for services to start..."
	@sleep 10
	@echo "Running E2E tests..."
	@E2E_BASE_URL=http://localhost:9991 E2E_SECRET=test-secret-key-for-e2e go test -v ./src/test/e2e/...
	@echo "Cleaning up E2E environment..."
	@docker compose -f src/test/e2e/docker-compose.yml down

# Build binary for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@go install github.com/goreleaser/goreleaser/v2@latest 2>/dev/null || true
	@GITHUB_REPOSITORY_OWNER=local goreleaser build --snapshot --clean --single-target

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf dist/
	@cd src/test/e2e && docker compose down 2>/dev/null || true
	@go clean -testcache

# Simulate GitHub Actions workflow locally
gha:
	@echo "============================================"
	@echo "Simulating GitHub Actions CI workflow"
	@echo "============================================"
	@echo ""
	@echo "=== Job: test ==="
	@echo "Step 1/4: Checkout code ✓"
	@echo "Step 2/4: Set up Go 1.24 ✓"
	@echo "Step 3/4: Install dependencies..."
	@go install github.com/goreleaser/goreleaser/v2@latest 2>/dev/null || true
	@go mod download
	@go mod verify
	@echo "Step 4/5: Run unit tests..."
	@CLOUD_UPDATE_SECRET=test go test -v -race $$(go list ./src/... | grep -v /e2e)
	@echo ""
	@echo "Step 5/5: Run E2E tests..."
	@echo "Building Linux binary for E2E tests..."
	@GOOS=linux GOARCH=amd64 GITHUB_REPOSITORY_OWNER=local goreleaser build --snapshot --clean --single-target
	@echo "Starting E2E test environment..."
	@docker compose -f src/test/e2e/docker-compose.yml down 2>/dev/null || true
	@docker compose -f src/test/e2e/docker-compose.yml up -d --build alpine
	@echo "Waiting for services to start..."
	@sleep 10
	@echo "Running E2E tests..."
	@E2E_BASE_URL=http://localhost:9991 E2E_SECRET=test-secret-key-for-e2e go test -v ./src/test/e2e/...
	@echo "Cleaning up E2E environment..."
	@docker compose -f src/test/e2e/docker-compose.yml down
	@echo ""
	@echo "✅ Job: test completed successfully"
	@echo ""
	@echo "=== Job: build (depends on test) ==="
	@echo "Step 1/5: Checkout code ✓"
	@echo "Step 2/5: Set up Go 1.24 ✓"
	@echo "Step 3/5: Install dependencies..."
	@go install github.com/goreleaser/goreleaser/v2@latest 2>/dev/null || true
	@go mod download
	@go mod verify
	@echo "Step 4/5: Build..."
	@export PATH=$$PATH:$$(go env GOPATH)/bin && \
		GITHUB_REPOSITORY_OWNER=local goreleaser build --snapshot --clean --single-target
	@echo "Step 5/5: Test binary..."
	@ls -la dist/
	@./dist/cloud-update_*/cloud-update --version
	@./dist/cloud-update_*/cloud-update --help
	@echo ""
	@echo "============================================"
	@echo "✅ GitHub Actions simulation completed!"
	@echo "============================================"