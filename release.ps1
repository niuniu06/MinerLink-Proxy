param (
    [Parameter(Mandatory=$true)]
    [string]$Token,
    
    [Parameter(Mandatory=$true)]
    [string]$Tag,
    
    [Parameter(Mandatory=$true)]
    [string]$Body
)

$repo = 'niuniu06/MinerLink-Proxy'
$headers = @{
    Authorization = "Bearer $Token"
    Accept = "application/vnd.github.v3+json"
    "User-Agent" = "PowerShell-Release-Script"
}

$jsonBody = @{
    tag_name = $Tag
    name = "MinerLink Proxy $Tag"
    body = $Body
} | ConvertTo-Json

Write-Host "Creating release $Tag..."
$res = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases" -Method Post -Headers $headers -Body $jsonBody -ContentType "application/json"
$releaseId = $res.id
Write-Host "Release ID: $releaseId"

function Upload-Asset($filePath, $contentType) {
    $fileName = Split-Path $filePath -Leaf
    $uploadUrl = "https://uploads.github.com/repos/$repo/releases/$releaseId/assets?name=$fileName"
    Write-Host "Uploading $fileName..."
    Invoke-RestMethod -Uri $uploadUrl -Method Post -Headers @{ Authorization = "Bearer $Token"; Accept = "application/vnd.github.v3+json"; "Content-Type" = $contentType } -InFile $filePath
}

Upload-Asset "MinerLink-Proxy-Linux.zip" "application/zip"
Upload-Asset "MinerLink-Proxy-Windows.zip" "application/zip"
Write-Host "All done!"
