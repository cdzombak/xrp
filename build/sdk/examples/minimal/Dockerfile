# This Dockerfile uses the XRP plugin build template
# It will automatically use exact dependency versions from the specified XRP release

ARG XRP_VERSION=latest
FROM ghcr.io/cdzombak/xrp-builder:${XRP_VERSION} AS builder
WORKDIR /workspace

# Copy plugin source
COPY . .

# Generate go.mod with XRP's exact dependency versions if available
RUN if [ -f /xrp-go.mod ]; then \
        echo "Using XRP dependency versions from /xrp-go.mod"; \
        # Extract Go version from XRP's go.mod
        GO_VERSION=$(grep '^go ' /xrp-go.mod | cut -d' ' -f2 || echo "1.21"); \
        PLUGIN_NAME="minimal-plugin"; \
        # Create new go.mod with exact versions
        { \
            echo "module ${PLUGIN_NAME}"; \
            echo ""; \
            echo "go ${GO_VERSION}"; \
            echo ""; \
            echo "require ("; \
            echo "    github.com/cdzombak/xrp v0.0.0-00010101000000-000000000000"; \
            # Extract and add other dependencies from XRP's go.mod
            grep -E '^[[:space:]]*github\.com/beevik/etree|^[[:space:]]*golang\.org/x/net' /xrp-go.mod | sed 's/^[[:space:]]*/    /' || true; \
            echo ")"; \
            echo ""; \
            echo "replace github.com/cdzombak/xrp => /xrp-source"; \
        } > go.mod; \
        echo "Generated go.mod:"; \
        cat go.mod; \
    else \
        echo "Warning: No XRP dependency information available, using plugin's go.mod"; \
    fi

# Initialize module and download dependencies
RUN go mod tidy

# Build the plugin
RUN CGO_ENABLED=1 go build -buildmode=plugin -o plugin.so .

# Output stage - copy built plugin
FROM scratch AS output
COPY --from=builder /workspace/plugin.so /plugin.so