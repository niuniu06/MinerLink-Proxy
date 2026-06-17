$append = [System.IO.File]::ReadAllText("append.md", [System.Text.Encoding]::UTF8)
$existing = [System.IO.File]::ReadAllText("AI_DEVELOPER_NOTES.md", [System.Text.Encoding]::UTF8)
$combined = $existing + $append
[System.IO.File]::WriteAllText("AI_DEVELOPER_NOTES.md", $combined, (New-Object System.Text.UTF8Encoding($false)))
Write-Host "Appended successfully!"
