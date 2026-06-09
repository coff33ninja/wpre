param(
    [string]$Action = "resolve",
    [string]$SID = "",
    [string]$Username = "",
    [string]$OutputPath = "sid_result.json"
)

function Resolve-SID {
    param($SidValue)
    try {
        $obj = New-Object System.Security.Principal.SecurityIdentifier($SidValue)
        $user = $obj.Translate([System.Security.Principal.NTAccount])
        return [PSCustomObject]@{
            SID = $SidValue
            Username = $user.Value
            Resolved = $true
        }
    } catch {
        return [PSCustomObject]@{
            SID = $SidValue
            Username = $null
            Resolved = $false
        }
    }
}

function Resolve-Username {
    param($Name)
    try {
        $obj = New-Object System.Security.Principal.NTAccount($Name)
        $sid = $obj.Translate([System.Security.Principal.SecurityIdentifier])
        return [PSCustomObject]@{
            Username = $Name
            SID = $sid.Value
            Resolved = $true
        }
    } catch {
        return [PSCustomObject]@{
            Username = $Name
            SID = $null
            Resolved = $false
        }
    }
}

function Get-AllUserSIDs {
    $profileList = Get-ItemProperty "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\ProfileList\*"
    $results = @()
    foreach ($p in $profileList) {
        $results += [PSCustomObject]@{
            SID = $p.PSChildName
            ProfilePath = $p.ProfileImagePath
            IsLoaded = (Get-Process -IncludeUserName -ErrorAction SilentlyContinue |
                Where-Object UserName -like "*$($p.PSChildName)*").Count -gt 0
        }
    }
    return $results
}

switch ($Action) {
    "resolve-sid" { $result = Resolve-SID -SidValue $SID }
    "resolve-user" { $result = Resolve-Username -Name $Username }
    "list-all" { $result = Get-AllUserSIDs }
    default { Write-Error "Unknown action: $Action"; exit 1 }
}

$result | ConvertTo-Json -Depth 5 | Out-File -FilePath $OutputPath -Encoding UTF8
Write-Output "SID operation complete: $Action"
