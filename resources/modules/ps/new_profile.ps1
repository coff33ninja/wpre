param(
    [string]$Username = "NewUser",
    [string]$FullName = "",
    [string]$Password = "",
    [string]$OutputPath = "new_profile_result.json"
)

function New-TargetProfile {
    param($User, $Full, $Pass)
    $securePass = ConvertTo-SecureString -String $Pass -AsPlainText -Force
    try {
        $params = @{
            Name = $User
            Password = $securePass
            PasswordNeverExpires = $true
            AccountNeverExpires = $true
        }
        if ($Full) { $params.FullName = $Full }
        $null = New-LocalUser @params

        $sid = (Get-LocalUser -Name $User).SID.Value
        $profilePath = "C:\Users\$User"

        return [PSCustomObject]@{
            Success = $true
            Username = $User
            SID = $sid
            ProfilePath = $profilePath
        }
    } catch {
        return [PSCustomObject]@{
            Success = $false
            Username = $User
            Error = $_.Exception.Message
        }
    }
}

$result = New-TargetProfile -User $Username -Full $FullName -Pass $Password
$result | ConvertTo-Json | Out-File -FilePath $OutputPath -Encoding UTF8
Write-Output "New profile result: $($result.Success)"
if (-not $result.Success) { exit 1 }
