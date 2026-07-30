# UIF-406 complete repository verification

## Scope

UIF-406 runs the complete verification surface required by the feature task
list after the production UI, browser/accessibility and capacity slices:

```sh
make fmt
make arch-check
make generate-check
make lint
make test
make test-integration
make test-e2e
```

The commands were executed from the repository root on macOS 26.6/arm64. The
container smoke used Docker Desktop engine 29.6.2 and the repository
`Dockerfile`.

## Results

| Command | Result |
| --- | --- |
| `make fmt` | Passed; `gofmt` produced no tracked source difference. |
| `make arch-check` | Passed: `tests/architecture`. |
| `make generate-check` | Passed: SQLC source check and generated TypeScript client check. |
| `make lint` | Passed: architecture and OpenAPI contract tests plus `go vet ./...`. |
| `make test` | Passed: all Go packages. |
| `make test-integration` | Passed: `tests/integration`. |
| `make test-e2e` | Passed: production image build and application container runtime smoke printed `application container smoke passed`. |

`git diff --check` also passed, and the worktree contained no changes after the
verification commands.

## Docker infrastructure recovery

The first `make test-e2e` attempt did not reach product assertions. Docker
Desktop's containerd metadata returned an I/O error left by the earlier host
disk-pressure incident. A normal Docker restart confirmed the VM still had
EXT4/containerd I/O failures.

Inspection found approximately 15 GiB of duplicated, ignored
`build/**/.trivy-cache` vulnerability-database downloads. Only those
reconstructible cache directories were removed:

- committed and ignored source files were untouched;
- existing SBOM, vulnerability report, notice and release evidence files were
  retained;
- no Docker factory reset, data purge, volume deletion or image deletion was
  performed;
- no media or application data was touched.

Host free space increased from approximately 4.2 GiB to 19 GiB. After a normal
Docker Desktop restart, the daemon API recovered and the original
`make test-e2e` command passed without changing the test or product code.

## Conclusion

UIF-406 is complete. Every command listed by the feature task ran
successfully against the current branch. This verification does not itself
sign UIF-S4 or replace the documentation reconciliation and affected Stage 5
Gate reruns owned by UIF-407 and UIF-408.
