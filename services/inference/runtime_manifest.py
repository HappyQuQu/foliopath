"""One owner for fixed per-request subsets of the image's complete pinned manifests."""
import json
from pathlib import Path

from cache_snapshot import canonical_manifest
import tag_contract


def request_entries(kind, root=None):
    root = Path(root) if root is not None else Path(__file__).parent
    if kind == "tag":
        entries = json.loads((root / "tag-manifest.json").read_bytes())
        if entries != tag_contract.MANIFEST:
            raise ValueError("Tag model not enabled or manifest changed")
        return entries
    entries = json.loads((root / "cache-manifest.json").read_bytes())
    canonical_manifest(entries)
    if kind == "face":
        selected = [e for e in entries if e["path"].startswith("facial-recognition/buffalo_l/")]
        required = {f"facial-recognition/buffalo_l/{k}/model.onnx" for k in ("detection", "recognition")}
    elif kind in ("visual", "textual"):
        prefix = "clip/ViT-B-32__openai/"
        excluded = prefix + ("textual/" if kind == "visual" else "visual/")
        selected = [e for e in entries if e["path"].startswith(prefix) and not e["path"].startswith(excluded)]
        required = {prefix + "config.json", prefix + kind + "/model.onnx"}
    else:
        raise ValueError("Unsupported capability")
    if not required <= {e["path"] for e in selected}:
        raise ValueError("Capability not enabled")
    canonical_manifest(selected)
    return selected
