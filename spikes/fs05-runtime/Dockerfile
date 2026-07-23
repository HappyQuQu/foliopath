# syntax=docker/dockerfile:1.7

FROM golang:1.26.4-bookworm@sha256:b305420a68d0f229d91eb3b3ed9e519fcf2cf5461da4bef997bf927e8c0bfd2b AS build
ARG VERSION=stage0
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY migrations ./migrations
COPY internal ./internal
COPY spikes/fs05-runtime ./spikes/fs05-runtime
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o /out/foliopath-runtime-spike ./spikes/fs05-runtime

FROM debian:bookworm-slim@sha256:7b140f374b289a7c2befc338f42ebe6441b7ea838a042bbd5acbfca6ec875818
ARG VERSION=stage0
LABEL org.opencontainers.image.title="FolioPath FS-05 runtime spike" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.description="Stage 0 evidence only; not a production release"
RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
       ca-certificates ffmpeg libvips42 tzdata \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd --system --gid 65532 foliopath \
    && useradd --system --uid 65532 --gid 65532 \
       --home-dir /nonexistent --shell /usr/sbin/nologin foliopath \
    && install -d -o 65532 -g 65532 -m 0750 /app/data \
    && install -d -o 65532 -g 65532 -m 0555 /library
COPY --from=build --chown=65532:65532 /out/foliopath-runtime-spike /app/foliopath-runtime-spike
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=6 \
    CMD ["/app/foliopath-runtime-spike", "healthcheck"]
ENTRYPOINT ["/app/foliopath-runtime-spike"]
CMD ["serve"]
