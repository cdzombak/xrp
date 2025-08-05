# Build stage
FROM golang:1.24.5-bookworm AS builder
RUN apt-get update && apt-get install build-essential -y

WORKDIR /app

# Copy go mod files first for better caching
COPY ../go.mod go.sum ./
RUN go mod download

# Copy source code
COPY .. .

# Build main application first to ensure all dependencies are resolved
RUN CGO_ENABLED=1 go build -ldflags="-X main.version=docker" -o xrp .

# Create plugins directory and build plugins using the same build environment
RUN mkdir -p plugins
RUN CGO_ENABLED=1 go build -buildmode=plugin -o plugins/html_modifier.so examples/plugins/html_modifier.go
RUN CGO_ENABLED=1 go build -buildmode=plugin -o plugins/xml_transformer.so examples/plugins/xml_transformer.go

# Runtime stage
FROM debian:bookworm-slim

# Install ca-certificates for HTTPS connections
#RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary and plugins from builder
COPY --from=builder /app/xrp .
COPY --from=builder /app/plugins ./plugins/

# Copy configuration
COPY examples/docker-config.json config.json

# Create non-root user
RUN adduser --disabled-password --shell /bin/sh --gecos "" xrp
USER xrp

EXPOSE 8080

CMD ["./xrp", "-config", "config.json", "-addr", ":8080"]
