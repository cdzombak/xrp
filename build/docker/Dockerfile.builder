FROM debian:bookworm-slim AS builder-base

ARG XRP_VERSION=development
ARG GO_VERSION=1.24.5

# Install dependencies with pinned versions
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc=4:12.2.0-3 \
    g++=4:12.2.0-3 \
    libc6-dev=2.36-9+deb12u10 \
    make=4.3-4.1 \
    git=1:2.39.5-0+deb12u2 \
    ca-certificates=20230311 \
    wget=1.21.3-1+deb12u1 \
    curl=7.88.1-10+deb12u12 \
    jq=1.6-2.1 \
    && rm -rf /var/lib/apt/lists/*

# Determine Go architecture based on build platform
ARG TARGETPLATFORM
RUN case "${TARGETPLATFORM}" in \
        "linux/amd64") GOARCH="amd64" ;; \
        "linux/arm64") GOARCH="arm64" ;; \
        "linux/arm/v7") GOARCH="armv6l" ;; \
        *) echo "Unsupported platform: ${TARGETPLATFORM}" && exit 1 ;; \
    esac \
    && wget -q https://go.dev/dl/go${GO_VERSION}.linux-${GOARCH}.tar.gz \
    && tar -C /usr/local -xzf go${GO_VERSION}.linux-${GOARCH}.tar.gz \
    && rm go${GO_VERSION}.linux-${GOARCH}.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}"
ENV CGO_ENABLED=1
ENV XRP_VERSION=${XRP_VERSION}

COPY go.mod /xrp-go.mod
COPY go.sum /xrp-go.sum
COPY . /xrp-source/
WORKDIR /xrp-source
RUN go mod download

LABEL org.opencontainers.image.source="https://github.com/cdzombak/xrp"
LABEL org.opencontainers.image.description="XRP Plugin Builder Image"
LABEL org.opencontainers.image.licenses="GPL-3.0"
LABEL org.opencontainers.image.version="${XRP_VERSION}"
LABEL xrp.version="${XRP_VERSION}"
LABEL go.version="${GO_VERSION}"
LABEL debian.version="bookworm"
