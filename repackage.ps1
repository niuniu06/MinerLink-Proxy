Remove-Item MinerLink-Proxy-Windows-Hotfix.exe
Remove-Item MinerLink-Proxy-Linux
Remove-Item MinerLink-Proxy-Windows.zip
Remove-Item MinerLink-Proxy-Linux.zip

="windows"
="amd64"
Write-Host "Building Windows..."
go build -ldflags="-w -s" -o MinerLink-Proxy-Windows-Hotfix.exe ./cmd/proxy
Compress-Archive -Path MinerLink-Proxy-Windows-Hotfix.exe -DestinationPath MinerLink-Proxy-Windows.zip -Force

="linux"
="amd64"
Write-Host "Building Linux..."
go build -ldflags="-w -s" -o MinerLink-Proxy-Linux ./cmd/proxy
Compress-Archive -Path MinerLink-Proxy-Linux -DestinationPath MinerLink-Proxy-Linux.zip -Force

$token = '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'
$repo = 'niuniu06/MinerLink-Proxy'
$tag = 'v2.2.36'
$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
}

Write-Host "Deleting old release..."
Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/345437248" -Method Delete -Headers $headers

# Create Release
$body = @{
    tag_name = $tag
    name = "Release $tag"
    body = "Fix critical issue with initial difficulty assignment for new miner connections."
    draft = $false
    prerelease = $false
} | ConvertTo-Json

Write-Host "Creating new release..."
$res = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $body -ContentType "application/json"
$releaseId = $res.id
Write-Host "Release created with ID $releaseId"

# Upload assets
$uploadUrl = "https://uploads.github.com/repos/$repo/releases/$releaseId/assets?name="

Write-Host "Uploading MinerLink-Proxy-Windows.zip..."
Invoke-RestMethod -Uri ($uploadUrl + "MinerLink-Proxy-Windows.zip") -Method Post -Headers $headers -ContentType "application/zip" -InFile "MinerLink-Proxy-Windows.zip"

Write-Host "Uploading MinerLink-Proxy-Linux.zip..."
Invoke-RestMethod -Uri ($uploadUrl + "MinerLink-Proxy-Linux.zip") -Method Post -Headers $headers -ContentType "application/zip" -InFile "MinerLink-Proxy-Linux.zip"

Write-Host "Done!"
