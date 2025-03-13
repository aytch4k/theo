# Theo Makefile

# Variables
BINARY_NAME=theo
DIST_DIR=./dist
DATA_DIR=./data
GO_FILES=$(shell find . -name "*.go" -type f)
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildDate=$(BUILD_DATE) -X main.CommitHash=$(COMMIT_HASH)"

# Default target
.PHONY: all
all: build

# Build the application
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/theo

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(DIST_DIR)

# Build for all platforms using Docker
.PHONY: dist
dist:
	@echo "Building distribution packages..."
	@./scripts/build.sh

# Run the application
.PHONY: run
run:
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME) $(ARGS)

# Build Docker image
.PHONY: docker-build
docker-build:
	@echo "Building Docker image..."
	@docker-compose build dev

# Run Docker container
.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	@docker-compose run --rm dev $(ARGS)

# Run tests in Docker
.PHONY: docker-test
docker-test:
	@echo "Running tests in Docker..."
	@docker-compose run --rm test

# Import a Git repository
.PHONY: import-git
import-git:
	@echo "Importing Git repository..."
	@REPO_ID=$(REPO_ID) OWNER_ID=$(OWNER_ID) USER_ID=$(USER_ID) SOURCE_REPO=$(SOURCE_REPO) docker-compose run --rm import-git

# Show repository status
.PHONY: status
status:
	@echo "Showing repository status..."
	@REPO_ID=$(REPO_ID) docker-compose run --rm status

# Export chains
.PHONY: export
export:
	@echo "Exporting chains..."
	@REPO_ID=$(REPO_ID) JSONL=$(JSONL) docker-compose run --rm export

# Add a commit
.PHONY: commit
commit:
	@echo "Adding commit..."
	@REPO_ID=$(REPO_ID) USER_ID=$(USER_ID) MESSAGE="$(MESSAGE)" BRANCH=$(BRANCH) docker-compose run --rm commit

# Add a tag
.PHONY: tag
tag:
	@echo "Adding tag..."
	@REPO_ID=$(REPO_ID) USER_ID=$(USER_ID) TAG_NAME=$(TAG_NAME) MESSAGE="$(MESSAGE)" docker-compose run --rm tag

# Record a branch deletion
.PHONY: branch-delete
branch-delete:
	@echo "Recording branch deletion..."
	@REPO_ID=$(REPO_ID) USER_ID=$(USER_ID) BRANCH=$(BRANCH) MESSAGE="$(MESSAGE)" docker-compose run --rm branch-delete

# Help
.PHONY: help
help:
	@echo "Theo Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all            Build the application (default)"
	@echo "  build          Build the application"
	@echo "  test           Run tests"
	@echo "  clean          Clean build artifacts"
	@echo "  dist           Build distribution packages for all platforms"
	@echo "  run            Run the application (use ARGS=\"--cmd help\" to pass arguments)"
	@echo "  docker-build   Build Docker image"
	@echo "  docker-run     Run Docker container (use ARGS=\"--cmd help\" to pass arguments)"
	@echo "  docker-test    Run tests in Docker"
	@echo "  import-git     Import a Git repository (use REPO_ID, OWNER_ID, USER_ID, SOURCE_REPO)"
	@echo "  status         Show repository status (use REPO_ID)"
	@echo "  export         Export chains (use REPO_ID, JSONL)"
	@echo "  commit         Add a commit (use REPO_ID, USER_ID, MESSAGE, BRANCH)"
	@echo "  tag            Add a tag (use REPO_ID, USER_ID, TAG_NAME, MESSAGE)"
	@echo "  branch-delete  Record a branch deletion (use REPO_ID, USER_ID, BRANCH, MESSAGE)"
	@echo "  help           Show this help message"