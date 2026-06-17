$token = "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"
$headers = @{
    "Authorization" = "token $token"
    "Accept" = "application/vnd.github.v3+json"
}

# Delete the v2.0.76-beta tag
try {
    $response = Invoke-RestMethod -Uri "https://api.github.com/repos/niuniu06/MinerLink-Proxy/git/refs/tags/v2.0.76-beta" -Method DELETE -Headers $headers -ErrorAction Stop
    Write-Host "Tag deleted successfully."
} catch {
    Write-Host "Failed to delete tag or it doesn't exist: $($_.Exception.Message)"
}

# Revert main branch to 5ea09a7ede09b67a2b5b12ed8e3bca419bc8938b
$body = @{
    sha = "5ea09a7ede09b67a2b5b12ed8e3bca419bc8938b"
    force = $true
} | ConvertTo-Json

try {
    $response = Invoke-RestMethod -Uri "https://api.github.com/repos/niuniu06/MinerLink-Proxy/git/refs/heads/main" -Method PATCH -Headers $headers -Body $body -ErrorAction Stop
    Write-Host "Main branch reverted successfully."
} catch {
    Write-Host "Failed to revert main branch: $($_.Exception.Message)"
}
