param(
    [Parameter(Mandatory = $true)]
    [string]$MailPVPath,
    [Parameter(Mandatory = $true)]
    [string]$OutputPath
)

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing

$form = New-Object System.Windows.Forms.Form
$form.Text = "WPRE — Mail Password Recovery"
$form.Size = New-Object System.Drawing.Size(820, 640)
$form.StartPosition = "CenterScreen"
$form.TopMost = $true
$form.Icon = $null

$header = New-Object System.Windows.Forms.Label
$header.Location = New-Object System.Drawing.Point(12, 12)
$header.Size = New-Object System.Drawing.Size(780, 60)
$header.Text = "Step 1: Click 'Launch MailPV' below.`nStep 2: In MailPV, press Ctrl+A to select all rows, then Ctrl+C to copy.`nStep 3: Click into the text area below and press Ctrl+V to paste.`nStep 4: Click 'Parse & Save'."
$header.Font = New-Object System.Drawing.Font("Segoe UI", 10)

$launchBtn = New-Object System.Windows.Forms.Button
$launchBtn.Location = New-Object System.Drawing.Point(12, 80)
$launchBtn.Size = New-Object System.Drawing.Size(140, 30)
$launchBtn.Text = "Launch MailPV"
$launchBtn.Add_Click({
    try {
        Start-Process -FilePath $MailPVPath -WindowStyle Normal
    } catch {
        [System.Windows.Forms.MessageBox]::Show("Failed to launch MailPV: $($_.Exception.Message)", "Error")
    }
})

$pasteBox = New-Object System.Windows.Forms.TextBox
$pasteBox.Location = New-Object System.Drawing.Point(12, 120)
$pasteBox.Size = New-Object System.Drawing.Size(780, 400)
$pasteBox.Multiline = $true
$pasteBox.ScrollBars = "Both"
$pasteBox.Font = New-Object System.Drawing.Font("Consolas", 9)
$pasteBox.AcceptsTab = $true
$pasteBox.WordWrap = $false

$saveBtn = New-Object System.Windows.Forms.Button
$saveBtn.Location = New-Object System.Drawing.Point(160, 80)
$saveBtn.Size = New-Object System.Drawing.Size(140, 30)
$saveBtn.Text = "Parse & Save"

$skipBtn = New-Object System.Windows.Forms.Button
$skipBtn.Location = New-Object System.Drawing.Point(310, 80)
$skipBtn.Size = New-Object System.Drawing.Size(140, 30)
$skipBtn.Text = "Skip (no data)"

$statusLabel = New-Object System.Windows.Forms.Label
$statusLabel.Location = New-Object System.Drawing.Point(12, 530)
$statusLabel.Size = New-Object System.Drawing.Size(780, 30)
$statusLabel.Text = "Paste MailPV data above, then click Parse & Save."
$statusLabel.Font = New-Object System.Drawing.Font("Segoe UI", 9, [System.Drawing.FontStyle]::Italic)

$saveBtn.Add_Click({
    $raw = $pasteBox.Text.Trim()
    if ([string]::IsNullOrEmpty($raw)) {
        [System.Windows.Forms.MessageBox]::Show("Paste data from MailPV first.", "No Data")
        return
    }

    $lines = $raw -split "`r`n|`n"
    $accounts = @()
    $lineCount = 0
    $errorCount = 0

    foreach ($line in $lines) {
        $trimmed = $line.Trim()
        if ([string]::IsNullOrEmpty($trimmed)) { continue }
        $lineCount++

        $fields = $trimmed -split "`t"
        if ($fields.Count -lt 7) {
            $errorCount++
            continue
        }

        $account = @{
            email = if ($fields[0]) { $fields[0].Trim() } else { "" }
            application = if ($fields[1]) { $fields[1].Trim() } else { "" }
            displayName = if ($fields[2]) { $fields[2].Trim() } else { "" }
            server = if ($fields[3]) { $fields[3].Trim() } else { "" }
            port = if ($fields[4]) { $fields[4].Trim() } else { "" }
            ssl = if ($fields[5]) { $fields[5].Trim() } else { "" }
            serverType = if ($fields[6]) { $fields[6].Trim() } else { "" }
            userName = if ($fields[7]) { $fields[7].Trim() } else { "" }
            password = if ($fields[8]) { $fields[8].Trim() } else { "" }
            profile = if ($fields[9]) { $fields[9].Trim() } else { "" }
            passwordStrength = if ($fields[10]) { $fields[10].Trim() } else { "" }
            smtpServer = if ($fields[11]) { $fields[11].Trim() } else { "" }
            smtpPort = if ($fields[12]) { $fields[12].Trim() } else { "" }
        }
        $accounts += $account
    }

    $result = @{
        success = $true
        totalAccounts = $accounts.Count
        accounts = $accounts
        rawLinesParsed = $lineCount
        parseErrors = $errorCount
        source = "MailPV v1.93+ (manual copy-paste)"
        timestamp = (Get-Date -Format "o")
    }

    $json = $result | ConvertTo-Json -Depth 3
    $null = New-Item -ItemType Directory -Path (Split-Path $OutputPath -Parent) -Force
    $json | Out-File -FilePath $OutputPath -Encoding UTF8

    $summary = "Saved $($accounts.Count) account(s) to:`n$OutputPath`n`nLines parsed: $lineCount`nErrors: $errorCount"
    [System.Windows.Forms.MessageBox]::Show($summary, "Saved Successfully", "OK", "Information")
    $statusLabel.Text = "Saved $($accounts.Count) accounts — you may close this window."
    $saveBtn.Enabled = $false
    $launchBtn.Enabled = $false
})

$skipBtn.Add_Click({
    $result = @{
        success = $true
        totalAccounts = 0
        accounts = @()
        rawLinesParsed = 0
        parseErrors = 0
        source = "MailPV (skipped by user)"
        timestamp = (Get-Date -Format "o")
    }
    $json = $result | ConvertTo-Json -Depth 3
    $null = New-Item -ItemType Directory -Path (Split-Path $OutputPath -Parent) -Force
    $json | Out-File -FilePath $OutputPath -Encoding UTF8
    $form.Close()
})

$form.Controls.AddRange(@($header, $launchBtn, $pasteBox, $saveBtn, $skipBtn, $statusLabel))
$form.Topmost = $true
$form.ShowDialog() | Out-Null
