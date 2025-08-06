# Makefile template for XRP plugin authors
# Copy this to your plugin repository and customize as needed

XRP_VERSION ?= v1.0.0
PLUGIN_NAME := $(shell basename $(PWD))
PLUGIN_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")

# Default platforms - matches XRP multi-platform support
PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v7

.PHONY: build
build: ## Build plugin for all platforms using XRP builder image
	@echo "Building $(PLUGIN_NAME) plugin for XRP $(XRP_VERSION) with exact dependency versions"
	docker buildx build \
		--platform $(PLATFORMS) \
		--build-arg XRP_VERSION=$(XRP_VERSION) \
		--output type=local,dest=./dist \
		--target output \
		.

.PHONY: build-single
build-single: ## Build plugin for current platform only (faster for development)
	@echo "Building $(PLUGIN_NAME) plugin for current platform with exact dependency versions"
	docker buildx build \
		--platform linux/amd64 \
		--build-arg XRP_VERSION=$(XRP_VERSION) \
		--output type=local,dest=./dist \
		--target output \
		.

.PHONY: test
test: ## Test plugin compatibility with target XRP version
	@echo "Testing $(PLUGIN_NAME) plugin compatibility with XRP $(XRP_VERSION)"
	@if [ ! -f dist/plugin.so ]; then \
		echo "Plugin not built. Run 'make build-single' first."; \
		exit 1; \
	fi
	docker run --rm \
		-v $(PWD)/dist:/plugins:ro \
		ghcr.io/cdzombak/xrp:$(XRP_VERSION) \
		-validate-plugin /plugins/plugin.so

.PHONY: test-env
test-env: ## Start local test environment (XRP + Redis + Nginx)
	@echo "Starting test environment for $(PLUGIN_NAME) plugin"
	@echo "XRP proxy: http://localhost:8080"
	@echo "Backend: http://localhost:8081"
	@echo "Redis: localhost:6379"
	docker-compose -f docker-compose.test.yml up

.PHONY: compatibility-check
compatibility-check: ## Check compatibility with XRP version
	@echo "Checking compatibility with XRP $(XRP_VERSION)"
	@echo "Builder image: ghcr.io/cdzombak/xrp-builder:$(XRP_VERSION)"
	@docker pull ghcr.io/cdzombak/xrp-builder:$(XRP_VERSION) >/dev/null 2>&1 && \
		echo "✅ Builder image available" || \
		echo "❌ Builder image not found - check XRP_VERSION"
	@docker pull ghcr.io/cdzombak/xrp:$(XRP_VERSION) >/dev/null 2>&1 && \
		echo "✅ Runtime image available" || \
		echo "❌ Runtime image not found - check XRP_VERSION"

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf dist/

.PHONY: help
help: ## Show available targets
	@echo "XRP Plugin Build System"
	@echo "Plugin: $(PLUGIN_NAME) v$(PLUGIN_VERSION)"
	@echo "Target XRP: $(XRP_VERSION)"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Variables:"
	@echo "  XRP_VERSION        XRP version to build against (default: $(XRP_VERSION))"
	@echo "  PLATFORMS          Target platforms (default: $(PLATFORMS))"
	@echo ""
	@echo "Quick start:"
	@echo "  make build-single   # Fast development build"
	@echo "  make test           # Test plugin compatibility"
	@echo "  make build          # Full multi-platform build"

default: help