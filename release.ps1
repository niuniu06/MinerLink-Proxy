$token = "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"
$repo = "niuniu06/MinerLink-Proxy"
$tag = "v2.2.19"

$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
    "User-Agent" = "PowerShell-Release-Script"
}

$body = @{
    tag_name = $tag
    name = $tag
    body = "Release v2.2.19: Cleaned up obsolete UI features and natively integrated Silent Auto-Reconnect."
} | ConvertTo-Json

Write-Host "Creating release $tag..."
$res = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $body -ContentType "application/json"
$uploadUrl = $res.upload_url -replace '\{.*\}$', ''

function Upload-Asset($filePath, $contentType) {
    $fileName = Split-Path $filePath -Leaf
    $uri = "$uploadUrl?name=$fileName"
    Write-Host "Uploading $fileName to $uri..."
    Invoke-RestMethod -Uri $uri -Method Post -Headers $headers -InFile $filePath -ContentType $contentType
}

Upload-Asset "MinerLink-Proxy-Linux.zip" "application/zip"
Upload-Asset "MinerLink-Proxy-Windows.zip" "application/zip"
Upload-Asset "install.sh" "text/plain"

Write-Host "All done!"
