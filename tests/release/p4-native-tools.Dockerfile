FROM golang:1.26.5-trixie@sha256:4ee9ffa999b4583ce281939cdff828763083610292f252279a0cee77473bd9a7
# Isolated acceptance driver, never a production deployment unit.
RUN apt-get update \
 && apt-get install --yes --no-install-recommends nodejs ffmpeg jq docker-cli \
 && docker --version \
    && rm -rf /var/lib/apt/lists/*
