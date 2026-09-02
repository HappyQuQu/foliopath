GO ?= go
NPM ?= npm
PYTHON ?= python3
OASDIFF_VERSION ?= v1.17.0
SQLC_VERSION ?= v1.31.1
GO_FILES := $(shell rg --files -g '*.go')

.PHONY: fmt fmt-check arch-check release-docs-check release-readiness-check release-ready storyboard-readiness-check storyboard-ready verify-release-image-evidence verify-supply-chain-evidence verify-storyboard-evidence verify-intelligent-media-native-evidence verify-intelligent-media-native-model-evidence verify-intelligent-media-quality verify-intelligent-media-face-quality verify-intelligent-media-supply-chain verify-intelligent-media-s2-evidence contract-check compatibility-check generate generate-sql generate-check generate-sql-check web-check openapi-lint lint test test-race test-libvips test-integration test-e2e test-web-e2e test-web-release-e2e test-web-chrome-stable test-browser-capacity test-storyboard-browser-capacity test-face-capacity test-release-image test-release-upgrade test-release-capacity test-intelligent-media-offline test-intelligent-media-privacy test-storyboard-runtime test-storyboard-vertical release-capacity spike-ai spike-capacity spike-vips spike-runtime sbom provenance release-notices scan-release-image capacity-trend

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

verify-intelligent-media-native-evidence:
	@test -n "$(EVIDENCE_DIR)" || (echo "EVIDENCE_DIR is required" >&2; exit 2)
	@test -n "$(RELEASE_SHA)" || (echo "RELEASE_SHA is required" >&2; exit 2)
	@test -n "$(WORKFLOW_RUN_ID)" || (echo "WORKFLOW_RUN_ID is required" >&2; exit 2)
	@test -n "$(WORKFLOW_RUN_ATTEMPT)" || (echo "WORKFLOW_RUN_ATTEMPT is required" >&2; exit 2)
	$(GO) run ./tests/release/intelligent_media_native_evidence \
		-dir "$(EVIDENCE_DIR)" -commit "$(RELEASE_SHA)" \
		-run-id "$(WORKFLOW_RUN_ID)" -run-attempt "$(WORKFLOW_RUN_ATTEMPT)" \
		-output "$(SUMMARY_FILE)"

verify-intelligent-media-native-model-evidence:
	@test -n "$(EVIDENCE_DIR)" || (echo "EVIDENCE_DIR is required" >&2; exit 2)
	@test -n "$(RELEASE_SHA)" || (echo "RELEASE_SHA is required" >&2; exit 2)
	@test -n "$(WORKFLOW_RUN_ID)" || (echo "WORKFLOW_RUN_ID is required" >&2; exit 2)
	@test -n "$(WORKFLOW_RUN_ATTEMPT)" || (echo "WORKFLOW_RUN_ATTEMPT is required" >&2; exit 2)
	$(GO) run ./tests/release/intelligent_media_native_evidence \
		-dir "$(EVIDENCE_DIR)" -commit "$(RELEASE_SHA)" \
		-run-id "$(WORKFLOW_RUN_ID)" -run-attempt "$(WORKFLOW_RUN_ATTEMPT)" \
		-require-model -output "$(SUMMARY_FILE)"

verify-intelligent-media-quality:
	@test -n "$(QUALITY_INPUT)" || (echo "QUALITY_INPUT is required" >&2; exit 2)
	@test -n "$(DATASET_MANIFEST)" || (echo "DATASET_MANIFEST is required" >&2; exit 2)
	@test -n "$(RELEASE_SHA)" || (echo "RELEASE_SHA is required" >&2; exit 2)
	cd spikes/int001-ai && $(GO) run . quality-score \
		-input "$(abspath $(QUALITY_INPUT))" \
		-dataset-manifest "$(abspath $(DATASET_MANIFEST))" \
		-commit "$(RELEASE_SHA)" \
		$(if $(SUMMARY_FILE),-output "$(abspath $(SUMMARY_FILE))",)

verify-intelligent-media-face-quality:
	@test -n "$(FACE_QUALITY_INPUT)" || (echo "FACE_QUALITY_INPUT is required" >&2; exit 2)
	@test -n "$(DATASET_MANIFEST)" || (echo "DATASET_MANIFEST is required" >&2; exit 2)
	@test -n "$(RELEASE_SHA)" || (echo "RELEASE_SHA is required" >&2; exit 2)
	cd spikes/int001-ai && $(GO) run . face-quality-score \
		-input "$(abspath $(FACE_QUALITY_INPUT))" \
		-dataset-manifest "$(abspath $(DATASET_MANIFEST))" \
		-commit "$(RELEASE_SHA)" \
		$(if $(SUMMARY_FILE),-output "$(abspath $(SUMMARY_FILE))",)

