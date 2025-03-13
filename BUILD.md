# Building and Releasing Theo

This document describes how to build, test, and release Theo using Docker.

## Prerequisites

- Docker
- Docker Compose

## Build Process

Theo uses Docker to ensure consistent builds across different platforms. The build process is orchestrated using Docker Compose and a set of Dockerfiles.

### Dockerfile Overview

- `Dockerfile`: Used for development and production builds
- `Dockerfile.test`: Used for running tests
- `Dockerfile.build`: Used for building cross-platform binaries
- `docker-compose.yml`: Orchestrates the build, test, and release process

## Building Theo

### Using the Build Script

The easiest way to build Theo is to use the provided build script:

```bash
# Make the script executable
chmod +x scripts/build.sh

# Show help
./scripts/build.sh --help

# Run tests
./scripts/build.sh --test

# Build for current platform
./scripts/build.sh --build

# Build for all platforms
./scripts/build.sh --cross

# Create release packages
./scripts/build.sh --release

# Run tests, build for all platforms, and create release packages
./scripts/build.sh --all
```

### Using Docker Compose Directly

You can also use Docker Compose directly:

```bash
# Run tests
docker-compose run --rm test

# Build for development
docker-compose build dev

# Build for all platforms
docker-compose run --rm build

# Create release packages
docker-compose run --rm release
```

## Release Process

The release process creates cross-platform binaries for the following platforms:

- Linux (amd64)
- Linux (arm64)
- macOS (amd64)
- macOS (arm64)
- Windows (amd64)

The binaries are compressed using UPX where possible and packaged as ZIP files in the `dist` directory.

## Development

For development, you can run Theo in a Docker container:

```bash
docker-compose up dev
```

This will start Theo in development mode, with the data directory mounted as a volume.

## Continuous Integration

The Dockerfiles and Docker Compose configuration can be used in a CI/CD pipeline to automate the build and release process.

Example GitHub Actions workflow:

```yaml
name: Build and Release

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v1
      - name: Build and test
        run: ./scripts/build.sh --all
      - name: Upload artifacts
        uses: actions/upload-artifact@v2
        with:
          name: theo-binaries
          path: dist/*.zip
      - name: Create Release
        if: startsWith(github.ref, 'refs/tags/')
        uses: softprops/action-gh-release@v1
        with:
          files: dist/*.zip
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Troubleshooting

If you encounter any issues with the build process, try the following:

1. Make sure Docker and Docker Compose are installed and running
2. Clean the Docker cache: `docker system prune -a`
3. Rebuild the images: `docker-compose build --no-cache`
4. Check the Docker logs: `docker-compose logs`