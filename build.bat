set GOOS=linux 
set GOARCH=amd64 
go build -ldflags=" -w "-s -o MinerLink-Proxy-linux-amd64 ./cmd/proxy 
