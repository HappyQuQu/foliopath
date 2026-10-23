# Activation final availability-revision CAS

Date: 2026-08-28
Scope: `INT-214`, synthetic SQLite and worker tests only

## Risk closed

Activation admission binds a model availability revision before runtime loading. A load can be long enough for the
source to become unavailable and later recover, advancing that revision while the model is again `available`.
Checking only the state at the final commit would allow the stale request to replace the active generation.

The final SQLite transaction now compares both the operation revision and the admission-bound availability revision
before it retires the old generation, inserts the new generation, or switches either active pointer. A mismatch returns
`ErrPreconditionFailed` with no generation or pointer mutation. The activation worker then reloads the operation and
converges it to `cancelled` when cancellation won, or to `failed/model_unavailable` while it is still running.

## Executed verification

```text
go test ./internal/aimodel -run 'TestActivationWorker(LoadsOnlyAvailableReviewedModelThroughSourcePort|FailsOperationWhenFinalAvailabilityCASIsStale)$' -count=1
ok github.com/HappyQuQu/foliopath/internal/aimodel

go test ./internal/store/sqlite -run 'TestAIModelActivation(AtomicallyReplacesOldGeneration|FailurePreservesOldActiveGeneration|RejectsAvailabilityRevisionChangedDuringLoad)$' -count=1
ok github.com/HappyQuQu/foliopath/internal/store/sqlite
```

The SQLite regression advances `availability_revision` after claim and before commit. It proves the old generation
remains active, the stale generation is absent, and the commit fails closed. The worker regression injects that final
CAS failure and proves the durable operation does not remain `running`.

## Remaining limits

This does not close `INT-214`. The production catalog is intentionally empty until a model package is approved, and
the production text tokenizer/encoder composition remains blocked on ADR-0014. No real reviewed-model activation or
image-to-text search quality claim is made here.
