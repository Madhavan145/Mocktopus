# Build all Go services for Linux (cross-compile from Windows)
# Usage: .\build.ps1

$ErrorActionPreference = "Stop"
$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"

$services = @(
    @{ Name = "auth-service";            Binary = "auth" },
    @{ Name = "chaos-service";           Binary = "chaos" },
    @{ Name = "cloud-catalog-service";   Binary = "cloud-catalog" },
    @{ Name = "cost-service";            Binary = "cost" },
    @{ Name = "gateway-service";         Binary = "gateway" },
    @{ Name = "iac-service";             Binary = "iac" },
    @{ Name = "metrics-service";         Binary = "metrics" },
    @{ Name = "project-service";         Binary = "project" },
    @{ Name = "recommendation-service";  Binary = "recommendation" },
    @{ Name = "replay-service";          Binary = "replay" },
    @{ Name = "simulation-service";      Binary = "simulation" },
    @{ Name = "template-service";        Binary = "template" },
    @{ Name = "topology-service";        Binary = "topology" },
    @{ Name = "admin-service";           Binary = "admin" }
)

Write-Host "Building all services for linux/amd64..." -ForegroundColor Cyan

foreach ($svc in $services) {
    $svcDir = "services/$($svc.Name)"
    $outPath = "$svcDir/bin/$($svc.Binary)"
    
    Write-Host "  Building $($svc.Name)..." -NoNewline
    
    Push-Location $svcDir
    try {
        go build -ldflags="-s -w" -o "bin/$($svc.Binary)" ./cmd/server
        if ($LASTEXITCODE -ne 0) {
            Write-Host " FAILED" -ForegroundColor Red
            exit 1
        }
        Write-Host " OK" -ForegroundColor Green
    } finally {
        Pop-Location
    }
}

Write-Host "`nAll services built successfully!" -ForegroundColor Green
Write-Host "Run 'docker compose up --build' to start." -ForegroundColor Yellow
