
# Docker build configuration
REGISTRY ?= ghcr.io
IMAGE_NAME ?= cdzombak/xrp
VERSION ?= $(shell git describe --tags --always --dirty)
PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v7

default: help
.PHONY: help  # via https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help: ## Print help
	@grep -E '^[a-zA-Z_-\/]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: clean
clean: ## Clean build artifacts
	rm -f xrp coverage.out coverage.html
	rm -f examples/plugins/*.so
	rm -rf dist/
	rm -f xrp-image.tar

.PHONY: deps
deps: deps/go ## Install dependencies

.PHONY: deps/go
deps/go: ## Install Go dependencies
	go mod download
	go mod tidy

.PHONY: lint/go
lint/go: ## Lint Go code
	golangci-lint run
	go vet ./...

.PHONY: build/go
build/go: ## Build xrp using local Go
	go build -o xrp .

.PHONY: build/example-plugins
build/example-plugins: ## Build example plugins using local Go
	go build -buildmode=plugin -o examples/plugins/html_modifier.so examples/plugins/html_modifier.go
	go build -buildmode=plugin -o examples/plugins/xml_transformer.so examples/plugins/xml_transformer.go

.PHONY: build/builder
build/builder: ## Build & push the Docker build container image
	PUSH=true REGISTRY=$(REGISTRY) IMAGE_NAME=$(IMAGE_NAME)-builder VERSION=$(VERSION) PLATFORMS=$(PLATFORMS) \
		./build/scripts/build.sh builder

.PHONY: build/binaries
build/binaries: build/builder ## Build binaries (only) using Docker build container
	REGISTRY=$(REGISTRY) IMAGE_NAME=$(IMAGE_NAME) VERSION=$(VERSION) PLATFORMS=$(PLATFORMS) \
		./build/scripts/build.sh binaries

.PHONY: build/image
build/image: build/builder ## Build xrp Docker image using Docker build container
	REGISTRY=$(REGISTRY) IMAGE_NAME=$(IMAGE_NAME) VERSION=$(VERSION) PLATFORMS=$(PLATFORMS) \
		./build/scripts/build.sh image

.PHONY: test/go
test/go: ## Run xrp test suite, with coverage
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: test/docker
test/docker: build/builder ## Run xrp test suite in Docker
	REGISTRY=$(REGISTRY) IMAGE_NAME=$(IMAGE_NAME) VERSION=$(VERSION) \
		./build/scripts/build.sh test

.PHONY: ci/local
ci/local: test/docker build/binaries build/image ## Run a complete local build + tests in Docker
	@echo "✅ All builds completed successfully"
	@echo "📦 Binaries in ./dist/"
	@echo "🐳 Image tagged as $(REGISTRY)/$(IMAGE_NAME):$(VERSION)"
