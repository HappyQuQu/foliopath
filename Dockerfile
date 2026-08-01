# syntax=docker/dockerfile:1.7

FROM node:22.22.0-bookworm-slim@sha256:dd9d21971ec4395903fa6143c2b9267d048ae01ca6d3ea96f16cb30df6187d94 AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --ignore-scripts
COPY web ./
RUN mkdir -p /src/internal/webassets \
    && npm run build

FROM debian:trixie-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd AS expat
RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
       build-essential cmake \
    && rm -rf /var/lib/apt/lists/*
ADD --checksum=sha256:3ad89b8588e6644bd4e49981480d48b21289eebbcd4f0a1a4afb1c29f99b6ab4 \
    https://github.com/libexpat/libexpat/releases/download/R_2_8_2/expat-2.8.2.tar.xz \
    /tmp/expat.tar.xz
RUN mkdir -p /src/expat \
    && tar --extract --file /tmp/expat.tar.xz \
       --directory /src/expat --strip-components=1 \
    && cmake -S /src/expat -B /build/expat \
       -DCMAKE_BUILD_TYPE=Release \
       -DCMAKE_INSTALL_PREFIX=/opt/expat \
       -DCMAKE_INSTALL_LIBDIR=lib \
       -DBUILD_SHARED_LIBS=ON \
       -DEXPAT_BUILD_DOCS=OFF \
       -DEXPAT_BUILD_EXAMPLES=OFF \
       -DEXPAT_BUILD_FUZZERS=OFF \
       -DEXPAT_BUILD_PKGCONFIG=ON \
       -DEXPAT_BUILD_TESTS=OFF \
       -DEXPAT_BUILD_TOOLS=OFF \
       -DEXPAT_SHARED_LIBS=ON \
    && cmake --build /build/expat --parallel "$(nproc)" \
    && cmake --install /build/expat \
    && install -D -m 0644 /src/expat/COPYING \
       /opt/expat/share/licenses/expat/COPYING \
    && install -d /pkg/DEBIAN /pkg/opt \
    && cp -a /opt/expat /pkg/opt/expat \
    && architecture=$(dpkg --print-architecture) \
    && printf '%s\n' \
       'Package: foliopath-expat' \
       'Version: 2.8.2-1' \
       "Architecture: ${architecture}" \
       'Maintainer: FolioPath release tooling' \
       'Homepage: https://github.com/libexpat/libexpat' \
       'Depends: libc6' \
       'Provides: libexpat1' \
       'Conflicts: libexpat1' \
       'Replaces: libexpat1' \
       'Description: FolioPath fixed-source Expat runtime' \
       ' Minimal shared Expat build for the image metadata runtime.' \
       >/pkg/DEBIAN/control \
    && dpkg-deb --build --root-owner-group \
       /pkg /foliopath-expat_2.8.2-1.deb

FROM debian:trixie-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd AS glib
RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
       build-essential libffi-dev libpcre2-dev meson ninja-build \
       patch pkg-config python3 zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*
ADD --checksum=sha256:ab24d24e698dfa1e408b7bcdb508f4aafc906185a8b8ce72fdf79bbbdc9b383b \
    https://download.gnome.org/sources/glib/2.88/glib-2.88.3.tar.xz \
    /tmp/glib.tar.xz
ADD --checksum=sha256:27d9d53dd539175b034f3670d0b1d1fefbe0311cf63a928895d1809e23ddfbbf \
    https://gitlab.gnome.org/GNOME/glib/-/commit/656ad4582cb1d7a7fa8bafe3ce8aec6aa3c17da0.patch \
    /tmp/glib-cve-2026-58016-assertions.patch
ADD --checksum=sha256:e16c1df025f22c106456bb8c5a45213ed37942e7f22bc40622b9f9ab54dd80b2 \
    https://gitlab.gnome.org/GNOME/glib/-/commit/c9da977c178fbfc0e4caf99f9fdf5dc433d6fcc2.patch \
    /tmp/glib-cve-2026-58016-fix.patch
COPY tests/release/glib_cve_2026_58016.c /tmp/glib-cve-2026-58016.c
RUN mkdir -p /src/glib \
    && tar --extract --file /tmp/glib.tar.xz \
       --directory /src/glib --strip-components=1 \
    && patch --directory=/src/glib --strip=1 \
       </tmp/glib-cve-2026-58016-assertions.patch \
    && sed '/^diff --git a\/gio\/tests\/gdbus-introspection.c b\/gio\/tests\/gdbus-introspection.c$/,$d' \
       /tmp/glib-cve-2026-58016-fix.patch \
       >/tmp/glib-cve-2026-58016-runtime.patch \
    && patch --directory=/src/glib --strip=1 \
       </tmp/glib-cve-2026-58016-runtime.patch \
    && meson setup /build/glib /src/glib \
       --buildtype=release \
       --prefix=/opt/glib \
       --libdir=lib \
       -Ddocumentation=false \
       -Ddtrace=disabled \
       -Dglib_debug=disabled \
       -Dinstalled_tests=false \
       -Dintrospection=disabled \
       -Dlibelf=disabled \
       -Dlibmount=disabled \
       -Dman-pages=disabled \
       -Dnls=disabled \
       -Dselinux=disabled \
       -Dsysprof=disabled \
       -Dsystemtap=disabled \
       -Dtests=false \
    && meson compile -C /build/glib \
    && meson install -C /build/glib \
    && PKG_CONFIG_PATH=/opt/glib/lib/pkgconfig \
       cc -o /tmp/glib-cve-2026-58016 \
       /tmp/glib-cve-2026-58016.c \
       $(PKG_CONFIG_PATH=/opt/glib/lib/pkgconfig \
         pkg-config --cflags --libs gio-2.0 gmodule-2.0) \
    && LD_LIBRARY_PATH=/opt/glib/lib /tmp/glib-cve-2026-58016 \
    && install -D -m 0644 /src/glib/COPYING \
       /opt/glib/share/licenses/glib/COPYING \
    && install -d /pkg/DEBIAN /pkg/opt \
    && cp -a /opt/glib /pkg/opt/glib \
    && architecture=$(dpkg --print-architecture) \
    && printf '%s\n' \
       'Package: foliopath-glib' \
       'Version: 2.88.3-1' \
       "Architecture: ${architecture}" \
       'Maintainer: FolioPath release tooling' \
       'Homepage: https://gitlab.gnome.org/GNOME/glib' \
       'Depends: libc6, libatomic1, libffi8, libpcre2-8-0, zlib1g' \
       'Description: FolioPath fixed-source minimal GLib runtime' \
       ' GLib 2.88.3 with the upstream CVE-2026-58016 fix and without libmount or SELinux integration.' \
       >/pkg/DEBIAN/control \
    && dpkg-deb --build --root-owner-group \
       /pkg /foliopath-glib_2.88.3-1.deb

FROM debian:trixie-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd AS vips
RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
       build-essential libexpat1-dev libexif-dev \
       libffi-dev libjpeg62-turbo-dev libpcre2-dev libpng-dev \
       libwebp-dev meson ninja-build pkg-config \
    && rm -rf /var/lib/apt/lists/*
COPY --from=expat /opt/expat /opt/expat
COPY --from=glib /opt/glib /opt/glib
ADD --checksum=sha256:d114d7c132ec5b45f116d654e17bb4af84561e3041183cd4bfd79abfb85cf724 \
    https://github.com/libvips/libvips/releases/download/v8.16.1/vips-8.16.1.tar.xz \
    /tmp/vips.tar.xz
RUN mkdir -p /src/vips \
    && tar --extract --file /tmp/vips.tar.xz \
       --directory /src/vips --strip-components=1 \
    && PKG_CONFIG_PATH=/opt/glib/lib/pkgconfig:/opt/expat/lib/pkgconfig \
       LD_LIBRARY_PATH=/opt/glib/lib \
       meson setup /src/vips/build /src/vips \
       --buildtype=release \
       --prefix=/opt/vips \
       --libdir=lib \
       -Dauto_features=disabled \
       -Ddeprecated=false \
       -Dexamples=false \
       -Dcplusplus=false \
       -Dmodules=disabled \
       -Dintrospection=disabled \
       -Djpeg=enabled \
       -Dpng=enabled \
       -Dwebp=enabled \
       -Dexif=enabled \
       -Dnsgif=true \
       -Dppm=false \
       -Danalyze=false \
       -Dradiance=false \
    && meson compile -C /src/vips/build \
    && meson install -C /src/vips/build \
    && install -D -m 0644 /src/vips/LICENSE \
       /opt/vips/share/licenses/libvips/LICENSE \
    && install -D -m 0644 /src/vips/libvips/foreign/libnsgif/COPYING \
       /opt/vips/share/licenses/libvips/NSGIF-COPYING \
    && install -d /pkg/DEBIAN /pkg/opt \
    && cp -a /opt/vips /pkg/opt/vips \
    && architecture=$(dpkg --print-architecture) \
    && printf '%s\n' \
       'Package: foliopath-libvips' \
       'Version: 8.16.1-1' \
       "Architecture: ${architecture}" \
       'Maintainer: FolioPath release tooling' \
       'Homepage: https://github.com/libvips/libvips' \
       'Depends: libc6, libatomic1, foliopath-expat, foliopath-glib, libexif12, libffi8, libjpeg62-turbo, libpng16-16t64, libwebp7, libwebpdemux2, libwebpmux3, zlib1g' \
       'Description: FolioPath minimal libvips runtime' \
       ' Fixed-source libvips build limited to the MVP image format contract.' \
       >/pkg/DEBIAN/control \
    && dpkg-deb --build --root-owner-group \
       /pkg /foliopath-libvips_8.16.1-1.deb

FROM debian:trixie-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd AS ffmpeg
RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
       build-essential libwebp-dev nasm pkg-config zlib1g-dev \
    && rm -rf /var/lib/apt/lists/*
ADD --checksum=sha256:e3963a50831c985933e1a625ed566ec4c7adb5c012c34fa9f84438e1d61bdacc \
    https://github.com/FFmpeg/FFmpeg/archive/refs/tags/n7.1.5.tar.gz \
    /tmp/ffmpeg.tar.gz
RUN mkdir -p /src/ffmpeg \
    && tar --extract --gzip --file /tmp/ffmpeg.tar.gz \
       --directory /src/ffmpeg --strip-components=1 \
    && cd /src/ffmpeg \
    && ./configure \
       --prefix=/opt/ffmpeg-build \
       --disable-autodetect \
       --disable-debug \
       --disable-doc \
       --disable-everything \
       --disable-network \
       --enable-ffmpeg \
       --enable-ffprobe \
       --enable-libwebp \
       --enable-zlib \
       --enable-protocol=file,pipe \
       --enable-demuxer=mov,matroska,avi,image2 \
       --enable-decoder=h264,hevc,mpeg4,mjpeg,webp,png,gif,ffv1,vp8,vp9,av1,prores,mpeg1video,mpeg2video,theora,vc1 \
       --enable-parser=h264,hevc,mpeg4video,mpegvideo,vp8,vp9,av1 \
       --enable-filter=scale,setsar,xstack \
       --enable-encoder=libwebp,png \
       --enable-muxer=image2pipe \
    && make -j"$(nproc)" \
    && make install \
    && install -D -m 0755 /opt/ffmpeg-build/bin/ffmpeg \
       /pkg/opt/ffmpeg/bin/ffmpeg \
    && install -D -m 0755 /opt/ffmpeg-build/bin/ffprobe \
       /pkg/opt/ffmpeg/bin/ffprobe \
    && install -D -m 0644 LICENSE.md \
       /pkg/opt/ffmpeg/share/licenses/ffmpeg/LICENSE.md \
    && install -D -m 0644 COPYING.LGPLv2.1 \
       /pkg/opt/ffmpeg/share/licenses/ffmpeg/COPYING.LGPLv2.1 \
    && install -D -m 0644 COPYING.LGPLv3 \
       /pkg/opt/ffmpeg/share/licenses/ffmpeg/COPYING.LGPLv3 \
    && install -d /pkg/DEBIAN \
    && architecture=$(dpkg --print-architecture) \
    && printf '%s\n' \
       'Package: foliopath-ffmpeg' \
       'Version: 7.1.5-2' \
       "Architecture: ${architecture}" \
       'Maintainer: FolioPath release tooling' \
       'Homepage: https://ffmpeg.org/' \
       'Depends: libc6, libwebp7, zlib1g' \
       'Description: FolioPath minimal FFmpeg runtime' \
       ' Fixed-source FFmpeg build limited to the MVP video processing contract.' \
       >/pkg/DEBIAN/control \
    && dpkg-deb --build --root-owner-group \
       /pkg /foliopath-ffmpeg_7.1.5-2.deb

FROM golang:1.26.5-trixie@sha256:4ee9ffa999b4583ce281939cdff828763083610292f252279a0cee77473bd9a7 AS build
ARG VERSION=stage5-candidate
WORKDIR /src
RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
       libexpat1-dev libexif-dev libjpeg62-turbo-dev \
       libffi-dev libpcre2-dev libpng-dev libwebp-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*
COPY --from=vips /opt/vips /opt/vips
COPY --from=expat /opt/expat /opt/expat
COPY --from=glib /opt/glib /opt/glib
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY migrations ./migrations
COPY --from=web /src/internal/webassets/dist ./internal/webassets/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    PKG_CONFIG_PATH=/opt/vips/lib/pkgconfig:/opt/glib/lib/pkgconfig:/opt/expat/lib/pkgconfig \
    LD_LIBRARY_PATH=/opt/glib/lib \
    CGO_LDFLAGS="-L/opt/vips/lib -L/opt/glib/lib -L/opt/expat/lib" \
    CGO_ENABLED=1 go build -tags=libvips -trimpath \
      -ldflags="-s -w -X main.version=${VERSION}" \
      -o /out/foliopath ./cmd/foliopath

FROM debian:trixie-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd AS runtime-assemble
RUN apt-get update \
    && apt-get install --yes --no-install-recommends \
       libatomic1 libexpat1 libexif12 libffi8 \
       libjpeg62-turbo libpng16-16t64 libwebp7 libwebpdemux2 \
       libwebpmux3 libpcre2-8-0 zlib1g \
    && rm -rf /var/lib/apt/lists/*
COPY --from=vips /foliopath-libvips_8.16.1-1.deb /tmp/foliopath-libvips.deb
COPY --from=ffmpeg /foliopath-ffmpeg_7.1.5-2.deb /tmp/foliopath-ffmpeg.deb
COPY --from=expat /foliopath-expat_2.8.2-1.deb /tmp/foliopath-expat.deb
COPY --from=glib /foliopath-glib_2.88.3-1.deb /tmp/foliopath-glib.deb
RUN dpkg --auto-deconfigure --install \
       /tmp/foliopath-expat.deb \
       /tmp/foliopath-glib.deb \
       /tmp/foliopath-libvips.deb /tmp/foliopath-ffmpeg.deb \
    && rm /tmp/foliopath-expat.deb \
       /tmp/foliopath-glib.deb \
       /tmp/foliopath-libvips.deb /tmp/foliopath-ffmpeg.deb
COPY --from=build /out/foliopath /app/foliopath
RUN set -eu; \
    mkdir -p /rootfs/app/data /rootfs/library /rootfs/opt \
      /rootfs/usr/local/bin /rootfs/var/lib/dpkg/status.d; \
    for package in \
      libatomic1 \
      libexif12 \
      libffi8 \
      libjpeg62-turbo \
      libpcre2-8-0 \
      libpng16-16t64 \
      libsharpyuv0 \
      libwebp7 \
      libwebpdemux2 \
      libwebpmux3 \
      zlib1g; do \
        dpkg-query --status "${package}" \
          >"/rootfs/var/lib/dpkg/status.d/${package}"; \
        dpkg-query --listfiles "${package}" | while IFS= read -r path; do \
          if [ -f "${path}" ] || [ -L "${path}" ]; then \
            cp -a --parents "${path}" /rootfs; \
          fi; \
        done; \
    done; \
    cp -a /app/foliopath /rootfs/app/foliopath; \
    cp -a /opt/expat /rootfs/opt/expat; \
    cp -a /opt/ffmpeg /rootfs/opt/ffmpeg; \
    cp -a /opt/glib /rootfs/opt/glib; \
    cp -a /opt/vips /rootfs/opt/vips; \
    dpkg-query --status foliopath-ffmpeg \
      >/rootfs/var/lib/dpkg/status.d/foliopath-ffmpeg; \
    dpkg-query --status foliopath-expat \
      >/rootfs/var/lib/dpkg/status.d/foliopath-expat; \
    dpkg-query --status foliopath-glib \
      >/rootfs/var/lib/dpkg/status.d/foliopath-glib; \
    dpkg-query --status foliopath-libvips \
      >/rootfs/var/lib/dpkg/status.d/foliopath-libvips; \
    ln -s /opt/ffmpeg/bin/ffmpeg /rootfs/usr/local/bin/ffmpeg; \
    ln -s /opt/ffmpeg/bin/ffprobe /rootfs/usr/local/bin/ffprobe; \
    chmod 0750 /rootfs/app/data; \
    chmod 0555 /rootfs/library

FROM gcr.io/distroless/cc-debian13@sha256:d97bc0a941b8d4be647dc0ee75b264ddbb772f1ac5ba690a4309c00723b23775
ARG VERSION=stage5-candidate
LABEL org.opencontainers.image.title="FolioPath" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.description="Stage 5 release candidate; not a stable release"
COPY --from=runtime-assemble /rootfs/ /
ENV LD_LIBRARY_PATH=/opt/vips/lib:/opt/glib/lib:/opt/expat/lib \
    FOLIOPATH_LISTEN=0.0.0.0:8080 \
    TZ=UTC
USER 0:0
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=6 \
    CMD ["/app/foliopath", "healthcheck"]
ENTRYPOINT ["/app/foliopath"]
CMD ["serve"]
