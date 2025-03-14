#!/bin/bash

# This script connects to each storage backend and verifies that data is being written correctly

# Set up error handling
set -e
trap 'echo "Error: Command failed with exit code $?"; exit 1' ERR

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print section header
print_header() {
  echo -e "\n${BLUE}=== $1 ===${NC}\n"
}

# Function to print success message
print_success() {
  echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error message
print_error() {
  echo -e "${RED}✗ $1${NC}"
}

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
  print_error "Docker is not running. Please start Docker and try again."
  exit 1
fi

# Check if storage containers are running
print_header "Checking if storage containers are running"
AEROSPIKE_RUNNING=$(docker ps | grep aerospike | wc -l)
IPFS_RUNNING=$(docker ps | grep ipfs | wc -l)

if [ $AEROSPIKE_RUNNING -eq 0 ] || [ $IPFS_RUNNING -eq 0 ]; then
  echo "Starting storage containers..."
  docker-compose -f docker-compose-storage.yml up -d
  sleep 10
else
  print_success "Storage containers are already running"
fi

# Verify File Storage
print_header "Verifying File Storage"
if [ -d "test-data" ]; then
  print_success "File storage directory exists"
  
  # List files in the directory
  echo "Files in test-data directory:"
  find test-data -type f | grep -v "\.git" | sort
else
  print_error "File storage directory does not exist"
fi

# Verify SQLite Storage
print_header "Verifying SQLite Storage"
SQLITE_FILES=$(find test-data -name "*.db" | wc -l)
if [ $SQLITE_FILES -gt 0 ]; then
  print_success "SQLite database files exist"
  
  # List SQLite files
  echo "SQLite database files:"
  find test-data -name "*.db"
else
  print_error "No SQLite database files found"
fi

# Verify Aerospike Storage
print_header "Verifying Aerospike Storage"
# Check if Aerospike is running
if [ $AEROSPIKE_RUNNING -gt 0 ]; then
  print_success "Aerospike is running"
  
  # Check Aerospike namespace info
  echo "Aerospike namespace info:"
  docker exec theo-aerospike-1 asinfo -v "namespace/theo"
else
  print_error "Aerospike is not running"
fi

# Verify IPFS Storage
print_header "Verifying IPFS Storage"
# Check if IPFS is running
if [ $IPFS_RUNNING -gt 0 ]; then
  print_success "IPFS is running"
  
  # Check IPFS node info
  echo "IPFS node info:"
  docker exec theo-ipfs-1 ipfs id | grep "ID\|Addresses"
  
  # Check IPFS pins
  echo "IPFS pins (first 10):"
  docker exec theo-ipfs-1 ipfs pin ls --type=recursive | head -10
else
  print_error "IPFS is not running"
fi

# Run the blockchain integration test to create test data
print_header "Running blockchain integration test to create test data"
echo "This will create test data in all storage backends..."
go test ./test -run TestBlockchainIntegration -v

print_header "Verification Complete"
echo "Storage verification completed. Check the output above to ensure data was written correctly to all storage backends."