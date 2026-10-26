# syntax=docker/dockerfile:1.7

ARG BASE_IMAGE
FROM ${BASE_IMAGE}

ARG ORT_ARCHIVE_URL
ARG ORT_ARCHIVE_SHA256
ARG ORT_COMMIT=da9b5e364c465de65c49d91e696cd6485270757f

ADD ${ORT_ARCHIVE_URL} /tmp/onnxruntime.tgz
RUN echo "${ORT_ARCHIVE_SHA256}  /tmp/onnxruntime.tgz" | sha256sum --check --strict - \
    && mkdir -p /opt/onnxruntime \
    && tar --extract --gzip --file /tmp/onnxruntime.tgz \
       --directory /opt/onnxruntime --strip-components=1 \
    && test "$(cat /opt/onnxruntime/VERSION_NUMBER)" = "1.28.0" \
    && test "$(cat /opt/onnxruntime/GIT_COMMIT_ID)" = "${ORT_COMMIT}" \
    && rm /tmp/onnxruntime.tgz

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    PKG_CONFIG_PATH=/opt/vips/lib/pkgconfig:/opt/glib/lib/pkgconfig:/opt/expat/lib/pkgconfig \
    LD_LIBRARY_PATH=/opt/vips/lib:/opt/glib/lib:/opt/expat/lib \
    CGO_CFLAGS="-I/opt/onnxruntime/include" \
    CGO_LDFLAGS="-L/opt/onnxruntime/lib -Wl,-rpath,/opt/onnxruntime/lib -lonnxruntime -L/opt/vips/lib -L/opt/glib/lib -L/opt/expat/lib" \
    CGO_ENABLED=1 go test -c -trimpath -tags "libvips onnxruntime" \
      -o /out/face-candidate.test ./internal/inference/faceonnx

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go test -c -trimpath \
      -o /out/face-capacity.test ./internal/face

ENV LD_LIBRARY_PATH=/opt/onnxruntime/lib:/opt/vips/lib:/opt/glib/lib:/opt/expat/lib
ENTRYPOINT ["/out/face-candidate.test"]
