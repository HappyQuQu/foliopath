"""Read expected compatibility identity from image files, without starting HTTP."""
import json
import sys
from pathlib import Path
from server import expected_identity, expected_tag_identity

root = Path(__file__).parent
tag = sys.argv[1:] == ["--tag"]
if sys.argv[1:] and not tag:
    raise SystemExit("usage: image_identity.py [--tag]")
entries = json.loads((root / ("tag-manifest.json" if tag else "cache-manifest.json")).read_bytes())
print(json.dumps(expected_tag_identity(entries) if tag else expected_identity(entries), separators=(",", ":")))
