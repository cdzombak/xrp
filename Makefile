.PHONY: build test clean install run example-plugins docker-up docker-down docker-logs docker-build docker-restart dev-env
.PHONY: build-binaries build-image build-builder test-docker save-image load-image push-image ci-local

# Docker build configuration
REGISTRY ?= ghcr.io
IMAGE_NAME ?= cdzombak/xrp
VERSION ?= $(shell git describe --tags --always --dirty)
PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v7

# Traditional Go build
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
	rm -rf dist/
	rm -f xrp-image.tar

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

# Lint code
lint:
	go fmt ./...
	go vet ./...

# Check for security issues (requires gosec)
security:
	gosec ./...

# Docker build targets

# Build binaries only (no push)
build-binaries:
	docker buildx build \
		--platform $(PLATFORMS) \
		--target binary \
		--output type=local,dest=./dist \
		--build-arg VERSION=$(VERSION) \
		-f build/docker/Dockerfile.xrp .

# Build Docker image locally (no push)
build-image:
	docker buildx build \
		--platform $(PLATFORMS) \
		--target runtime \
		--tag $(REGISTRY)/$(IMAGE_NAME):$(VERSION) \
		--load \
		-f build/docker/Dockerfile.xrp .

# Run tests in Docker
test-docker:
	docker buildx build \
		--target test \
		--progress plain \
		-f build/docker/Dockerfile.xrp .

# Build and export image to tar
save-image:
	docker buildx build \
		--platform $(PLATFORMS) \
		--target runtime \
		--tag $(REGISTRY)/$(IMAGE_NAME):$(VERSION) \
		--output type=docker,dest=./xrp-image.tar \
		-f build/docker/Dockerfile.xrp .

# Load saved image
load-image:
	docker load < xrp-image.tar

# Push pre-built image (requires load-image first)
push-image:
	docker push $(REGISTRY)/$(IMAGE_NAME):$(VERSION)

# Build builder image locally
build-builder:
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--tag $(REGISTRY)/$(IMAGE_NAME)/builder:$(VERSION) \
		--build-arg XRP_VERSION=$(VERSION) \
		--load \
		-f build/docker/Dockerfile.builder .

# Complete local build + test workflow
ci-local: test-docker build-binaries build-image
	@echo "✅ All builds completed successfully"
	@echo "📦 Binaries in ./dist/"
	@echo "🐳 Image tagged as $(REGISTRY)/$(IMAGE_NAME):$(VERSION)"

# Build using scripts
build-script-all:
	./build/scripts/build.sh all

build-script-binaries:
	./build/scripts/build.sh binaries

build-script-test:
	./build/scripts/build.sh test
