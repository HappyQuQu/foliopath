GO ?= go
NPM ?= npm
OASDIFF_VERSION ?= v1.17.0
SQLC_VERSION ?= v1.31.1
GO_FILES := $(shell rg --files -g '*.go')

.PHONY: fmt fmt-check arch-check release-docs-check release-readiness-check release-ready storyboard-readiness-check storyboard-ready verify-release-image-evidence verify-supply-chain-evidence verify-storyboard-evidence contract-check compatibility-check generate generate-sql generate-check generate-sql-check web-check openapi-lint lint test test-race test-libvips test-integration test-e2e test-web-e2e test-web-release-e2e test-web-chrome-stable test-browser-capacity test-storyboard-browser-capacity test-release-image test-release-upgrade test-release-capacity test-storyboard-runtime test-storyboard-vertical release-capacity spike-capacity spike-vips spike-runtime sbom provenance release-notices scan-release-image capacity-trend

fmt:
	gofmt -w $(GO_FILES)

fmt-check:
	@test -z "$$(gofmt -l $(GO_FILES))"

arch-check:
	$(GO) test ./tests/architecture/...

release-docs-check:
	$(GO) test -run '^TestReleaseDocumentationMatchesCandidateBoundaries$$' \
		./tests/architecture/...

release-readiness-check:
	$(GO) test -run '^TestReleaseReadinessManifestFailsClosed$$' \
		./tests/architecture/...

release-ready:
	FOLIOPATH_REQUIRE_RELEASE_GO=1 $(GO) test -count=1 \
		-run '^TestReleaseReadinessManifestFailsClosed$$' \
		./tests/architecture/...

storyboard-readiness-check:
	$(GO) test -run '^TestStoryboardReadinessManifestFailsClosed$$' \
		./tests/architecture/...

storyboard-ready:
	FOLIOPATH_REQUIRE_STORYBOARD_GO=1 $(GO) test -count=1 \
		-run '^TestStoryboardReadinessManifestFailsClosed$$' \
		./tests/architecture/...

verify-release-image-evidence:
	@test -n "$(EVIDENCE_DIR)" || (echo "EVIDENCE_DIR is required" >&2; exit 2)
	@test -n "$(RELEASE_SHA)" || (echo "RELEASE_SHA is required" >&2; exit 2)
	$(GO) run ./tests/release/evidence \
		-dir "$(EVIDENCE_DIR)" -commit "$(RELEASE_SHA)"

verify-supply-chain-evidence:
	@test -n "$(EVIDENCE_DIR)" || (echo "EVIDENCE_DIR is required" >&2; exit 2)
	@test -n "$(RELEASE_SHA)" || (echo "RELEASE_SHA is required" >&2; exit 2)
	$(GO) run ./tests/release/supplychain_evidence \
		-dir "$(EVIDENCE_DIR)" -commit "$(RELEASE_SHA)" \
		-output "$(SUMMARY_FILE)"

verify-storyboard-evidence:
	@test -n "$(EVIDENCE_DIR)" || (echo "EVIDENCE_DIR is required" >&2; exit 2)
	@test -n "$(RELEASE_SHA)" || (echo "RELEASE_SHA is required" >&2; exit 2)
	$(GO) run ./tests/release/storyboard_evidence \
		-dir "$(EVIDENCE_DIR)" -commit "$(RELEASE_SHA)" \
		-output "$(SUMMARY_FILE)"

contract-check:
	$(GO) test -count=1 ./tests/contract/...

compatibility-check:
	@test -n "$(OPENAPI_BASELINE)" || (echo "OPENAPI_BASELINE is required" >&2; exit 2)
	$(GO) run github.com/oasdiff/oasdiff@$(OASDIFF_VERSION) breaking \
		--fail-on WARN "$(OPENAPI_BASELINE)" api/openapi.yaml

generate: generate-sql
	cd web && $(NPM) run generate:api

generate-sql:
	$(GO) run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate \
		-f internal/store/sqlite/sqlc.yaml

generate-check: generate-sql-check
	cd web && $(NPM) run generate:check

generate-sql-check:
	GO=$(GO) SQLC_VERSION=$(SQLC_VERSION) scripts/check-sqlc.sh

