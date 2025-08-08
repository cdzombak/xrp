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
- `max_response_size_mb`: The maximum response size to process via plugins and cache. If a response exceeds this size, it is streamed through to the client unchanged without plugin processing or caching.
- `mime_types`: A list of MIME type configuration objects. These specify the plugins that will run on responses with the specified MIME type.
- `redis`: Redis cache backend configuration.
- `health_port`: Port for the health check endpoint server (default: 8081)

## Health Check Endpoint

XRP provides a dedicated health check endpoint on a separate port (default: 8081) that can be used by container orchestrators, load balancers, and monitoring systems to determine when the proxy is ready to handle traffic.

- **GET `/health`** on the health port:
  - Returns `102 Processing` with body `starting` during startup (while plugins are loading)
  - Returns `200 OK` with body `ok` when fully ready to serve traffic
  - Returns `102 Processing` during configuration reloads

This endpoint is useful for:
- Kubernetes readiness probes
- Docker health checks
- Load balancer health monitoring
- Service mesh integration

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

You'll need to write your custom plugins depending on your needs. See the [Plugin Development](#plugin-development) section below for more information. Build the plugin binaries for the exact XRP version your server is running. The resulting plugin `.so` binaries must be accessible to XRP and references in your configuration.

### Docker

Docker images for `xrp` are available from GHCR. To ensure compatibility with your plugins, I recommend using the Docker image tagged with the exact XRP version your plugins were built for.

See [the example Docker Compose file](deployment/docker-compose.prod.yml) for details.

### Debian/Ubuntu via apt repository

[Install my Debian repository](https://www.dzombak.com/blog/2025/06/updated-instructions-for-installing-my-debian-package-repositories/) if you haven't already:

```shell
sudo mkdir -p /etc/apt/keyrings
curl -fsSL https://dist.cdzombak.net/keys/dist-cdzombak-net.gpg -o /etc/apt/keyrings/dist-cdzombak-net.gpg
sudo chmod 644 /etc/apt/keyrings/dist-cdzombak-net.gpg
sudo mkdir -p /etc/apt/sources.list.d
sudo curl -fsSL https://dist.cdzombak.net/cdzombak-oss.sources -o /etc/apt/sources.list.d/cdzombak-oss.sources
sudo chmod 644 /etc/apt/sources.list.d/cdzombak-oss.sources
sudo apt update
```

Then install `xrp` via `apt`:

```shell
sudo apt install xrp
```

### Manual from release artifacts

Pre-built binaries for Linux on amd64/arm64 are downloadable from each [GitHub Release](https://github.com/cdzombak/xrp/releases). Debian packages are available as well.

Copy the appropriate binary for your architecture and run it using your tools of choice.

### From source

To build binaries yourself, check out this repository and check out the Git tag for the xrp version you want to build. Then run:

```shell
make build/binaries
```

Copy the resulting binary for your architecture from `dist/` and run it using your tools of choice.

### systemd

If you've installed XRP binaries, whether from the apt repository, release artifacts, or from source, you can use [the provided systemd unit file](deployment/systemd/xrp.service) to run XRP as a service.

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

GNU GPL v3.0; see [LICENSE](LICENSE) in this repo.

## Author

Chris Dzombak
- [dzombak.com](https://www.dzombak.com)
- [GitHub @cdzombak](https://github.com/cdzombak)

## Special Thanks

Thanks to [Namespace](https://namespace.so) for providing GitHub Actions runners for this project.
