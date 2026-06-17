$text = [System.IO.File]::ReadAllText("install.sh", [System.Text.Encoding]::UTF8)
$text = $text.Replace("`r`n", "`n")
[System.IO.File]::WriteAllText("install.sh", $text, (New-Object System.Text.UTF8Encoding($false)))
Write-Host "Fixed CRLF with UTF-8 No BOM"
