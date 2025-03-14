#!/bin/bash

# This script connects to each storage backend, creates test data, and verifies that data is being written correctly

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

# Create a simple test program to verify storage backends
print_header "Creating test program"

cat > ./cmd/verify-storage/main.go << 'EOF'
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gold2th/theo/internal/blockchain/block"
	"github.com/gold2th/theo/internal/storage"
	"github.com/gold2th/theo/internal/storage/factory"
)

func main() {
	ctx := context.Background()
	
	// Create and verify all storage backends
	verifyFileStorage(ctx)
	verifySQLiteStorage(ctx)
	verifyAerospikeStorage(ctx)
	verifyIPFSStorage(ctx)
}

func verifyFileStorage(ctx context.Context) {
	fmt.Println("\n=== Verifying File Storage ===\n")
	
	config := &storage.StorageConfig{
		Type: storage.StorageTypeFile,
		Path: "test-data/file-storage-verify",
	}
	
	verifyStorage(ctx, config, "file")
}

func verifySQLiteStorage(ctx context.Context) {
	fmt.Println("\n=== Verifying SQLite Storage ===\n")
	
	config := &storage.StorageConfig{
		Type:             storage.StorageTypeSQLite,
		ConnectionString: "test-data/sqlite-storage-verify.db",
	}
	
	verifyStorage(ctx, config, "sqlite")
}

func verifyAerospikeStorage(ctx context.Context) {
	fmt.Println("\n=== Verifying Aerospike Storage ===\n")
	
	config := &storage.StorageConfig{
		Type:               storage.StorageTypeAerospike,
		AerospikeHost:      "localhost",
		AerospikePort:      3000,
		AerospikeNamespace: "theo",
		AerospikeSet:       "verify-test",
	}
	
	verifyStorage(ctx, config, "aerospike")
}

func verifyIPFSStorage(ctx context.Context) {
	fmt.Println("\n=== Verifying IPFS Storage ===\n")
	
	config := &storage.StorageConfig{
		Type:         storage.StorageTypeIPFS,
		IPFSHost:     "localhost",
		IPFSPort:     5001,
		IPFSProtocol: "http",
		Path:         "test-data/ipfs-cache-verify",
	}
	
	verifyStorage(ctx, config, "ipfs")
}

func verifyStorage(ctx context.Context, config *storage.StorageConfig, name string) {
	fmt.Printf("Creating and verifying %s storage...\n", name)
	
	// Create storage backend
	s, err := factory.NewStorage(config)
	if err != nil {
		fmt.Printf("Failed to create %s storage: %v\n", name, err)
		return
	}
	
	// Open storage
	err = s.Open(ctx)
	if err != nil {
		fmt.Printf("Failed to open %s storage: %v\n", name, err)
		return
	}
	defer s.Close()
	
	// Create test chain
	chainType := "verify-test"
	repoID := fmt.Sprintf("repo-%s", name)
	
	// Create blocks
	blocks := make([]*block.Block, 0, 5)
	prevHash := ""
	
	for i := 0; i < 5; i++ {
		data := map[string]interface{}{
			"message":   fmt.Sprintf("Test data for block %d in %s storage", i, name),
			"timestamp": time.Now().Unix(),
			"index":     i,
		}
		
		dataBytes, err := json.Marshal(data)
		if err != nil {
			fmt.Printf("Failed to marshal data: %v\n", err)
			return
		}
		
		b := &block.Block{
			Context:      "https://devhub-git.org/contexts/block.jsonld",
			ID:           fmt.Sprintf("chain://%s/%s/block%d", chainType, repoID, i),
			Index:        uint64(i),
			Timestamp:    time.Now().Unix(),
			Data:         dataBytes,
			PreviousHash: prevHash,
		}
		
		// Calculate hash
		hash, err := b.CalculateHash()
		if err != nil {
			fmt.Printf("Failed to calculate hash: %v\n", err)
			return
		}
		b.Hash = hash
		prevHash = hash
		
		blocks = append(blocks, b)
	}
	
	// Save chain
	err = s.SaveChain(ctx, chainType, repoID, blocks)
	if err != nil {
		fmt.Printf("Failed to save chain to %s storage: %v\n", name, err)
		return
	}
	
	// Set IPFS CID
	cid := fmt.Sprintf("QmTest%s123456789", name)
	err = s.SetIPFSCID(ctx, chainType, repoID, cid)
	if err != nil {
		fmt.Printf("Failed to set IPFS CID in %s storage: %v\n", name, err)
		return
	}
	
	// Load chain to verify
	loadedBlocks, err := s.LoadChain(ctx, chainType, repoID)
	if err != nil {
		fmt.Printf("Failed to load chain from %s storage: %v\n", name, err)
		return
	}
	
	// Print loaded blocks
	fmt.Printf("Successfully loaded %d blocks from %s storage\n", len(loadedBlocks), name)
	for i, b := range loadedBlocks {
		fmt.Printf("Block %d:\n", i)
		fmt.Printf("  Hash: %s\n", b.Hash)
		fmt.Printf("  Previous Hash: %s\n", b.PreviousHash)
		fmt.Printf("  Index: %d\n", b.Index)
		fmt.Printf("  Timestamp: %d\n", b.Timestamp)
		
		var data map[string]interface{}
		if err := json.Unmarshal(b.Data, &data); err != nil {
			fmt.Printf("  Data: %s\n", string(b.Data))
		} else {
			fmt.Printf("  Data: %v\n", data)
		}
		fmt.Println()
	}
	
	// Get IPFS CID
	loadedCID, err := s.GetIPFSCID(ctx, chainType, repoID)
	if err != nil {
		fmt.Printf("Failed to get IPFS CID from %s storage: %v\n", name, err)
		return
	}
	
	fmt.Printf("IPFS CID: %s\n", loadedCID)
	
	fmt.Printf("Successfully created and verified test data for %s storage\n", name)
}
EOF

# Create directory for verify-storage
mkdir -p ./cmd/verify-storage

# Build the verify-storage program
echo "Building verify-storage program..."
go build -o ./cmd/verify-storage/verify-storage ./cmd/verify-storage/main.go

# Run the verify-storage program
print_header "Running verification"
./cmd/verify-storage/verify-storage

print_header "Verification Complete"
echo "Storage verification completed. Check the output above to ensure data was written correctly to all storage backends."