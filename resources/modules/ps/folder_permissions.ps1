param(
    [string]$Action = "backup",
    [string]$SourcePath = "",
    [string]$TargetPath = "",
    [string]$TargetUser = "",
    [string]$OutputPath = "permissions_result.json"
)

function Backup-Permissions {
    param($Path)
    try {
        $acl = Get-Acl -Path $Path -ErrorAction Stop
        $rules = @()
        foreach ($access in $acl.Access) {
            $rules += [PSCustomObject]@{
                IdentityReference = $access.IdentityReference.Value
                FileSystemRights = $access.FileSystemRights.ToString()
                AccessControlType = $access.AccessControlType.ToString()
                IsInherited = $access.IsInherited
            }
        }
        return [PSCustomObject]@{
            Path = $Path
            Owner = $acl.Owner
            Group = $acl.Group
            Rules = $rules
            Success = $true
        }
    } catch {
        return [PSCustomObject]@{
            Path = $Path
            Success = $false
            Error = $_.Exception.Message
        }
    }
}

function Apply-Permissions {
    param($Path, $User)
    try {
        $acl = Get-Acl -Path $Path -ErrorAction Stop
        $permission = "$($User):FullControl"
        $accessRule = New-Object System.Security.AccessControl.FileSystemAccessRule(
            $User, "FullControl", "ContainerInherit,ObjectInherit",
            "None", "Allow"
        )
        $acl.SetAccessRule($accessRule)
        Set-Acl -Path $Path -AclObject $acl

        return [PSCustomObject]@{
            Path = $Path
            User = $User
            Success = $true
        }
    } catch {
        return [PSCustomObject]@{
            Path = $Path
            User = $User
            Success = $false
            Error = $_.Exception.Message
        }
    }
}

function Take-Ownership {
    param($Path)
    try {
        takeown /F $Path /R /D Y | Out-Null
        icacls $Path /grant "$env:USERNAME:(OI)(CI)F" /T | Out-Null
        return [PSCustomObject]@{
            Path = $Path
            Success = $true
        }
    } catch {
        return [PSCustomObject]@{
            Path = $Path
            Success = $false
            Error = $_.Exception.Message
        }
    }
}

switch ($Action) {
    "backup" {
        if (-not $SourcePath) { Write-Error "SourcePath required"; exit 1 }
        $result = Backup-Permissions -Path $SourcePath
    }
    "apply" {
        if (-not $TargetPath -or -not $TargetUser) { Write-Error "TargetPath and TargetUser required"; exit 1 }
        $result = Apply-Permissions -Path $TargetPath -User $TargetUser
    }
    "take-ownership" {
        if (-not $SourcePath) { Write-Error "SourcePath required"; exit 1 }
        $result = Take-Ownership -Path $SourcePath
    }
    default { Write-Error "Unknown action: $Action"; exit 1 }
}

$result | ConvertTo-Json -Depth 10 | Out-File -FilePath $OutputPath -Encoding UTF8
Write-Output "Permissions operation complete: $Action"