verify-intelligent-media-supply-chain:
	@test -n "$(SUPPLY_CHAIN_INPUT)" || (echo "SUPPLY_CHAIN_INPUT is required" >&2; exit 2)
	@test -n "$(RELEASE_SHA)" || (echo "RELEASE_SHA is required" >&2; exit 2)
	$(GO) run ./tests/release/intelligent_media_supplychain_evidence \
		-input "$(SUPPLY_CHAIN_INPUT)" -commit "$(RELEASE_SHA)" \
		-output "$(SUMMARY_FILE)"

verify-intelligent-media-s2-evidence:
	@test -n "$(QUALITY_SUMMARY)" || (echo "QUALITY_SUMMARY is required" >&2; exit 2)
	@test -n "$(FACE_QUALITY_SUMMARY)" || (echo "FACE_QUALITY_SUMMARY is required" >&2; exit 2)
	@test -n "$(NATIVE_SUMMARY)" || (echo "NATIVE_SUMMARY is required" >&2; exit 2)
	@test -n "$(SUPPLY_CHAIN_SUMMARY)" || (echo "SUPPLY_CHAIN_SUMMARY is required" >&2; exit 2)
	@test -n "$(RELEASE_SHA)" || (echo "RELEASE_SHA is required" >&2; exit 2)
	$(GO) run ./tests/release/intelligent_media_s2_evidence \
		-quality "$(QUALITY_SUMMARY)" -face-quality "$(FACE_QUALITY_SUMMARY)" -native "$(NATIVE_SUMMARY)" \
		-supply-chain "$(SUPPLY_CHAIN_SUMMARY)" -commit "$(RELEASE_SHA)" \
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

test-face-capacity:
	FOLIOPATH_RUN_CAPACITY_TEST=1 $(GO) test ./internal/face \
		-run '^TestClusterFaces100KCapacity$$' -count=1 -v

test-release-image:
	tests/release/image_smoke.sh

test-release-upgrade:
	@test -n "$(PREVIOUS_IMAGE)" || (echo "PREVIOUS_IMAGE is required" >&2; exit 2)
	@test -n "$(IMAGE)" || (echo "IMAGE is required" >&2; exit 2)
	tests/release/upgrade_rollback_smoke.sh "$(PREVIOUS_IMAGE)" "$(IMAGE)"

test-release-capacity:
	tests/release/capacity_smoke.sh

test-intelligent-media-offline:
	tests/release/intelligent_media_offline_smoke.sh

test-intelligent-media-privacy:
	$(GO) test -count=1 ./internal/app \
		-run '^TestJSONLoggerProducesStructuredEventsAndRedactsSensitiveAttributes$$'
	$(GO) test -count=1 ./internal/api \
		-run '^Test(FaceReadRoutesUsePrivacySafeWireDTOs|MediaDiagnosticsHTTP(ReturnsSafeAttemptHistory|ListsSafeFailureDetails)|SemanticSearchMasksQueryFromErrorsAndRequestLogs)$$'
	$(GO) test -count=1 ./internal/store/sqlite \
		-run '^Test(DerivedFaceClearPreservesManualStateAndOriginalAssetMetadata|ManualFaceClearRequiresExactImpactAndRetainsDerivedAndPeople|SecureDeleteRemovesDeletedPayloadFromLiveDatabaseFiles)$$'
	$(GO) test -count=1 ./tests/architecture \
		-run '^TestFacePrivacyProjectionRemainsClosed$$'

test-storyboard-runtime:
	tests/release/storyboard_ffmpeg_smoke.sh

test-storyboard-vertical:
	tests/release/storyboard_vertical_smoke.sh

release-capacity:
	FOLIOPATH_CAPACITY=1 FOLIOPATH_CAPACITY_ENFORCE_BUDGET=1 \
		tests/release/capacity_smoke.sh

spike-ai:
	cd spikes/int001-ai && $(GO) test ./...
	cd spikes/int001-ai && $(PYTHON) -m unittest face_functional_smoke_test.py face_arcface_functional_smoke_test.py
	$(GO) test ./spikes/int001-model-package-v2
	$(GO) test ./spikes/int001-vips-input

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
