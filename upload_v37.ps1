Remove-Item MinerLink-Proxy-Windows-Hotfix.exe
Remove-Item MinerLink-Proxy-Linux
Remove-Item MinerLink-Proxy-Windows.zip
Remove-Item MinerLink-Proxy-Linux.zip

$env:GOOS="windows"
$env:GOARCH="amd64"
go build -ldflags="-w -s" -o MinerLink-Proxy-Windows-Hotfix.exe ./cmd/proxy
Compress-Archive -Path MinerLink-Proxy-Windows-Hotfix.exe -DestinationPath MinerLink-Proxy-Windows.zip

$env:GOOS="linux"
$env:GOARCH="amd64"
go build -ldflags="-w -s" -o MinerLink-Proxy-Linux ./cmd/proxy
Compress-Archive -Path MinerLink-Proxy-Linux -DestinationPath MinerLink-Proxy-Linux.zip

$token = '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'
$repo = 'niuniu06/MinerLink-Proxy'
$tag = 'v2.2.37'
$headers = @{
    Authorization = "Bearer $token"
    Accept = "application/vnd.github.v3+json"
}

# Create Release
$body = @{
    tag_name = $tag
    name = "Release $tag"
    body = "Fix high-hash rejections for low hashrate miners by clamping Vardiff."
    draft = $false
    prerelease = $false
} | ConvertTo-Json

$res = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $body -ContentType "application/json"
$releaseId = $res.id

# Upload assets
$uploadUrl = "https://uploads.github.com/repos/$repo/releases/$releaseId/assets?name="
Invoke-RestMethod -Uri ($uploadUrl + "MinerLink-Proxy-Windows.zip") -Method Post -Headers $headers -ContentType "application/zip" -InFile "MinerLink-Proxy-Windows.zip"
Invoke-RestMethod -Uri ($uploadUrl + "MinerLink-Proxy-Linux.zip") -Method Post -Headers $headers -ContentType "application/zip" -InFile "MinerLink-Proxy-Linux.zip"
Write-Host "Done v2.2.37!"
