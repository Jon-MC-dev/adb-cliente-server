#!/bin/bash

# Build script for multi-platform Go client

echo "Building ADB Remote Client for multiple platforms..."

# Create dist directory
mkdir -p dist

# Download dependencies
go mod tidy

# Build for Windows (64-bit)
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o dist/adb-remote-client-windows-amd64.exe main.go

# Build for Linux (64-bit)
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o dist/adb-remote-client-linux-amd64 main.go

# Build for macOS (64-bit Intel)
echo "Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o dist/adb-remote-client-macos-amd64 main.go

# Build for macOS (ARM64 - M1/M2)
echo "Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o dist/adb-remote-client-macos-arm64 main.go

# Build for Linux ARM64
echo "Building for Linux (arm64)..."
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o dist/adb-remote-client-linux-arm64 main.go

echo "Build complete! Binaries are in the 'dist' directory:"
ls -la dist/

echo ""
echo "Usage examples:"
echo "  ./adb-remote-client-linux-amd64"
echo "  ./adb-remote-client-linux-amd64 http://server-ip:5001"