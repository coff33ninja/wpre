param(
    [string]$Action = "export",
    [string]$RegPath = "",
    [string]$OutputFile = "",
    [string]$OutputPath = "registry_result.json"
)

function Export-RegistryKey {
    param($Path, $File)
    try {
        $parent = Split-Path $File -Parent
        if ($parent -and -not (Test-Path $parent)) {
            New-Item -Path $parent -ItemType Directory -Force | Out-Null
        }
        reg export "$Path" "$File" /y 2>$null | Out-Null
        if (Test-Path $File) {
            return [PSCustomObject]@{
                Path = $Path
                ExportedTo = $File
                Success = $true
            }
        }
        throw "Export command did not produce output file"
    } catch {
        return [PSCustomObject]@{
            Path = $Path
            Success = $false
            Error = $_.Exception.Message
        }
    }
}

function Snapshot-UserHive {
    param($OutputDir)
    $results = @()
    $keys = @(
        "HKCU\Software\Microsoft\Windows\CurrentVersion\Explorer"
        "HKCU\Software\Microsoft\Windows\CurrentVersion\Run"
        "HKCU\Software\Microsoft\Office"
        "HKCU\Software\Google\Chrome"
        "HKCU\Software\Microsoft\Edge"
        "HKCU\Software\Mozilla"
        "HKCU\Software\Microsoft\OneDrive"
    )
    foreach ($key in $keys) {
        $safeName = $key -replace '[/:\\]', '_'
        $outFile = Join-Path $OutputDir "$safeName.reg"
        $results += Export-RegistryKey -Path $key -File $outFile
    }
    return $results
}

switch ($Action) {
    "export" {
        if (-not $RegPath -or -not $OutputFile) {
            Write-Error "RegPath and OutputFile required"; exit 1
        }
        $result = Export-RegistryKey -Path $RegPath -File $OutputFile
    }
    "snapshot-user" {
        $dir = if ($OutputFile) { $OutputFile } else { "registry_snapshot" }
        if (-not (Test-Path $dir)) { New-Item -Path $dir -ItemType Directory -Force | Out-Null }
        $result = Snapshot-UserHive -OutputDir $dir
    }
    default { Write-Error "Unknown action: $Action"; exit 1 }
}

$result | ConvertTo-Json -Depth 5 | Out-File -FilePath $OutputPath -Encoding UTF8
Write-Output "Registry operation complete: $Action"
