#!/bin/bash

# This script runs the database verification in Docker containers

# Set the script to exit on error
set -e

# Print header
echo "====================================="
echo " Theo Database Verification"
echo "====================================="

# Start the Docker containers
echo "Starting Docker containers..."
docker-compose -f docker-compose-db-verify.yml up -d

# Wait for containers to be ready
echo "Waiting for containers to be ready..."
sleep 15

# Run the verification script and follow logs
echo "Running database verification..."
docker-compose -f docker-compose-db-verify.yml logs -f db-verify

# Clean up
echo "Cleaning up..."
docker-compose -f docker-compose-db-verify.yml down

# Print success message
echo "====================================="
echo " Database verification complete!"
echo "====================================="