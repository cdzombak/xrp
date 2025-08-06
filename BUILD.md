# XRP Build System

This document describes the containerized build system for XRP and provides guidance for both XRP developers and plugin authors.

## Overview

The XRP build system provides:
- **Multi-architecture builds** for linux/amd64, linux/arm64, linux/arm/v7
- **Containerized compilation** with pinned Go and dependency versions
- **Plugin SDK** for external plugin development
- **CI/CD integration** with GitHub Actions
- **Reproducible builds** across development and production environments

## For XRP Developers

### Prerequisites

- Docker with buildx support
- Git
- Make (optional, for convenience)

### Quick Start

```bash
# Run all tests and build everything locally
make ci-local

# Build binaries only
make build-binaries

# Build Docker image
make build-image

# Run tests in Docker
make test-docker
```

### Build Targets

| Target | Description |
|--------|-------------|
| `make build` | Traditional Go build (single platform) |
| `make build-binaries` | Multi-arch binaries via Docker |
| `make build-image` | Multi-arch Docker image |
| `make build-builder` | Builder base image |
| `make test-docker` | Run tests in Docker environment |
| `make ci-local` | Complete CI workflow locally |

### Build Scripts

```bash
# Use build scripts directly
./build/scripts/build.sh all          # Build everything
./build/scripts/build.sh binaries     # Build binaries only
./build/scripts/build.sh test         # Run tests
./build/scripts/build.sh image        # Build Docker image
```

### Environment Variables

```bash
REGISTRY=ghcr.io                      # Container registry
IMAGE_NAME=cdzombak/xrp              # Image name
VERSION=v1.0.0                       # Version tag
PLATFORMS=linux/amd64,linux/arm64   # Target platforms
PUSH=true                            # Push images to registry
```

## For Plugin Authors

### Quick Start

1. **Download the Plugin SDK:**
   ```bash
   curl -L https://github.com/cdzombak/xrp/releases/latest/download/xrp-plugin-sdk.tar.gz | tar xz
   ```

2. **Copy templates to your plugin repository:**
   ```bash
   cp xrp-plugin-sdk/Dockerfile.plugin ./Dockerfile
   cp xrp-plugin-sdk/Makefile ./Makefile
   cp xrp-plugin-sdk/docker-compose.test.yml ./
   ```

3. **Write your plugin:**
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
   
   func GetPlugin() xrpplugin.Plugin {
       return &MyPlugin{}
   }
   ```

4. **Build your plugin:**
   ```bash
   make build XRP_VERSION=v1.0.0
   ```

### Plugin Development Workflow

```bash
# Build for all platforms
make build

# Build for current platform only
make build-single

# Test compatibility
make test

# Start local test environment
make test-env
```

### Version Compatibility

Always specify the XRP version you're targeting:

```bash
# Check compatibility
make compatibility-check XRP_VERSION=v1.0.0

# Build against specific version
make build XRP_VERSION=v1.0.0
```

### Using GitHub Actions

Add this to your plugin repository's `.github/workflows/build.yml`:

```yaml
name: Build Plugin
on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: cdzombak/xrp/.github/actions/build-xrp-plugin@v1.0.0
        with:
          xrp-version: v1.0.0
      - uses: actions/upload-artifact@v3
        with:
          name: plugins
          path: dist/
```

### Local Testing

The SDK includes a complete test environment:

```bash
# Start test environment (XRP + Redis + Nginx)
docker-compose -f docker-compose.test.yml up

# Your plugin will be available at http://localhost:8080
# Backend test server at http://localhost:8081
```

## Architecture Details

### Multi-Stage Docker Build

```
Source Code
     ↓
Builder Image (Debian + Go + XRP deps)
     ↓
Compilation (per architecture)
     ↓
Binary Export ← → Runtime Image
```

### Builder Image

- **Base:** Debian Bookworm (for CGO support)
- **Go Version:** 1.21.6 (pinned)
- **Dependencies:** GCC, build tools, XRP plugin interface
- **Architectures:** linux/amd64, linux/arm64

### Plugin Interface Versioning

Each XRP release publishes:
- Builder image: `ghcr.io/cdzombak/xrp/builder:v1.0.0`
- Compatibility manifest: `compatibility.json`
- Plugin SDK: `xrp-plugin-sdk.tar.gz`

## CI/CD Pipeline

### GitHub Actions Workflow

1. **Build Stage:** Multi-arch binary compilation
2. **Test Stage:** Run Go tests in Docker
3. **Image Stage:** Build Docker images (no push)
4. **Push Stage:** Push images (main/tags only)
5. **Release Stage:** Create GitHub release with artifacts

### Caching Strategy

- **Build cache:** Docker layer caching per target
- **Registry cache:** GitHub Actions cache for buildx
- **Artifact cache:** Binary artifacts between jobs

## Version Management

### XRP Releases

Each XRP release includes:
- Multi-arch binaries
- Docker images (runtime + builder)
- Plugin SDK with templates
- Compatibility manifest
- Checksums and signatures

### Plugin Compatibility

```json
{
  "xrp_version": "v1.0.0",
  "builder_image": "ghcr.io/cdzombak/xrp/builder:v1.0.0",
  "go_version": "1.21.6",
  "supported_platforms": ["linux/amd64", "linux/arm64", "linux/arm/v7"],
  "plugin_interface": {
    "package": "github.com/cdzombak/xrp/pkg/xrpplugin",
    "version": "v1.0.0"
  }
}
```

## Troubleshooting

### Common Issues

**CGO Errors:**
```bash
# Ensure CGO is enabled and GCC is available
docker run --rm ghcr.io/cdzombak/xrp/builder:v1.0.0 go env CGO_ENABLED
```

**Plugin Loading Failures:**
```bash
# Validate plugin compatibility
docker run --rm -v $(pwd)/dist:/plugins:ro \
  ghcr.io/cdzombak/xrp:v1.0.0 -validate-plugin /plugins/plugin.so
```

**Build Cache Issues:**
```bash
# Clear buildx cache
docker buildx prune --all

# Disable cache
make build-binaries BUILDX_ARGS="--no-cache"
```

### Debug Commands

```bash
# Inspect builder image
docker run --rm -it ghcr.io/cdzombak/xrp/builder:v1.0.0 bash

# Check available platforms
docker buildx ls

# Verify multi-arch image
docker buildx imagetools inspect ghcr.io/cdzombak/xrp:v1.0.0
```

## Best Practices

### For XRP Developers

- Always test multi-arch builds before releasing
- Update builder image when Go version changes
- Maintain compatibility manifest accuracy
- Use semantic versioning for releases

### For Plugin Authors

- Pin XRP version for reproducible builds
- Test against multiple XRP versions when possible
- Use the provided SDK templates as starting points
- Follow the plugin interface exactly
- Include proper error handling in plugin code

## Migration Guide

### From Traditional Go Builds

Replace:
```bash
go build -buildmode=plugin -o plugin.so .
```

With:
```bash
make build XRP_VERSION=v1.0.0
```

### From Custom Docker Builds

Use the provided builder image instead of custom Go installations:
```dockerfile
FROM ghcr.io/cdzombak/xrp/builder:v1.0.0 AS builder
# ... your build steps
```