#!/bin/bash
set -e

# Define colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Print with color
print_green() {
    echo -e "${GREEN}$1${NC}"
}

print_yellow() {
    echo -e "${YELLOW}$1${NC}"
}

print_red() {
    echo -e "${RED}$1${NC}"
}

# Create dist directory if it doesn't exist
mkdir -p dist

# Function to show help
show_help() {
    echo "Usage: $0 [options]"
    echo ""
    echo "Options:"
    echo "  -h, --help      Show this help message"
    echo "  -t, --test      Run tests"
    echo "  -b, --build     Build for current platform"
    echo "  -c, --cross     Build for all platforms"
    echo "  -r, --release   Create release packages"
    echo "  -a, --all       Run tests, build for all platforms, and create release packages"
    echo ""
    echo "Examples:"
    echo "  $0 --test       # Run tests"
    echo "  $0 --build      # Build for current platform"
    echo "  $0 --cross      # Build for all platforms"
    echo "  $0 --release    # Create release packages"
    echo "  $0 --all        # Run tests, build for all platforms, and create release packages"
}

# Parse arguments
if [ $# -eq 0 ]; then
    show_help
    exit 0
fi

RUN_TESTS=false
BUILD_CURRENT=false
BUILD_CROSS=false
CREATE_RELEASE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -t|--test)
            RUN_TESTS=true
            shift
            ;;
        -b|--build)
            BUILD_CURRENT=true
            shift
            ;;
        -c|--cross)
            BUILD_CROSS=true
            shift
            ;;
        -r|--release)
            CREATE_RELEASE=true
            shift
            ;;
        -a|--all)
            RUN_TESTS=true
            BUILD_CROSS=true
            CREATE_RELEASE=true
            shift
            ;;
        *)
            print_red "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Run tests
if [ "$RUN_TESTS" = true ]; then
    print_yellow "Running tests..."
    docker-compose run --rm test
    print_green "Tests completed successfully!"
fi

# Build for current platform
if [ "$BUILD_CURRENT" = true ]; then
    print_yellow "Building for current platform..."
    docker-compose build dev
    print_green "Build completed successfully!"
fi

# Build for all platforms
if [ "$BUILD_CROSS" = true ]; then
    print_yellow "Building for all platforms..."
    docker-compose run --rm build
    print_green "Cross-platform build completed successfully!"
fi

# Create release packages
if [ "$CREATE_RELEASE" = true ]; then
    print_yellow "Creating release packages..."
    docker-compose run --rm release
    print_green "Release packages created successfully in ./dist directory!"
fi

print_green "All operations completed successfully!"