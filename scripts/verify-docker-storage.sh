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

# Create a Dockerfile for the verification container
print_header "Creating verification container"

cat > ./Dockerfile.verify << 'EOF'
FROM golang:1.20-alpine

# Install dependencies
RUN apk add --no-cache git make gcc libc-dev curl jq sqlite

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the rest of the code
COPY . .

# Set environment variables
ENV CGO_ENABLED=1
ENV GO111MODULE=on
ENV AEROSPIKE_HOST=aerospike
ENV AEROSPIKE_PORT=3000
ENV AEROSPIKE_NAMESPACE=theo
ENV IPFS_HOST=ipfs
ENV IPFS_PORT=5001
ENV IPFS_PROTOCOL=http

# Build the verification program
RUN go build -o verify-storage ./cmd/verify-storage/main.go

# Default command
CMD ["./verify-storage"]
EOF

# Create a docker-compose file for verification
cat > ./docker-compose-verify.yml << 'EOF'
version: '3.8'

services:
  verify:
    build:
      context: .
      dockerfile: Dockerfile.verify
    depends_on:
      - aerospike
      - ipfs
    volumes:
      - ./:/app
    environment:
      - AEROSPIKE_HOST=aerospike
      - AEROSPIKE_PORT=3000
      - AEROSPIKE_NAMESPACE=theo
      - IPFS_HOST=ipfs
      - IPFS_PORT=5001
      - IPFS_PROTOCOL=http
    networks:
      - default

  # Aerospike
  aerospike:
    image: aerospike/aerospike-server:latest
    ports:
      - "3000:3000"
      - "3001:3001"
      - "3002:3002"
    volumes:
      - aerospike-data:/opt/aerospike/data
    environment:
      - NAMESPACE=theo
    healthcheck:
      test: ["CMD", "asinfo", "-v", "status"]
      interval: 10s
      timeout: 5s
      retries: 3
    networks:
      - default

  # IPFS
  ipfs:
    image: ipfs/kubo:latest
    ports:
      - "4001:4001"
      - "5001:5001"
      - "8080:8080"
    volumes:
      - ipfs-data:/data/ipfs
    environment:
      - IPFS_PROFILE=server
    healthcheck:
      test: ["CMD", "ipfs", "id"]
      interval: 10s
      timeout: 5s
      retries: 3
    networks:
      - default

  # IndexedDB doesn't need a container as it's browser-based
  # But we can add a simple web server to serve test pages
  indexeddb-test:
    image: nginx:alpine
    ports:
      - "8000:80"
    volumes:
      - ./test/indexeddb:/usr/share/nginx/html
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:80"]
      interval: 10s
      timeout: 5s
      retries: 3
    networks:
      - default

networks:
  default:

volumes:
  aerospike-data:
  ipfs-data:
EOF

# Create the verification program
print_header "Creating verification program"

mkdir -p ./cmd/verify-storage

cat > ./cmd/verify-storage/main.go << 'EOF'
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
	"os"
	"os/exec"
	"strings"

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
	
	// Verify data in Docker containers
	verifyAerospikeDocker()
	verifyIPFSDocker()
}

func verifyFileStorage(ctx context.Context) {
	fmt.Println("\n=== Verifying File Storage ===\n")
	
	config := &storage.StorageConfig{
		Type: storage.StorageTypeFile,
		Path: "test-data/file-storage-verify",
	}
	
	verifyStorage(ctx, config, "file")
	
	// Verify files on disk
	fmt.Println("\nVerifying files on disk:")
	cmd := exec.Command("find", "test-data/file-storage-verify", "-type", "f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to list files: %v\n", err)
		return
	}
	
	fmt.Println(string(output))
	
	// Read a file to verify content
	files := strings.Split(string(output), "\n")
	for _, file := range files {
		if strings.Contains(file, "verify-test:repo-file") && !strings.Contains(file, "cid_mapping") {
			fmt.Printf("Reading file: %s\n", file)
			data, err := os.ReadFile(file)
			if err != nil {
				fmt.Printf("Failed to read file: %v\n", err)
				continue
			}
			fmt.Printf("File content: %s\n", string(data))
			break
		}
	}
}

func verifySQLiteStorage(ctx context.Context) {
	fmt.Println("\n=== Verifying SQLite Storage ===\n")
	
	config := &storage.StorageConfig{
		Type:             storage.StorageTypeSQLite,
		ConnectionString: "test-data/sqlite-storage-verify.db",
	}
	
	verifyStorage(ctx, config, "sqlite")
	
	// Verify SQLite database
	fmt.Println("\nVerifying SQLite database:")
	cmd := exec.Command("ls", "-la", "test-data/sqlite-storage-verify.db")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to check SQLite database: %v\n", err)
		return
	}
	
	fmt.Println(string(output))
	
	// Query SQLite database
	cmd = exec.Command("sqlite3", "test-data/sqlite-storage-verify.db", ".tables")
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to query SQLite database: %v\n", err)
		return
	}
	
	fmt.Printf("SQLite tables: %s\n", string(output))
	
	// Query chains table
	cmd = exec.Command("sqlite3", "test-data/sqlite-storage-verify.db", "SELECT * FROM chains;")
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to query chains table: %v\n", err)
		return
	}
	
	fmt.Printf("Chains table: %s\n", string(output))
	
	// Query blocks table
	cmd = exec.Command("sqlite3", "test-data/sqlite-storage-verify.db", "SELECT id, chain_type, repo_id, block_hash, block_index FROM blocks LIMIT 5;")
	output, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to query blocks table: %v\n", err)
		return
	}
	
	fmt.Printf("Blocks table: %s\n", string(output))
}

