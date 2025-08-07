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

## Key Principles

- **Use TDD**

### Error Handling
- **Fail gracefully**: Plugin errors don't crash the proxy
- **Validate early**: Catch configuration errors at startup
- **Use structured logging**: `slog` with context throughout

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
- Responses exceeding `max_response_size_mb` are not cached

### Response Size Handling
- Responses over `max_response_size_mb` are streamed through unchanged
- No plugin processing or caching for oversized responses
- Full response content is preserved and returned to client
- Memory protection prevents excessive buffer allocation

### Configuration Hot-Reload
- SIGHUP triggers config reload
- New plugins are loaded, old ones remain until replaced
- Invalid configs are rejected, keeping current configuration
