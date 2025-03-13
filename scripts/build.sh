#!/bin/bash
set -e

# Create dist directory if it doesn't exist
mkdir -p dist

# Build for Linux
echo "Building for Linux (amd64)..."
GOOS=linux GOARCH=amd64 go build -o dist/theo-linux-amd64 ./cmd/theo

# Build for macOS
echo "Building for macOS (amd64)..."
GOOS=darwin GOARCH=amd64 go build -o dist/theo-darwin-amd64 ./cmd/theo

# Build for macOS ARM (M1/M2)
echo "Building for macOS (arm64)..."
GOOS=darwin GOARCH=arm64 go build -o dist/theo-darwin-arm64 ./cmd/theo

# Build for Windows
echo "Building for Windows (amd64)..."
GOOS=windows GOARCH=amd64 go build -o dist/theo-windows-amd64.exe ./cmd/theo

# Create archives
echo "Creating archives..."
cd dist

# Linux archive
tar -czf theo-linux-amd64.tar.gz theo-linux-amd64
echo "Created theo-linux-amd64.tar.gz"

# macOS archives
tar -czf theo-darwin-amd64.tar.gz theo-darwin-amd64
echo "Created theo-darwin-amd64.tar.gz"
tar -czf theo-darwin-arm64.tar.gz theo-darwin-arm64
echo "Created theo-darwin-arm64.tar.gz"

# Windows archive
if command -v zip >/dev/null 2>&1; then
  zip theo-windows-amd64.zip theo-windows-amd64.exe
  echo "Created theo-windows-amd64.zip"
else
  echo "Warning: zip command not found, skipping Windows archive creation"
fi

echo "Build complete! Binaries and archives are in the dist directory."