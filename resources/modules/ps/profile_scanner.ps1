param(
    [string]$OutputPath = "profile_scan_result.json"
)

function Get-UserProfiles {
    $profiles = @()
    $profileList = Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList\*"
    foreach ($profile in $profileList) {
        if ($profile.ProfileImagePath) {
            $profiles += [PSCustomObject]@{
                SID = $profile.PSChildName
                Path = $profile.ProfileImagePath
                Exists = Test-Path $profile.ProfileImagePath
            }
        }
    }
    return $profiles
}

function Get-OneDriveStatus {
    $onedrive = @{
        ProcessRunning = $null -ne (Get-Process -Name "OneDrive" -ErrorAction SilentlyContinue)
        Installed = Test-Path "$env:LOCALAPPDATA\Microsoft\OneDrive"
        FolderPaths = @()
    }
    $possiblePaths = @(
        "$env:USERPROFILE\OneDrive*"
    )
    foreach ($pattern in $possiblePaths) {
        $matches = Get-ChildItem -Path $pattern -Directory -ErrorAction SilentlyContinue
        foreach ($m in $matches) {
            $onedrive.FolderPaths += $m.FullName
        }
    }
    return $onedrive
}

$result = @{
    Timestamp = (Get-Date).ToString("o")
    ComputerName = $env:COMPUTERNAME
    CurrentUser = $env:USERNAME
    Profiles = Get-UserProfiles
    OneDrive = Get-OneDriveStatus
    SystemInfo = @{
        OSVersion = (Get-CimInstance Win32_OperatingSystem).Version
        IsElevated = ([Security.Principal.WindowsPrincipal]::new(
            [Security.Principal.WindowsIdentity]::GetCurrent()
        )).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    }
}

$result | ConvertTo-Json -Depth 10 | Out-File -FilePath $OutputPath -Encoding UTF8
Write-Output "Profile scan complete: $OutputPath"
