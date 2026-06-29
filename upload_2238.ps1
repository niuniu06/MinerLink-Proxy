Remove-Item MinerLink-Proxy-Windows-Hotfix.exe -ErrorAction SilentlyContinue
Remove-Item MinerLink-Proxy-Linux -ErrorAction SilentlyContinue
Remove-Item MinerLink-Proxy-Windows.zip -ErrorAction SilentlyContinue
Remove-Item MinerLink-Proxy-Linux.zip -ErrorAction SilentlyContinue

="windows"
="amd64"
Write-Host "Building Windows..."
go build -ldflags="-w -s" -o MinerLink-Proxy-Windows-Hotfix.exe ./cmd/proxy
Compress-Archive -Path MinerLink-Proxy-Windows-Hotfix.exe -DestinationPath MinerLink-Proxy-Windows.zip

="linux"
="amd64"
Write-Host "Building Linux..."
go build -ldflags="-w -s" -o MinerLink-Proxy-Linux ./cmd/proxy
tar -czf MinerLink-Proxy-Linux.zip MinerLink-Proxy-Linux

 = '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'
 = 'niuniu06/MinerLink-Proxy'
 = 'v2.2.38'
 = @{
    Authorization = "Bearer "
    Accept = "application/vnd.github.v3+json"
}

# Create Release
 = @{
    tag_name = 
    name = 
    draft = False
    prerelease = False
} | ConvertTo-Json

 = Invoke-RestMethod -Uri "https://api.github.com/repos//releases" -Method Post -Headers  -Body  -ContentType "application/json"
 = .upload_url -replace '\{.*\}', ''

# Upload Windows
Invoke-RestMethod -Uri "?name=MinerLink-Proxy-Windows.zip" -Method Post -Headers  -InFile MinerLink-Proxy-Windows.zip -ContentType "application/zip"
# Upload Linux
Invoke-RestMethod -Uri "?name=MinerLink-Proxy-Linux.zip" -Method Post -Headers  -InFile MinerLink-Proxy-Linux.zip -ContentType "application/zip"

Write-Host "Done!"
