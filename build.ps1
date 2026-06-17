Write-Host "Compiling Vue 3 Frontend..."
Set-Location frontend
npm run build
Set-Location ..

Write-Host "Copying Frontend Dist to Go UI package..."
if (Test-Path "internal/ui/dist") {
    Remove-Item -Recurse -Force "internal/ui/dist"
}
Copy-Item -Path "frontend/dist" -Destination "internal/ui/dist" -Recurse -Force

Write-Host "Building Tunnel Clients..."
if (!(Test-Path "internal/api/downloads")) {
    New-Item -ItemType Directory -Force -Path "internal/api/downloads" | Out-Null
}
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -ldflags="-w -s" -o internal/api/downloads/local-tunnel-linux-amd64 ./cmd/local
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -ldflags="-w -s" -o internal/api/downloads/local-tunnel-windows-amd64.exe ./cmd/local
Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue

Write-Host "Building Go Binary..."
go build -ldflags="-w -s" -o proxy.exe ./cmd/proxy

Write-Host "Build Complete! Check proxy.exe"
