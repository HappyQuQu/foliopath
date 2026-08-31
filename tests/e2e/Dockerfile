# syntax=docker/dockerfile:1.7

# This image exists only to exercise the real application binary in CI. It is
# not the release image and intentionally includes curl for internal probes.
FROM golang:1.26.5-trixie@sha256:4ee9ffa999b4583ce281939cdff828763083610292f252279a0cee77473bd9a7 AS build
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

FROM debian:trixie-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd
ARG VERSION=s1-smoke
LABEL org.opencontainers.image.title="FolioPath application smoke" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.description="Stage 1 test evidence only; not a release image"
RUN apt-get update \
    && apt-get install --yes --no-install-recommends ca-certificates curl libvips42 socat \
    && rm -rf /var/lib/apt/lists/* \
    && install -d -m 0750 /app/data \
    && install -d -m 0555 /library
COPY --from=build /out/foliopath /app/foliopath
EXPOSE 8080
HEALTHCHECK --interval=2s --timeout=2s --start-period=2s --retries=15 \
    CMD ["curl", "--fail", "--silent", "--show-error", "http://127.0.0.1:8080/health/ready"]
ENTRYPOINT ["/app/foliopath"]
CMD ["serve"]
