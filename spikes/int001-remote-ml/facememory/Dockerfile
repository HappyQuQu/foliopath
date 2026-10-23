# Build from already loaded, pinned local images; no model/media download.
ARG BUILD_IMAGE=foliopath:local-face-diagnostic-builder
ARG RUNTIME_IMAGE=foliopath:local-face-retry-20261023
FROM ${BUILD_IMAGE} AS build
COPY go.mod go.sum /src/
COPY internal/media/imagevips/ /src/internal/media/imagevips/
COPY spikes/int001-remote-ml/facememory/main.go /src/spikes/int001-remote-ml/facememory/main.go
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build PKG_CONFIG_PATH=/opt/vips/lib/pkgconfig:/opt/glib/lib/pkgconfig:/opt/expat/lib/pkgconfig LD_LIBRARY_PATH=/opt/glib/lib CGO_LDFLAGS="-L/opt/vips/lib -L/opt/glib/lib -L/opt/expat/lib" CGO_ENABLED=1 go build -tags=libvips -o /out/face-memory /src/spikes/int001-remote-ml/facememory/main.go
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build PKG_CONFIG_PATH=/opt/vips/lib/pkgconfig:/opt/glib/lib/pkgconfig:/opt/expat/lib/pkgconfig LD_LIBRARY_PATH=/opt/glib/lib CGO_LDFLAGS="-L/opt/vips/lib -L/opt/glib/lib -L/opt/expat/lib" CGO_ENABLED=1 go test -c -tags=libvips -o /out/imagevips.test ./internal/media/imagevips
FROM ${RUNTIME_IMAGE}
COPY --from=build /out/face-memory /app/face-memory
COPY --from=build /out/imagevips.test /app/imagevips.test
ENTRYPOINT ["/app/face-memory"]
