# Smart Fish Feeder Backend - PowerShell Build Script
# Equivalent to Makefile targets for Windows PowerShell

param(
    [Parameter(Position=0)]
    [string]$Target = "help"
)

# Configuration
$BINARY_NAME = "smart-fish-feeder"
$BINARY_EXE = "$BINARY_NAME.exe"
$MAIN_PATH = "./cmd/server"
$TEST_TIMEOUT = "30s"
$COVERAGE_OUT = "coverage.out"
$COVERAGE_HTML = "coverage.html"
$MIN_COVERAGE = 40
$MAX_CYCLOMATIC_COMPLEXITY = 10

# Helper function to run commands
function Invoke-GoCommand {
    param([string]$Command, [string]$Description)
    Write-Host "`n=== $Description ===" -ForegroundColor Cyan
    Invoke-Expression $Command
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: $Description failed with exit code $LASTEXITCODE" -ForegroundColor Red
        exit $LASTEXITCODE
    }
}

# Build targets
function Build {
    Write-Host "Building $BINARY_NAME..." -ForegroundColor Green
    go build -o $BINARY_EXE -v $MAIN_PATH
}

function Build-Linux {
    Write-Host "Building for Linux..." -ForegroundColor Green
    $env:CGO_ENABLED = "0"
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -o "${BINARY_NAME}_unix" -v $MAIN_PATH
    Remove-Item Env:CGO_ENABLED, Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
}

function Clean {
    Write-Host "Cleaning..." -ForegroundColor Green
    go clean
    Remove-Item -Force -ErrorAction SilentlyContinue $BINARY_EXE, "${BINARY_NAME}_unix", $COVERAGE_OUT, $COVERAGE_HTML
}

# Test targets
function Test {
    Test-Unit
    Test-Algorithms
    Test-Integration
}

function Test-Unit {
    Invoke-GoCommand "go test -v -timeout $TEST_TIMEOUT ./internal/services/..." "Running unit tests"
}

function Test-Algorithms {
    Invoke-GoCommand "go test -v -timeout $TEST_TIMEOUT ./internal/algorithms/..." "Running algorithm tests"
}

function Test-Integration {
    Invoke-GoCommand "go test -v -timeout $TEST_TIMEOUT -tags=integration ./..." "Running integration tests"
}

function Test-Property {
    Invoke-GoCommand "go test -v -timeout $TEST_TIMEOUT -tags=property ./internal/services/..." "Running property-based tests"
}

# Coverage targets
function Coverage {
    Write-Host "Generating test coverage..." -ForegroundColor Green
    go test -coverprofile=$COVERAGE_OUT -covermode=atomic ./...
    go tool cover -html=$COVERAGE_OUT -o $COVERAGE_HTML
    go tool cover -func=$COVERAGE_OUT
}

function Coverage-Check {
    Coverage
    Write-Host "Checking coverage threshold..." -ForegroundColor Green
    $coverageOutput = go tool cover -func=$COVERAGE_OUT | Select-String "total:"
    if ($coverageOutput -match "(\d+\.?\d*)%") {
        $coverage = [float]$Matches[1]
        if ($coverage -lt $MIN_COVERAGE) {
            Write-Host "X Coverage $coverage% is below minimum $MIN_COVERAGE%" -ForegroundColor Red
            exit 1
        } else {
            Write-Host "OK Coverage $coverage% meets minimum $MIN_COVERAGE%" -ForegroundColor Green
        }
    }
}

# Quality targets
function Fmt {
    Write-Host "Formatting code..." -ForegroundColor Green
    gofmt -s -w .
}

function Fmt-Check {
    Write-Host "Checking code formatting..." -ForegroundColor Green
    $unformatted = gofmt -l .
    if ($unformatted) {
        Write-Host "X The following files are not formatted:" -ForegroundColor Red
        Write-Host $unformatted
        exit 1
    } else {
        Write-Host "OK All files are properly formatted" -ForegroundColor Green
    }
}

function Lint {
    Invoke-GoCommand "golangci-lint run --timeout=5m" "Running linter"
}

function Vet {
    Invoke-GoCommand "go vet ./..." "Running go vet"
}

function Security {
    Invoke-GoCommand "gosec -quiet ./..." "Running security scan"
}

# Benchmark targets
function Benchmark {
    Invoke-GoCommand "go test -bench=. -benchmem ./internal/algorithms/..." "Running benchmarks"
}

function Profile-CPU {
    Write-Host "Running CPU profiling..." -ForegroundColor Green
    go test -cpuprofile=cpu.prof -bench=. ./internal/algorithms/...
    go tool pprof cpu.prof
}

function Profile-Mem {
    Write-Host "Running memory profiling..." -ForegroundColor Green
    go test -memprofile=mem.prof -bench=. ./internal/algorithms/...
    go tool pprof mem.prof
}

# Dependency management
function Deps {
    Invoke-GoCommand "go mod download" "Downloading dependencies"
}

