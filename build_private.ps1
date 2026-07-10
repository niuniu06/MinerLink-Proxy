$ErrorActionPreference = "Stop"
Write-Host "Building Private MinerLink-Proxy v2.2.107-beta..."
Copy-Item -Path frontend/src/components/ConfigModal_private.vue -Destination frontend/src/components/ConfigModal.vue -Force
node -e "const fs=require('fs'); fs.writeFileSync('frontend/index.html', fs.readFileSync('frontend/index.html', 'utf8').replace(/<title>.*<\/title>/, '<title>MinerLink-Proxy</title>'));"
node -e "const fs=require('fs'); let c=fs.readFileSync('frontend/src/components/ConfigModal.vue', 'utf8'); c=c.replace(/<option value=\x22DOGE\x22>.*<\/option>\r?\n?\s*/g, ''); fs.writeFileSync('frontend/src/components/ConfigModal.vue', c);"

Set-Location frontend
cmd.exe /c npm run build
Set-Location ..

if (Test-Path internal/ui/dist) { Remove-Item -Recurse -Force internal/ui/dist }
Copy-Item -Path frontend/dist -Destination internal/ui/dist -Recurse -Force

$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -ldflags="-w -s" -o MinerLink-Proxy-linux-amd64 ./cmd/proxy

$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -ldflags="-w -s" -o MinerLink-Proxy-windows-amd64.exe ./cmd/proxy

Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue

Compress-Archive -Path MinerLink-Proxy-linux-amd64 -DestinationPath MinerLink-Proxy-Linux.zip -Force
Compress-Archive -Path MinerLink-Proxy-windows-amd64.exe -DestinationPath MinerLink-Proxy-Windows.zip -Force

Write-Host "Build complete! Uploading via python script..."
git checkout HEAD -- frontend/src/components/ConfigModal.vue frontend/index.html

# python upload_private.py
