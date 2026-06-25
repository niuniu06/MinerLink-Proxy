$token = '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'
$repo = "niuniu06/MinerLink-Proxy"
$tag = "v2.2.27"

$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
    "User-Agent" = "PowerShell-Release-Script"
}

$body = @{
    tag_name = $tag
    name = $tag
    body = "Release v2.2.27: Complete Scanner (Probe) UI Filtering. Scanners hitting directly-exposed Stratum ports (e.g., 10690) no longer appear in the UI as '0-share' workers. All connections are completely invisible in the dashboard until they submit at least 1 valid share, permanently preventing the UI count from increasing and decreasing wildly due to scanner noise."
} | ConvertTo-Json

Write-Host "Creating release $tag..."
$res = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $body -ContentType "application/json"
$uploadUrl = $res.upload_url -replace '\{.*\}$', ''

Write-Host "Upload URL: $uploadUrl"
