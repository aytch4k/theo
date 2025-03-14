#!/bin/bash

# This script runs all tests and verifies the storage backends

# Set up error handling
set -e
trap 'echo "Error: Command failed with exit code $?"; exit 1' ERR

# Start Docker containers for storage backends
echo "Starting Docker containers for storage backends..."
docker-compose -f docker-compose-storage.yml up -d

# Wait for containers to be ready
echo "Waiting for containers to be ready..."
sleep 15

# Run unit tests
echo "Running unit tests..."
go test ./internal/... -v

# Run integration tests
echo "Running integration tests..."
go test ./test -v

# Run storage verification
echo "Running storage verification..."
go run ./scripts/verify-storage.go

# Run blockchain validation tests
echo "Running blockchain validation tests..."
go test ./test -run TestBlockchainIntegration -v

# Clean up
echo "Cleaning up..."
docker-compose -f docker-compose-storage.yml down

echo "All tests completed successfully!"