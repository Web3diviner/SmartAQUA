# Smart Fish Feeder Backend - Production Grade Makefile

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint
BINARY_NAME=smart-fish-feeder
BINARY_UNIX=$(BINARY_NAME)_unix
MAIN_PATH=./cmd/server

# Test parameters
TEST_TIMEOUT=30s
COVERAGE_OUT=coverage.out
COVERAGE_HTML=coverage.html

# Quality gates
MIN_COVERAGE=40
MAX_CYCLOMATIC_COMPLEXITY=10

.PHONY: all build clean test coverage lint fmt vet security benchmark \
        test-unit test-integration test-algorithms test-property \
        pre-commit install-hooks quality-gate help

# Default target
all: clean fmt lint vet test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) -o $(BINARY_NAME) -v $(MAIN_PATH)

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v $(MAIN_PATH)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)
	rm -f $(COVERAGE_OUT)
	rm -f $(COVERAGE_HTML)

# Run all tests
test: test-unit test-algorithms test-integration

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./internal/services/...

# Run algorithm tests
test-algorithms:
	@echo "Running algorithm tests..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) ./internal/algorithms/...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -tags=integration ./...

# Run property-based tests
test-property:
	@echo "Running property-based tests..."
	$(GOTEST) -v -timeout $(TEST_TIMEOUT) -tags=property ./internal/services/...

# Generate test coverage
coverage:
	@echo "Generating test coverage..."
	$(GOTEST) -coverprofile=$(COVERAGE_OUT) -covermode=atomic ./...
	$(GOCMD) tool cover -html=$(COVERAGE_OUT) -o $(COVERAGE_HTML)
	$(GOCMD) tool cover -func=$(COVERAGE_OUT)

# Check coverage meets minimum threshold
coverage-check: coverage
	@echo "Checking coverage threshold..."
	@COVERAGE=$$($(GOCMD) tool cover -func=$(COVERAGE_OUT) | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < $(MIN_COVERAGE)" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below minimum $(MIN_COVERAGE)%"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets minimum $(MIN_COVERAGE)%"; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Check formatting
fmt-check:
	@echo "Checking code formatting..."
	@UNFORMATTED=$$($(GOFMT) -l .); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "❌ The following files are not formatted:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	else \
		echo "✅ All files are properly formatted"; \
	fi

# Run linter
lint:
	@echo "Running linter..."
	$(GOLINT) run --timeout=5m

# Run vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

# Security scan
security:
	@echo "Running security scan..."
	gosec -quiet ./...

# Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./internal/algorithms/...

# Performance profiling
profile-cpu:
	@echo "Running CPU profiling..."
	$(GOTEST) -cpuprofile=cpu.prof -bench=. ./internal/algorithms/...
	$(GOCMD) tool pprof cpu.prof

profile-mem:
	@echo "Running memory profiling..."
	$(GOTEST) -memprofile=mem.prof -bench=. ./internal/algorithms/...
	$(GOCMD) tool pprof mem.prof

# Dependency management
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

deps-update:
	@echo "Updating dependencies..."
	$(GOMOD) tidy

deps-verify:
	@echo "Verifying dependencies..."
	$(GOMOD) verify

# Install development tools
install-tools:
	@echo "Installing development tools..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	$(GOGET) github.com/fzipp/gocyclo/cmd/gocyclo@latest

# Install git hooks
install-hooks:
	@echo "Installing git hooks..."
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
	cp scripts/pre-push .git/hooks/pre-push
	chmod +x .git/hooks/pre-push

# Pre-commit checks (run before every commit)
pre-commit: fmt-check vet lint test-unit test-algorithms coverage-check security
	@echo "✅ All pre-commit checks passed!"

# Quality gate (comprehensive checks)
quality-gate: clean deps-verify fmt-check vet lint security test coverage-check benchmark
	@echo "✅ All quality gates passed!"

# Cyclomatic complexity check
complexity:
	@echo "Checking cyclomatic complexity..."
	@COMPLEX=$$(gocyclo -over $(MAX_CYCLOMATIC_COMPLEXITY) .); \
	if [ -n "$$COMPLEX" ]; then \
		echo "❌ Functions with high cyclomatic complexity:"; \
		echo "$$COMPLEX"; \
		exit 1; \
	else \
		echo "✅ All functions have acceptable complexity"; \
	fi

# Docker targets
docker-build:
	@echo "Building Docker image..."
	docker build -t smart-fish-feeder:latest .

docker-test:
	@echo "Running tests in Docker..."
	docker run --rm smart-fish-feeder:latest make test

# Database targets
db-migrate:
	@echo "Running database migrations..."
	$(GOCMD) run cmd/migrate/main.go

db-seed:
	@echo "Seeding database..."
	$(GOCMD) run cmd/seed/main.go

# Development server
dev:
	@echo "Starting development server..."
	$(GOCMD) run $(MAIN_PATH)

# Production build
prod-build: quality-gate
	@echo "Building production binary..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) \
		-ldflags="-w -s" \
		-o $(BINARY_NAME) \
		$(MAIN_PATH)

# Help target
help:
	@echo "Smart Fish Feeder Backend - Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build         - Build the application"
	@echo "  build-linux   - Build for Linux"
	@echo "  prod-build    - Production build with quality gates"
	@echo "  clean         - Clean build artifacts"
	@echo ""
	@echo "Test targets:"
	@echo "  test          - Run all tests"
	@echo "  test-unit     - Run unit tests"
	@echo "  test-algorithms - Run algorithm tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-property - Run property-based tests"
	@echo "  coverage      - Generate test coverage report"
	@echo "  coverage-check - Check coverage meets minimum threshold"
	@echo ""
	@echo "Quality targets:"
	@echo "  fmt           - Format code"
	@echo "  fmt-check     - Check code formatting"
	@echo "  lint          - Run linter"
	@echo "  vet           - Run go vet"
	@echo "  security      - Run security scan"
	@echo "  complexity    - Check cyclomatic complexity"
	@echo "  pre-commit    - Run pre-commit checks"
	@echo "  quality-gate  - Run comprehensive quality checks"
	@echo ""
	@echo "Performance targets:"
	@echo "  benchmark     - Run benchmarks"
	@echo "  profile-cpu   - CPU profiling"
	@echo "  profile-mem   - Memory profiling"
	@echo ""
	@echo "Development targets:"
	@echo "  dev           - Start development server"
	@echo "  install-tools - Install development tools"
	@echo "  install-hooks - Install git hooks"
	@echo "  deps          - Download dependencies"
	@echo "  deps-update   - Update dependencies"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-test   - Run tests in Docker"
	@echo ""
	@echo "Database targets:"
	@echo "  db-migrate    - Run database migrations"
	@echo "  db-seed       - Seed database"