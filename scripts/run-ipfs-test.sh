#!/bin/bash

# This script starts a standalone IPFS container and tests basic functionality
# Following the official IPFS Docker documentation: https://docs.ipfs.tech/install/run-ipfs-inside-docker/

# Print header
echo "====================================="
echo " IPFS Standalone Test"
echo "====================================="

# Stop any existing IPFS containers
echo "Stopping any existing IPFS containers..."
docker-compose -f docker-compose-ipfs.yml down

# Start the IPFS container
echo "Starting IPFS container..."
docker-compose -f docker-compose-ipfs.yml up -d ipfs

# Wait for IPFS to start
echo "Waiting for IPFS to initialize and start..."
echo "This may take up to 30 seconds..."
sleep 30

# Check if container is running
echo "Checking if IPFS container is running..."
if ! docker ps | grep -q ipfs-node; then
    echo "❌ IPFS container is not running. Check logs with:"
    echo "docker logs ipfs-node"
    exit 1
fi

echo "✅ IPFS container is running"

# Test IPFS API
echo -e "\nTesting IPFS API..."
echo "Checking IPFS version:"
curl -s -X POST http://localhost:5001/api/v0/version | jq

# Create and add a test file
echo -e "\nCreating test file..."
echo "Hello IPFS from $(date)" > /tmp/ipfs-test.txt

echo "Adding file to IPFS..."
ADD_RESULT=$(curl -s -X POST -F file=@/tmp/ipfs-test.txt http://localhost:5001/api/v0/add)
echo "Add result: $ADD_RESULT"

# Extract hash using jq if available, otherwise use grep
if command -v jq &> /dev/null; then
    HASH=$(echo $ADD_RESULT | jq -r '.Hash')
else
    HASH=$(echo $ADD_RESULT | grep -o '"Hash":"[^"]*"' | grep -o '[^"]*$')
fi

if [ -z "$HASH" ]; then
    echo "❌ Failed to extract hash from IPFS response"
    echo "Raw response: $ADD_RESULT"
    exit 1
fi

echo "File added with hash: $HASH"

# Pin the file to ensure it persists
echo -e "\nPinning file to IPFS..."
PIN_RESULT=$(curl -s -X POST "http://localhost:5001/api/v0/pin/add?arg=$HASH")
echo "Pin result: $PIN_RESULT"

# Initialize MFS if needed
echo -e "\nInitializing IPFS MFS..."
MKDIR_RESULT=$(curl -s -X POST "http://localhost:5001/api/v0/files/mkdir?arg=/test-files&parents=true")

# Add the file to MFS (Mutable File System) to make it visible in the web interface
echo -e "\nAdding file to IPFS MFS (visible in web interface)..."
MFS_RESULT=$(curl -s -X POST "http://localhost:5001/api/v0/files/cp?arg=/ipfs/$HASH&arg=/test-files/ipfs-test.txt")
echo "MFS result: $MFS_RESULT"

# List files in MFS to verify
echo -e "\nListing files in MFS:"
curl -s -X POST "http://localhost:5001/api/v0/files/ls?arg=/test-files" | jq

# Print success message
echo -e "\n====================================="
echo " IPFS test completed!"
echo "====================================="

# Ask if user wants to stop the container
read -p "Do you want to stop the IPFS container? (y/n): " STOP_CONTAINER
if [[ $STOP_CONTAINER == "y" || $STOP_CONTAINER == "Y" ]]; then
    echo "Stopping IPFS container..."
    docker-compose -f docker-compose-ipfs.yml down
    echo "IPFS container stopped."
else
    echo "IPFS container is still running. To stop it later, run:"
    echo "docker-compose -f docker-compose-ipfs.yml down"
fi