.PHONY: build run test clean docker-up docker-down migrate-up migrate-down e2e-test e2e-up e2e-down e2e-logs

# Build the application
build:
	go build -o bin/posts-service cmd/main.go

# Run the application
run:
	go run cmd/main.go

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Start local development environment
docker-up:
	docker-compose up -d

# Stop local development environment
docker-down:
	docker-compose down

# Run database migrations up
migrate-up:
	@echo "Running database migrations..."
	@for file in migrations/*.up.sql; do \
		echo "Applying $$file"; \
		psql $(DATABASE_URL) -f $$file; \
	done

# Run database migrations down
migrate-down:
	@echo "Rolling back database migrations..."
	@for file in $$(ls migrations/*.down.sql | sort -r); do \
		echo "Rolling back $$file"; \
		psql $(DATABASE_URL) -f $$file; \
	done

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
	@echo "  test           Run unit tests (deprecated - use E2E tests)"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  e2e-test       Run complete E2E test suite (recommended)"
	@echo "  e2e-up         Start E2E test environment"
	@echo "  e2e-down       Stop E2E test environment"
	@echo "  e2e-test-only  Run E2E tests (environment must be running)"
	@echo "  e2e-logs       View E2E environment logs"
	@echo ""
	@echo "Database:"
	@echo "  migrate-up     Run database migrations"
	@echo "  migrate-down   Rollback database migrations"
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