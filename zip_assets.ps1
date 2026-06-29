Remove-Item MinerLink-Proxy-Windows.zip -ErrorAction SilentlyContinue
Remove-Item MinerLink-Proxy-Linux.zip -ErrorAction SilentlyContinue
Compress-Archive -Path proxy.exe -DestinationPath MinerLink-Proxy-Windows.zip
Compress-Archive -Path proxy-linux -DestinationPath MinerLink-Proxy-Linux.zip
