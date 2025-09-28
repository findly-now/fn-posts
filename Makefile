.PHONY: build run clean docker-up docker-down deploy-schema e2e-test e2e-up e2e-down e2e-logs fmt lint tidy install-deps dev help e2e-test-only e2e-test-coverage

# Build the application
build:
	go build -o bin/posts-service cmd/main.go

# Run the application
run:
	go run cmd/main.go

# Note: This service uses E2E tests only - no unit tests

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Start local development environment
docker-up:
	docker-compose up -d

# Stop local development environment
docker-down:
	docker-compose down

# Deploy database schema using script.sql
deploy-schema:
	@echo "Deploying database schema..."
	psql "$(DATABASE_URL)" -f script.sql
	@echo "✅ Database schema deployed"

# Format code
fmt:
	go fmt ./...

# Run linter
lint:
	golangci-lint run

# Tidy dependencies
tidy:
	go mod tidy

# Install development dependencies
install-deps:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run the service with live reload (requires air)
dev:
	air

# =============================================================================
# E2E TESTING COMMANDS
# =============================================================================

# Run E2E tests (complete flow: up -> test -> down)
e2e-test:
	@echo "🚀 Starting E2E test environment..."
	docker-compose -f tests/e2e/docker-compose.e2e.yml up -d --build
	@echo "⏳ Waiting for services to be ready..."
	sleep 30
	@echo "🧪 Running E2E tests..."
	go test -v ./tests/e2e/... || (echo "❌ E2E tests failed" && docker-compose -f tests/e2e/docker-compose.e2e.yml down && exit 1)
	@echo "✅ E2E tests passed"
	@echo "🧹 Cleaning up test environment..."
	docker-compose -f tests/e2e/docker-compose.e2e.yml down
	@echo "✅ E2E test complete"

# Start E2E test environment (for development/debugging)
e2e-up:
	@echo "🚀 Starting E2E test environment..."
	docker-compose -f tests/e2e/docker-compose.e2e.yml up -d --build
	@echo "⏳ Waiting for services to be ready..."
	sleep 30
	@echo "✅ E2E environment ready"
	@echo "📝 Posts API: http://localhost:8081"
	@echo "📝 MinIO Console: http://localhost:9002"
	@echo "📝 Health Check: curl http://localhost:8081/health"

# Stop E2E test environment
e2e-down:
	@echo "🧹 Stopping E2E test environment..."
	docker-compose -f tests/e2e/docker-compose.e2e.yml down
	@echo "✅ E2E environment stopped"

# View E2E environment logs
e2e-logs:
	docker-compose -f tests/e2e/docker-compose.e2e.yml logs -f

# Run E2E tests only (environment must be running)
e2e-test-only:
	@echo "🧪 Running E2E tests (assuming environment is already running)..."
	go test -v ./tests/e2e/...

# Run E2E tests with coverage
e2e-test-coverage:
	@echo "🚀 Starting E2E test environment..."
	docker-compose -f tests/e2e/docker-compose.e2e.yml up -d --build
	@echo "⏳ Waiting for services to be ready..."
	sleep 30
	@echo "🧪 Running E2E tests with coverage..."
	go test -v -coverprofile=e2e-coverage.out ./tests/e2e/... || (echo "❌ E2E tests failed" && docker-compose -f tests/e2e/docker-compose.e2e.yml down && exit 1)
	go tool cover -html=e2e-coverage.out -o e2e-coverage.html
	@echo "✅ E2E tests passed, coverage report generated: e2e-coverage.html"
	@echo "🧹 Cleaning up test environment..."
	docker-compose -f tests/e2e/docker-compose.e2e.yml down

# Show help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Development:"
	@echo "  build          Build the application"
	@echo "  run            Run the application"
	@echo "  dev            Run with live reload"
	@echo "  docker-up      Start local development environment"
	@echo "  docker-down    Stop local development environment"
	@echo ""
	@echo "Testing:"
	@echo "  e2e-test       Run complete E2E test suite (recommended)"
	@echo "  e2e-up         Start E2E test environment"
	@echo "  e2e-down       Stop E2E test environment"
	@echo "  e2e-test-only  Run E2E tests (environment must be running)"
	@echo "  e2e-logs       View E2E environment logs"
	@echo ""
	@echo "Database:"
	@echo "  deploy-schema  Deploy database schema using script.sql"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt            Format code"
	@echo "  lint           Run linter"
	@echo "  tidy           Tidy dependencies"
	@echo ""
	@echo "Utility:"
	@echo "  clean          Clean build artifacts"
	@echo "  install-deps   Install development dependencies"
	@echo "  help           Show this help message"