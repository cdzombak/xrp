# XRP Build System

This document describes the simplified XRP build system that uses local source dependencies and intelligent platform detection for fast local development and multi-architecture CI/CD builds.

## Overview

The XRP build system provides:
- **Local source dependencies** - Uses current working directory's go.mod/go.sum for consistency
- **Smart platform detection** - Single platform for local development, multi-arch for CI/CD
- **Containerized compilation** with pinned Go and dependency versions
- **Unified build script** as single source of truth
- **Make target delegation** for convenient workflows

## For XRP Developers

### Prerequisites

- Docker with buildx support
- Git
- Make (optional, for convenience)

### Quick Start

```bash
# Local development (fast, single platform)
make build/go                    # Build with local Go
make test/go                     # Run tests with local Go  
make build/example-plugins       # Build example plugins

# Docker builds (consistent, containerized)
make test/docker                 # Run tests in Docker
make build/binaries             # Build binaries for current platform
make build/image                # Build Docker image for current platform
make ci/local                   # Complete CI workflow locally

# Production/CI builds (push to registry, multi-platform)
make build/builder              # Build and push builder image (multi-arch)
```

### Build System Architecture

The build system uses **intelligent platform detection**:
- **Local builds** (`PUSH=false`): Use `linux/amd64` only for speed
- **CI/Push builds** (`PUSH=true`): Use full `PLATFORMS` list for compatibility

**Delegation pattern**:
- **Make targets** provide clean interface and set environment variables
- **Build script** (`build/scripts/build.sh`) contains all Docker buildx logic
- **Builder image** uses local source tree dependencies for consistency

### Make Targets

| Target | Description | Platforms | Purpose |
|--------|-------------|-----------|---------|
| `make build/go` | Local Go build | Current | Fast development |
| `make test/go` | Local Go tests with coverage | Current | Fast testing |
| `make build/example-plugins` | Build example plugins | Current | Plugin development |
| `make test/docker` | Docker-based tests | `linux/amd64` | Consistent testing |
| `make build/binaries` | Docker binaries build | `linux/amd64` | Local binaries |
| `make build/image` | Docker image build | `linux/amd64` | Local image |
| `make build/builder` | Builder image (push) | Multi-arch | CI/CD infrastructure |
| `make ci/local` | Complete local CI | `linux/amd64` | Pre-commit validation |

### Build Script Commands

```bash
# Local development (single platform)
./build/scripts/build.sh test         # Run tests  
./build/scripts/build.sh binaries     # Build binaries
./build/scripts/build.sh image        # Build Docker image
./build/scripts/build.sh all          # Build everything

# CI/Production (multi-platform push)
PUSH=true ./build/scripts/build.sh builder    # Build and push builder image
PUSH=true ./build/scripts/build.sh image      # Build and push runtime image
PUSH=true ./build/scripts/build.sh binaries   # Build multi-arch binaries
```

### Environment Variables

```bash
REGISTRY=ghcr.io                           # Container registry
IMAGE_NAME=cdzombak/xrp                   # Base image name  
VERSION=v1.0.0                            # Version tag (default: git describe)
PLATFORMS=linux/amd64,linux/arm64,linux/arm/v7  # Target platforms
PUSH=true                                 # Push to registry (enables multi-arch)
```

### Key Features

**Local Source Dependencies:**
- Builder image uses current working directory's `go.mod` and `go.sum`
- No more version mismatches between local development and Docker builds
- Consistent dependency resolution across all build methods

**Smart Platform Detection:**
- Local builds: Fast single-platform (`linux/amd64`) for development
- Push builds: Full multi-architecture support for production deployment
- Automatic detection based on `PUSH` environment variable

## For Plugin Authors

### Using XRP Builder Image

The XRP builder image contains all dependencies needed to build plugins compatible with a specific XRP version:

```dockerfile
# Use XRP builder image matching your target XRP version
FROM ghcr.io/cdzombak/xrp-builder:v1.0.0 AS builder

# Copy your plugin source
COPY .. /plugin-source/
WORKDIR /plugin-source

# Build your plugin
RUN CGO_ENABLED=1 go build -buildmode=plugin -o plugin.so .

# Multi-stage build for deployment
FROM alpine:latest
COPY --from=builder /plugin-source/plugin.so /plugins/
```

### Plugin Development Workflow

1. **Choose XRP version to target:**
   ```bash
   XRP_VERSION=v1.0.0  # Pin to specific XRP release
   ```

