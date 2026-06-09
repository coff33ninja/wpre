param(
    [switch]$Verbose,
    [string]$Package = "./..."
)

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

Write-Step "Running Go tests"
$goArgs = @("test")
if ($Verbose) { $goArgs += "-v" }
$goArgs += $Package
& go $goArgs

if ($LASTEXITCODE -ne 0) {
    Write-Host "FAIL: Go tests failed" -ForegroundColor Red
    exit 1
}

Write-Step "All tests passed" -ForegroundColor Green
