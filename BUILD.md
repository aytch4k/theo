# Building and Releasing Theo

This document describes how to build, test, and release Theo using Docker.

## Prerequisites

- Docker
- Docker Compose

## Development Build

To build and run Theo in development mode:

```bash
docker-compose up dev
```

This will build Theo and run it in a Docker container with live code reloading.

## Running Tests

To run the tests:

```bash
docker-compose up test
```

This will run all the tests in the Theo codebase.

## Building Releases

### Building for a Specific Platform

To build Theo for a specific platform:

```bash
# For Linux
docker-compose up build-linux

# For macOS
docker-compose up build-macos

# For Windows
docker-compose up build-windows
```

### Building for All Platforms

To build Theo for all supported platforms:

```bash
docker-compose up build-all
```

This will create binaries for Linux, macOS (Intel and Apple Silicon), and Windows in the `dist` directory.

## Release Artifacts

After building, the following artifacts will be available in the `dist` directory:

- `theo-linux-amd64` - Linux binary
- `theo-linux-amd64.tar.gz` - Linux archive
- `theo-darwin-amd64` - macOS Intel binary
- `theo-darwin-amd64.tar.gz` - macOS Intel archive
- `theo-darwin-arm64` - macOS Apple Silicon binary
- `theo-darwin-arm64.tar.gz` - macOS Apple Silicon archive
- `theo-windows-amd64.exe` - Windows binary
- `theo-windows-amd64.zip` - Windows archive

## Manual Build

If you prefer to build without Docker, you can use the build script directly:

```bash
./scripts/build.sh
```

This requires Go 1.21 or later to be installed on your system.

## Cross-Compilation Details

The build process uses Go's cross-compilation capabilities to build binaries for different platforms. The Dockerfiles and build scripts handle the necessary environment setup for each target platform.

## CI/CD Integration

The Docker-based build system can be easily integrated into CI/CD pipelines by running the appropriate Docker Compose commands in your CI/CD configuration.