2. **Create your plugin:**
   ```go
   // main.go
   package main
   
   import (
       "context"
       "net/url"
       "golang.org/x/net/html"
       "github.com/beevik/etree"
       "github.com/cdzombak/xrp/pkg/xrpplugin"
   )
   
   type MyPlugin struct{}
   
   func (p *MyPlugin) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
       // Your HTML processing logic
       return nil
   }
   
   func (p *MyPlugin) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
       // Your XML processing logic  
       return nil
   }
   
   // Export struct value (not pointer) for plugin system
   var MyPluginInstance = MyPlugin{}
   ```

3. **Build with Docker:**
   ```bash
   # Single platform (development)
   docker build -t my-plugin .
   
   # Multi-platform (production)
   docker buildx build --platform linux/amd64,linux/arm64 -t my-plugin .
   ```

### Local Testing Setup

Create a `docker-compose.yml` for testing:

```yaml
version: '3.8'
services:
  xrp:
    image: ghcr.io/cdzombak/xrp:v1.0.0
    ports:
      - "8080:8080"
    volumes:
      - ./dist:/plugins:ro
    environment:
      - PLUGINS_DIR=/plugins
      - UPSTREAM_URL=http://nginx:80
    depends_on:
      - nginx
      - redis

  nginx:
    image: nginx:alpine
    volumes:
      - ./test-content:/usr/share/nginx/html:ro

  redis:
    image: redis:alpine
```

```bash
# Start test environment
docker-compose up

# Test your plugin
curl http://localhost:8080
```

## Architecture Details

### Multi-Stage Docker Build

```
Local Source Tree
      ↓
Builder Image (Debian + Go + current deps)
      ↓
Compilation (platform-specific)
      ↓
Binary Export ← → Runtime Image
```

### Builder Image

- **Base:** Debian Bookworm (CGO support)
- **Go Version:** 1.24.5 (pinned)
- **Dependencies:** Current source tree's go.mod/go.sum
- **Platforms:** `linux/amd64` (local), `linux/amd64,linux/arm64` (push)

### Version Management

**XRP Releases include:**
- Multi-arch binaries (`dist/xrp-{os}-{arch}`)
- Docker images (`ghcr.io/cdzombak/xrp:version`)
- Builder images (`ghcr.io/cdzombak/xrp-builder:version`)
- Plugin interface (`pkg/xrpplugin` Go module)

**Version Detection:**
- Uses `.version.sh` script with `git describe`
- Development builds: `commit-dirty` format
- Release builds: `v1.0.0` semantic versioning

## Troubleshooting

### Common Issues

**Docker buildx not available:**
```bash
docker buildx create --use
```

**CGO build failures:**
```bash
# Verify CGO is enabled in builder
docker run --rm ghcr.io/cdzombak/xrp-builder:latest go env CGO_ENABLED
```

**Platform compatibility issues:**
```bash
# Check available platforms
docker buildx ls

# Force specific platform for testing
docker run --platform linux/amd64 ghcr.io/cdzombak/xrp:latest
```

**Build cache issues:**
```bash
# Clear build cache
docker buildx prune --all

# Rebuild without cache
docker buildx build --no-cache ...
```

### Debug Commands

```bash
# Inspect built images
docker buildx imagetools inspect ghcr.io/cdzombak/xrp:latest

# Test plugin loading
docker run --rm -v $(pwd)/plugin.so:/test.so:ro \
  ghcr.io/cdzombak/xrp:latest -validate-plugin /test.so

# Interactive builder environment
docker run --rm -it ghcr.io/cdzombak/xrp-builder:latest bash
```

## Best Practices

### For XRP Developers

- Use `make ci/local` before pushing changes
- Pin Go version in builder Dockerfile when updating
- Test both local and Docker builds regularly
- Use semantic versioning for releases
- Update builder image tag when dependencies change

### For Plugin Authors  

- Always pin XRP version for reproducible builds
- Use the XRP builder image for consistency
- Test plugins against multiple XRP versions when possible
- Follow the exact plugin interface (struct values, not pointers)
- Include proper error handling and logging

### Performance Tips

- Use local Go builds (`make build/go`) for rapid development
- Use Docker builds (`make test/docker`) for validation
- Set `PUSH=false` (default) for fast single-platform local builds
- Set `PUSH=true` only for CI/CD multi-platform builds
- Leverage Docker layer caching with consistent build patterns
