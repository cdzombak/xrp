# XRP - HTML/XML-aware Reverse Proxy

An HTML/XML-aware reverse proxy that allows modifying responses via plugins.

- **Plugin-based content modification** - Go plugins modify HTML/XML responses
- **Redis caching** - HTTP-compliant caching

## Run: Quick Demo

```shell
cd examples
docker compose build && docker compose up

# In another terminal:
curl localhost:8080
```

## Configuration

Create a `config.json` file based on `deployment/config.example.json`. This file configures the proxy server, content modification plugins, Redis cache, and certain policies. It contains the following top-level keys:

- `backend_url`: The upstream URL to proxy requests to
- `cookie_denylist`: If a request has a cookie whose name is listed in the denylist, the response is not cached in Redis
- `max_response_size_mb`: The maximum response size to process via plugins and cache. If a response exceeds this size, it is simply returned to the client without modification or caching.
- `mime_types`: A list of MIME type configuration objects. These specify the plugins that will run on responses with the specified MIME type.
- `redis`: Redis cache backend configuration.

## Installation & Running

XRP is inserted between your web server and your application backend. So, instead of:

```
nginx  ->  app (e.g. Ghost)
```

You'll run:

```
nginx  ->  xrp ->  app
```

The exact details of how to implement this will vary depending on your setup.

### Docker

TK

### Debian/Ubuntu via apt repository

TK

### Manual from release artifacts

TK

### From source

TK

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
- **Dependency Management**: [PLUGIN_DEPENDENCY_MANAGEMENT.md](doc/PLUGIN_DEPENDENCY_MANAGEMENT.md) - Docker-based builds with version enforcement
- **Build System**: [BUILD.md](doc/BUILD.md) - XRP build system documentation

## License

TK

## Author

TK
