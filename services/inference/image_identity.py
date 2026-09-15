"""Read expected compatibility identity from image files, without starting HTTP."""
import json
from pathlib import Path
from server import expected_identity

root = Path(__file__).parent
entries = json.loads((root / "cache-manifest.json").read_bytes())
print(json.dumps(expected_identity(entries), separators=(",", ":")))
