# syntax=docker/dockerfile:1.7

# This image is an arm64-only INT-S0 evidence fixture. It is deliberately
# separate from FolioPath's production Dockerfile and dependency graph.
FROM golang:1.26.5-trixie@sha256:4ee9ffa999b4583ce281939cdff828763083610292f252279a0cee77473bd9a7 AS build

ADD --checksum=sha256:e15ff8b5d85afe6c144d97c6fd432254bf76a219daaf17658087d6ecb3e8f0bb \
    https://github.com/microsoft/onnxruntime/releases/download/v1.28.0/onnxruntime-linux-aarch64-1.28.0.tgz \
    /tmp/onnxruntime.tgz
RUN mkdir -p /opt/onnxruntime \
    && tar --extract --gzip --file /tmp/onnxruntime.tgz \
       --directory /opt/onnxruntime --strip-components=1 \
    && test "$(cat /opt/onnxruntime/VERSION_NUMBER)" = "1.28.0" \
    && test "$(cat /opt/onnxruntime/GIT_COMMIT_ID)" = \
       "da9b5e364c465de65c49d91e696cd6485270757f"

WORKDIR /src
COPY spikes/int001-ort-capi/go.mod ./
COPY spikes/int001-ort-capi/main.go ./
RUN CGO_ENABLED=1 \
    CGO_CFLAGS="-I/opt/onnxruntime/include" \
    CGO_LDFLAGS="-L/opt/onnxruntime/lib -lonnxruntime" \
    go build -trimpath -ldflags="-s -w" -o /out/int001-ort-capi .

FROM gcr.io/distroless/cc-debian13:nonroot@sha256:b5c859c8f9538074fa248ea12aad3552d29fb437d2b546ea65504b4af125e59f
LABEL org.opencontainers.image.title="FolioPath INT-008 ORT C API spike" \
      org.opencontainers.image.version="onnxruntime-1.28.0-arm64" \
      org.opencontainers.image.description="Isolated evidence image; not a FolioPath production image"
COPY --from=build /opt/onnxruntime/lib/libonnxruntime.so.1.28.0 /opt/onnxruntime/lib/libonnxruntime.so.1.28.0
COPY --from=build /opt/onnxruntime/lib/libonnxruntime.so.1 /opt/onnxruntime/lib/libonnxruntime.so.1
COPY --from=build /opt/onnxruntime/LICENSE /usr/share/licenses/onnxruntime/LICENSE
COPY --from=build /opt/onnxruntime/ThirdPartyNotices.txt /usr/share/licenses/onnxruntime/ThirdPartyNotices.txt
COPY --from=build /out/int001-ort-capi /app/int001-ort-capi
ENV LD_LIBRARY_PATH=/opt/onnxruntime/lib
USER 65532:65532
ENTRYPOINT ["/app/int001-ort-capi"]
