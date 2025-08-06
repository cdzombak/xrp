FROM debian:bookworm-slim AS builder-base

ARG XRP_VERSION=development
ARG GO_VERSION=1.21.6

# Install dependencies with pinned versions
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc=4:12.2.0-3 \
    g++=4:12.2.0-3 \
    libc6-dev=2.36-9+deb12u3 \
    make=4.3-4.1 \
    git=1:2.39.2-1.1 \
    ca-certificates=20230311 \
    wget=1.21.3-1 \
    && rm -rf /var/lib/apt/lists/*

# Install Go
RUN wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz \
    && rm go${GO_VERSION}.linux-amd64.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"
ENV CGO_ENABLED=1
ENV XRP_VERSION=${XRP_VERSION}

# Pre-download XRP plugin interface if available
RUN if [ "${XRP_VERSION}" != "development" ]; then \
        go install github.com/cdzombak/xrp/pkg/xrpplugin@${XRP_VERSION} || \
        echo "Warning: Could not pre-install XRP plugin interface"; \
    fi

# Add metadata
LABEL org.opencontainers.image.source="https://github.com/cdzombak/xrp"
LABEL org.opencontainers.image.version="${XRP_VERSION}"
LABEL xrp.version="${XRP_VERSION}"
LABEL go.version="${GO_VERSION}"
LABEL debian.version="bookworm"