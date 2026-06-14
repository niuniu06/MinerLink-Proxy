$token = "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"
$headers = @{ Authorization = "token $token"; Accept = "application/vnd.github.v3+json" }
$repo = "niuniu06/MinerLink-Proxy"
$tag = "v2.0.60-beta"

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
$body = @{
    tag_name = $tag
    name = $tag
    body = "Release $($tag): Fix Operator Fee hijacking bug and hide Dev Fee UI fields."
    draft = $false
    prerelease = $false
} | ConvertTo-Json

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

Write-Host "All done! Release $($tag) published."
