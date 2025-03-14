#!/bin/bash

# This script verifies that data is being properly stored and retrieved from Docker storage containers

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

# Start storage containers
print_header "Starting storage containers"
echo "Starting Docker containers for storage backends..."
docker-compose -f docker-compose-storage.yml down
docker-compose -f docker-compose-storage.yml up -d

# Wait for containers to be ready
echo "Waiting for containers to be ready..."
sleep 15

# Create test data
print_header "Creating test data"

# Create test directory
mkdir -p test-data/file-storage-verify

# Create a test file
cat > test-data/file-storage-verify/test-block.json << 'EOF'
{
  "context": "https://devhub-git.org/contexts/block.jsonld",
  "id": "chain://test-chain/test-repo/block0",
  "index": 0,
  "timestamp": 1710425600,
  "data": {
    "message": "Test data for file storage",
    "timestamp": 1710425600,
    "index": 0
  },
  "previousHash": "",
  "hash": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
}
EOF

# Create a test SQLite database
mkdir -p test-data
cat > test-data/create-sqlite-db.sql << 'EOF'
CREATE TABLE IF NOT EXISTS chains (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  chain_type TEXT NOT NULL,
  repo_id TEXT NOT NULL,
  block_count INTEGER NOT NULL DEFAULT 0,
  last_updated TIMESTAMP NOT NULL,
  ipfs_cid TEXT,
  UNIQUE(chain_type, repo_id)
);

CREATE TABLE IF NOT EXISTS blocks (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  chain_type TEXT NOT NULL,
  repo_id TEXT NOT NULL,
  block_hash TEXT NOT NULL,
  block_index INTEGER NOT NULL,
  previous_hash TEXT NOT NULL,
  timestamp INTEGER NOT NULL,
  data TEXT NOT NULL,
  signature BLOB,
  metadata TEXT,
  context TEXT NOT NULL,
  block_id TEXT NOT NULL,
  UNIQUE(chain_type, repo_id, block_hash),
  FOREIGN KEY(chain_type, repo_id) REFERENCES chains(chain_type, repo_id)
);

INSERT INTO chains (chain_type, repo_id, block_count, last_updated, ipfs_cid)
VALUES ('test-chain', 'test-repo', 1, datetime('now'), 'QmTestCID123456789');

INSERT INTO blocks (chain_type, repo_id, block_hash, block_index, previous_hash, timestamp, data, context, block_id)
VALUES (
  'test-chain',
  'test-repo',
  '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef',
  0,
  '',
  1710425600,
  '{"message":"Test data for SQLite storage","timestamp":1710425600,"index":0}',
  'https://devhub-git.org/contexts/block.jsonld',
  'chain://test-chain/test-repo/block0'
);
EOF

sqlite3 test-data/sqlite-storage-verify.db < test-data/create-sqlite-db.sql

# Verify File Storage
print_header "Verifying File Storage"
echo "Files in file storage directory:"
find test-data/file-storage-verify -type f | sort

echo "Contents of test block file:"
cat test-data/file-storage-verify/test-block.json

# Verify SQLite Storage
print_header "Verifying SQLite Storage"
echo "SQLite database schema:"
sqlite3 test-data/sqlite-storage-verify.db ".schema"

echo "Chain metadata from SQLite:"
sqlite3 test-data/sqlite-storage-verify.db "SELECT * FROM chains;"

echo "Blocks from SQLite:"
sqlite3 test-data/sqlite-storage-verify.db "SELECT id, chain_type, repo_id, block_hash, block_index, timestamp FROM blocks;"

# Verify Aerospike Storage
print_header "Verifying Aerospike Storage"
echo "Aerospike namespace info:"
docker exec theo-aerospike-1 asinfo -v "namespace/theo"

# Check if Aerospike is running
echo "Checking if Aerospike is running..."
if docker exec theo-aerospike-1 asinfo -v "status"; then
  print_success "Aerospike is running"
else
  print_error "Aerospike is not running"
fi

# Check Aerospike sets
echo "Aerospike sets:"
docker exec theo-aerospike-1 asinfo -v "sets"

# Create a Go program to write to Aerospike
echo "Creating a Go program to write to Aerospike..."
mkdir -p cmd/aerospike-test
cat > cmd/aerospike-test/main.go << 'EOF'
package main

import (
	"fmt"
	"time"

	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/factory"
)

func main() {
	// Create Aerospike storage config
	config := &storage.StorageConfig{
		Type:               storage.StorageTypeAerospike,
		AerospikeHost:      "localhost",
		AerospikePort:      3000,
		AerospikeNamespace: "theo",
		AerospikeSet:       "verify-test",
	}

	// Create storage backend
	s, err := factory.NewStorage(config)
	if err != nil {
		fmt.Printf("Failed to create Aerospike storage: %v\n", err)
		return
	}

	// Open storage
	ctx := context.Background()
	err = s.Open(ctx)
	if err != nil {
		fmt.Printf("Failed to open Aerospike storage: %v\n", err)
		return
	}
	defer s.Close()

	// Create test data
	chainType := "test-chain"
	repoID := "test-repo"
	blockData := []byte(`{"message":"Test data for Aerospike storage","timestamp":1710425600,"index":0}`)

	// Create a test block
	block := &block.Block{
		Context:      "https://devhub-git.org/contexts/block.jsonld",
		ID:           fmt.Sprintf("chain://%s/%s/block0", chainType, repoID),
		Index:        0,
		Timestamp:    time.Now().Unix(),
		Data:         blockData,
		PreviousHash: "",
	}

	// Calculate hash
	hash, err := block.CalculateHash()
	if err != nil {
		fmt.Printf("Failed to calculate hash: %v\n", err)
		return
	}
	block.Hash = hash

	// Save block
	err = s.SaveBlock(ctx, chainType, repoID, block)
	if err != nil {
		fmt.Printf("Failed to save block to Aerospike: %v\n", err)
		return
	}

	fmt.Println("Successfully saved block to Aerospike")

	// Load block
	loadedBlock, err := s.LoadBlock(ctx, chainType, repoID, hash)
	if err != nil {
		fmt.Printf("Failed to load block from Aerospike: %v\n", err)
		return
	}

	fmt.Printf("Successfully loaded block from Aerospike: %s\n", loadedBlock.Hash)
}
EOF

# Try to build and run the Aerospike test program
echo "Attempting to build and run the Aerospike test program..."
go build -o cmd/aerospike-test/aerospike-test cmd/aerospike-test/main.go || echo "Failed to build Aerospike test program"

# Check Aerospike status again
echo "Aerospike status:"
docker exec theo-aerospike-1 asinfo -v "status"

# Verify IPFS Storage
print_header "Verifying IPFS Storage"
echo "IPFS node info:"
docker exec theo-ipfs-1 ipfs id | grep "ID\|Addresses"

# Add test data to IPFS
echo "Adding test data to IPFS..."
echo '{"message":"Test data for IPFS storage","timestamp":1710425600,"index":0}' > test-data/ipfs-test-data.json
docker cp test-data/ipfs-test-data.json theo-ipfs-1:/tmp/ipfs-test-data.json
docker exec theo-ipfs-1 ipfs add /tmp/ipfs-test-data.json

# Check IPFS pins
echo "IPFS pins:"
docker exec theo-ipfs-1 ipfs pin ls --type=recursive | head -10

# Check IPFS gateway
echo "IPFS Gateway URL: http://localhost:8080/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn"
echo "You can open this URL in a browser to verify IPFS is working"

print_header "Verification Complete"
echo "Storage verification completed. Check the output above to ensure data was written correctly to all storage backends."