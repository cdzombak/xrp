# Plugin Dependency Management

## Problem

Go plugins require identical dependency versions between the main binary and the plugin at runtime. Version mismatches can cause:
- Runtime panics
- Interface compatibility errors  
- Subtle bugs due to behavior differences

## Solution

XRP's plugin build system enforces dependency version consistency through a builder image that uses the exact same source tree and dependency versions as the target XRP version.

## How It Works

### 1. Builder Image (`Dockerfile.builder`)

The builder image:
- **Uses local source dependencies** from the current XRP working directory
- Copies `go.mod` and `go.sum` to `/xrp-go.mod` and `/xrp-go.sum`
- Copies full XRP source to `/xrp-source/` for plugin builds
- Pre-downloads all dependencies with `go mod download`

**Key Change**: No more downloading from GitHub releases - uses current source tree for perfect consistency.

### 2. Plugin Build Process

Plugin builds using the XRP builder image:
- Start from `ghcr.io/cdzombak/xrp-builder:VERSION` 
- Have access to exact XRP dependencies via `/xrp-source/`
- Can use `WORKDIR /xrp-source` and build against local XRP interfaces
- Build with CGO enabled and same Go version as XRP

### 3. Version Alignment

```bash
# Builder image contains dependencies from XRP source at build time
XRP_VERSION=v1.0.0 make build/builder    # Creates builder with v1.0.0 deps
XRP_VERSION=development make build/builder # Creates builder with current deps
```

## Usage

### For Plugin Authors

1. **Use XRP builder image matching your target version**:
   ```dockerfile
   FROM ghcr.io/cdzombak/xrp-builder:v1.0.0 AS builder
   
   # Copy your plugin source
   COPY . /plugin-source/
   WORKDIR /plugin-source
   
   # Build with exact XRP dependencies available
   RUN CGO_ENABLED=1 go build -buildmode=plugin -o plugin.so .
   ```

2. **Create a simple Makefile**:
   ```makefile
   XRP_VERSION ?= v1.0.0
   REGISTRY ?= ghcr.io
   IMAGE_NAME ?= myuser/my-plugin
   
   .PHONY: build
   build: ## Build plugin using XRP builder image
   	docker buildx build \
   		--build-arg XRP_VERSION=$(XRP_VERSION) \
   		--platform linux/amd64,linux/arm64 \
   		--output type=local,dest=./dist \
   		.
   
   .PHONY: build-single
   build-single: ## Build plugin for current platform only
   	docker buildx build \
   		--build-arg XRP_VERSION=$(XRP_VERSION) \
   		--platform linux/amd64 \
   		--output type=local,dest=./dist \
   		.
   ```

3. **Build your plugin**:
   ```bash
   make build XRP_VERSION=v1.0.0      # Multi-platform build
   make build-single XRP_VERSION=v1.0.0  # Single-platform build
   ```

### For XRP Development

**Local plugin development** (uses current source tree):
```bash
# Build example plugins with current XRP dependencies
make build/example-plugins

# Build specific plugin
go build -buildmode=plugin -o plugin.so examples/plugins/html_modifier.go
```

**Consistent plugin testing**:
```bash
# Build builder image with current source
make build/builder  # Creates local builder with current deps

# Test plugin against current XRP
docker run --rm -v $(pwd)/plugin.so:/test.so:ro \
  ghcr.io/cdzombak/xrp-builder:$(git describe --always) \
  go tool objdump -t /test.so
```

## Version Compatibility Matrix

| XRP Source State | Builder Build | Plugin Build | Dependencies |
|------------------|---------------|--------------|--------------|
| `v1.0.0` tag     | `make build/builder` | Uses release deps | Exact v1.0.0 versions |
| `main` branch    | `make build/builder` | Uses current deps | Current go.mod/go.sum |
| Local changes    | `make build/builder` | Uses local deps | Modified dependencies |
| Development      | Local plugins | Uses current deps | Current repo state |

