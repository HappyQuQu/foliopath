# Local incremental candidate, not a canonical full-release build.
FROM foliopath:local-face-diagnostic-builder@sha256:41cdae585410a8ad7fa5dc18d8f0940c3825343233ee3973a189fe903bc428b3 AS build
COPY go.mod go.sum /src/
COPY internal/media/imagevips/ /src/internal/media/imagevips/
COPY internal/inference/remoteml/ /src/internal/inference/remoteml/
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build PKG_CONFIG_PATH=/opt/vips/lib/pkgconfig:/opt/glib/lib/pkgconfig:/opt/expat/lib/pkgconfig LD_LIBRARY_PATH=/opt/glib/lib CGO_LDFLAGS="-L/opt/vips/lib -L/opt/glib/lib -L/opt/expat/lib" CGO_ENABLED=1 go build -tags=libvips -trimpath -ldflags="-s -w -X main.version=stage5-candidate" -o /out/foliopath ./cmd/foliopath
FROM foliopath:local-face-memory-20261023@sha256:c9f28d7a6f4deb1d7d83bc7b73f0fd45e41609d546ac77a3b9c9543b652d9ec2
COPY --from=build /out/foliopath /app/foliopath
