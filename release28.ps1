$token = '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'
$repo = "niuniu06/MinerLink-Proxy"
$tag = "v2.2.28"

$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
    "User-Agent" = "PowerShell-Release-Script"
}

$body = @{
    tag_name = $tag
    name = $tag
    body = "Release v2.2.28: Real Stable Sorting. Fixed a script execution issue where the A-Z sorting logic was not correctly embedded in the previous release. The miner list is now definitively sorted by Status (Online first) and then strictly Alphabetical (A-Z) by Worker name. And yes, the 117 reconnections on port 2288 were exactly the proxy intercepting and dumping the scanners!"
} | ConvertTo-Json

Write-Host "Creating release $tag..."
$res = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $body -ContentType "application/json"
$uploadUrl = $res.upload_url -replace '\{.*\}$', ''

Write-Host "Upload URL: $uploadUrl"