## Architecture Benefits

### 1. Source Truth Consistency
- Builder image dependencies **always** match source tree
- No version drift between XRP development and plugin builds
- Consistent behavior across development and CI/CD environments

### 2. Development Workflow
- Fast local plugin builds with `go build -buildmode=plugin`
- Consistent containerized builds with XRP builder image
- Easy testing against different XRP versions

### 3. CI/CD Integration
- Builder images tagged with exact XRP version
- Plugin builds reference specific builder version
- Reproducible builds across different environments

### 4. Multi-Platform Support
- Single builder image supports multiple architectures
- Plugin builds inherit platform support from XRP
- Consistent CGO environment across platforms

## Best Practices

### For Plugin Authors

1. **Pin XRP Version**: Always specify exact XRP version for production plugins
   ```bash
   make build XRP_VERSION=v1.0.0  # Not "latest"
   ```

2. **Test Multiple Versions**: Test your plugin against different XRP versions
   ```bash
   make build XRP_VERSION=v1.0.0
   make build XRP_VERSION=v1.1.0
   # Compare behavior
   ```

3. **Use Multi-Stage Builds**: Keep plugin images minimal
   ```dockerfile
   FROM ghcr.io/cdzombak/xrp-builder:v1.0.0 AS builder
   # ... build steps ...
   
   FROM alpine:latest
   COPY --from=builder /plugin-source/plugin.so /plugins/
   ```

### For XRP Developers

1. **Update Builder Images**: Rebuild builder when dependencies change
   ```bash
   # After updating go.mod
   make build/builder  # Updates local builder
   PUSH=true make build/builder  # Pushes updated builder
   ```

2. **Version Builder Images**: Tag builder images with XRP version
   ```bash
   VERSION=v1.0.0 PUSH=true make build/builder
   # Creates: ghcr.io/cdzombak/xrp-builder:v1.0.0
   ```

3. **Test Plugin Compatibility**: Verify plugins work with XRP changes
   ```bash
   make build/example-plugins  # Build with current deps
   make test/docker            # Test XRP with plugins
   ```

## Migration Guide

### From Custom Plugin Builds

Replace custom Docker builds:
```dockerfile
# Old approach
FROM golang:1.24
RUN go install github.com/cdzombak/xrp/pkg/xrpplugin@latest
# ... custom dependency management

# New approach  
FROM ghcr.io/cdzombak/xrp-builder:v1.0.0
# Dependencies already managed correctly
```

### From Direct Go Builds

For production plugins, replace:
```bash
# Old: Dependency versions may not match XRP
go build -buildmode=plugin -o plugin.so .

# New: Guaranteed compatibility
make build XRP_VERSION=v1.0.0
```

### From Version Guessing

Replace version assumptions:
```bash
# Old: Hope dependencies align
go mod tidy && go build -buildmode=plugin

# New: Explicit version alignment  
make build XRP_VERSION=$(XRP_TARGET_VERSION)
```

## Troubleshooting

### Plugin Loading Failures
```bash
# Check plugin was built with correct XRP version
docker run --rm -v $(pwd)/plugin.so:/test.so:ro \
  ghcr.io/cdzombak/xrp:v1.0.0 -validate-plugin /test.so

# Verify dependency versions in plugin
go version -m plugin.so
```

### Build Failures
```bash
# Ensure builder image exists for target XRP version
docker pull ghcr.io/cdzombak/xrp-builder:v1.0.0

# Check builder image has correct dependencies
docker run --rm ghcr.io/cdzombak/xrp-builder:v1.0.0 \
  cat /xrp-source/go.mod
```

### Version Mismatches
```bash
# Rebuild builder with current source
make build/builder

# Verify builder uses current dependencies
docker run --rm ghcr.io/cdzombak/xrp-builder:$(git describe --always) \
  go list -m all | grep github.com/cdzombak/xrp
```