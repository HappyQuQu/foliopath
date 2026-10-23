# Local test runner only, not a deployment service. Build the repository's
# foliopath-web-check:local image first (pinned Node and package lock).
FROM foliopath-web-check:local
COPY .node-version /tmp/foliopath-node-version
RUN test "$(node --version)" = "v$(cat /tmp/foliopath-node-version)" \
    && npx playwright install --with-deps chromium firefox webkit
