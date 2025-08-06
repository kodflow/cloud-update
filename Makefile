.PHONY: build install clean test run lint coverage bench release-dry release

# Variables
BINARY_NAME=cloud-update
BUILD_DIR=build
INSTALL_DIR=/opt/cloud-update
SERVICE_FILE=src/init/systemd/cloud-update.service
CONFIG_DIR=/etc/cloud-update
VERSION=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git rev-parse --short HEAD)
DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-s -w -X cloud-update/src/internal/version.Version=$(VERSION) -X cloud-update/src/internal/version.Commit=$(COMMIT) -X cloud-update/src/internal/version.Date=$(DATE)"

# Build the binary
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME) src/cmd/cloud-update/main.go

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 src/cmd/cloud-update/main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 src/cmd/cloud-update/main.go
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 src/cmd/cloud-update/main.go
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 src/cmd/cloud-update/main.go

# Install the service
install: build
	@echo "Installing $(BINARY_NAME)..."
	sudo mkdir -p $(INSTALL_DIR)
	sudo mkdir -p $(CONFIG_DIR)
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/
	sudo cp $(SERVICE_FILE) /etc/systemd/system/
	@echo "Installation complete!"
	@echo "Don't forget to:"
	@echo "1. Edit $(CONFIG_DIR)/config.env with your configuration"
	@echo "2. Run 'sudo systemctl daemon-reload'"
	@echo "3. Run 'sudo systemctl enable cloud-update'"
	@echo "4. Run 'sudo systemctl start cloud-update'"

# Uninstall the service
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	sudo systemctl stop cloud-update || true
	sudo systemctl disable cloud-update || true
	sudo rm -f /etc/systemd/system/cloud-update.service
	sudo rm -rf $(INSTALL_DIR)
	sudo systemctl daemon-reload

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf dist/
	go clean -testcache

# Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not installed. Install it with:"; \
		echo "curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin"; \
	fi

# Run locally for development
run:
	@echo "Running $(BINARY_NAME) locally..."
	@echo "Make sure to set environment variables:"
	@echo "export CLOUD_UPDATE_SECRET=your-secret-key"
	@echo "export CLOUD_UPDATE_PORT=9999"
	go run $(LDFLAGS) src/cmd/cloud-update/main.go

# Generate a secret key
generate-secret:
	@echo "Generated secret key:"
	@openssl rand -hex 32

# Check version
version:
	@go run $(LDFLAGS) src/cmd/cloud-update/main.go -version

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	go mod tidy

# Security scan
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install it with:"; \
		echo "go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
	fi

# Dry run release with goreleaser
release-dry:
	@echo "Running goreleaser in dry-run mode..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --clean --skip=publish; \
	else \
		echo "goreleaser not installed. Install it from https://goreleaser.com/install/"; \
	fi

# Create a release with goreleaser
release:
	@echo "Creating release with goreleaser..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --clean; \
	else \
		echo "goreleaser not installed. Install it from https://goreleaser.com/install/"; \
	fi

# Run E2E tests
e2e:
	@echo "Running E2E tests with Docker..."
	@if [ -f src/test/e2e/run-e2e-tests.sh ]; then \
		./src/test/e2e/run-e2e-tests.sh; \
	else \
		echo "E2E test script not found"; \
		exit 1; \
	fi

# Run E2E tests for a specific distribution
e2e-distro:
	@if [ -z "$(DISTRO)" ]; then \
		echo "Usage: make e2e-distro DISTRO=alpine"; \
		echo "Available: alpine, ubuntu, debian, rockylinux"; \
		exit 1; \
	fi
	@echo "Running E2E tests for $(DISTRO)..."
	docker build -f src/test/e2e/Dockerfile.$(DISTRO) -t cloud-update-$(DISTRO):test .
	docker run -d --name cloud-update-test -p 9999:9999 \
		-e CLOUD_UPDATE_PORT=9999 \
		-e CLOUD_UPDATE_SECRET=test-secret-key-for-e2e \
		cloud-update-$(DISTRO):test
	@sleep 5
	E2E_BASE_URL=http://localhost:9999 E2E_SECRET=test-secret-key-for-e2e go test -v ./src/test/e2e/...
	@docker stop cloud-update-test && docker rm cloud-update-test

# CI/CD checks (runs all quality checks)
ci: tidy fmt lint test coverage security
	@echo "All CI checks passed!"

# Full CI with E2E tests
ci-full: ci e2e
	@echo "All CI checks including E2E tests passed!"

