"""ADR-0026: reviewed photo-category policy, separate from legacy identities."""
import hashlib
import json
import math
from pathlib import Path

import tag_contract as legacy
from cache_snapshot import canonical_manifest

HERE = Path(__file__).parent
MODEL = legacy.MODEL
MODEL_SHA256 = legacy.MODEL_SHA256
MANIFEST = legacy.MANIFEST
THRESHOLD = legacy.THRESHOLD
MAX_LABELS = legacy.MAX_LABELS


def categories(root=HERE):
    values = json.loads((Path(root) / "photo_categories.json").read_bytes())
    if not isinstance(values, dict) or not 1 <= len(values) <= 256 or not legacy.SAFE_LABELS <= values.keys():
        raise ValueError("invalid photo catalog")
    for key, names in values.items():
        if not isinstance(key, str) or not key or len(key) > 32 or not all(c in "abcdefghijklmnopqrstuvwxyz_" for c in key):
            raise ValueError("invalid photo class")
        if not isinstance(names, list) or not 3 <= len(names) <= 8 or not all(isinstance(n, str) and 1 <= len(n) <= 32 for n in names):
            raise ValueError("invalid photo labels")
    return values


SAFE_LABELS = frozenset(categories())
POLICY = {**legacy.POLICY, "version": 2, "labels": sorted(SAFE_LABELS),
          "name_mapping": "curation-photo-category-key-v2"}
POLICY_SHA256 = hashlib.sha256(json.dumps(POLICY, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def normalize(result):
    if not isinstance(result, dict) or set(result) != {"labels"} or not isinstance(result["labels"], list) or len(result["labels"]) > MAX_LABELS:
        raise ValueError()
    seen, previous = set(), None
    for item in result["labels"]:
        if not isinstance(item, dict) or set(item) != {"name", "confidence"}:
            raise ValueError()
        name, score = item["name"], item["confidence"]
        if not isinstance(name, str) or name not in SAFE_LABELS or name in seen or type(score) not in (float, int) or not math.isfinite(score) or not THRESHOLD <= score <= 1:
            raise ValueError()
        order = (-score, name)
        if previous is not None and order < previous:
            raise ValueError()
        previous = order
        seen.add(name)
    return result


def expected_identity(root=HERE):
    import server
    root = Path(root)
    if json.loads((root / "tag-manifest.json").read_bytes()) != MANIFEST:
        raise ValueError("unexpected tag manifest")
    digest = hashlib.sha256(server.runtime_digest().encode())
    # Localized names/aliases do not alter inference. Allowed class keys are
    # covered by policy_sha256; do not invalidate models for a display correction.
    for name in ("photo_policy.py", "photo_server.py", "photo_worker.py", "photo_runtime.py"):
        digest.update(name.encode() + b"\0" + (root / name).read_bytes())
    return {"version": 1, "protocol": "foliopath-tag-v1", "model": MODEL,
            "model_sha256": MODEL_SHA256, "policy_sha256": POLICY_SHA256,
            "runtime_contract_sha256": digest.hexdigest(),
            "cache_manifest_sha256": hashlib.sha256(canonical_manifest(MANIFEST)).hexdigest()}


if __name__ == "__main__":
    print(json.dumps(expected_identity(), separators=(",", ":")))
