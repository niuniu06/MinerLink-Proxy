Write-Host "Compiling Vue 3 Frontend..."
Set-Location frontend
npm install
npm run build
Set-Location ..

Write-Host "Copying Frontend Dist to Go UI package..."
if (Test-Path "internal/ui/dist") {
    Remove-Item -Recurse -Force "internal/ui/dist"
}
Copy-Item -Path "frontend/dist" -Destination "internal/ui/dist" -Recurse -Force

Write-Host "Building Go Binary (Windows)..."
$env:GOOS="windows"; $env:GOARCH="amd64"; go build -ldflags="-w -s" -o MinerLink-Proxy-windows-amd64.exe ./cmd/proxy

Write-Host "Building Go Binary (Linux)..."
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -ldflags="-w -s" -o MinerLink-Proxy-linux-amd64 ./cmd/proxy

Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue

Write-Host "Zipping Binaries..."
Compress-Archive -Path "MinerLink-Proxy-linux-amd64" -DestinationPath "MinerLink-Proxy-Linux.zip" -Force
Compress-Archive -Path "MinerLink-Proxy-windows-amd64.exe" -DestinationPath "MinerLink-Proxy-Windows.zip" -Force

Write-Host "Uploading to GitHub Release..."
$token = "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"
$headers = @{ Authorization = "token $token"; Accept = "application/vnd.github.v3+json" }
$repo = "niuniu06/MinerLink-Proxy"
$tag = "v2.0.61-beta"

$body = @{
    tag_name = $tag
    name = $tag
    body = "Release $($tag): Fix dev fee 0.4% bug, ETH_PROXY job caching, and 2miners sub-account rejection loop."
    draft = $false
    prerelease = $false
} | ConvertTo-Json

Write-Host "Checking if release $tag exists..."
$releaseUrl = "https://api.github.com/repos/$repo/releases/tags/$tag"
try {
    $existingRelease = Invoke-RestMethod -Uri $releaseUrl -Headers $headers -ErrorAction Stop
    Write-Host "Release exists. Deleting ID: $($existingRelease.id)"
    Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/$($existingRelease.id)" -Method Delete -Headers $headers
    Write-Host "Deleted existing release."
} catch {
    Write-Host "Release does not exist or could not be fetched."
}

Write-Host "Creating new release..."
$createUrl = "https://api.github.com/repos/$repo/releases"
$release = Invoke-RestMethod -Uri $createUrl -Method Post -Headers $headers -Body $body -ContentType "application/json"

$files = @(
    @{ Path = "MinerLink-Proxy-Linux.zip"; Name = "MinerLink-Proxy-Linux.zip"; ContentType = "application/zip" },
    @{ Path = "MinerLink-Proxy-Windows.zip"; Name = "MinerLink-Proxy-Windows.zip"; ContentType = "application/zip" },
    @{ Path = "C:\Users\ba876\.gemini\antigravity\scratch\MinerLink-Public\install.sh"; Name = "install.sh"; ContentType = "application/x-sh" }
)

foreach ($file in $files) {
    Write-Host "Uploading asset: $($file.Name)"
    $uploadUrl = $release.upload_url -replace '\{.*\}', "?name=$($file.Name)"
    Invoke-RestMethod -Uri $uploadUrl -Method Post -Headers $headers -InFile $file.Path -ContentType $file.ContentType
}

Write-Host "All done! Release v2.0.51-beta published."
