# Smart Fish Feeder - Pre-commit Hook (PowerShell)
# This script runs comprehensive checks before allowing commits

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "Running pre-commit checks..." -ForegroundColor Cyan
Write-Host ""

# Configuration
$MIN_COVERAGE = 40

# Helper functions
function Write-Status {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

function Write-Err {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Test-Command {
    param([string]$Command)
    $null = Get-Command $Command -ErrorAction SilentlyContinue
    return $?
}

# Check if we are in the backend directory
function Test-ProjectDirectory {
    if (-not (Test-Path "go.mod")) {
        Write-Err "Not in Go project directory. Please run from backend directory."
        exit 1
    }
}

# Check for required tools
function Test-RequiredTools {
    Write-Status "Checking required tools..."
    
    $missingTools = @()
    
    if (-not (Test-Command "go")) { $missingTools += "go" }
    if (-not (Test-Command "golangci-lint")) { $missingTools += "golangci-lint" }
    if (-not (Test-Command "gosec")) { $missingTools += "gosec" }
    
    if ($missingTools.Count -gt 0) {
        Write-Err ("Missing required tools: " + ($missingTools -join ", "))
        Write-Status "Run .\build.ps1 install-tools to install missing tools"
        exit 1
    }
    
    Write-Success "All required tools are available"
}

# Check Go modules
function Test-GoModules {
    Write-Status "Verifying Go modules..."
    
    go mod verify
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Go module verification failed"
        exit 1
    }
    
    go mod tidy
    
    Write-Success "Go modules verified"
}

# Format check
function Test-Formatting {
    Write-Status "Checking code formatting..."
    
    $unformattedFiles = gofmt -l .
    if ($unformattedFiles) {
        Write-Err "The following files are not formatted:"
        Write-Host $unformattedFiles
        Write-Status "Run .\build.ps1 fmt to format files"
        exit 1
    }
    
    Write-Success "All files are properly formatted"
}

# Vet check
function Test-GoVet {
    Write-Status "Running go vet..."
    
    go vet ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Err "go vet failed"
        exit 1
    }
    
    Write-Success "go vet passed"
}

# Lint check
function Test-Linting {
    Write-Status "Running linter..."
    
    golangci-lint run --timeout=5m
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Linting failed"
        exit 1
    }
    
    Write-Success "Linting passed"
}

# Security check (optional - gosec can be very slow)
function Test-Security {
    Write-Status "Running security scan..."
    Write-Warn "Note: gosec can take 3-5+ minutes. Press Ctrl+C to skip if needed."
    
    $result = gosec -quiet ./... 2>&1
    
    if ($LASTEXITCODE -ne 0) {
        # Check if there are actual issues (not just nosec comments)
        $issueCount = ($result | Select-String "Issues : (\d+)" | ForEach-Object { $_.Matches.Groups[1].Value })
        if ($issueCount -gt 0) {
            Write-Err "Security scan found $issueCount issues:"
            Write-Host $result
            exit 1
        }
    }
    
    Write-Success "Security scan passed"
}

# Unit tests
function Test-UnitTests {
    Write-Status "Running unit tests..."
    
    go test -timeout=30s ./internal/services/...
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Unit tests failed"
        exit 1
    }
    
    Write-Success "Unit tests passed"
}

# Algorithm tests
function Test-AlgorithmTests {
    Write-Status "Running algorithm tests..."
    
    go test -timeout=30s ./internal/algorithms/...
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Algorithm tests failed"
        exit 1
    }
    
    Write-Success "Algorithm tests passed"
}

# Coverage check
function Test-Coverage {
    Write-Status "Checking test coverage..."
    
    $coverageFile = "coverage.out"
    
    go test -coverprofile=$coverageFile -covermode=atomic ./...
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Coverage generation failed"
        exit 1
    }
    
    $coverageOutput = go tool cover -func=$coverageFile | Select-String "total:"
    if ($coverageOutput -match "(\d+\.?\d*)%") {
        $coverage = [float]$Matches[1]
        
        if ($coverage -lt $MIN_COVERAGE) {
            Write-Err "Coverage $coverage% is below minimum $MIN_COVERAGE%"
            Remove-Item -Force $coverageFile -ErrorAction SilentlyContinue
            exit 1
        }
        
        Write-Success "Coverage $coverage% meets minimum $MIN_COVERAGE%"
    }
    
    Remove-Item -Force $coverageFile -ErrorAction SilentlyContinue
}

