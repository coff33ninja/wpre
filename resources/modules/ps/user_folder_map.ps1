param(
    [string]$Action = "resolve",
    [string]$Username = "",
    [string]$OutputPath = "folder_map_result.json"
)

function Get-KnownFolderPath {
    param([string]$FolderId)
    $type = [Type]::GetTypeFromCLSID("03837531-098B-4DA5-AF68-1F07D8A64F9D")
    if (-not $type) { return $null }
    $shell = [System.Activator]::CreateInstance($type)
    $folder = $shell.NameSpace($FolderId)
    if ($folder) { return $folder.Self.Path }
    return $null
}

$knownFolders = @{
    "Desktop"    = [System.Environment+SpecialFolder]::Desktop
    "Documents"  = [System.Environment+SpecialFolder]::MyDocuments
    "Downloads"  = [System.Environment+SpecialFolder]::UserProfiles  # special
    "Pictures"   = [System.Environment+SpecialFolder]::MyPictures
    "Videos"     = [System.Environment+SpecialFolder]::MyVideos
    "Music"      = [System.Environment+SpecialFolder]::MyMusic
    "AppDataRoaming" = [System.Environment+SpecialFolder]::ApplicationData
    "AppDataLocal"   = [System.Environment+SpecialFolder]::LocalApplicationData
}

$result = @{}
foreach ($name in $knownFolders.Keys) {
    $value = [System.Environment]::GetFolderPath($knownFolders[$name])
    if ($name -eq "Downloads") {
        $value = Join-Path $env:USERPROFILE "Downloads"
    }
    $result[$name] = @{
        Path = $value
        Exists = Test-Path $value
    }
}

if ($Username) {
    $userProfileDir = "C:\Users\$Username"
    $result["UserProfile"] = @{
        Path = $userProfileDir
        Exists = Test-Path $userProfileDir
    }
    foreach ($name in @("Desktop", "Documents", "Downloads", "Pictures", "Videos", "Music")) {
        $userPath = Join-Path $userProfileDir $name
        $result["$name($Username)"] = @{
            Path = $userPath
            Exists = Test-Path $userPath
        }
    }
}

$result | ConvertTo-Json -Depth 5 | Out-File -FilePath $OutputPath -Encoding UTF8
Write-Output "Folder map resolved"
