# Local test runner only, not a deployment service. Build the repository's
# foliopath-web-check:local image first (pinned Node and package lock).
FROM foliopath-web-check:local
RUN npx playwright install --with-deps chromium firefox webkit
