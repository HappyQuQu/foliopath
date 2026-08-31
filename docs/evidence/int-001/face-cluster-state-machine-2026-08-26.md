# Anonymous face cluster and manual-person state-machine spike

Status: **synthetic state-semantics evidence only; face quality and INT-S0 remain No-Go**.

## Question

The required workflow is background anonymous grouping first, followed by user
curation:

1. create a named person from an anonymous group;
2. merge another anonymous group into that person;
3. assign an individual face to an existing person; and
4. preserve every user decision across later model generations.

The safety question is whether lower-confidence suggestions or a later
recluster can silently become manual truth.

## Isolated implementation

[`facecluster.go`](../../../spikes/int001-ai/facecluster.go) is inside the
non-production `INT-001` module. It uses deterministic, complete-link-style
synthetic grouping solely to exercise ownership and transitions. It is not the
selected production clustering algorithm.

The state split is explicit:

- unassigned observations may enter anonymous clusters;
- a core member must satisfy the core threshold against every existing core
  member and may not violate a `cannot-link`;
- an edge face is only a suggestion and is never included in bulk create/merge;
- creating a person or merging a group writes only core faces as manual
  assignments;
- an edge or standalone face requires an explicit individual assignment;
- named/manual faces are excluded from every later background clustering run;
- model generation changes affect derived plans, not persons, assignments or
  `cannot-link` state;
- an assignment to another person cannot be overwritten through the automatic
  path, and a `cannot-link` conflicting with one named person fails closed.
- person creation validates the whole stale anonymous plan before changing
  persons or assignments, so a conflict leaves neither partial membership nor
  an empty person.
- assigning a core, edge or standalone face checks all cannot-links already in
  the destination person before committing.

Anonymous cluster IDs derive from sorted core face IDs, so input enumeration
order does not change the ID in this spike.

## Executed cases

`go test -count=1 ./...` in `spikes/int001-ai` passed on macOS arm64 and covers:

- initial anonymous core/edge grouping;
- person creation from a core group;
- explicit confirmation of an edge face;
- merging a second anonymous core group into the existing person;
- individual face assignment;
- creating a separate named person;
- reclustering with a different generation and looser thresholds without any
  manual reassignment or named-person merge;
- `cannot-link` exclusion;
- rejection of automatic manual-assignment overwrite;
- rejection of a stale group action without leaving an empty person or partial
  assignments;
- rejection of assigning a face across an existing cannot-link;
- rejection of a constraint that conflicts with an existing named person; and
- stable anonymous cluster IDs under input reordering.

## Result and limitation

The user-requested order and correction boundaries are implementable without
letting the model own person identity. The crucial decision is that an edge is
not part of a bulk-confirmed group; it stays a suggestion until the user acts.

This closes only a state-transition subproof. It does not establish detection
recall, embedding ROC, the 99.5% core precision gate, real cosplay/person
failure modes, stale-face transactions, concurrency, API authorization,
privacy, audit history or backup/restore. The greedy synthetic clusterer is not
approved for S2. Those missing facts keep `INT-005`, `INT-007`, `INT-012` and
INT-S0 open.
