# xrp: HTML/XML-aware reverse proxy

## Overview

`xrp` is a reverse proxy that can parse and modify HTML and XML responses from backend servers. It is extensible via Golang plugins, allowing users to modify the response bodies of HTML/XML content in a flexible and performant manner.

## Functional Requirements

- The configuration JSON file specifies a list of MIME types and the plugin(s), in order, that `xrp` will call for each MIME type.
- The config JSON is validated against a JSON schema.
- The config JSON only allows the user to specify MIME types that are known to be HTML/XML.

### Request Handling

- Files that are not HTML/XML should streamed directly from backend to the client, not buffered in memory.
- Incoming request bodies are not modified. They are streamed to the backend, not buffered in memory.

### Plugins

- Plugins are configured via the configuration JSON file.
- Plugins are used to modify the response body of HTML/XML responses.
- A Plugin interface is defined that is shared and can be imported by plugins without importing all of `xrp`.
- The Plugin interface has two methods. These methods are expected to modify the tree in place, so they do not return a new tree:
    - ProcessHTMLTree takes a `*html.Node` and returns an error.
    - ProcessXMLTree takes a `*etree.Document` and returns an error.
- If a plugin does not implement the required method (e.g. the plugin is supposed to run on the HTML MIME type but only implements ProcessXMLTree), the program exits with an error.

### Caching

- Responses for the configured MIME types are cached in Redis. Responses for unconfigured MIME types are not cached at all.
- The Redis details are specified in the configuration JSON file.
- Caching is only performed for successful responses (HTTP 200 OK).
- Caching is only done for GET requests.
- Caching is done using the `Cache-Control` and `Expires` headers to determine cacheability.
- Responses with a `Vary` header are cached separately for each variation..
- Caching obeys HTTP caching headers, including `Cache-Control`, `Expires`, and `ETag`.
- Cached responses are stored with a key that includes the URL path and query parameters.
- Responses including a Set-Cookie header are not cached.
- A cookie name denylist can be specified in the configuration JSON file. Responses to requests that include cookies matching the denylist are not cached.

## Technical/Implementation Requirements

- Implemented in Go using https://pkg.go.dev/net/http/httputil#ReverseProxy
- Plugins are implemented using the https://pkg.go.dev/plugin package.
- The configuration JSON file is read at startup and reloaded on SIGHUP.
- XML trees are handled using the https://github.com/beevik/etree package.
- HTML trees are handled using the Go standard library's `html` package.
- Logging is done using the Golang standard library's `slog` package.
- The code follows best practices for idiomatic Go. The code is readable and maintainable.
- The implementation must have good test coverage with unit tests!
