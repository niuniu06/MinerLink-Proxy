#!/bin/bash
# build.sh - Cross-platform build script for Go Proxy

echo "Compiling Vue 3 Frontend..."
cd frontend
npm install
npm run build
cd ..

echo "Copying Frontend Dist to Go UI package..."
rm -rf internal/ui/dist
cp -R frontend/dist internal/ui/dist

echo "Building Go Binary for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o proxy-linux-amd64 ./cmd/proxy

echo "Building Go Binary for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o proxy-windows-amd64.exe ./cmd/proxy

echo "Build Complete! Check proxy-linux-amd64 and proxy-windows-amd64.exe"
