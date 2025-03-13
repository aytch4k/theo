# Building Theo with Docker

This document provides instructions for building, testing, and releasing the Theo project using Docker. This approach ensures consistent builds across different platforms and environments.

## Prerequisites

- Docker
- Docker Compose
- Make (optional, for using the Makefile)

## Quick Start

The easiest way to build Theo is to use the provided Makefile:

```bash
# Build for all platforms
make dist-all

# Run tests
make docker-test

# Create a release
make release
```

All build artifacts will be placed in the `dist/` directory.

## Manual Build Process

If you prefer not to use Make, you can use Docker Compose directly:

```bash
# Create the dist directory
mkdir -p dist

# Build for Linux AMD64
docker-compose run --rm build-linux-amd64

# Build for Linux ARM64
docker-compose run --rm build-linux-arm64

# Build for macOS AMD64
docker-compose run --rm build-darwin-amd64

# Build for macOS ARM64
docker-compose run --rm build-darwin-arm64

# Build for Windows AMD64
docker-compose run --rm build-windows-amd64

# Run tests
docker-compose run --rm test
```

## Build Artifacts

The build process creates the following artifacts in the `dist/` directory:

- `theo-linux-amd64`: Linux AMD64 binary
- `theo-linux-arm64`: Linux ARM64 binary
- `theo-darwin-amd64`: macOS AMD64 binary
- `theo-darwin-arm64`: macOS ARM64 binary
- `theo-windows-amd64.exe`: Windows AMD64 binary

Each binary comes with:
- A SHA256 checksum file (`.sha256`)
- A compressed archive (`.tar.gz` for Unix platforms, `.zip` for Windows)

## Customizing the Build

You can customize the build by modifying the environment variables in `docker-compose.yml` or by creating your own Docker Compose override file.

## Docker Images

The build process also creates a Docker image that can be used to run Theo:

```bash
# Build the Docker image
docker-compose build build

# Run Theo in Docker
docker run --rm -v $(pwd)/data:/app/data theo:latest --help
```

## CI/CD Integration

This Docker-based build system can be easily integrated into CI/CD pipelines. Here's an example for GitHub Actions:

```yaml
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
    - uses: actions/checkout@v2
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v1
    - name: Build and Test
      run: |
        make docker-test
        make dist-all
    - name: Upload artifacts
      uses: actions/upload-artifact@v2
      with:
        name: theo-binaries
        path: dist/
```

## Troubleshooting

If you encounter any issues with the build process, try the following:

1. Make sure Docker and Docker Compose are up to date
2. Clean the build environment: `make clean`
3. Rebuild the Docker images: `docker-compose build --no-cache`
4. Check the Docker logs: `docker-compose logs`

### Common Issues

#### Docker Compose Version Warning

If you see a warning about the `version` attribute being obsolete, you can ignore it. We've removed this attribute from our docker-compose.yml file.

#### Build Failures

If you encounter build failures, check the following:

- Ensure all required tools are installed in the Docker images
- Check that the build script has the correct permissions
- Verify that the Docker volumes are mounted correctly

## Advanced Usage

### Building with Custom Tags

You can build Theo with custom version tags by setting the `VERSION` environment variable:

```bash
VERSION=v1.0.0 make dist-all
```

### Building with Custom Flags

You can pass custom build flags by modifying the `build.sh` script or by overriding the `LDFLAGS` environment variable:

```bash
LDFLAGS="-X main.Version=v1.0.0 -s -w" make dist-all