param(
    [string]$Action = "create",
    [string]$Username = "WPRE_TempAdmin",
    [string]$Password = "",
    [string]$OutputPath = "temp_profile_result.json"
)

function New-TempProfile {
    param($User, $Pass)

    $securePass = ConvertTo-SecureString -String $Pass -AsPlainText -Force
    try {
        $null = New-LocalUser -Name $User -Password $securePass -PasswordNeverExpires -AccountNeverExpires
        $null = Add-LocalGroupMember -Group "Administrators" -Member $User

        return [PSCustomObject]@{
            Success = $true
            Username = $User
            Action = "created"
        }
    } catch {
        return [PSCustomObject]@{
            Success = $false
            Username = $User
            Action = "create_failed"
            Error = $_.Exception.Message
        }
    }
}

function Remove-TempProfile {
    param($User)
    try {
        $null = Remove-LocalUser -Name $User -ErrorAction SilentlyContinue

        $profilePath = "C:\Users\$User"
        if (Test-Path $profilePath) {
            Remove-Item -Path $profilePath -Recurse -Force -ErrorAction SilentlyContinue | Out-Null
        }

        return [PSCustomObject]@{
            Success = $true
            Username = $User
            Action = "removed"
        }
    } catch {
        return [PSCustomObject]@{
            Success = $false
            Username = $User
            Action = "remove_failed"
            Error = $_.Exception.Message
        }
    }
}

function Set-AutoLogin {
    param($User, $Pass)
    $regPath = "HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon"
    try {
        Set-ItemProperty -Path $regPath -Name "DefaultUserName" -Value $User
        Set-ItemProperty -Path $regPath -Name "DefaultPassword" -Value $Pass
        Set-ItemProperty -Path $regPath -Name "AutoAdminLogon" -Value "1"
        return $true
    } catch {
        return $false
    }
}

switch ($Action) {
    "create" {
        if (-not $Password) {
            Add-Type -AssemblyName System.Web
            $Password = [System.Web.Security.Membership]::GeneratePassword(16, 4)
        }
        $result = New-TempProfile -User $Username -Pass $Password
        if ($result.Success) {
            $null = Set-AutoLogin -User $Username -Pass $Password
        }
        $result | ConvertTo-Json | Out-File -FilePath $OutputPath -Encoding UTF8
        Write-Output "Temp profile action: create -> $($result.Success)"
    }
    "remove" {
        $result = Remove-TempProfile -User $Username
        $result | ConvertTo-Json | Out-File -FilePath $OutputPath -Encoding UTF8
        Write-Output "Temp profile action: remove -> $($result.Success)"
    }
    default {
        Write-Error "Unknown action: $Action"
        exit 1
    }
}
