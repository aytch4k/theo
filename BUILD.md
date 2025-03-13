# Building and Distributing Theo

This document explains how to build, test, and distribute Theo using Docker and the provided build tools.

## Prerequisites

- Docker and Docker Compose
- Git
- Make (optional, for using the Makefile)

## Building Theo

### Using Docker (Recommended)

The easiest way to build Theo is using Docker, which ensures consistent builds across different environments.

```bash
# Build the Docker image
docker-compose build dev

# Run Theo in Docker
docker-compose run --rm dev --cmd help
```

### Using Make

If you have Make installed, you can use the provided Makefile for common tasks:

```bash
# Build Theo
make build

# Run tests
make test

# Build Docker image
make docker-build

# Run Theo in Docker
make docker-run ARGS="--cmd help"
```

## Cross-Platform Building

Theo can be built for multiple platforms using the provided build script:

```bash
# Build for all supported platforms
./scripts/build.sh
```

Or using Make:

```bash
make dist
```

This will create binaries for the following platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

The binaries and distribution packages will be available in the `dist` directory.

## Testing

Run the tests using Docker:

```bash
docker-compose run --rm test
```

Or using Make:

```bash
make docker-test
```

## Using Theo with Docker

The docker-compose.yml file provides services for all Theo commands:

```bash
# Import a Git repository
docker-compose run --rm import-git

# Show repository status
docker-compose run --rm status

# Export chains
docker-compose run --rm export

# Add a commit
docker-compose run --rm commit

# Add a tag
docker-compose run --rm tag

# Record a branch deletion
docker-compose run --rm branch-delete
```

You can pass environment variables to customize the commands:

```bash
REPO_ID=my-repo USER_ID=alice SOURCE_REPO=https://github.com/example/repo.git docker-compose run --rm import-git
```

## Environment Variables

The following environment variables can be used with the Docker services:

- `REPO_ID`: Repository ID (default: "example")
- `OWNER_ID`: Owner ID (default: "owner")
- `USER_ID`: User ID (default: "user")
- `SOURCE_REPO`: Source Git repository URL (default: "https://github.com/example/repo.git")
- `MESSAGE`: Commit/tag/branch-delete message
- `BRANCH`: Branch name (default: "main")
- `TAG_NAME`: Tag name (default: "v1.0.0")
- `JSONL`: Set to any value to enable JSONL output for exports

## Distribution

The build script creates distribution packages for all supported platforms:

1. Binaries for each platform
2. SHA256 checksums for each binary
3. Archives (tar.gz for Linux/macOS, zip for Windows) containing:
   - The binary
   - README.md
   - LICENSE (if present)
   - Documentation (if present)

## Continuous Integration

You can integrate the build process into your CI/CD pipeline:

```yaml
# Example GitHub Actions workflow
name: Build and Test

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    - name: Set up Docker
      uses: docker/setup-buildx-action@v2
    - name: Build and Test
      run: |
        docker-compose build test
        docker-compose run --rm test
    - name: Build Distribution
      run: ./scripts/build.sh
    - name: Upload Artifacts
      uses: actions/upload-artifact@v3
      with:
        name: theo-binaries
        path: dist/
```

## Customizing the Build

You can customize the build by modifying the following files:

- `Dockerfile`: The main Docker image for running Theo
- `Dockerfile.test`: Docker image for running tests
- `Dockerfile.build`: Docker image for building cross-platform binaries
- `docker-compose.yml`: Docker Compose services for various Theo commands
- `scripts/build.sh`: Script for building cross-platform binaries
- `Makefile`: Make targets for common tasks

## Troubleshooting

### Permission Issues

If you encounter permission issues with the build script:

```bash
chmod +x scripts/build.sh
```

### Docker Volume Mounting

If you have issues with Docker volume mounting, ensure your Docker has permission to access the project directory.

### Cross-Platform Build Failures

If cross-platform builds fail, you may need to install additional dependencies in the Dockerfile.build file.