# syntax=docker/dockerfile:1.7

FROM debian:trixie-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd
RUN apt-get update \
    && apt-get install --yes --no-install-recommends ffmpeg \
    && rm -rf /var/lib/apt/lists/*
COPY generate-release-fixtures.sh /usr/local/bin/generate-release-fixtures
ENTRYPOINT ["/usr/local/bin/generate-release-fixtures"]
