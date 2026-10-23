# Managed published-package orphan reconciliation

Status: **production startup recovery implemented and passed; reviewed release
catalog remains empty**.

Managed installation publishes immutable package bytes before its short SQLite
registration transaction. A process death between those operations can
therefore leave a complete content-addressed final package without an installed
model record. Startup now closes that gap through one bounded owner chain:

1. `files.ManagedModelStore.Reconcile` removes interrupted staging and returns
   at most 256 sorted content hashes for syntactically valid final-directory
   names. It returns no path.
2. `aimodel.ManagedOrphanService` accepts only a complete, non-truncated report.
3. Each hash must match one unique built-in catalog entry for the current CPU
   architecture and pass the production managed-package validator, including
   manifest equality, regular/non-executable files, sizes and SHA-256 values.
4. The existing `aimodel.Service` idempotently registers the exact package as a
   managed, available model. Registration never changes the active model or
   active semantic generation.

Unknown, corrupt, wrong-architecture and incomplete-scan finals are not
registered, activated or deleted. This preserves fail-closed behavior and lets
an exact reviewed package recover without requiring its original `/models`
source to remain mounted.

The app-level corruption/recovery vertical now begins from this real orphan
reconciliation path rather than calling model registration directly. It proves
the recovered package initially appears as inactive, can subsequently be
activated through the durable activation transaction, and retains the earlier
corruption/restart/recovery guarantees.

Focused verification:

```sh
go test ./internal/aimodel ./internal/files ./internal/app -count=1
```

Production currently constructs an empty reviewed catalog, so this mechanism
correctly registers nothing in a normal build until model compliance and
supply-chain review provides an accepted catalog entry. This evidence does not
approve a model or close the install-worker/native-inference release gates.
