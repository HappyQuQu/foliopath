# Managed model corruption and availability recovery

Status: **production ownership/failure-semantics subproof passed; S2A remains incomplete**.

The app-level test
[`TestManagedModelCorruptionMarksUnavailableWithoutRetiringActiveGeneration`](../../../internal/app/ai_models_test.go)
uses only temporary application data. It constructs a synthetic package that
passes the same built-in catalog and managed-package validator used by the
production composition, registers it through `aimodel.Service`, and commits an
active semantic generation through the durable activation transaction.

Before corruption the fixture also writes one derived embedding bound to that
generation and creates a read-only original-media sentinel outside application
data. It then replaces the published `text_encoder.onnx` bytes with a different
payload and invokes the production availability service through the managed
activation-source boundary. Observed result:

- the model changed from `available` revision 1 to `unavailable` revision 2;
- the active model pointer remained unchanged;
- the active semantic generation remained present and active;
- the existing embedding remained present;
- the unavailable state survived a real database-component stop/start;
- restoring the exact reviewed bytes after restart changed the model back to
  `available` revision 3;
- the active generation and existing embedding remained present after recovery;
- the sentinel original media retained the same SHA-256, byte size, and mtime.

The focused verification command was:

```sh
go test ./internal/app \
  -run TestManagedModelCorruptionMarksUnavailableWithoutRetiringActiveGeneration \
  -count=1
```

This proves the production database/filesystem ownership and recovery semantics
for an already-published managed package across a database restart. The package
bytes are synthetic and are not an approved release model. The media assertion
covers this corruption/recovery path only; it is not evidence for every AI job.
The test does not replace native ONNX execution, process strong-kill, disk-full,
native amd64, or final signed supply-chain evidence.
