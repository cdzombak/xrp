# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files first for better caching
COPY ../go.mod go.sum ./
RUN go mod download

# Copy source code
COPY .. .

# Build example plugins first
RUN CGO_ENABLED=0 go build -buildmode=plugin -o plugins/html_modifier.so examples/plugins/html_modifier.go
RUN CGO_ENABLED=0 go build -buildmode=plugin -o plugins/xml_modifier.so examples/plugins/xml_modifier.go

# Build main application
RUN CGO_ENABLED=0 go build -ldflags="-X main.version=docker" -o xrp .

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary and plugins from builder
COPY --from=builder /app/xrp .
COPY --from=builder /app/plugins ./plugins/

# Copy configuration
COPY docker-config.json config.json

# Create non-root user
RUN adduser -D -s /bin/sh xrp
USER xrp

EXPOSE 8080

CMD ["./xrp", "-config", "config.json", "-addr", ":8080"]
