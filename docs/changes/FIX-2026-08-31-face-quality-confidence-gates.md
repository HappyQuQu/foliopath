# Face quality confidence and bias evidence gates

Date: 2026-08-31

Gate: `POST-MVP-5 / INT-250` (existing S2C quality evidence slice).

Affected invariant: anonymous core groups may be used for whole-group assignment
only after representative, governed evidence demonstrates at least 99.5% precision;
missing bias evidence must fail closed.

## Change

The face-quality verifier now evaluates approved thresholds against Wilson 95%
confidence bounds: lower bounds for detection, slice and verification recall and for
core/edge precision, and the upper bound for verification false-positive rate. A
small all-correct sample can no longer pass a high-confidence release threshold.

Every detection record must provide exactly one evaluable value for skin tone, age,
lighting, occlusion and people count. Each dimension requires at least two groups,
and every group needs at least 20 expected faces. Blank, `unknown`, `unlabeled` and
`unspecified` values fail closed. These checks prevent a single placeholder bucket
from satisfying the bias matrix structurally.

## Evidence

```sh
cd spikes/int001-ai
go test ./...
```

The suite covers a valid 50-identity/1,000-image matrix, single-bucket and unlabeled
bias evidence, and perfect but statistically insufficient pair samples.
