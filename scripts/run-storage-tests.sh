#!/bin/bash
set -e

# Change to the project root directory
cd "$(dirname "$0")/.."

# Build the Docker images
echo "Building Docker images..."
docker-compose -f docker-compose-storage.yml build

# Start the services
echo "Starting services..."
docker-compose -f docker-compose-storage.yml up -d

# Wait for services to be ready
echo "Waiting for services to be ready..."
sleep 10

# Run the tests
echo "Running tests..."
docker-compose -f docker-compose-storage.yml run theo go test ./test -v

# Clean up
echo "Cleaning up..."
docker-compose -f docker-compose-storage.yml down

echo "Done!"