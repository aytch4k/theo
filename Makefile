.PHONY: all build test clean dist dist-all

# Default target
all: build

# Create dist directory
dist:
	mkdir -p dist

# Build the application locally
build:
	go build -o theo ./cmd/theo

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf dist theo

# Build using Docker
docker-build: dist
	docker-compose build build
	docker-compose run --rm build

# Run tests using Docker
docker-test:
	docker-compose build test
	docker-compose run --rm test

# Build for all platforms using Docker
dist-all: dist
	docker-compose build build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64
	docker-compose run --rm build-linux-amd64
	docker-compose run --rm build-linux-arm64
	docker-compose run --rm build-darwin-amd64
	docker-compose run --rm build-darwin-arm64
	docker-compose run --rm build-windows-amd64

# Build for Linux AMD64
dist-linux-amd64: dist
	docker-compose run --rm build-linux-amd64

# Build for Linux ARM64
dist-linux-arm64: dist
	docker-compose run --rm build-linux-arm64

# Build for macOS AMD64
dist-darwin-amd64: dist
	docker-compose run --rm build-darwin-amd64

# Build for macOS ARM64
dist-darwin-arm64: dist
	docker-compose run --rm build-darwin-arm64

# Build for Windows AMD64
dist-windows-amd64: dist
	docker-compose run --rm build-windows-amd64

# Release process
release: clean dist-all
	@echo "Release artifacts created in dist/ directory"
	@ls -la dist/

# Install locally
install: build
	cp theo /usr/local/bin/theo

# Uninstall
uninstall:
	rm -f /usr/local/bin/theo