func verifyAerospikeStorage(ctx context.Context) {
	fmt.Println("\n=== Verifying Aerospike Storage ===\n")
	
	// Get Aerospike host from environment or use default
	host := os.Getenv("AEROSPIKE_HOST")
	if host == "" {
		host = "localhost"
	}
	
	config := &storage.StorageConfig{
		Type:               storage.StorageTypeAerospike,
		AerospikeHost:      host,
		AerospikePort:      3000,
		AerospikeNamespace: "theo",
		AerospikeSet:       "verify-test",
	}
	
	verifyStorage(ctx, config, "aerospike")
}

func verifyIPFSStorage(ctx context.Context) {
	fmt.Println("\n=== Verifying IPFS Storage ===\n")
	
	// Get IPFS host from environment or use default
	host := os.Getenv("IPFS_HOST")
	if host == "" {
		host = "localhost"
	}
	
	config := &storage.StorageConfig{
		Type:         storage.StorageTypeIPFS,
		IPFSHost:     host,
		IPFSPort:     5001,
		IPFSProtocol: "http",
		Path:         "test-data/ipfs-cache-verify",
	}
	
	verifyStorage(ctx, config, "ipfs")
}

func verifyAerospikeDocker() {
	fmt.Println("\n=== Verifying Aerospike in Docker ===\n")
	
	// Check if we're running in Docker
	inDocker := os.Getenv("AEROSPIKE_HOST") == "aerospike"
	
	if inDocker {
		fmt.Println("Running in Docker, using direct commands")
		
		// Check Aerospike namespace
		cmd := exec.Command("asinfo", "-h", "aerospike", "-v", "namespace/theo")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to check Aerospike namespace: %v\n", err)
			return
		}
		
		fmt.Printf("Aerospike namespace info: %s\n", string(output))
		
		// Check Aerospike sets
		cmd = exec.Command("asinfo", "-h", "aerospike", "-v", "sets/theo")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to check Aerospike sets: %v\n", err)
			return
		}
		
		fmt.Printf("Aerospike sets: %s\n", string(output))
	} else {
		fmt.Println("Not running in Docker, using docker exec commands")
		
		// Check Aerospike namespace
		cmd := exec.Command("docker", "exec", "theo-aerospike-1", "asinfo", "-v", "namespace/theo")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to check Aerospike namespace: %v\n", err)
			return
		}
		
		fmt.Printf("Aerospike namespace info: %s\n", string(output))
		
		// Check Aerospike sets
		cmd = exec.Command("docker", "exec", "theo-aerospike-1", "asinfo", "-v", "sets/theo")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to check Aerospike sets: %v\n", err)
			return
		}
		
		fmt.Printf("Aerospike sets: %s\n", string(output))
	}
}

func verifyIPFSDocker() {
	fmt.Println("\n=== Verifying IPFS in Docker ===\n")
	
	// Check if we're running in Docker
	inDocker := os.Getenv("IPFS_HOST") == "ipfs"
	
	if inDocker {
		fmt.Println("Running in Docker, using direct commands")
		
		// Check IPFS node info
		cmd := exec.Command("ipfs", "--api", "/ip4/ipfs/tcp/5001", "id")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to check IPFS node info: %v\n", err)
			return
		}
		
		fmt.Printf("IPFS node info: %s\n", string(output))
		
		// Check IPFS pins
		cmd = exec.Command("ipfs", "--api", "/ip4/ipfs/tcp/5001", "pin", "ls")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to check IPFS pins: %v\n", err)
			return
		}
		
		fmt.Printf("IPFS pins: %s\n", string(output))
	} else {
		fmt.Println("Not running in Docker, using docker exec commands")
		
		// Check IPFS node info
		cmd := exec.Command("docker", "exec", "theo-ipfs-1", "ipfs", "id")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to check IPFS node info: %v\n", err)
			return
		}
		
		fmt.Printf("IPFS node info: %s\n", string(output))
		
		// Check IPFS pins
		cmd = exec.Command("docker", "exec", "theo-ipfs-1", "ipfs", "pin", "ls")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to check IPFS pins: %v\n", err)
			return
		}
		
		fmt.Printf("IPFS pins: %s\n", string(output))
		
		// Check IPFS gateway
		fmt.Println("\nIPFS Gateway URL: http://localhost:8080/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn")
		fmt.Println("You can open this URL in a browser to verify IPFS is working")
	}
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

# Build and run the verification in Docker
print_header "Running verification in Docker"
echo "Building and running verification container..."
docker-compose -f docker-compose-verify.yml build verify
docker-compose -f docker-compose-verify.yml run --rm verify

# Check IPFS gateway
print_header "Checking IPFS Gateway"
echo "IPFS Gateway URL: http://localhost:8080/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn"
echo "You can open this URL in a browser to verify IPFS is working"

# Check Aerospike
print_header "Checking Aerospike"
echo "Aerospike namespace info:"
docker exec theo-aerospike-1 asinfo -v "namespace/theo"

print_header "Verification Complete"
echo "Storage verification completed. Check the output above to ensure data was written correctly to all storage backends."