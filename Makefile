GO ?= go
NPM ?= npm
OASDIFF_VERSION ?= v1.17.0
SQLC_VERSION ?= v1.31.1
GO_FILES := $(shell rg --files -g '*.go')

.PHONY: fmt fmt-check arch-check contract-check compatibility-check generate generate-sql generate-check generate-sql-check web-check openapi-lint lint test test-race test-integration spike-capacity spike-vips spike-runtime sbom capacity-trend

fmt:
	gofmt -w $(GO_FILES)

fmt-check:
	@test -z "$$(gofmt -l $(GO_FILES))"

arch-check:
	$(GO) test ./tests/architecture/...

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

test-integration:
	$(GO) test ./tests/integration/...

spike-capacity:
	FOLIOPATH_CAPACITY=1 GOMAXPROCS=4 $(GO) test -timeout=20m -count=1 -run '^Test(CapacityBaseline|DirectoryRollupDeepChainBaseline)$$' -v ./tests/performance

spike-vips:
	cd spikes/fs03-vips && MALLOC_ARENA_MAX=2 timeout 2m $(GO) test -count=1 -v ./...

spike-runtime:
	docker build -f spikes/fs05-runtime/Dockerfile \
		-t foliopath-fs05:local --build-arg VERSION=stage0-local .
	spikes/fs05-runtime/verify.sh foliopath-fs05:local

sbom:
	@test -n "$(IMAGE)" || (echo "IMAGE is required" >&2; exit 2)
	scripts/generate-sbom.sh "$(IMAGE)" "$${SBOM_OUTPUT:-build/sbom}"

capacity-trend:
	@set -e; for tier in "1000 10000" "5000 50000" "10000 100000"; do \
		set -- $$tier; \
		FOLIOPATH_CAPACITY=1 FOLIOPATH_CAPACITY_ENFORCE_BUDGET=1 \
		FOLIOPATH_CAPACITY_DIRS=$$1 FOLIOPATH_CAPACITY_ASSETS=$$2 \
		GOMAXPROCS=4 $(GO) test -timeout=20m -count=1 \
		-run '^TestCapacityBaseline$$' -v ./tests/performance; \
	done
