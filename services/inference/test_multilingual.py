import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import AsyncMock, patch

from fastapi.testclient import TestClient
from cache_snapshot import InvalidCache, canonical_manifest
import multilingual_contract as contract
import multilingual_server as service
import server as legacy
from scheduler import Busy


class MultilingualTests(unittest.TestCase):
    def entry(self, path, data=b"model"):
        return {"path": contract.PREFIX + path, "size": len(data), "sha256": hashlib.sha256(data).hexdigest()}

    def test_snapshot_and_subsets(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            entries = [self.entry(p) for p in ("config.json", "visual/model.onnx", "textual/model.onnx", "textual/external.weight")]
            (root / "multilingual-manifest.json").write_text(json.dumps(entries))
            for entry in entries:
                target = root / "source" / entry["path"]
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_bytes(b"model")
            selected = contract.request_entries("textual", root)
            self.assertEqual(len(selected), 3)
            self.assertNotIn(entries[1], selected)
            snapshot = contract.verified_snapshot(root / "source", root, selected)
            self.assertEqual((snapshot / entries[3]["path"]).read_bytes(), b"model")
            (root / "source" / entries[3]["path"]).write_bytes(b"other")
            self.assertEqual((snapshot / entries[3]["path"]).read_bytes(), b"model")
            with self.assertRaises(InvalidCache):
                contract.verified_snapshot(root / "source", root, selected)

    def test_legacy_budget_stays_fixed_and_extended_budget_is_bounded(self):
        entries = [self.entry(str(i)) for i in range(3)]
        for entry in entries: entry["size"] = 2**30
        with self.assertRaises(InvalidCache): canonical_manifest(entries)
        self.assertTrue(contract.manifest_bytes(entries))
        with self.assertRaises(InvalidCache): contract.manifest_bytes(entries + [dict(entries[0])])
        for path in ("../escape", "textual/../../escape", "a//b"):
            with self.subTest(path=path), self.assertRaises(InvalidCache): contract.manifest_bytes([self.entry(path)])
        with self.assertRaises(InvalidCache): contract.manifest_bytes([self.entry("x")]*513)

    def test_http_uses_multilingual_model_and_shared_scheduler(self):
        client = TestClient(service.application)
        identity = {"runtime_contract_sha256":"a"*64,"cache_manifest_sha256":"b"*64}
        data = {"entries":json.dumps({"clip":{"textual":{"modelName":contract.MODEL}}}),"text":"汽车"}
        self.assertEqual(client.get("/api/v1/identity").status_code,503)
        with patch.object(service,"identity",identity), patch.object(legacy,"run_inference",AsyncMock(return_value={"clip":"[1]"})) as infer:
            response = client.post("/predict",data=data)
            self.assertEqual(response.status_code,200)
            self.assertEqual(response.headers["X-FolioPath-Runtime"],"a"*64)
            self.assertEqual(infer.call_args.args[1:],("textual","汽车".encode()))
            data["entries"] = json.dumps({"clip":{"textual":{"modelName":"ViT-B-32__openai"}}})
            self.assertEqual(client.post("/predict",data=data).status_code,400)
            self.assertEqual(client.post("/predict",content=b"x"*(legacy.MAX_BODY+1)).status_code,413)
        # The overlay must not overwrite legacy capability identities.
        tag_identity={"runtime_contract_sha256":"c"*64,"model_sha256":"d"*64,"policy_sha256":"e"*64,"cache_manifest_sha256":"f"*64}
        with patch.object(legacy,"tag_identity",tag_identity):
            self.assertEqual(client.get("/api/v1/tag/identity").json(),tag_identity)

    def test_changed_source_or_symlinks_fail_closed(self):
        with tempfile.TemporaryDirectory() as temp:
            root=Path(temp)
            entry=self.entry("textual/model.onnx")
            source=root/"source"
            target=source/entry["path"]
            target.parent.mkdir(parents=True)
            outside=root/"outside"
            outside.write_bytes(b"model")
            target.symlink_to(outside)
            with self.assertRaises(InvalidCache): contract.verified_snapshot(source,root,[entry])
            self.assertEqual(list(root.glob("multilingual-*")),[])


if __name__ == "__main__":
    unittest.main()
