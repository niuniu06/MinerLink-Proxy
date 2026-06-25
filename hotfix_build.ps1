Write-Host "Building Go Binary (Windows)..."
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -ldflags="-w -s" -o MinerLink-Proxy-windows-amd64.exe ./cmd/proxy

Write-Host "Building Go Binary (Linux)..."
$env:GOOS="linux"
$env:GOARCH="amd64"
go build -ldflags="-w -s" -o MinerLink-Proxy-linux-amd64 ./cmd/proxy

Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue

Write-Host "Zipping Binaries..."
Compress-Archive -Path "MinerLink-Proxy-linux-amd64" -DestinationPath "MinerLink-Proxy-Linux.zip" -Force
Compress-Archive -Path "MinerLink-Proxy-windows-amd64.exe" -DestinationPath "MinerLink-Proxy-Windows.zip" -Force
