$token = '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'
$repo = "niuniu06/MinerLink-Proxy"
$tag = "v2.2.22"

$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
    "User-Agent" = "PowerShell-Release-Script"
}

$body = @{
    tag_name = $tag
    name = $tag
    body = "Release v2.2.22: Optimized Anti-Probe mechanism. Empty wallet probe connections (e.g., from LB health checks or network scanners) are now 'quarantined' instead of being abruptly disconnected. The proxy replies with a fake success to keep the scanner happy and the TCP connection open, but completely hides it from the UI list and avoids forwarding it to F2Pool. This prevents scanners from erroneously reporting the node as dead."
} | ConvertTo-Json

Write-Host "Creating release $tag..."
$res = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $body -ContentType "application/json"
$uploadUrl = $res.upload_url -replace '\{.*\}$', ''

Write-Host "Upload URL: $uploadUrl"
