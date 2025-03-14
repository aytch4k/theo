#!/bin/bash

# This script starts a standalone IPFS container and tests basic functionality
# Following the official IPFS Docker documentation: https://docs.ipfs.tech/install/run-ipfs-inside-docker/

# Set the script to exit on error
set -e

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
echo "This may take up to 60 seconds..."
sleep 60

# Check if container is running
echo "Checking if IPFS container is running..."
if ! docker ps | grep -q ipfs-node; then
    echo "❌ IPFS container is not running. Check logs with:"
    echo "docker logs ipfs-node"
    exit 1
fi

echo "✅ IPFS container is running"

# Test IPFS API with retries
echo -e "\nTesting IPFS API..."
MAX_RETRIES=5
RETRY_COUNT=0
API_READY=false

while [ $RETRY_COUNT -lt $MAX_RETRIES ] && [ "$API_READY" = false ]; do
    echo "Attempt $(($RETRY_COUNT+1))/$MAX_RETRIES: Checking IPFS version..."
    VERSION_RESULT=$(curl -s -X POST http://localhost:5001/api/v0/version)
    
    if [[ $VERSION_RESULT == *"Version"* ]]; then
        echo "✅ IPFS API is ready"
        echo "$VERSION_RESULT" | jq
        API_READY=true
    else
        echo "⏳ IPFS API not ready yet, waiting 10 seconds..."
        RETRY_COUNT=$((RETRY_COUNT+1))
        sleep 10
    fi
done

if [ "$API_READY" = false ]; then
    echo "❌ IPFS API did not become ready after $MAX_RETRIES attempts"
    echo "Check the container logs with: docker logs ipfs-node"
    exit 1
fi

# Create and add a test file
echo -e "\nCreating test file..."
TEST_CONTENT="Hello IPFS from $(date)"
echo "$TEST_CONTENT" > /tmp/ipfs-test.txt

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

# Retrieve the file
echo -e "\nRetrieving file from IPFS API..."
CAT_RESULT=$(curl -s "http://localhost:5001/api/v0/cat?arg=$HASH")
echo "$CAT_RESULT"

# Verify content matches
if [ "$CAT_RESULT" = "$TEST_CONTENT" ]; then
    echo "✅ Content verification successful"
else
    echo "❌ Content verification failed"
    echo "Expected: $TEST_CONTENT"
    echo "Got: $CAT_RESULT"
fi

echo -e "\nRetrieving file from IPFS Gateway..."
GATEWAY_RESULT=$(curl -s "http://localhost:8080/ipfs/$HASH")
echo "$GATEWAY_RESULT"

# List IPFS peers
echo -e "\nListing IPFS peers:"
curl -s -X POST http://localhost:5001/api/v0/swarm/peers | jq

# Print node info
echo -e "\nIPFS node info:"
curl -s -X POST http://localhost:5001/api/v0/id | jq

# Print success message
echo -e "\n====================================="
echo " IPFS test completed successfully!"
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