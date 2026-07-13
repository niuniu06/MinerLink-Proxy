$token = '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'
$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
    "User-Agent" = "PowerShell-Release-Script"
}

Invoke-RestMethod -Uri "https://api.github.com/repos/niuniu06/MinerLink-Proxy/releases/tags/v2.2.31" -Headers $headers | Tee-Object -Variable release
$releaseId = $release.id

$files = @(
    "MinerLink-Proxy-Windows.zip",
    "MinerLink-Proxy-Linux.zip"
)

foreach ($file in $files) {
    $filePath = "$file"
    $uploadUrl = "https://uploads.github.com/repos/niuniu06/MinerLink-Proxy/releases/$releaseId/assets?name=$file"
    Invoke-RestMethod -Uri $uploadUrl -Method Post -Headers @{ Authorization = "Bearer $token"; Accept = "application/vnd.github.v3+json"; "Content-Type" = "application/octet-stream" } -InFile $filePath
    Write-Host "Uploaded $file"
}
