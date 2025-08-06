# XRP Plugin SDK

This SDK provides everything you need to develop XRP plugins with guaranteed dependency compatibility using the XRP builder image system.

## Quick Start

1. **Copy these files to your plugin repository:**
   - `Dockerfile.plugin` → `Dockerfile`
   - `Makefile` → `Makefile`
   - `docker-compose.test.yml` → `docker-compose.test.yml`
   - `test-config.json` → `test-config.json`

2. **Write your plugin** (see `examples/minimal/main.go`)

3. **Build and test:**
   ```bash
   make build XRP_VERSION=v1.0.0
   make test
   ```

## How It Works

The XRP Plugin SDK uses the **XRP builder image system** to ensure perfect dependency compatibility:

- **Builder Image**: Contains exact XRP dependencies from the source tree at build time
- **Version Alignment**: Plugin dependencies automatically match the target XRP version
- **Multi-Platform Support**: Build for multiple architectures using the same dependency versions

## Files in this SDK

| File | Description |
|------|-------------|
| `Dockerfile.plugin` | Multi-arch build template using XRP builder image |
| `Makefile` | Build commands with smart platform detection |
| `docker-compose.test.yml` | Local test environment (XRP + Redis + Nginx) |
| `test-config.json` | XRP configuration for testing |
| `test-content/` | Sample HTML content for testing |
| `examples/minimal/` | Complete minimal plugin example |

## Plugin Structure

Your plugin must implement the XRP plugin interface and export properly:

```go
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

// IMPORTANT: Export struct value (not pointer) for plugin system compatibility
var MyPluginInstance = MyPlugin{}
```

**Critical Notes:**
- Export struct **values**, not pointers (use `var MyPluginInstance = MyPlugin{}`)
- This avoids pointer-to-pointer issues in the Go plugin system
- XRP loads plugins using reflection fallback for maximum compatibility

## Build System Features

### Smart Platform Detection

The SDK automatically uses the optimal build strategy:

```bash
# Development builds (single platform, faster)
make build-single XRP_VERSION=v1.0.0

# Production builds (multi-platform)
make build XRP_VERSION=v1.0.0
```

### Dependency Consistency

The XRP builder image ensures your plugin uses **exactly** the same dependency versions as your target XRP version:

```dockerfile
# Your Dockerfile (using SDK template)
FROM ghcr.io/cdzombak/xrp-builder:v1.0.0
# Dependencies automatically match XRP v1.0.0
```

### Version Compatibility

Always specify the XRP version you're targeting:

```bash
# Pin to specific version for reproducible builds
make build XRP_VERSION=v1.0.0

# Check what versions are available
make compatibility-check XRP_VERSION=v1.0.0
```

## Development Workflow

### 1. Local Testing Setup

Use the included test environment:

```bash
# Start complete test stack
docker-compose -f docker-compose.test.yml up

# Access points:
# - XRP proxy: http://localhost:8080
# - Backend: http://localhost:8081  
# - Redis: localhost:6379

# Test your plugin
curl http://localhost:8080
```

### 2. Build Commands

```bash
# Fast development build (current platform)
make build-single XRP_VERSION=v1.0.0

# Full production build (all platforms) 
make build XRP_VERSION=v1.0.0

# Test plugin compatibility
make test

# Clean build artifacts
make clean
```

### 3. Plugin Validation

```bash
# Verify plugin loads correctly
make test XRP_VERSION=v1.0.0

# Check plugin exports
go tool objdump -t dist/plugin.so | grep MyPluginInstance

# Validate with XRP binary
docker run --rm -v $(pwd)/dist:/plugins:ro \
  ghcr.io/cdzombak/xrp:v1.0.0 -validate-plugin /plugins/plugin.so
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Build Plugin
on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        xrp-version: [v1.0.0, v1.1.0]
    steps:
      - uses: actions/checkout@v4
      - name: Build plugin
        run: make build XRP_VERSION=${{ matrix.xrp-version }}
      - name: Test plugin
        run: make test XRP_VERSION=${{ matrix.xrp-version }}
      - uses: actions/upload-artifact@v4
        with:
          name: plugin-${{ matrix.xrp-version }}
          path: dist/
```

### Automated Testing

```yaml
  test-compatibility:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Test against multiple XRP versions
        run: |
          for version in v1.0.0 v1.1.0; do
            make build XRP_VERSION=$version
            make test XRP_VERSION=$version
          done
```

## Architecture Benefits

### 1. Perfect Dependency Matching
- Plugin dependencies **exactly** match XRP binary dependencies
- No runtime version conflicts or interface mismatches
- Consistent behavior across development and production

### 2. Multi-Platform Support
- Single build command produces all architectures
- Consistent CGO environment across platforms
- Same dependency versions on all platforms

### 3. Version Isolation
- Different XRP versions use different builder images
- No cross-contamination between plugin builds
- Reproducible builds for any XRP version

## Best Practices

### Development
- Use `make build-single` for fast iteration
- Test against the exact XRP version you'll deploy with
- Keep plugins simple and focused on single responsibility

### Production
- Always pin XRP version: `XRP_VERSION=v1.0.0` (never "latest")
- Build for all target platforms: `make build`
- Test plugin compatibility before deployment

### Version Management
- Test against multiple XRP versions when possible
- Update XRP version dependency gradually across environments
- Use semantic versioning for your own plugin releases

## Troubleshooting

### Plugin Loading Issues
```bash
# Check plugin exports are correct
go tool nm dist/plugin.so | grep -i plugin

# Verify plugin structure
docker run --rm -v $(pwd)/dist:/plugins:ro \
  ghcr.io/cdzombak/xrp:v1.0.0 -validate-plugin /plugins/plugin.so
```

### Build Failures
```bash
# Ensure builder image exists
docker pull ghcr.io/cdzombak/xrp-builder:v1.0.0

# Check for correct plugin interface
grep -r "xrpplugin.Plugin" .
```

### Dependency Issues
```bash
# Verify dependencies match XRP version
docker run --rm ghcr.io/cdzombak/xrp-builder:v1.0.0 \
  cat /xrp-source/go.mod

# Force rebuild without cache
docker buildx build --no-cache ...
```

## Examples

See `examples/minimal/` for a complete working plugin that demonstrates:
- Proper interface implementation
- Correct export pattern (struct values, not pointers)  
- HTML and XML processing
- Build configuration

## Support

- **XRP Documentation**: [GitHub Repository](https://github.com/cdzombak/xrp)
- **Plugin Interface**: `pkg/xrpplugin/interface.go`
- **Build System**: `BUILD.md` 
- **Dependency Management**: `PLUGIN_DEPENDENCY_MANAGEMENT.md`