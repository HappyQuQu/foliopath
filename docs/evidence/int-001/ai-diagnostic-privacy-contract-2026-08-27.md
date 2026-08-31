# AI diagnostic privacy contract spike

Status: **closed-field contract subproof only; production privacy testing is not complete**.

The isolated AI spike now has a typed diagnostic snapshot whose JSON shape is
closed to model ID/version/digest, stable component state/error code, four fixed
aggregate counters, and three fixed resource totals. It deliberately exposes no
field for query text, person names, media paths, face crops, embeddings/vectors,
or raw runtime errors.

`make spike-ai` passed on Darwin/arm64 with Go 1.26.5. Tests assert the exact
nested JSON key set, scan the serialized field names for forbidden categories,
and reject raw/path-like error and identifier values, unknown states, invalid
digests, and negative counters. The adjacent JSON records the source digests and
negative matrix.

This does not test production behavior: there is no approved AI API, production
diagnostic bundle, model runtime, or log call yet. S2/S4 still need tests across
the real DTO, logger, error mapper, API response, support bundle, database and
post-deletion artifacts. Therefore the production executable-test item,
`INT-015`, `INT-406`, and INT-S0 remain open.