function Deps-Update {
    Invoke-GoCommand "go mod tidy" "Updating dependencies"
}

function Deps-Verify {
    Invoke-GoCommand "go mod verify" "Verifying dependencies"
}

# Install development tools
function Install-Tools {
    Write-Host "Installing development tools..." -ForegroundColor Green
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install github.com/securego/gosec/v2/cmd/gosec@latest
    go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
}

# Pre-commit checks
function Pre-Commit {
    Fmt-Check
    Vet
    Lint
    Test-Unit
    Test-Algorithms
    Coverage-Check
    Security
    Write-Host "`nOK All pre-commit checks passed!" -ForegroundColor Green
}

# Quality gate
function Quality-Gate {
    Clean
    Deps-Verify
    Fmt-Check
    Vet
    Lint
    Security
    Test
    Coverage-Check
    Benchmark
    Write-Host "`nOK All quality gates passed!" -ForegroundColor Green
}

# Cyclomatic complexity check
function Complexity {
    Write-Host "Checking cyclomatic complexity..." -ForegroundColor Green
    $complex = gocyclo -over $MAX_CYCLOMATIC_COMPLEXITY .
    if ($complex) {
        Write-Host "X Functions with high cyclomatic complexity:" -ForegroundColor Red
        Write-Host $complex
        exit 1
    } else {
        Write-Host "OK All functions have acceptable complexity" -ForegroundColor Green
    }
}

# Docker targets
function Docker-Build {
    Invoke-GoCommand "docker build -t smart-fish-feeder:latest ." "Building Docker image"
}

function Docker-Test {
    Invoke-GoCommand "docker run --rm smart-fish-feeder:latest make test" "Running tests in Docker"
}

# Development server
function Dev {
    Write-Host "Starting development server..." -ForegroundColor Green
    go run $MAIN_PATH
}

# Production build
function Prod-Build {
    Quality-Gate
    Write-Host "Building production binary..." -ForegroundColor Green
    $env:CGO_ENABLED = "0"
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    go build -ldflags="-w -s" -o $BINARY_NAME $MAIN_PATH
    Remove-Item Env:CGO_ENABLED, Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
}

# All target
function All {
    Clean
    Fmt
    Lint
    Vet
    Test
    Build
}

# Help
function Show-Help {
    Write-Host @"

Smart Fish Feeder Backend - PowerShell Build Script
====================================================

Usage: .\build.ps1 <target>

Build targets:
  build         - Build the application
  build-linux   - Build for Linux
  prod-build    - Production build with quality gates
  clean         - Clean build artifacts
  all           - Clean, format, lint, vet, test, build

Test targets:
  test          - Run all tests
  test-unit     - Run unit tests
  test-algorithms - Run algorithm tests
  test-integration - Run integration tests
  test-property - Run property-based tests
  coverage      - Generate test coverage report
  coverage-check - Check coverage meets minimum threshold

Quality targets:
  fmt           - Format code
  fmt-check     - Check code formatting
  lint          - Run linter
  vet           - Run go vet
  security      - Run security scan
  complexity    - Check cyclomatic complexity
  pre-commit    - Run pre-commit checks
  quality-gate  - Run comprehensive quality checks

Performance targets:
  benchmark     - Run benchmarks
  profile-cpu   - CPU profiling
  profile-mem   - Memory profiling

Development targets:
  dev           - Start development server
  install-tools - Install development tools
  deps          - Download dependencies
  deps-update   - Update dependencies
  deps-verify   - Verify dependencies

Docker targets:
  docker-build  - Build Docker image
  docker-test   - Run tests in Docker

"@ -ForegroundColor Yellow
}

# Main execution
switch ($Target.ToLower()) {
    "build"         { Build }
    "build-linux"   { Build-Linux }
    "clean"         { Clean }
    "test"          { Test }
    "test-unit"     { Test-Unit }
    "test-algorithms" { Test-Algorithms }
    "test-integration" { Test-Integration }
    "test-property" { Test-Property }
    "coverage"      { Coverage }
    "coverage-check" { Coverage-Check }
    "fmt"           { Fmt }
    "fmt-check"     { Fmt-Check }
    "lint"          { Lint }
    "vet"           { Vet }
    "security"      { Security }
    "benchmark"     { Benchmark }
    "profile-cpu"   { Profile-CPU }
    "profile-mem"   { Profile-Mem }
    "deps"          { Deps }
    "deps-update"   { Deps-Update }
    "deps-verify"   { Deps-Verify }
    "install-tools" { Install-Tools }
    "pre-commit"    { Pre-Commit }
    "quality-gate"  { Quality-Gate }
    "complexity"    { Complexity }
    "docker-build"  { Docker-Build }
    "docker-test"   { Docker-Test }
    "dev"           { Dev }
    "prod-build"    { Prod-Build }
    "all"           { All }
    "help"          { Show-Help }
    default         { 
        Write-Host "Unknown target: $Target" -ForegroundColor Red
        Show-Help 
    }
}
