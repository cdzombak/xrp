#!/bin/bash
# XRP plugin build helper script

set -euo pipefail

# Configuration
XRP_VERSION="${XRP_VERSION:-v1.0.0}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64,linux/arm/v7}"
PLUGIN_NAME="${PLUGIN_NAME:-$(basename "$(pwd)")}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

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
    log "Checking prerequisites for plugin build..."
    
    if ! command -v docker &> /dev/null; then
        error "Docker is required but not installed"
    fi
    
    if ! docker buildx version &> /dev/null; then
        error "Docker buildx is required but not available"
    fi
    
    if [[ ! -f "Dockerfile" ]]; then
        error "No Dockerfile found. Use 'init' command to create one."
    fi
    
    if [[ ! -f "go.mod" ]]; then
        error "No go.mod found. This must be run from a Go plugin directory."
    fi
}

# Initialize plugin directory
init_plugin() {
    local plugin_name="${1:-$PLUGIN_NAME}"
    
    log "Initializing plugin directory for: $plugin_name"
    
    if [[ -f "Dockerfile" ]] || [[ -f "Makefile" ]]; then
        warn "Plugin files already exist. Use --force to overwrite."
        if [[ "${FORCE:-false}" != "true" ]]; then
            exit 1
        fi
    fi
    
    # Create necessary files
    curl -sL "https://github.com/cdzombak/xrp/releases/download/$XRP_VERSION/xrp-plugin-sdk.tar.gz" | tar xz --strip-components=1
    
    log "Plugin initialized. Edit main.go and run 'build' to compile."
}

# Validate plugin compatibility
validate_plugin() {
    log "Validating plugin compatibility with XRP $XRP_VERSION"
    
    # Check if the plugin builds
    docker buildx build \
        --build-arg XRP_VERSION="$XRP_VERSION" \
        --target output \
        --output type=local,dest=./dist \
        .
    
    # Test loading the plugin (if XRP supports validation)
    if docker run --rm ghcr.io/cdzombak/xrp:$XRP_VERSION --help | grep -q "validate-plugin"; then
        docker run --rm \
            -v "$(pwd)/dist:/plugins:ro" \
            ghcr.io/cdzombak/xrp:$XRP_VERSION \
            -validate-plugin /plugins/plugin.so
    else
        warn "Plugin validation not supported in XRP $XRP_VERSION"
    fi
    
    log "Plugin validation completed"
}

# Build plugin
build_plugin() {
    log "Building plugin: $PLUGIN_NAME for XRP $XRP_VERSION"
    log "Target platforms: $PLATFORMS"
    
    docker buildx build \
        --platform "$PLATFORMS" \
        --build-arg XRP_VERSION="$XRP_VERSION" \
        --output type=local,dest=./dist \
        --target output \
        .
    
    log "Plugin built successfully in ./dist/"
    
    # List generated files
    if [[ -d "./dist" ]]; then
        echo "Generated files:"
        find ./dist -name "*.so" -exec ls -la {} \;
    fi
}

# Test plugin locally
test_plugin() {
    log "Testing plugin locally with Docker Compose"
    
    if [[ ! -f "docker-compose.test.yml" ]]; then
        error "No docker-compose.test.yml found. Run 'init' first."
    fi
    
    # Build plugin first
    build_plugin
    
    # Start test environment
    XRP_VERSION="$XRP_VERSION" docker-compose -f docker-compose.test.yml up -d
    
    log "Test environment started. XRP available at http://localhost:8080"
    log "Backend available at http://localhost:8081"
    log "Use 'docker-compose -f docker-compose.test.yml logs -f' to view logs"
    log "Use 'docker-compose -f docker-compose.test.yml down' to stop"
}

# Show usage
usage() {
    echo "Usage: $0 [OPTIONS] COMMAND"
    echo ""
    echo "Commands:"
    echo "  init [name]  Initialize plugin directory with templates"
    echo "  build        Build plugin for all platforms"
    echo "  validate     Validate plugin compatibility"
    echo "  test         Start local test environment"
    echo ""
    echo "Options:"
    echo "  --force      Force overwrite existing files (with init)"
    echo "  --version    Set XRP version (default: $XRP_VERSION)"
    echo "  --platforms  Set target platforms (default: $PLATFORMS)"
    echo ""
    echo "Environment variables:"
    echo "  XRP_VERSION  XRP version to build against (default: $XRP_VERSION)"
    echo "  PLATFORMS    Target platforms (default: $PLATFORMS)"
    echo "  PLUGIN_NAME  Plugin name (default: current directory name)"
}

# Main execution
main() {
    case "${1:-}" in
        init)
            init_plugin "${2:-}"
            ;;
        build)
            check_prerequisites
            build_plugin
            ;;
        validate)
            check_prerequisites
            validate_plugin
            ;;
        test)
            check_prerequisites
            test_plugin
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