$token = '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'
$repo = 'niuniu06/MinerLink-Proxy'
$tag = 'v2.2.35'
$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
    "User-Agent" = "PowerShell-Release-Script"
}

# Create Release
$body = @{
    tag_name = $tag
    name = "MinerLink Proxy $tag"
    body = "Fix upstream pool difficulty relay bug causing S19 disconnects."
    draft = $false
    prerelease = $false
} | ConvertTo-Json

$releaseRes = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $body
$uploadUrl = $releaseRes.upload_url -replace '\{\?name,label\}',''

# Upload Assets
function Upload-Asset {
    param([string]$path, [string]$name)
    $url = "$uploadUrl?name=$name"
    $bytes = [System.IO.File]::ReadAllBytes($path)
    $uploadHeaders = $headers.Clone()
    $uploadHeaders.Add("Content-Type", "application/octet-stream")
    Invoke-RestMethod -Uri $url -Method Post -Headers $uploadHeaders -Body $bytes
    Write-Host "Uploaded $name"
}

Upload-Asset "C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\MinerLink-Proxy-Linux.zip" "MinerLink-Proxy-Linux.zip"
Upload-Asset "C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\MinerLink-Proxy-Windows.zip" "MinerLink-Proxy-Windows.zip"
Upload-Asset "C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\MinerLink-Proxy-Windows-Hotfix.exe" "MinerLink-Proxy-Windows-Hotfix.exe"
Upload-Asset "C:\Users\ba876\.gemini\antigravity\scratch\go-proxy\install.sh" "install.sh"
