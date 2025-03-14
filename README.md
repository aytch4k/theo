# Theo - Blockchain Storage and Validation System

Theo is a blockchain-based storage and validation system designed to provide secure, distributed storage for various types of data. It supports multiple storage backends and includes comprehensive blockchain validation and synchronization capabilities.

## Features

- Multiple storage backends:
  - File-based storage
  - SQLite storage
  - Aerospike storage
  - IPFS storage
  - IndexedDB storage (for browser environments)
- Blockchain validation and verification
- Chain synchronization with conflict resolution
- Support for linked JSON data
- Comprehensive test suite

## Getting Started

### Prerequisites

- Go 1.19 or later
- Docker and Docker Compose (for running storage backends)

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/gold2th/theo.git
   cd theo
   ```

2. Build the project:
   ```
   make build
   ```

### Running Tests

To run all tests and verify the storage backends:

```
./scripts/run-all-tests.sh
```

This script will:
1. Start Docker containers for the storage backends
2. Run unit tests
3. Run integration tests
4. Run storage verification
5. Run blockchain validation tests
6. Clean up Docker containers

### Running Storage Verification

To verify the storage backends:

```
./scripts/run-verify-storage.sh
```

## Storage Backends

### File Storage

File storage saves blockchain data to the local filesystem. It's suitable for development and testing.

### SQLite Storage

SQLite storage uses an embedded SQLite database to store blockchain data. It's suitable for applications that need a lightweight, self-contained database.

### Aerospike Storage

Aerospike storage uses an Aerospike database for high-performance, distributed storage. It's suitable for applications that need high throughput and low latency.

### IPFS Storage

IPFS storage uses the InterPlanetary File System for decentralized, content-addressed storage. It's suitable for applications that need distributed, peer-to-peer storage.

### IndexedDB Storage

IndexedDB storage uses the browser's IndexedDB API for client-side storage. It's suitable for web applications that need to store data in the browser.

## Blockchain Validation and Synchronization

Theo includes a comprehensive blockchain validation and synchronization system. It can validate blockchain integrity, synchronize chains between different storage backends, and resolve conflicts using various strategies.

### Validation

The blockchain validator ensures that:
- Each block has a valid hash
- Each block has a valid previous hash
- Block indices are sequential
- Timestamps are in ascending order (optional)

### Synchronization

The chain synchronizer supports various strategies for resolving conflicts:
- Keep the longest chain
- Keep the newest chain
- Keep the local chain
- Keep the remote chain
- Merge chains

## Docker Support

Theo includes Docker support for running the storage backends and tests. The following Docker Compose files are provided:

- `docker-compose.yml`: Runs the main Theo service
- `docker-compose-storage.yml`: Runs the storage backends
- `docker-compose-test.yml`: Runs the test suite
- `docker-compose-verify.yml`: Runs the storage verification

## License

This project is licensed under the MIT License - see the LICENSE file for details.