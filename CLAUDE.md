# CLAUDE.md - XRP Development Guide

## Project Overview

XRP is an HTML/XML-aware reverse proxy built in Go that supports:
- Plugin-based content modification for HTML/XML responses
- Redis-based caching with HTTP compliance
- Configuration hot-reloading via SIGHUP

## Architecture

```
xrp/
├── main.go                     # Entry point
├── internal/                   # Private packages
│   ├── config/                 # Configuration handling
│   ├── cache/                  # Redis caching
│   ├── plugins/                # Plugin management
│   └── proxy/                  # Core proxy logic
├── pkg/xrpplugin/              # Plugin interface (public)
└── examples/                   # Example plugins and Docker setup
```

## Building & Testing

### Quick Build
```bash
go build .                      # Main binary
make example-plugins           # Build example plugins
```

### Testing
```bash
go test ./internal/... -short  # Fast unit tests
go test ./...                  # All tests including integration
```

### Docker Testing (Complete Stack)
```bash
cd examples/
timeout 30 docker compose build && timeout 30 docker compose up
```
This starts Nginx (backend), Redis (cache), and XRP (proxy) with example plugins.

## Plugin Development

### Plugin Structure
Plugins export struct values (not pointers) to avoid Go plugin system complexities:

```go
package main

import (
    "context"
    "net/url"
    "golang.org/x/net/html"
    "github.com/beevik/etree"
    "xrp/pkg/xrpplugin"
)

type MyPlugin struct{}

func (p *MyPlugin) ProcessHTMLTree(ctx context.Context, url *url.URL, node *html.Node) error {
    // Modify HTML tree in place
    return nil
}

func (p *MyPlugin) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
    // Modify XML document in place
    return nil
}

// Export as struct value (not pointer)
var MyPluginInstance = MyPlugin{}
```

### Building Plugins
```bash
go build -buildmode=plugin -o my_plugin.so my_plugin.go
```

## Key Implementation Details

### Plugin System
- Plugins are Go shared libraries (`.so` files)
- Export struct values to avoid pointer-to-pointer issues
- Interface validation happens at load time using reflection fallback
- Plugin failures are graceful (log error, continue without plugin)

### Caching Logic
- Only cache GET requests with 200 status
- Respect `Cache-Control`, `Expires`, `ETag` headers
- Generate keys from URL + query + Vary headers
- Cookie denylist prevents caching certain requests

### Configuration Hot-Reload
- SIGHUP triggers config reload
- New plugins are loaded, old ones remain until replaced
- Invalid configs are rejected, keeping current configuration

## Error Handling
- **Fail gracefully**: Plugin errors don't crash the proxy
- **Validate early**: Catch configuration errors at startup
- **Use structured logging**: `slog` with context throughout

## Dependencies
- `golang.org/x/net/html`: HTML parsing
- `github.com/beevik/etree`: XML processing
- `github.com/redis/go-redis/v9`: Redis client