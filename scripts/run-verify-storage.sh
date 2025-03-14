#!/bin/bash

# This script runs the verify-storage.go script inside the Docker container

# Start the Docker containers
echo "Starting Docker containers..."
docker-compose -f docker-compose-verify.yml up -d

# Wait for containers to be ready
echo "Waiting for containers to be ready..."
sleep 15

# Run the verification script and follow logs
echo "Running verification script inside Docker container..."
docker-compose -f docker-compose-verify.yml logs -f theo-verify

# Clean up
echo "Cleaning up..."
docker-compose -f docker-compose-verify.yml down

# Print success message
echo "Storage verification complete!"