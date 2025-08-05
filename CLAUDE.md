# CLAUDE.md - Development Guidelines for XRP

This file contains important guidelines and context for AI agents working on the XRP codebase.

## Project Overview

XRP is an HTML/XML-aware reverse proxy built in Go that supports:
- Plugin-based content modification for HTML/XML responses
- Redis-based caching with HTTP compliance
- Configuration hot-reloading via SIGHUP
- Response header injection (version and cache status)

## Architecture & Design Principles

### Core Components

1. **Main Package** (`main.go`): Entry point, signal handling, server lifecycle
2. **Proxy Package** (`internal/proxy/`): Core reverse proxy logic, response modification
3. **Config Package** (`internal/config/`): Configuration parsing and validation
4. **Cache Package** (`internal/cache/`): Redis caching with HTTP compliance
5. **Plugins Package** (`internal/plugins/`): Plugin loading and management
6. **Plugin Interface** (`pkg/plugin/`): Shared interface for external plugins

### Key Design Decisions

- **Streaming for non-HTML/XML**: Non-target MIME types are streamed directly without buffering
- **In-place modification**: Plugins modify trees in place to avoid copying large documents
- **HTTP compliance**: Caching respects standard HTTP headers and behaviors
- **Type safety**: Strong typing with interface-based plugin system
- **Error handling**: Graceful degradation - processing errors don't crash the proxy

## Testing Strategy

### Current Test Coverage

- **Config**: Full validation, loading, and schema compliance tests
- **Cache**: Redis integration, TTL calculation, header parsing, key generation
- **Plugins**: Plugin loading, validation, interface compliance
- **Proxy**: Basic functionality, header injection, utility functions

### Testing Guidelines

1. **Use `-short` flag**: Skip integration tests during development
2. **Mock external dependencies**: Redis, plugin loading for unit tests
3. **Test error conditions**: Invalid configs, plugin failures, cache misses
4. **Interface testing**: Verify plugin interface compliance

### Test File Organization

- Simple unit tests in `*_test.go` files
- Complex integration tests can be split into separate files
- Use table-driven tests for multiple scenarios
- Mock heavy dependencies (Redis, file system, plugins)

## Code Organization

### Package Structure

```
xrp/
├── main.go                     # Entry point
├── internal/                   # Private packages
│   ├── config/                 # Configuration handling
│   ├── cache/                  # Redis caching
│   ├── plugins/                # Plugin management
│   └── proxy/                  # Core proxy logic
├── pkg/                        # Public packages
│   └── plugin/                 # Plugin interface
└── examples/                   # Example plugins
```

### Import Guidelines

- Use `internal/` packages for implementation details
- Keep `pkg/plugin` minimal and dependency-free for external plugin authors
- Group imports: stdlib, external, internal

## Critical Implementation Details

### Plugin System

- Plugins are Go shared libraries (`.so` files)
- Must export a symbol matching the configured name
- Interface validation happens at load time
- Plugin failures should be graceful (log error, continue without plugin)

### Caching Logic

- Only cache GET requests with 200 status
- Respect `Cache-Control`, `Expires`, `ETag` headers
- Generate keys from URL + query + Vary headers
- Cookie denylist prevents caching certain requests
- TTL calculation prefers explicit headers over defaults

### Response Headers

- `X-XRP-Version`: Always added to responses going through XRP
- `X-XRP-Cache`: "HIT" for cached responses, "MISS" for fresh responses
- Version comes from `main.version` variable (set at build time)

### Configuration Hot-Reload

- SIGHUP triggers config reload
- New plugins are loaded, old ones remain until replaced
- Invalid configs are rejected, keeping current configuration
- Backend URL changes require new reverse proxy instance

## Development Workflow

### Building

```bash
go build .                      # Main binary
make build                      # Alternative
make example-plugins           # Build example plugins
```

### Testing

```bash
go test ./internal/... -short  # Fast unit tests
go test ./...                  # All tests including integration
make test-coverage             # With coverage report
```

### Adding New Features

1. **Start with tests**: Write failing tests first
2. **Update SPEC.md**: Document new requirements
3. **Implement incrementally**: Small, focused commits
4. **Update documentation**: README.md and code comments
5. **Test integration**: End-to-end testing with real plugins/Redis

## Common Pitfalls & Solutions

### Plugin Development

- **Issue**: Type assertion failures in plugin loading
- **Solution**: Use interface{} and type assertion with ok checks
- **Issue**: Plugin panics crashing the proxy
- **Solution**: Recover from panics in plugin execution

### Testing Challenges

- **Issue**: Redis dependency in tests
- **Solution**: Use build tags or skip integration tests in CI
- **Issue**: Complex mocking for interfaces
- **Solution**: Create simple test implementations, avoid over-mocking

### Configuration Validation

- **Issue**: Runtime config errors
- **Solution**: Validate at load time, provide clear error messages
- **Issue**: MIME type restrictions
- **Solution**: Maintain whitelist of supported HTML/XML types

## Performance Considerations

### Memory Usage

- Stream non-target content (don't buffer large files)
- Limit response size processing (`max_response_size_mb`)
- Cache eviction based on Redis TTL settings

### Concurrency

- Read-write mutex for configuration updates
- Plugin instances are shared across requests (must be thread-safe)
- Redis connection pooling via go-redis client

## Security Considerations

- **Plugin security**: Plugins run with full process privileges
- **Cache poisoning**: Validate cache keys and content
- **Header injection**: Sanitize configuration-provided values
- **DoS protection**: Response size limits, request timeouts

## Dependencies

### Core Dependencies

- `golang.org/x/net/html`: HTML parsing
- `github.com/beevik/etree`: XML processing
- `github.com/redis/go-redis/v9`: Redis client

### Development Dependencies

- Standard Go testing framework
- No external testing frameworks to keep it simple

## Future Extension Points

- Additional MIME type support (with careful consideration)
- Plugin hot-reloading without process restart
- Metrics and monitoring integration
- Multiple backend support (load balancing)
- Custom cache backends beyond Redis

## Error Handling Philosophy

- **Fail gracefully**: Plugin errors shouldn't crash the proxy
- **Log comprehensively**: Use structured logging with context
- **Degrade functionality**: Continue operating with reduced capability
- **Validate early**: Catch configuration errors at startup

## Debugging Tips

- Use `slog` for structured logging throughout
- Add request tracing with unique IDs
- Monitor Redis operations for cache debugging
- Plugin debugging requires separate compilation and testing

Remember: The goal is reliability and performance for high-traffic reverse proxy scenarios while maintaining extensibility through the plugin system.