# Build check
function Test-Build {
    Write-Status "Checking build..."
    
    $tempBinary = Join-Path $env:TEMP "smart-fish-feeder-test.exe"
    
    go build -o $tempBinary ./cmd/server
    if ($LASTEXITCODE -ne 0) {
        Write-Err "Build failed"
        exit 1
    }
    
    Remove-Item -Force $tempBinary -ErrorAction SilentlyContinue
    Write-Success "Build successful"
}

# Check for large files
function Test-LargeFiles {
    Write-Status "Checking for large files..."
    
    $largeFiles = Get-ChildItem -Recurse -File -ErrorAction SilentlyContinue | 
        Where-Object { $_.Length -gt 1MB -and $_.FullName -notmatch "\.git|vendor" } |
        Select-Object -ExpandProperty FullName
    
    if ($largeFiles) {
        Write-Warn "Large files detected:"
        $largeFiles | ForEach-Object { Write-Host "  $_" }
        Write-Status "Consider using Git LFS for large files"
    } else {
        Write-Success "No large files detected"
    }
}

# Check for sensitive data (only actual hardcoded secrets, not variable names)
function Test-SensitiveData {
    Write-Status "Checking for sensitive data..."
    
    $foundSensitive = $false
    
    # Check for .env file with real secrets (not .env.example)
    if (Test-Path ".env") {
        $envContent = Get-Content ".env" -ErrorAction SilentlyContinue
        $realSecrets = $envContent | Where-Object { 
            $_ -match "^(PASSWORD|SECRET|API_KEY|PRIVATE_KEY)=.+" -and 
            $_ -notmatch "=\s*$" -and 
            $_ -notmatch "=your_" -and 
            $_ -notmatch "=changeme"
        }
        if ($realSecrets) {
            Write-Warn "Potential secrets in .env file - ensure this is not committed"
            $foundSensitive = $true
        }
    }
    
    # Check for AWS credentials in config files
    $awsKeys = Get-ChildItem -Recurse -Include *.yaml,*.yml,*.json -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -notmatch "\.git|vendor" } |
        Select-String -Pattern "AKIA[0-9A-Z]{16}" -ErrorAction SilentlyContinue
    
    if ($awsKeys) {
        Write-Err "AWS access key found in config files"
        $foundSensitive = $true
    }
    
    # Check for private keys in config files (not test files)
    $privateKeys = Get-ChildItem -Recurse -Include *.yaml,*.yml,*.json,*.pem -ErrorAction SilentlyContinue |
        Where-Object { $_.FullName -notmatch "\.git|vendor|_test\.go" } |
        Select-String -Pattern "BEGIN.*PRIVATE KEY" -ErrorAction SilentlyContinue
    
    if ($privateKeys) {
        Write-Err "Private key found in config files"
        $foundSensitive = $true
    }
    
    if ($foundSensitive) {
        Write-Err "Potential sensitive data found"
        Write-Status "Please review and remove sensitive data before committing"
        exit 1
    }
    
    Write-Success "No sensitive data detected"
}

# Main execution
function Main {
    Write-Host "Smart Fish Feeder Pre-commit Checks" -ForegroundColor Cyan
    Write-Host "====================================" -ForegroundColor Cyan
    
    Test-ProjectDirectory
    Test-RequiredTools
    Test-GoModules
    Test-Formatting
    Test-GoVet
    Test-Linting
    Test-Security
    Test-SensitiveData
    Test-LargeFiles
    Test-UnitTests
    Test-AlgorithmTests
    Test-Coverage
    Test-Build
    
    Write-Host ""
    Write-Success "All pre-commit checks passed!"
    Write-Status "Commit is ready to proceed"
}

# Run main function
Main
