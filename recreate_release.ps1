$token = "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"
$repo = "niuniu06/MinerLink-Proxy"
$tag = "v2.2.20"

$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
    "User-Agent" = "PowerShell-Release-Script"
}

$body = @{
    tag_name = $tag
    name = $tag
    body = "Release v2.2.20: Cleaned up obsolete UI features and natively integrated Silent Auto-Reconnect. No more dropped connections during fee shifts. (Rebuilt with fixed version badge)"
} | ConvertTo-Json

Write-Host "Creating release $tag..."
$res = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $body -ContentType "application/json"
$uploadUrl = $res.upload_url -replace '\{.*\}$', ''

Write-Host "Upload URL: $uploadUrl"

Invoke-RestMethod -Uri "$uploadUrl?name=MinerLink-Proxy-Linux.zip" -Method Post -Headers $headers -InFile "MinerLink-Proxy-Linux.zip" -ContentType "application/zip"
Invoke-RestMethod -Uri "$uploadUrl?name=MinerLink-Proxy-Windows.zip" -Method Post -Headers $headers -InFile "MinerLink-Proxy-Windows.zip" -ContentType "application/zip"
Invoke-RestMethod -Uri "$uploadUrl?name=install.sh" -Method Post -Headers $headers -InFile "install.sh" -ContentType "text/plain"

Write-Host "All done!"
