#!/bin/bash
set -e

# Change to the project root directory
cd "$(dirname "$0")/.."

echo "Starting storage containers..."
docker-compose -f docker-compose-test.yml up -d aerospike ipfs

echo "Waiting for storage containers to be ready..."
# Wait for Aerospike to be ready
until docker-compose -f docker-compose-test.yml exec -T aerospike asinfo -v "status" > /dev/null 2>&1; do
  echo "Waiting for Aerospike..."
  sleep 2
done
echo "Aerospike is ready!"

# Wait for IPFS to be ready
until docker-compose -f docker-compose-test.yml exec -T ipfs ipfs id > /dev/null 2>&1; do
  echo "Waiting for IPFS..."
  sleep 2
done
echo "IPFS is ready!"

echo "Running integration tests..."
# Run the tests
if [ "$1" == "--docker" ]; then
  # Run tests in Docker
  docker-compose -f docker-compose-test.yml run --rm theo-test
else
  # Run tests locally
  export AEROSPIKE_HOST=$(docker-compose -f docker-compose-test.yml port aerospike 3000 | cut -d: -f1)
  export AEROSPIKE_PORT=$(docker-compose -f docker-compose-test.yml port aerospike 3000 | cut -d: -f2)
  export AEROSPIKE_NAMESPACE=theo
  export IPFS_HOST=$(docker-compose -f docker-compose-test.yml port ipfs 5001 | cut -d: -f1)
  export IPFS_PORT=$(docker-compose -f docker-compose-test.yml port ipfs 5001 | cut -d: -f2)
  export IPFS_PROTOCOL=http
  
  go test ./test -v
fi

# Check if tests passed
if [ $? -eq 0 ]; then
  echo "Integration tests passed!"
else
  echo "Integration tests failed!"
  exit 1
fi

# Clean up if not in CI environment
if [ -z "$CI" ]; then
  echo "Cleaning up containers..."
  docker-compose -f docker-compose-test.yml down -v
fi

echo "Done!"