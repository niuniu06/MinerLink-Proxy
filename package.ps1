="windows"
="amd64"
Write-Host "Building Windows..."
go build -ldflags="-w -s" -o MinerLink-Proxy-Windows-Hotfix.exe ./cmd/proxy
if (Test-Path MinerLink-Proxy-Windows.zip) { Remove-Item MinerLink-Proxy-Windows.zip }
Compress-Archive -Path MinerLink-Proxy-Windows-Hotfix.exe -DestinationPath MinerLink-Proxy-Windows.zip -Force

="linux"
="amd64"
Write-Host "Building Linux..."
go build -ldflags="-w -s" -o MinerLink-Proxy-Linux ./cmd/proxy
if (Test-Path MinerLink-Proxy-Linux.zip) { Remove-Item MinerLink-Proxy-Linux.zip }
Compress-Archive -Path MinerLink-Proxy-Linux -DestinationPath MinerLink-Proxy-Linux.zip -Force

Write-Host "Done packaging v2.2.36."
