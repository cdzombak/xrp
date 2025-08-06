# XRP Plugin SDK

This SDK provides everything you need to develop XRP plugins in external repositories.

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

## Files in this SDK

| File | Description |
|------|-------------|
| `Dockerfile.plugin` | Multi-arch build template |
| `Makefile` | Build commands and targets |
| `docker-compose.test.yml` | Local test environment |
| `test-config.json` | XRP configuration for testing |
| `test-content/` | Sample HTML content for testing |
| `examples/` | Example plugin implementations |

## Plugin Structure

Your plugin must export a `GetPlugin()` function:

```go
func GetPlugin() xrpplugin.Plugin {
    return &YourPlugin{}
}
```

And implement the `Plugin` interface:

```go
type Plugin interface {
    ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error
    ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error
}
```

## Version Compatibility

Check compatibility before building:

```bash
make compatibility-check XRP_VERSION=v1.0.0
```

Always build against a specific XRP version for reproducibility.

## Testing

The SDK includes a complete test environment:

```bash
# Start XRP + Redis + Nginx backend
docker-compose -f docker-compose.test.yml up

# Access test environment:
# - XRP proxy: http://localhost:8080
# - Backend: http://localhost:8081
# - Redis: localhost:6379
```

## CI/CD Integration

Use the XRP GitHub Action in your workflows:

```yaml
- uses: cdzombak/xrp/.github/actions/build-xrp-plugin@v1.0.0
  with:
    xrp-version: v1.0.0
```

## Support

- XRP Documentation: https://github.com/cdzombak/xrp
- Plugin Interface: `pkg/xrpplugin/interface.go`
- Examples: `build/sdk/examples/`