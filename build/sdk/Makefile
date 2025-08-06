# Makefile template for XRP plugin authors
# Copy this to your plugin repository and customize as needed

XRP_VERSION ?= v1.0.0
PLUGIN_NAME := $(shell basename $(PWD))
PLUGIN_VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")

# Default platforms to build for
PLATFORMS ?= linux/amd64,linux/arm64,linux/arm/v7

.PHONY: build
build:
	@echo "Building $(PLUGIN_NAME) plugin for XRP $(XRP_VERSION)"
	docker buildx build \
		--platform $(PLATFORMS) \
		--build-arg XRP_VERSION=$(XRP_VERSION) \
		--output type=local,dest=./dist \
		--target output \
		.

.PHONY: build-single
build-single:
	@echo "Building $(PLUGIN_NAME) plugin for current platform"
	docker buildx build \
		--build-arg XRP_VERSION=$(XRP_VERSION) \
		--output type=local,dest=./dist \
		--target output \
		.

.PHONY: test
test:
	@echo "Testing $(PLUGIN_NAME) plugin compatibility"
	docker run --rm \
		-v $(PWD)/dist:/plugins:ro \
		ghcr.io/cdzombak/xrp:$(XRP_VERSION) \
		-validate-plugin /plugins/plugin.so

.PHONY: compatibility-check
compatibility-check:
	@echo "Checking compatibility with XRP $(XRP_VERSION)"
	@curl -sL https://github.com/cdzombak/xrp/releases/download/$(XRP_VERSION)/compatibility.json \
		| jq -r '.builder_image'

.PHONY: clean
clean:
	rm -rf dist/

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build              - Build plugin for all platforms"
	@echo "  build-single       - Build plugin for current platform only"
	@echo "  test               - Test plugin compatibility"
	@echo "  compatibility-check - Check XRP version compatibility"
	@echo "  clean              - Remove build artifacts"
	@echo ""
	@echo "Variables:"
	@echo "  XRP_VERSION        - XRP version to build against (default: $(XRP_VERSION))"
	@echo "  PLATFORMS          - Target platforms (default: $(PLATFORMS))"