Write-Host "Compiling Vue 3 Frontend..."
Set-Location frontend
npm run build
Set-Location ..

Write-Host "Copying Frontend Dist to Go UI package..."
if (Test-Path "internal/ui/dist") {
    Remove-Item -Recurse -Force "internal/ui/dist"
}
Copy-Item -Path "frontend/dist" -Destination "internal/ui/dist" -Recurse -Force

Write-Host "Building Go Binary..."
go build -ldflags="-w -s" -o proxy.exe ./cmd/proxy

Write-Host "Build Complete! Check proxy.exe"
