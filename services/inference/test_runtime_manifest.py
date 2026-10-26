import json
from pathlib import Path
import tempfile
import unittest

from runtime_manifest import request_entries
import tag_contract


class ManifestTests(unittest.TestCase):
    def test_family_subset_is_complete_and_hash_preserving(self):
        all_entries = json.loads(Path(__file__).with_name("cache-manifest.json").read_text())
        for kind in ("visual", "textual"):
            selected = request_entries(kind)
            self.assertTrue(selected)
            self.assertTrue(all(entry in all_entries for entry in selected))
            paths = {entry["path"] for entry in selected}
            prefix = "clip/ViT-B-32__openai/"
            self.assertIn(prefix + "config.json", paths)
            self.assertIn(prefix + kind + "/model.onnx", paths)
            other = "textual" if kind == "visual" else "visual"
            self.assertNotIn(prefix + other + "/model.onnx", paths)
            self.assertFalse(any(path.startswith("facial-recognition/") for path in paths))
        # Default image may intentionally omit these optional capabilities.
        if any(entry["path"].startswith("facial-recognition/") for entry in all_entries):
            selected = request_entries("face")
            self.assertTrue(all(e in all_entries and e["path"].startswith("facial-recognition/buffalo_l/") for e in selected))
        tag_entries = json.loads(Path(__file__).with_name("tag-manifest.json").read_text())
        if tag_entries:
            self.assertEqual(request_entries("tag"), tag_contract.MANIFEST)

    def test_disabled_and_invalid_family_fail_closed(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / "tag-manifest.json").write_text("[]")
            (root / "cache-manifest.json").write_text(json.dumps([
                {"path": "clip/ViT-B-32__openai/config.json", "size": 1, "sha256": "a" * 64}]))
            for kind in ("visual", "textual", "face", "tag", "other"):
                with self.subTest(kind=kind), self.assertRaises(ValueError):
                    request_entries(kind, root)
