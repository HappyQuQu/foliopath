# INT-009 SQLite vector generation recovery evidence

Status: **bounded synthetic recovery evidence; INT-009B remains open**.

An isolated SQLite/WAL test created a committed active vector generation, then
started a real helper process that inserted a replacement generation and moved
the active pointer inside one uncommitted transaction. The parent killed the
helper after all replacement rows and the pointer update had executed but
before commit.

After reopening the database:

- `PRAGMA integrity_check` returned `ok`;
- the prior active generation and all 16 rows remained visible;
- no row from the uncommitted replacement generation was visible;
- a fresh transaction rebuilt 32 rows, moved the active pointer and removed the
  old generation atomically;
- every stored synthetic four-dimensional vector retained its exact 16-byte
  BLOB length.

The test `TestSQLiteVectorGenerationStrongKillRecovery` passed on macOS/arm64
and native Linux/arm64. It uses random/synthetic data and small row counts. It
does not measure real-embedding recall, 100k capacity, browse concurrency,
native Linux/amd64, backup size or the final 500 MiB budget.
