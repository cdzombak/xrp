# XRP - HTML/XML-aware Reverse Proxy

`xrp` is a reverse proxy that can parse and modify HTML and XML responses from backend servers. It is extensible via Golang plugins, allowing users to modify the response bodies of HTML/XML content in a flexible and performant manner.

## Features

- **Plugin-based content modification**: Support for Golang plugins to modify HTML and XML responses
- **Redis caching**: Intelligent caching with HTTP compliance for configured MIME types
- **HTML/XML processing**: Parse and modify HTML/XML content using Go's standard libraries
- **Configuration hot-reload**: Reload configuration on SIGHUP signal
- **Cookie denylist**: Configurable cookie-based cache exclusion
- **Multi-architecture support**: Docker builds for linux/amd64, linux/arm64, linux/arm/v7
- **Plugin SDK**: Complete toolkit for external plugin development

## Quick Start

### Prerequisites

- Go 1.21 or later
- Redis server (for caching)

### Building

#### Traditional Go Build
```bash
# Clone and build
git clone <repository-url>
cd xrp
make build

# Or just use go build
go build -o xrp .
```

#### Docker Multi-Architecture Build
```bash
# Build for all supported platforms
make build-binaries

# Build Docker image
make build-image

# Complete CI workflow locally
make ci-local
```

For detailed build instructions, see [BUILD.md](BUILD.md).

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

#### With Docker Compose (Recommended)

```bash
# Start the complete environment (Nginx + Redis + XRP)
make dev-env

# Or manually with docker-compose
docker-compose up -d

# View logs
make docker-logs

# Stop the environment
make docker-down
```

This starts:
- **Nginx** on port 8081 (backend server)
- **Redis** on port 6379 (cache)  
- **XRP** on port 8080 (reverse proxy)

#### Manual Setup

```bash
# Start Redis separately
docker run --name xrp-redis -p 6379:6379 -d redis:alpine

# Build and run XRP
make build
./xrp -config config.json -addr :8080
```

## Plugin Development

### For External Plugin Authors

Download the Plugin SDK and follow the quick start guide:

```bash
# Download SDK
curl -L https://github.com/cdzombak/xrp/releases/latest/download/xrp-plugin-sdk.tar.gz | tar xz

# Copy templates to your plugin repository
cp xrp-plugin-sdk/Dockerfile.plugin ./Dockerfile
cp xrp-plugin-sdk/Makefile ./Makefile

# Build your plugin
make build XRP_VERSION=v1.0.0
```

### Plugin Interface

Plugins must implement the `Plugin` interface and export a `GetPlugin()` function:

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

// Export the plugin
func GetPlugin() xrpplugin.Plugin {
    return &MyPlugin{}
}
```

### Building Plugins

#### With Plugin SDK (Recommended)
```bash
make build XRP_VERSION=v1.0.0  # Multi-arch build
make test                      # Test compatibility
```

#### Traditional Build
```bash
go build -buildmode=plugin -o my_plugin.so my_plugin.go
```

For complete plugin development guide, see [BUILD.md](BUILD.md#for-plugin-authors).

## Response Headers

XRP adds the following headers to responses:

- `X-XRP-Version`: Version of XRP that processed the response
- `X-XRP-Cache`: Either "HIT" (served from cache) or "MISS" (processed fresh)

## Development

### Quick Start

```bash
# Start complete development environment
make dev-env

# This is equivalent to:
docker-compose up -d
```

### Manual Development

```bash
# Install dependencies
make install

# Build example plugins
make example-plugins

# Run tests
make test

# Run with coverage
make test-coverage

# Build for local development
make build
```

### Docker Commands

```bash
# Build Docker images
make docker-build

# Start services
make docker-up

# Stop services  
make docker-down

# View logs
make docker-logs

# Restart just XRP
make docker-restart
```

## Configuration Reference

| Field | Type | Description |
|-------|------|-------------|
| `backend_url` | string | URL of the backend server to proxy to |
| `redis.addr` | string | Redis server address |
| `redis.password` | string | Redis password (optional) |
| `redis.db` | int | Redis database number |
| `mime_types` | array | List of MIME types and their associated plugins |
| `cookie_denylist` | array | Cookies that prevent caching when present |
| `request_timeout` | int | Timeout for backend requests in seconds (default: 30) |
| `max_response_size_mb` | int | Maximum response size to process in MB (default: 10) |

### Supported MIME Types

- `text/html`
- `application/xhtml+xml`
- `text/xml`
- `application/xml`
- `application/rss+xml`
- `application/atom+xml`

## Caching Behavior

XRP caches responses based on:

- HTTP cache headers (`Cache-Control`, `Expires`, `ETag`)
- Request method (only GET requests)
- Response status (only 200 OK)
- Cookie denylist (requests with denylisted cookies are not cached)
- Vary header support (different cache entries per variation)

## Architecture

```
Client -> XRP -> Backend
          |
          v
       Redis Cache
          |
          v
       Plugins
```

## License

[Add your license here]

## Contributing

[Add contribution guidelines here]
