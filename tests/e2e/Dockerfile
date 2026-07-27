# syntax=docker/dockerfile:1.7

# This image exists only to exercise the real application binary in CI. It is
# not the release image and intentionally includes curl for internal probes.
FROM golang:1.26.4-bookworm@sha256:b305420a68d0f229d91eb3b3ed9e519fcf2cf5461da4bef997bf927e8c0bfd2b AS build
ARG VERSION=s1-smoke
WORKDIR /src
RUN apt-get update \
    && apt-get install --yes --no-install-recommends libvips-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
RUN CGO_ENABLED=1 go build -tags=libvips -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/foliopath ./cmd/foliopath

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818
ARG VERSION=s1-smoke
LABEL org.opencontainers.image.title="FolioPath application smoke" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.description="Stage 1 test evidence only; not a release image"
RUN apt-get update \
    && apt-get install --yes --no-install-recommends ca-certificates curl libvips42 \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 65532 foliopath \
    && useradd --system --uid 65532 --gid 65532 \
       --home-dir /nonexistent --shell /usr/sbin/nologin foliopath \
    && install -d -o 65532 -g 65532 -m 0750 /app/data \
    && install -d -o 65532 -g 65532 -m 0555 /library
COPY --from=build --chown=65532:65532 /out/foliopath /app/foliopath
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=2s --timeout=2s --start-period=2s --retries=15 \
    CMD ["curl", "--fail", "--silent", "--show-error", "http://127.0.0.1:8080/health/ready"]
ENTRYPOINT ["/app/foliopath"]
CMD ["serve"]
