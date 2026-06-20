$token = "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"
$owner = "niuniu06"
$repo = "MinerLink-Proxy"
$tag = "v2.1.5-beta"
$commitish = "main"

$headers = @{
    "Authorization" = "token $token"
    "Accept" = "application/vnd.github.v3+json"
}

# 0. Try to get and delete existing release
try {
    $existingRelease = Invoke-RestMethod -Uri "https://api.github.com/repos/$owner/$repo/releases/tags/$tag" -Headers $headers -ErrorAction Stop
    Write-Host "Found existing release, deleting it..."
    Invoke-RestMethod -Uri "https://api.github.com/repos/$owner/$repo/releases/$($existingRelease.id)" -Method Delete -Headers $headers
    
    Write-Host "Deleting existing tag..."
    Invoke-RestMethod -Uri "https://api.github.com/repos/$owner/$repo/git/refs/tags/$tag" -Method Delete -Headers $headers
} catch {
    Write-Host "No existing release found or failed to delete."
}

# 1. Create Release
$releaseBody = @{
    tag_name = $tag
    target_commitish = $commitish
    name = $tag
    body = "Release v2.1.0-beta: [MAJOR UPDATE] Implemented Stateless Distributed Rotation Scheduler for perfectly smooth, visual-lossless fee routing. Fees are now dynamically distributed across active miners per account, guaranteeing exact 2% withdrawal while mathematically eliminating simultaneous miner restarts."
    draft = $false
    prerelease = $false
} | ConvertTo-Json

Write-Host "Creating release $tag..."
$releaseUrl = "https://api.github.com/repos/$owner/$repo/releases"
$releaseResponse = Invoke-RestMethod -Uri $releaseUrl -Method Post -Headers $headers -Body $releaseBody

if (-not $releaseResponse.id) {
    Write-Host "Failed to create release!"
    exit 1
}

$releaseId = $releaseResponse.id
Write-Host "Release created with ID: $releaseId"

# 2. Upload Assets
$uploadUrlBase = "https://uploads.github.com/repos/$owner/$repo/releases/$releaseId/assets?name="

function Upload-Asset {
    param(
        [string]$filePath,
        [string]$contentType
    )
    $fileName = [System.IO.Path]::GetFileName($filePath)
    $url = $uploadUrlBase + $fileName
    Write-Host "Uploading $fileName..."
    
    $fileBytes = [System.IO.File]::ReadAllBytes($filePath)
    $uploadHeaders = @{
        "Authorization" = "token $token"
        "Accept" = "application/vnd.github.v3+json"
        "Content-Type" = $contentType
    }
    
    try {
        Invoke-RestMethod -Uri $url -Method Post -Headers $uploadHeaders -Body $fileBytes
        Write-Host "Successfully uploaded ${fileName}"
    } catch {
        Write-Host "Failed to upload ${fileName}: $($_.Exception.Message)"
    }
}

Upload-Asset -filePath "MinerLink-Proxy-Linux.zip" -contentType "application/zip"
Upload-Asset -filePath "MinerLink-Proxy-Windows.zip" -contentType "application/zip"
Upload-Asset -filePath "install.sh" -contentType "text/x-sh"

Write-Host "All done!"
