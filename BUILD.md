# Building Theo

This document provides detailed instructions for building and developing Theo.

## Prerequisites

- Go 1.19 or later
- Docker and Docker Compose
- Make

## Building from Source

### Clone the Repository

```bash
git clone https://github.com/gold2th/theo.git
cd theo
```

### Build the Project

```bash
make build
```

This will create the `theo` executable in the project root directory.

### Build for Different Platforms

To build for different platforms, use the following commands:

```bash
# Build for Linux
make build-linux

# Build for macOS
make build-darwin

# Build for Windows
make build-windows
```

### Build Docker Image

```bash
make docker-build
```

## Development

### Running Tests

To run all tests:

```bash
make test
```

To run specific tests:

```bash
# Run unit tests
make test-unit

# Run integration tests
make test-integration

# Run storage tests
make test-storage
```

### Running Storage Backends

To start the storage backends:

```bash
docker-compose -f docker-compose-storage.yml up -d
```

To stop the storage backends:

```bash
docker-compose -f docker-compose-storage.yml down
```

### Running the Verification Script

To verify the storage backends:

```bash
./scripts/run-verify-storage.sh
```

### Running All Tests and Verification

To run all tests and verify the storage backends:

```bash
./scripts/run-all-tests.sh
```

## Project Structure

- `cmd/`: Command-line interface
- `internal/`: Internal packages
  - `blockchain/`: Blockchain implementation
    - `action/`: Action chains
    - `block/`: Block implementation
    - `master/`: Master chain
    - `sync/`: Chain synchronization
    - `validation/`: Chain validation
  - `git/`: Git integration
  - `storage/`: Storage backends
    - `aerospike/`: Aerospike storage
    - `file/`: File storage
    - `indexeddb/`: IndexedDB storage
    - `ipfs/`: IPFS storage
    - `sqlite/`: SQLite storage
    - `factory/`: Storage factory
- `scripts/`: Utility scripts
- `test/`: Integration tests

## Docker Compose Files

- `docker-compose.yml`: Main service
- `docker-compose-storage.yml`: Storage backends
- `docker-compose-test.yml`: Test environment
- `docker-compose-verify.yml`: Verification environment

## Makefile Targets

- `build`: Build the project
- `build-linux`: Build for Linux
- `build-darwin`: Build for macOS
- `build-windows`: Build for Windows
- `docker-build`: Build Docker image
- `test`: Run all tests
- `test-unit`: Run unit tests
- `test-integration`: Run integration tests
- `test-storage`: Run storage tests
- `clean`: Clean build artifacts
- `lint`: Run linters
- `fmt`: Format code
- `vet`: Run Go vet