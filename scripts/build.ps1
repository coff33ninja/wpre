param(
    [string]$OutputDir = ".",
    [switch]$CrossCompile
)

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

Write-Step "Building WPRE orchestrator"

Write-Step "Tidying dependencies"
go mod tidy

if ($CrossCompile) {
    Write-Step "Cross-compiling for Windows x64"
    $env:GOOS = "windows"
    $env:GOARCH = "amd64"
    go build -ldflags="-s -w" -o "$OutputDir\wpre-x64.exe" ./cmd/wpre

    Write-Step "Cross-compiling for Windows arm64"
    $env:GOARCH = "arm64"
    go build -ldflags="-s -w" -o "$OutputDir\wpre-arm64.exe" ./cmd/wpre

    Remove-Item Env:GOOS, Env:GOARCH -ErrorAction SilentlyContinue
} else {
    Write-Step "Building for native architecture"
    go build -ldflags="-s -w" -o "$OutputDir\wpre.exe" ./cmd/wpre
}

Write-Step "Build complete"
Get-ChildItem "$OutputDir\wpre*.exe" | ForEach-Object {
    Write-Host "  $($_.Name) - $([math]::Round($_.Length / 1MB, 2)) MB"
}
