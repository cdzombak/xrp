.PHONY: build test clean install run example-plugins

# Build the xrp binary
build:
	go build -o xrp .

# Run tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -f xrp coverage.out coverage.html
	rm -f examples/plugins/*.so

# Install dependencies
install:
	go mod download
	go mod tidy

# Run the server with example config
run: build
	./xrp -config config.example.json

# Build example plugins
example-plugins:
	go build -buildmode=plugin -o examples/plugins/html_modifier.so examples/plugins/html_modifier.go
	go build -buildmode=plugin -o examples/plugins/xml_transformer.so examples/plugins/xml_transformer.go

# Development setup
dev-setup: install example-plugins

# Start Redis for testing (requires Docker)
redis-start:
	docker run --name xrp-redis -p 6379:6379 -d redis:alpine

# Stop Redis
redis-stop:
	docker stop xrp-redis
	docker rm xrp-redis

# Lint code
lint:
	go fmt ./...
	go vet ./...

# Check for security issues (requires gosec)
security:
	gosec ./...