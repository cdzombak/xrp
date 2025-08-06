# XRP - HTML/XML-aware Reverse Proxy

An HTML/XML-aware reverse proxy that supports plugin-based content modification, Redis caching, and configuration hot-reloading.

## Features

- **Plugin-based content modification** - Go plugins modify HTML/XML responses
- **Redis caching** - HTTP-compliant caching with configurable MIME types  
- **Configuration hot-reload** - Reload config via SIGHUP signal
- **Multi-architecture support** - Docker builds for amd64, arm64, arm/v7

## Quick Start

### Prerequisites

- Go 1.24.5 or later
- Redis server (for caching)

### Building

```bash
go build .                    # Build main binary
make example-plugins         # Build example plugins
```

### Configuration

Create a `config.json` file based on the example:

```json
{
  "backend_url": "http://localhost:8081",
  "redis": {
    "addr": "localhost:6379",
    "password": "",
    "db": 0
  },
  "mime_types": [
    {
      "mime_type": "text/html",
      "plugins": [
        {
          "path": "./plugins/html_modifier.so",
          "name": "HTMLModifierPlugin"
        }
      ]
    }
  ],
  "cookie_denylist": ["session"],
  "request_timeout": 30,
  "max_response_size_mb": 10
}
```

### Running

```bash
# Docker setup (nginx + redis + xrp)
cd examples/
timeout 30 docker compose build && timeout 30 docker compose up

# Or run directly  
./xrp -config config.example.json
```

## Plugin Development

Plugins must implement the `Plugin` interface and export struct values (not pointers):

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
    // Modify HTML tree in place
    return nil
}

func (p *MyPlugin) ProcessXMLTree(ctx context.Context, url *url.URL, doc *etree.Document) error {
    // Modify XML document in place  
    return nil
}

// Export struct value (not pointer) for plugin system compatibility
var MyPluginInstance = MyPlugin{}
```

### Development Options

**Local development** (fast, uses current dependencies):
```bash
go build -buildmode=plugin -o plugin.so plugin.go
```

**Production builds** (guaranteed compatibility):
```bash
# Use XRP Plugin SDK for exact dependency matching
cp -r build/sdk/* my-plugin/
cd my-plugin/
make build XRP_VERSION=v1.0.0
```

### Documentation

- **Plugin SDK**: [build/sdk/README.md](build/sdk/README.md) - Complete plugin development guide
- **Dependency Management**: [PLUGIN_DEPENDENCY_MANAGEMENT.md](PLUGIN_DEPENDENCY_MANAGEMENT.md) - Docker-based builds with version enforcement
- **Build System**: [BUILD.md](BUILD.md) - XRP build system documentation

## Testing

```bash
go test ./...                          # All tests
go test ./internal/... -short          # Fast unit tests  
timeout 30 docker compose build && timeout 30 docker compose up  # Integration test
```

## How it Works

```
Client -> XRP -> Backend
          |
          v
       Redis Cache
          |
          v  
       Plugins
```

XRP intercepts responses, applies plugins to modify HTML/XML content, and caches results in Redis with HTTP compliance (respects `Cache-Control`, `ETag`, etc.).
