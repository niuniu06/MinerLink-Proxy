$token = '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'
$repo = "niuniu06/MinerLink-Proxy"
$tag = "v2.2.23"

$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
    "User-Agent" = "PowerShell-Release-Script"
}

$body = @{
    tag_name = $tag
    name = $tag
    body = "Release v2.2.23: True Stealth Fee Logs. All internal fee routing mechanics, connections, and fee share acceptances are now completely invisible in the web UI's Miner Logs. They are strictly confined to the global backend CLI console to ensure miners never see the fee mechanics on their individual log pages."
} | ConvertTo-Json

Write-Host "Creating release $tag..."
$res = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $body -ContentType "application/json"
$uploadUrl = $res.upload_url -replace '\{.*\}$', ''

Write-Host "Upload URL: $uploadUrl"
