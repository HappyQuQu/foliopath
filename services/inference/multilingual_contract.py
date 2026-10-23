"""ADR-0025 fixed semantic profile; legacy face/tag code and identity stay intact."""
import hashlib
import json
from pathlib import Path
import shutil
import tempfile

from cache_snapshot import InvalidCache, canonical_manifest, snapshot

MODEL = "XLM-Roberta-Large-Vit-B-16Plus"
DIMENSION = 640
PREFIX = f"clip/{MODEL}/"
HERE = Path(__file__).parent


def manifest_bytes(entries):
    if not isinstance(entries, list) or not 1 <= len(entries) <= 512:
        raise InvalidCache()
    seen = set()
    total = 0
    for entry in entries:
        # Reuse canonical lexical/type/per-file validation, not a weaker clone.
        canonical_manifest([entry])
        if not entry["path"].startswith(PREFIX) or entry["path"] in seen:
            raise InvalidCache()
        seen.add(entry["path"])
        total += entry["size"]
        if total > 4 * 2**30:
            raise InvalidCache()
    return json.dumps(sorted(entries, key=lambda e: e["path"]), sort_keys=True, separators=(",", ":")).encode()


def request_entries(kind, root=HERE):
    entries = json.loads((Path(root) / "multilingual-manifest.json").read_bytes())
    manifest_bytes(entries)
    if kind not in ("visual", "textual"):
        raise InvalidCache()
    other = PREFIX + ("textual/" if kind == "visual" else "visual/")
    selected = [e for e in entries if not e["path"].startswith(other)]
    required = {PREFIX + "config.json", PREFIX + kind + "/model.onnx"}
    if not required <= {e["path"] for e in selected}:
        raise InvalidCache()
    return selected


def verified_snapshot(source, parent, entries):
    manifest_bytes(entries)
    target = Path(tempfile.mkdtemp(prefix="multilingual-", dir=parent))
    try:
        # Partition only the size admission. All filesystem/byte validation stays
        # in the original anchored snapshot owner. Move verified private files,
        # never source paths; no duplicate full-cache allocation during merging.
        batch, size = [], 0
        batches = []
        for entry in entries:
            if size + entry["size"] > 2**31:
                batches.append(batch)
                batch, size = [], 0
            batch.append(entry)
            size += entry["size"]
        batches.append(batch)
        for batch in batches:
            partial = snapshot(source, parent, batch)
            try:
                for entry in batch:
                    destination = target / entry["path"]
                    destination.parent.mkdir(parents=True, exist_ok=True)
                    (partial / entry["path"]).rename(destination)
            finally:
                shutil.rmtree(partial)
        return target
    except BaseException:
        shutil.rmtree(target)
        raise


def expected_identity(root=HERE):
    import server
    root = Path(root)
    entries = json.loads((root / "multilingual-manifest.json").read_bytes())
    digest = hashlib.sha256(server.runtime_digest().encode())
    for name in ("multilingual_contract.py", "multilingual_server.py", "multilingual_worker.py", "runtime_entry.py"):
        digest.update(name.encode() + b"\0" + (root / name).read_bytes())
    return {"version": 1, "protocol": "foliopath-ml-v1", "model": MODEL, "dimension": DIMENSION,
            "cache_manifest_sha256": hashlib.sha256(manifest_bytes(entries)).hexdigest(),
            "runtime_contract_sha256": digest.hexdigest()}


if __name__ == "__main__":
    print(json.dumps(expected_identity(), separators=(",", ":")))
