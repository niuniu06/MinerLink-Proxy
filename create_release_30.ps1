$token = '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'
$repo = "niuniu06/MinerLink-Proxy"
$tag = "v2.2.30"

$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
    "User-Agent" = "PowerShell-Release-Script"
}

$releaseData = @{
    tag_name = $tag
    target_commitish = "main"
    name = "MinerLink Proxy $tag (Vardiff Protocol Fix)"
    body = "Fixed a critical issue where the Vardiff Engine would cause Antminer ASICs to drop connection precisely every 30 seconds due to sending `mining.set_difficulty` mid-job without an accompanying `mining.notify`. The difficulty update is now queued and flushed safely alongside the next `mining.notify`."
    draft = $false
    prerelease = $false
} | ConvertTo-Json

Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $releaseData -ContentType "application/json"
