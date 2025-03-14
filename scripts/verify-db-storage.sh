#!/bin/bash

# This script verifies that data is being properly stored and retrieved from Docker storage containers
# It uses SQL for SQLite and AQL for Aerospike

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

cat > ./Dockerfile.db-verify << 'EOF'
FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    curl \
    sqlite3 \
    jq \
    python3 \
    python3-pip \
    iputils-ping \
    net-tools \
    netcat \
    && rm -rf /var/lib/apt/lists/*

# Install Aerospike CLI tools
RUN pip3 install aerospike-cli

# Set working directory
WORKDIR /app

# Default command
CMD ["bash"]
EOF

# Build the verification container
echo "Building verification container..."
docker build -t theo-db-verify -f Dockerfile.db-verify .

# Create a script to run inside the verification container
print_header "Creating verification script"

mkdir -p ./scripts/db-verify
cat > ./scripts/db-verify/verify.sh << 'EOF'
#!/bin/bash

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

# Verify SQLite Storage
print_header "Verifying SQLite Storage"

# Create a test SQLite database
echo "Creating SQLite database..."
sqlite3 /app/test-data/sqlite-verify.db << 'SQLEOF'
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

-- Insert test data
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
SQLEOF

# Verify SQLite database
echo "SQLite database schema:"
sqlite3 /app/test-data/sqlite-verify.db ".schema"

echo "Chain metadata from SQLite:"
sqlite3 /app/test-data/sqlite-verify.db "SELECT * FROM chains;"

echo "Blocks from SQLite:"
sqlite3 /app/test-data/sqlite-verify.db "SELECT id, chain_type, repo_id, block_hash, block_index, timestamp FROM blocks;"

# Read data from SQLite
echo "Reading data from SQLite:"
sqlite3 /app/test-data/sqlite-verify.db "SELECT json_extract(data, '$.message') FROM blocks WHERE chain_type = 'test-chain' AND repo_id = 'test-repo';"

# Verify Aerospike Storage
print_header "Verifying Aerospike Storage"

# Check if Aerospike is reachable
echo "Checking if Aerospike is reachable..."
if ping -c 1 aerospike > /dev/null 2>&1; then
  print_success "Aerospike is reachable"
else
  print_error "Aerospike is not reachable"
  exit 1
fi

# Create Aerospike test data using aql
echo "Creating Aerospike test data..."

# Create a temporary AQL script
cat > /tmp/aerospike-test.aql << 'AQLEOF'
-- Create a set
CREATE SET theo.test;

-- Insert test data
INSERT INTO theo.test (PK, chain_type, repo_id, block_hash, block_index, data)
VALUES ('test-chain:test-repo:block0', 'test-chain', 'test-repo', '0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef', 0, '{"message":"Test data for Aerospike storage","timestamp":1710425600,"index":0}');
AQLEOF

# Run the AQL script
echo "Running AQL script..."
cat /tmp/aerospike-test.aql | aql -h aerospike

# Query Aerospike data
echo "Querying Aerospike data:"
echo "SELECT * FROM theo.test WHERE PK = 'test-chain:test-repo:block0';" | aql -h aerospike

# Verify IPFS Storage
print_header "Verifying IPFS Storage"

# Check if IPFS is reachable
echo "Checking if IPFS is reachable..."
if ping -c 1 ipfs > /dev/null 2>&1; then
  print_success "IPFS is reachable"
else
  print_error "IPFS is not reachable"
  exit 1
fi

# Add test data to IPFS
echo "Adding test data to IPFS..."
echo '{"message":"Test data for IPFS storage","timestamp":1710425600,"index":0}' > /tmp/ipfs-test-data.json

# Use curl to add data to IPFS
echo "Using curl to add data to IPFS..."
IPFS_RESPONSE=$(curl -s -X POST -F file=@/tmp/ipfs-test-data.json "http://ipfs:5001/api/v0/add")
echo "IPFS response: $IPFS_RESPONSE"

# Extract the hash from the response
IPFS_HASH=$(echo "$IPFS_RESPONSE" | jq -r '.Hash')
echo "IPFS hash: $IPFS_HASH"

if [ -n "$IPFS_HASH" ] && [ "$IPFS_HASH" != "null" ]; then
  print_success "Successfully added data to IPFS with hash: $IPFS_HASH"
  
  # Get the data back from IPFS
  echo "Getting data from IPFS:"
  curl -s "http://ipfs:5001/api/v0/cat?arg=$IPFS_HASH"
  
  # Check IPFS gateway
  echo -e "\nIPFS Gateway URL: http://localhost:8080/ipfs/$IPFS_HASH"
  echo "You can open this URL in a browser to verify IPFS is working"
else
  print_error "Failed to add data to IPFS"
fi

print_header "Verification Complete"
echo "Storage verification completed. Check the output above to ensure data was written correctly to all storage backends."
EOF

# Make the verification script executable
chmod +x ./scripts/db-verify/verify.sh

# Run the verification container
print_header "Running verification container"
echo "Running verification container..."

# Create test data directory
mkdir -p test-data

# Run the verification container
docker run --rm \
  --network theo_default \
  -v $(pwd)/test-data:/app/test-data \
  -v $(pwd)/scripts/db-verify:/app/scripts \
  theo-db-verify \
  /app/scripts/verify.sh

print_header "Verification Complete"
echo "Storage verification completed. Check the output above to ensure data was written correctly to all storage backends."