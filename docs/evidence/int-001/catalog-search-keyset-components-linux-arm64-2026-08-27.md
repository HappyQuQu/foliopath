# Production catalog search-keyset component diagnosis

Status: **existing baseline budget still fails; cost ownership is now split**.

The production catalog capacity test ran without an AI runtime in native
Linux/arm64 with 4 CPUs, 4 GiB, no network, 10,000 directories and 100,000
assets. The existing `s4-search-v1` limit was not changed. Its name-ascending
`asset` search still failed because fetching the first and second 100-item
pages together reached P95 358.623 ms against the 250 ms limit.

Additional benchmark-only timings show that this is not merely a misleading
two-page aggregate and not one isolated slow call:

- service first page: P95 167.755 ms;
- service second page: P95 190.937 ms;
- repository count: P95 66.987 ms;
- repository first-page list: P95 106.750 ms;
- repository second-page list: P95 130.316 ms.

Every service page currently obtains total/image/video counts before listing
the page, so repeated counting is material. Removing or caching that work alone
would still not prove the budget: the broad list query is also material and the
second page is slower than the first. The query combines a broad trigram FTS
candidate set, exact substring predicates, a derived directory-path name order
and a multi-column keyset predicate. Those are hypotheses for query-plan work,
not yet proven root causes.

This result does not authorize changing the threshold or production query. It
closes only the measurement ambiguity seen in the AI/no-AI paired runs. The
next approved step is a non-production query-plan/index-design spike followed
by the unchanged capacity test; any production search-contract or schema change
belongs to its existing MVP maintenance Gate and must not be smuggled into
`INT-S0`.

Machine-readable results are in
[`catalog-search-keyset-components-linux-arm64-2026-08-27.json`](catalog-search-keyset-components-linux-arm64-2026-08-27.json).
