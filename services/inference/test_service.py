import hashlib
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

from fastapi.testclient import TestClient
from cache_snapshot import InvalidCache, canonical_manifest, snapshot
import server

IDENTITY = {"version": 1, "runtime_contract_sha256": "a" * 64, "cache_manifest_sha256": "b" * 64}


class CacheTests(unittest.TestCase):
    def test_snapshot_is_independent_and_ignores_unlisted_files(self):
        with tempfile.TemporaryDirectory() as tmp:
            source = Path(tmp) / "source"
            source.mkdir()
            (source / "model").write_bytes(b"model")
            (source / "unlisted").write_bytes(b"not used")
            entries = [{"path": "model", "size": 5, "sha256": hashlib.sha256(b"model").hexdigest()}]
            result = snapshot(source, tmp, entries)
            (source / "model").write_bytes(b"other")
            self.assertEqual((result / "model").read_bytes(), b"model")
            self.assertFalse((result / "unlisted").exists())
            with self.assertRaises(InvalidCache):
                snapshot(source, tmp, entries)

    def test_invalid_manifest(self):
        for path in ("/absolute", "../escape", "a//b", "a/./b", "a\x00b"):
            with self.subTest(path=path), self.assertRaises(InvalidCache):
                canonical_manifest([{"path": path, "size": 0, "sha256": "a" * 64}])
        with self.assertRaises(InvalidCache):
            canonical_manifest([])

    def test_symlink_directory_and_leaf_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            source = Path(tmp) / "source"
            source.mkdir()
            outside = Path(tmp) / "outside"
            outside.mkdir()
            (outside / "model").write_bytes(b"model")
            (source / "dir").symlink_to(outside, target_is_directory=True)
            (source / "leaf").symlink_to(outside / "model")
            for path in ("dir/model", "leaf"):
                with self.subTest(path=path), self.assertRaises(InvalidCache):
                    snapshot(source, tmp, [{"path": path, "size": 5, "sha256": hashlib.sha256(b"model").hexdigest()}])
            self.assertEqual(list(Path(tmp).glob("verified-*")), [])


class HTTPTests(unittest.TestCase):
    def setUp(self):
        # No context manager: intentionally do not load real cache/lifespan here.
        self.client = TestClient(server.application)

    def test_not_ready(self):
        self.assertEqual(self.client.get("/api/v1/identity").status_code, 503)

    def test_identity_and_invalid_model(self):
        with patch.object(server, "identity", IDENTITY):
            self.assertEqual(self.client.get("/api/v1/identity").json(), IDENTITY)
            response = self.client.post("/predict", data={"entries": '{"clip":{"textual":{"modelName":"other"}}}', "text": "hello"})
            self.assertEqual(response.status_code, 400)
            self.assertEqual(response.json(), {"error": "invalid_input"})

    def test_large_chunked_body_rejected(self):
        response = self.client.post("/predict", content=b"x" * (server.MAX_BODY + 1))
        self.assertEqual(response.status_code, 413)

    def test_busy(self):
        with patch.object(server, "identity", IDENTITY), patch.object(server, "active", 2):
            self.assertEqual(self.client.post("/predict", content=b"x").status_code, 503)


if __name__ == "__main__":
    unittest.main()
