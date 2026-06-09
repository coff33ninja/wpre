param(
    [string]$Action = "profile",
    [string]$Username = "",
    [string]$OutputPath = "cleanup_result.json"
)

function Remove-UserProfile {
    param($User)
    try {
        Remove-LocalUser -Name $User -ErrorAction SilentlyContinue
        $profileDir = "C:\Users\$User"
        if (Test-Path $profileDir) {
            Remove-Item -Path $profileDir -Recurse -Force -ErrorAction SilentlyContinue
        }
        $regPath = "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList"
        $sids = Get-ChildItem $regPath -ErrorAction SilentlyContinue
        foreach ($sid in $sids) {
            $val = Get-ItemProperty -Path $sid.PSPath -Name "ProfileImagePath" -ErrorAction SilentlyContinue
            if ($val.ProfileImagePath -like "*$User*") {
                Remove-Item -Path $sid.PSPath -Recurse -Force -ErrorAction SilentlyContinue
            }
        }
        return [PSCustomObject]@{ Success = $true; User = $User; Action = "profile_removed" }
    } catch {
        return [PSCustomObject]@{ Success = $false; User = $User; Error = $_.Exception.Message }
    }
}

function Remove-AutoLogin {
    $regPath = "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon"
    try {
        Remove-ItemProperty -Path $regPath -Name "DefaultUserName" -ErrorAction SilentlyContinue
        Remove-ItemProperty -Path $regPath -Name "DefaultPassword" -ErrorAction SilentlyContinue
        Remove-ItemProperty -Path $regPath -Name "AutoAdminLogon" -ErrorAction SilentlyContinue
        return [PSCustomObject]@{ Success = $true; Action = "autologin_removed" }
    } catch {
        return [PSCustomObject]@{ Success = $false; Error = $_.Exception.Message }
    }
}

function Remove-MigrationVault {
    param([string]$VaultPath)
    try {
        if (Test-Path $VaultPath) {
            Remove-Item -Path $VaultPath -Recurse -Force
            return [PSCustomObject]@{ Success = $true; Path = $VaultPath; Action = "vault_removed" }
        }
        return [PSCustomObject]@{ Success = $true; Path = $VaultPath; Action = "vault_not_found" }
    } catch {
        return [PSCustomObject]@{ Success = $false; Error = $_.Exception.Message }
    }
}

switch ($Action) {
    "profile" {
        if (-not $Username) { Write-Error "Username required"; exit 1 }
        $result = Remove-UserProfile -User $Username
    }
    "autologin" { $result = Remove-AutoLogin }
    "vault" {
        $vaultPath = if ($args[0]) { $args[0] } else { "C:\MigrationVault" }
        $result = Remove-MigrationVault -VaultPath $vaultPath
    }
    "all" {
        $results = @()
        if ($Username) { $results += Remove-UserProfile -User $Username }
        $results += Remove-AutoLogin
        $result = $results
    }
    default { Write-Error "Unknown action: $Action"; exit 1 }
}

$result | ConvertTo-Json -Depth 5 | Out-File -FilePath $OutputPath -Encoding UTF8
Write-Output "Cleanup operation complete: $Action"
