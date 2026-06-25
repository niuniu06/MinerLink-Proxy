$token = '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'
$headers = @{
    Authorization = "Bearer $token"
    Accept = 'application/vnd.github.v3+json'
    "User-Agent" = 'PowerShell-Release-Script'
}

Invoke-RestMethod -Uri "https://uploads.github.com/repos/niuniu06/MinerLink-Proxy/releases/344789546/assets?name=MinerLink-Proxy-Linux.zip" -Method Post -Headers $headers -InFile 'MinerLink-Proxy-Linux.zip' -ContentType 'application/zip'
Invoke-RestMethod -Uri "https://uploads.github.com/repos/niuniu06/MinerLink-Proxy/releases/344789546/assets?name=MinerLink-Proxy-Windows.zip" -Method Post -Headers $headers -InFile 'MinerLink-Proxy-Windows.zip' -ContentType 'application/zip'
Invoke-RestMethod -Uri "https://uploads.github.com/repos/niuniu06/MinerLink-Proxy/releases/344789546/assets?name=install.sh" -Method Post -Headers $headers -InFile 'install.sh' -ContentType 'text/plain'
