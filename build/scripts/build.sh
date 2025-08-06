#!/bin/bash
# XRP multi-architecture build script

set -euo pipefail
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

REGISTRY="${REGISTRY:-ghcr.io}"
IMAGE_NAME="${IMAGE_NAME:-cdzombak/xrp}"
VERSION="${VERSION:-$("$SCRIPT_DIR"/../../.version.sh)}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64,linux/arm/v7}"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

# Check prerequisites
check_prerequisites() {
    log "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        error "Docker is required but not installed"
    fi
    
    if ! docker buildx version &> /dev/null; then
        error "Docker buildx is required but not available"
    fi
    
    if ! git rev-parse --git-dir > /dev/null 2>&1; then
        error "Must be run from within a git repository"
    fi
}

# Build binaries for all platforms
build_binaries() {
    log "Building XRP binaries for platforms: $PLATFORMS"
    
    docker buildx build \
        --platform "$PLATFORMS" \
        --target binary \
        --output type=local,dest=./dist \
        --build-arg VERSION="$VERSION" \
        -f build/docker/Dockerfile.xrp .
    
    log "Binaries built successfully in ./dist/"
}

# Run tests
run_tests() {
    log "Running tests in Docker..."
    
    docker buildx build \
        --target test \
        --progress plain \
        -f build/docker/Dockerfile.xrp .
    
    log "Tests completed successfully"
}

# Build Docker image
build_image() {
    local push_flag=""
    if [[ "${PUSH:-false}" == "true" ]]; then
        push_flag="--push"
        log "Building and pushing Docker image: $REGISTRY/$IMAGE_NAME:$VERSION"
    else
        push_flag="--load"
        log "Building Docker image: $REGISTRY/$IMAGE_NAME:$VERSION"
    fi
    
    docker buildx build \
        --platform "$PLATFORMS" \
        --target runtime \
        --tag "$REGISTRY/$IMAGE_NAME:$VERSION" \
        $push_flag \
        --build-arg VERSION="$VERSION" \
        -f build/docker/Dockerfile.xrp .
    
    if [[ "${PUSH:-false}" == "true" ]]; then
        log "Image pushed successfully"
    else
        log "Image built successfully"
    fi
}

# Build builder image
build_builder() {
    local push_flag=""
    if [[ "${PUSH:-false}" == "true" ]]; then
        push_flag="--push"
        log "Building and pushing builder image: $REGISTRY/$IMAGE_NAME/builder:$VERSION"
    else
        push_flag="--load"
        log "Building builder image: $REGISTRY/$IMAGE_NAME/builder:$VERSION"
    fi
    
    docker buildx build \
        --platform linux/amd64,linux/arm64 \
        --tag "$REGISTRY/$IMAGE_NAME/builder:$VERSION" \
        $push_flag \
        --build-arg XRP_VERSION="$VERSION" \
        -f build/docker/Dockerfile.builder .
    
    if [[ "${PUSH:-false}" == "true" ]]; then
        log "Builder image pushed successfully"
    else
        log "Builder image built successfully"
    fi
}

# Show usage
usage() {
    echo "Usage: $0 [OPTIONS] COMMAND"
    echo ""
    echo "Commands:"
    echo "  binaries    Build binaries for all platforms"
    echo "  test        Run tests in Docker"
    echo "  image       Build Docker image"
    echo "  builder     Build builder image"
    echo "  all         Build everything (binaries, test, image)"
    echo ""
    echo "Options:"
    echo "  --version   Set version (default: git describe)"
    echo "  --platforms Set target platforms (default: $PLATFORMS)"
    echo ""
    echo "Environment variables:"
    echo "  REGISTRY    Container registry (default: $REGISTRY)"
    echo "  IMAGE_NAME  Image name (default: $IMAGE_NAME)"
    echo "  VERSION     Version tag (default: git describe)"
    echo "  PLATFORMS   Target platforms (default: $PLATFORMS)"
    echo "  PUSH        Set to 'true' to push images"
}

# Main execution
main() {
    check_prerequisites
    
    case "${1:-}" in
        binaries)
            build_binaries
            ;;
        test)
            run_tests
            ;;
        image)
            build_image
            ;;
        builder)
            build_builder
            ;;
        all)
            run_tests
            build_binaries
            build_image
            ;;
        --help|-h|help)
            usage
            ;;
        *)
            echo "Unknown command: ${1:-}"
            echo ""
            usage
            exit 1
            ;;
    esac
}

main "$@"
