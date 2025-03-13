#!/bin/bash
set -e

# Define variables
DIST_DIR="./dist"
PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Print build information
echo "Building Theo version: $VERSION"
echo "Build date: $BUILD_DATE"
echo "Commit hash: $COMMIT_HASH"

# Create distribution directory
mkdir -p $DIST_DIR

# Build using Docker
echo "Building cross-platform binaries using Docker..."
docker-compose build build
docker-compose run --rm build

# Create checksums
echo "Creating checksums..."
cd $DIST_DIR
for file in theo_*; do
    if [ -f "$file" ]; then
        sha256sum "$file" > "$file.sha256"
    fi
done
cd ..

# Create archives
echo "Creating archives..."
cd $DIST_DIR
for file in theo_*; do
    if [ -f "$file" ] && [[ ! "$file" == *.sha256 ]]; then
        # Skip if archive already exists
        if [ -f "$file.tar.gz" ] || [ -f "$file.zip" ]; then
            continue
        fi
        
        # Create temporary directory
        TEMP_DIR="temp_$file"
        mkdir -p $TEMP_DIR
        
        # Copy binary and documentation
        cp "$file" "$TEMP_DIR/theo"
        cp ../README.md "$TEMP_DIR/"
        cp ../LICENSE "$TEMP_DIR/" 2>/dev/null || echo "No LICENSE file found"
        cp -r ../docs "$TEMP_DIR/" 2>/dev/null || echo "No docs directory found"
        
        # Create archive based on platform
        if [[ "$file" == *"windows"* ]]; then
            # For Windows, create a zip file
            zip -r "$file.zip" "$TEMP_DIR"
            echo "Created $file.zip"
        else
            # For Linux and macOS, create a tar.gz file
            tar -czf "$file.tar.gz" "$TEMP_DIR"
            echo "Created $file.tar.gz"
        fi
        
        # Clean up
        rm -rf "$TEMP_DIR"
    fi
done
cd ..

echo "Build completed successfully!"
echo "Binaries and archives are available in the $DIST_DIR directory"