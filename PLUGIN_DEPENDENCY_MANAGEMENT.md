# Plugin Dependency Management

## Problem

Go plugins require identical dependency versions between the main binary and the plugin at runtime. Version mismatches can cause:
- Runtime panics
- Interface compatibility errors  
- Subtle bugs due to behavior differences

## Solution

XRP's plugin build system now enforces dependency version consistency through Docker-based builds that extract exact dependency versions from XRP releases.

## How It Works

### 1. Builder Image (`Dockerfile.builder`)

The builder image:
- Downloads the specified XRP release source code
- Extracts `go.mod` and `go.sum` to `/xrp-go.mod` and `/xrp-go.sum`
- Copies full XRP source to `/xrp-source` for plugin builds

### 2. Plugin Template (`Dockerfile.plugin-template`)

Plugin builds:
- Use the builder image as base
- Generate a new `go.mod` with exact dependency versions from XRP
- Replace the XRP dependency with local source via `replace` directive
- Build the plugin with CGO enabled

### 3. SDK Makefile

The SDK Makefile now:
- Uses the plugin template Dockerfile
- Passes `XRP_VERSION` to ensure version alignment
- Supports multi-platform builds

## Usage

### For Plugin Authors

1. **Use the SDK Makefile template**:
   ```bash
   cp /path/to/xrp/build/sdk/Makefile ./Makefile
   ```

2. **Build your plugin**:
   ```bash
   make build XRP_VERSION=v1.0.0
   ```

3. **Your plugin will automatically use**:
   - Same Go version as XRP v1.0.0
   - Same `github.com/beevik/etree` version
   - Same `golang.org/x/net` version
   - Same build environment

### For XRP Development

Plugin examples in `examples/plugins/` still build with:
```bash
make example-plugins
```

These use local XRP dependencies and are built for development/testing.

## Version Compatibility

| XRP Version | Plugin Build | Dependencies |
|-------------|--------------|--------------|
| `development` | Uses local go.mod | Current repo versions |
| `v1.0.0` | Downloads release | Exact v1.0.0 versions |
| `latest` | Downloads latest tag | Latest release versions |

## Benefits

1. **Guaranteed Compatibility**: Plugin dependencies exactly match XRP binary
2. **Reproducible Builds**: Same XRP version = same plugin dependencies
3. **Multi-Platform**: Supports linux/amd64, linux/arm64, linux/arm/v7
4. **Version Isolation**: Different XRP versions can coexist
5. **CI/CD Ready**: Deterministic builds for automated systems

## Migration

Existing plugins can migrate by:

1. Copy the SDK Makefile template
2. Add a Dockerfile (use `build/sdk/examples/minimal/Dockerfile` as template)
3. Update build scripts to use `make build XRP_VERSION=<target-version>`

The old `go build -buildmode=plugin` approach will continue to work for development but may have version inconsistencies in production.