# Diagnostic candidate only; reuse verified runtime/frontend, replace Go binary.
# Binary must be built for the base image architecture from recorded source.
ARG BASE_IMAGE
FROM ${BASE_IMAGE}
COPY foliopath /app/foliopath