# Complete test suite - runs everything like GitHub Actions
tests:
	@echo "==============================================="
	@echo "🚀 Running Complete Test Suite"
	@echo "==============================================="
	@echo ""
	@echo "📋 Phase 1: Code Quality Checks"
	@echo "-----------------------------------------------"
	@echo "→ Checking go.mod..."
	@go mod tidy -v && git diff --exit-code go.mod go.sum || (echo "❌ go.mod/go.sum needs updating. Run 'go mod tidy'" && exit 1)
	@echo "✅ go.mod is clean"
	@echo ""
	@echo "→ Checking code formatting..."
	@gofmt -l src/ > /tmp/gofmt.txt && test ! -s /tmp/gofmt.txt || (echo "❌ Code needs formatting. Files:" && cat /tmp/gofmt.txt && exit 1)
	@echo "✅ Code formatting is correct"
	@echo ""
	@echo "→ Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m ./src/... || exit 1; \
	else \
		echo "⚠️  golangci-lint not installed. Install with:"; \
		echo "curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin"; \
		exit 1; \
	fi
	@echo "✅ Linting passed"
	@echo ""
	@echo "→ Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -quiet ./src/... || exit 1; \
	else \
		echo "⚠️  gosec not installed. Install with:"; \
		echo "go install github.com/securego/gosec/v2/cmd/gosec@latest"; \
		exit 1; \
	fi
	@echo "✅ Security scan passed"
	@echo ""
	@echo "📋 Phase 2: Unit Tests"
	@echo "-----------------------------------------------"
	@echo "→ Running unit tests with coverage..."
	@go test -v -race -coverprofile=coverage.txt -covermode=atomic ./src/... || exit 1
	@echo ""
	@echo "→ Coverage report:"
	@go tool cover -func=coverage.txt | tail -1
	@echo "✅ Unit tests passed"
	@echo ""
	@echo "📋 Phase 3: Build Validation"
	@echo "-----------------------------------------------"
	@echo "→ Building binary..."
	@make build > /dev/null 2>&1 || exit 1
	@echo "✅ Build successful"
	@echo ""
	@echo "→ Testing binary..."
	@./build/cloud-update --version > /dev/null || exit 1
	@echo "✅ Binary works"
	@echo ""
	@echo "📋 Phase 4: E2E Tests (Optional)"
	@echo "-----------------------------------------------"
	@echo "→ To run E2E tests, use: make tests-e2e"
	@echo ""
	@echo "==============================================="
	@echo "✅ All tests passed successfully!"
	@echo "==============================================="

# E2E tests only
tests-e2e:
	@echo "🐳 Running E2E tests with Docker..."
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "❌ Docker is not installed"; \
		exit 1; \
	fi
	@echo "→ Testing Alpine Linux..."
	@make e2e-distro DISTRO=alpine || exit 1
	@echo "→ Testing Ubuntu..."
	@make e2e-distro DISTRO=ubuntu || exit 1
	@echo "→ Testing Rocky Linux..."
	@make e2e-distro DISTRO=rockylinux || exit 1
	@echo "✅ All E2E tests passed!"

# Quick tests (no E2E)
tests-quick:
	@echo "⚡ Running quick tests (no E2E)..."
	@make tests

# Help
help:
	@echo "Available targets:"
	@echo "  build        - Build the binary"
	@echo "  build-all    - Build for multiple platforms"
	@echo "  install      - Install the service"
	@echo "  uninstall    - Uninstall the service"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  coverage     - Run tests with coverage"
	@echo "  bench        - Run benchmarks"
	@echo "  lint         - Run linter"
	@echo "  run          - Run locally for development"
	@echo "  generate-secret - Generate a secret key"
	@echo "  version      - Check version"
	@echo "  fmt          - Format code"
	@echo "  tidy         - Tidy dependencies"
	@echo "  security     - Run security scan"
	@echo "  e2e          - Run E2E tests with Docker"
	@echo "  e2e-distro   - Run E2E tests for specific distro (DISTRO=alpine)"
	@echo "  release-dry  - Dry run release with goreleaser"
	@echo "  release      - Create a release with goreleaser"
	@echo "  ci           - Run all CI checks"
	@echo "  ci-full      - Run all CI checks including E2E tests"
	@echo "  tests        - Run complete test suite (like GitHub Actions)"
	@echo "  tests-e2e    - Run E2E tests only"
	@echo "  tests-quick  - Run quick tests without E2E"
	@echo "  help         - Show this help message"