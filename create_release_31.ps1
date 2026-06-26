$token = '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'
$repo = "niuniu06/MinerLink-Proxy"
$tag = "v2.2.31"

$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
    "User-Agent" = "PowerShell-Release-Script"
}

$releaseData = @{
    tag_name = $tag
    target_commitish = "main"
    name = "MinerLink Proxy $tag (UI Submits Formatting)"
    body = "Updated the Miner Table UI to format large SUBMITS values using a 'K' suffix (e.g. 3.13K) and perfectly align the '有效' and '无效' rows for improved readability on high-submit clusters."
    draft = $false
    prerelease = $false
} | ConvertTo-Json

Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $releaseData -ContentType "application/json"
