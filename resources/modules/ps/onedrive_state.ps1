param(
    [string]$Action = "detect",
    [string]$OutputPath = "onedrive_result.json"
)

function Get-OneDriveDetailedStatus {
    $status = @{
        ProcessRunning = $null -ne (Get-Process -Name "OneDrive" -ErrorAction SilentlyContinue)
        ExecutablePath = $null
        Accounts = @()
        Folders = @()
        SyncStatus = "unknown"
    }
    $process = Get-Process -Name "OneDrive" -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($process) { $status.ExecutablePath = $process.Path }

    $regPaths = @(
        "HKCU:\Software\Microsoft\OneDrive\Accounts",
        "HKCU:\Software\Microsoft\OneDrive\Accounts\Business1"
    )
    foreach ($regPath in $regPaths) {
        if (Test-Path $regPath) {
            $acct = Get-ItemProperty -Path $regPath -ErrorAction SilentlyContinue
            if ($acct) {
                $status.Accounts += [PSCustomObject]@{
                    Path = $regPath
                    DisplayName = $acct.DisplayName
                    AccountType = $acct.AccountType
                    ServiceEndpointUri = $acct.ServiceEndpointUri
                }
            }
        }
    }
    $possiblePaths = Get-ChildItem "$env:USERPROFILE\OneDrive*" -Directory -ErrorAction SilentlyContinue
    foreach ($p in $possiblePaths) {
        $status.Folders += [PSCustomObject]@{
            Path = $p.FullName
            ItemCount = (Get-ChildItem -Path $p.FullName -Recurse -File -ErrorAction SilentlyContinue).Count
            SizeMB = [math]::Round(
                ((Get-ChildItem -Path $p.FullName -Recurse -File -ErrorAction SilentlyContinue |
                    Measure-Object -Property Length -Sum).Sum / 1MB), 2
            )
        }
    }
    return $status
}

function Invoke-OneDrivePause {
    try {
        $proc = Get-Process -Name "OneDrive" -ErrorAction SilentlyContinue
        if (-not $proc) { return @{Success = $false; Message = "OneDrive not running"} }

        $onedriveExe = $proc[0].Path
        if (-not $onedriveExe) { return @{Success = $false; Message = "Cannot find OneDrive.exe path"} }

        Start-Process -FilePath $onedriveExe -ArgumentList "/pause" -NoNewWindow -Wait
        return @{Success = $true; Message = "OneDrive sync paused"}
    } catch {
        return @{Success = $false; Message = $_.Exception.Message}
    }
}

function Invoke-OneDriveResume {
    try {
        $proc = Get-Process -Name "OneDrive" -ErrorAction SilentlyContinue
        if (-not $proc) { return @{Success = $false; Message = "OneDrive not running"} }

        $onedriveExe = $proc[0].Path
        if (-not $onedriveExe) { return @{Success = $false; Message = "Cannot find OneDrive.exe path"} }

        Start-Process -FilePath $onedriveExe -ArgumentList "/resume" -NoNewWindow -Wait
        return @{Success = $true; Message = "OneDrive sync resumed"}
    } catch {
        return @{Success = $false; Message = $_.Exception.Message}
    }
}

function Invoke-OneDriveSignOut {
    try {
        $proc = Get-Process -Name "OneDrive" -ErrorAction SilentlyContinue
        if (-not $proc) { return @{Success = $false; Message = "OneDrive not running"} }

        $onedriveExe = $proc[0].Path
        Start-Process -FilePath $onedriveExe -ArgumentList "/signout" -NoNewWindow -Wait
        return @{Success = $true; Message = "OneDrive signed out"}
    } catch {
        return @{Success = $false; Message = $_.Exception.Message}
    }
}

function Invoke-OneDriveShutdown {
    try {
        Get-Process -Name "OneDrive" -ErrorAction SilentlyContinue | Stop-Process -Force
        Start-Sleep -Seconds 2
        return @{Success = $true; Message = "OneDrive process stopped"}
    } catch {
        return @{Success = $false; Message = $_.Exception.Message}
    }
}

function Get-PlaceholderFiles {
    param([string]$Path)
    $placeholders = @()
    $files = Get-ChildItem -Path $Path -Recurse -File -ErrorAction SilentlyContinue
    foreach ($f in $files) {
        $attributes = (Get-ItemProperty -Path $f.FullName).Attributes
        if ($attributes -band 0x00400000) {
            $placeholders += $f.FullName
        }
    }
    return $placeholders
}

switch ($Action) {
    "detect" { $result = Get-OneDriveDetailedStatus }
    "pause" { $result = Invoke-OneDrivePause }
    "resume" { $result = Invoke-OneDriveResume }
    "signout" { $result = Invoke-OneDriveSignOut }
    "shutdown" { $result = Invoke-OneDriveShutdown }
    "placeholders" {
        $targetPath = $args[0]
        if (-not $targetPath) { Write-Error "Path argument required"; exit 1 }
        $result = @{
            Path = $targetPath
            PlaceholderCount = (Get-PlaceholderFiles -Path $targetPath).Count
            PlaceholderFiles = Get-PlaceholderFiles -Path $targetPath
        }
    }
    default { Write-Error "Unknown action: $Action"; exit 1 }
}

$result | ConvertTo-Json -Depth 10 | Out-File -FilePath $OutputPath -Encoding UTF8
Write-Output "OneDrive operation complete: $Action"
