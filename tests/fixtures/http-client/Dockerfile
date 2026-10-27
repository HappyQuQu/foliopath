# syntax=docker/dockerfile:1.7

# Test-only client. The production image intentionally contains no general HTTP CLI.
FROM debian:trixie-slim@sha256:020c0d20b9880058cbe785a9db107156c3c75c2ac944a6aa7ab59f2add76a7bd
RUN apt-get update \
    && apt-get install --yes --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*
ENTRYPOINT ["sleep", "infinity"]