web-check:
	cd web && $(NPM) run check

openapi-lint:
	cd web && $(NPM) run lint:openapi

lint: arch-check contract-check
	$(GO) vet ./...

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

test-libvips:
	docker build --target libvips-test --progress plain .

test-integration:
	$(GO) test ./tests/integration/...

test-e2e:
	tests/e2e/runtime_smoke.sh

test-web-e2e:
	tests/e2e/web_auth.sh

test-web-release-e2e:
	FOLIOPATH_E2E_SUITE=release tests/e2e/web_auth.sh

test-web-chrome-stable:
	FOLIOPATH_E2E_SUITE=chrome-stable tests/e2e/web_auth.sh

test-browser-capacity:
	cd web && npm run build:storybook
	cd web && FOLIOPATH_BROWSER_CAPACITY_ENFORCE=1 npm run test:capacity

test-storyboard-browser-capacity:
	cd web && npm run build:storybook
	cd web && npm run test:storyboard-capacity

test-release-image:
	tests/release/image_smoke.sh

test-release-upgrade:
	@test -n "$(PREVIOUS_IMAGE)" || (echo "PREVIOUS_IMAGE is required" >&2; exit 2)
	@test -n "$(IMAGE)" || (echo "IMAGE is required" >&2; exit 2)
	tests/release/upgrade_rollback_smoke.sh "$(PREVIOUS_IMAGE)" "$(IMAGE)"

test-release-capacity:
	tests/release/capacity_smoke.sh

test-storyboard-runtime:
	tests/release/storyboard_ffmpeg_smoke.sh

test-storyboard-vertical:
	tests/release/storyboard_vertical_smoke.sh

release-capacity:
	FOLIOPATH_CAPACITY=1 FOLIOPATH_CAPACITY_ENFORCE_BUDGET=1 \
		tests/release/capacity_smoke.sh

spike-capacity:
	FOLIOPATH_CAPACITY=1 FOLIOPATH_CAPACITY_ENFORCE_BUDGET=1 GOMAXPROCS=4 \
		$(GO) test -timeout=20m -count=1 \
		-run '^Test(CapacityBaseline|DirectoryRollupDeepChainBaseline)$$' \
		-v ./tests/performance

spike-vips:
	cd spikes/fs03-vips && MALLOC_ARENA_MAX=2 timeout 2m $(GO) test -count=1 -v ./...

spike-runtime:
	docker build -f spikes/fs05-runtime/Dockerfile \
		-t foliopath-fs05:local --build-arg VERSION=stage0-local .
	spikes/fs05-runtime/verify.sh foliopath-fs05:local

sbom:
	@test -n "$(IMAGE)" || (echo "IMAGE is required" >&2; exit 2)
	scripts/generate-sbom.sh "$(IMAGE)" "$${SBOM_OUTPUT:-build/sbom}"

provenance:
	@test -n "$(IMAGE)" || (echo "IMAGE is required" >&2; exit 2)
	@test -n "$(OUTPUT)" || (echo "OUTPUT is required" >&2; exit 2)
	scripts/generate-provenance.sh "$(IMAGE)" "$(OUTPUT)"

release-notices:
	@test -n "$(IMAGE)" || (echo "IMAGE is required" >&2; exit 2)
	scripts/collect-release-notices.sh "$(IMAGE)" "$${NOTICES_OUTPUT:-build/notices}"

scan-release-image:
	@test -n "$(IMAGE)" || (echo "IMAGE is required" >&2; exit 2)
	scripts/scan-release-image.sh "$(IMAGE)" "$${SCAN_OUTPUT:-build/security}"

capacity-trend:
	@set -e; for tier in "1000 10000" "5000 50000" "10000 100000"; do \
		set -- $$tier; \
		FOLIOPATH_CAPACITY=1 FOLIOPATH_CAPACITY_ENFORCE_BUDGET=1 \
		FOLIOPATH_CAPACITY_DIRS=$$1 FOLIOPATH_CAPACITY_ASSETS=$$2 \
		GOMAXPROCS=4 $(GO) test -timeout=20m -count=1 \
		-run '^TestCapacityBaseline$$' -v ./tests/performance; \
	done
