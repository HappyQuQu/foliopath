import hashlib
import io
import json
import re
import unittest
from unittest.mock import AsyncMock, patch

from fastapi.testclient import TestClient
from PIL import Image

import server
import tag_contract


def picture():
    stream = io.BytesIO()
    Image.new("RGB", (32, 24), (25, 70, 130)).save(stream, format="PNG")
    return stream.getvalue()


class TagTests(unittest.TestCase):
    def test_policy_has_only_fixed_canonical_classes_and_pinned_files(self):
        self.assertEqual(len(tag_contract.SAFE_LABELS), 92)
        for name in tag_contract.SAFE_LABELS:
            self.assertRegex(name, r"^[a-z_]{1,32}$")
        identity = server.expected_tag_identity(tag_contract.MANIFEST)
        self.assertEqual(set(identity), {"version", "protocol", "model", "model_sha256", "policy_sha256",
                                         "runtime_contract_sha256", "cache_manifest_sha256"})
        self.assertEqual(identity["protocol"], "foliopath-tag-v1")
        self.assertEqual(identity["policy_sha256"], hashlib.sha256(json.dumps(tag_contract.POLICY, sort_keys=True, separators=(",", ":")).encode()).hexdigest())
        with self.assertRaises(ValueError):
            server.expected_tag_identity([])

    def test_safe_finite_unique_bounded_whole_response(self):
        good = {"labels": [{"name": "flower", "confidence": 0.8}]}
        self.assertEqual(tag_contract.normalize(good), good)
        self.assertEqual(tag_contract.normalize({"labels": []}), {"labels": []})
        for result in ({"labels": good["labels"] * 2}, {"labels": good["labels"] * 21},
                       {"labels": [{"name": "unknown", "confidence": 0.8}]},
                       {"labels": [{"name": "flower", "confidence": value} for value in [0.8, 0.9]]}):
            with self.assertRaises(ValueError):
                tag_contract.normalize(result)
        for score in (float("nan"), float("inf"), -1, 1.1, True, None, "0.8", 0.39):
            with self.assertRaises(ValueError):
                tag_contract.normalize({"labels": [{"name": "flower", "confidence": score}]})

    def test_http_identity_headers_predict_and_reject_candidates(self):
        client = TestClient(server.application)
        self.assertEqual(client.get("/api/v1/tag/identity").status_code, 503)
        self.assertEqual(client.post("/api/v1/tag/predict").status_code, 503)
        identity = server.expected_tag_identity(tag_contract.MANIFEST)
        good = {"labels": [{"name": "flower", "confidence": 0.8}]}
        with patch.object(server, "tag_identity", identity), patch.object(server, "run_inference", AsyncMock(return_value=good)) as infer:
            self.assertEqual(client.get("/api/v1/tag/identity").json(), identity)
            response = client.post("/api/v1/tag/predict", files={"image": ("image.png", picture())})
            self.assertEqual(response.status_code, 200)
            self.assertEqual(response.json(), good)
            self.assertEqual(infer.call_args.args[1], "tag")
            for header, field in (("X-FolioPath-Tag-Model", "model_sha256"), ("X-FolioPath-Tag-Policy", "policy_sha256"),
                                  ("X-FolioPath-Tag-Cache", "cache_manifest_sha256"), ("X-FolioPath-Runtime", "runtime_contract_sha256")):
                self.assertEqual(response.headers[header], identity[field])
            for data in ({"labels": "flower"}, {"threshold": "0.1"}):
                self.assertEqual(client.post("/api/v1/tag/predict", files={"image": ("image.png", picture())}, data=data).status_code, 400)
            self.assertEqual(client.post("/api/v1/tag/predict", files={"image": ("image.png", b"bad")}).status_code, 400)
            oversized = client.post("/api/v1/tag/predict", content=b"x" * (server.MAX_BODY + 1))
            self.assertEqual(oversized.status_code, 413)
            self.assertEqual(oversized.headers["X-FolioPath-Tag-Cache"], identity["cache_manifest_sha256"])
            self.assertEqual(oversized.headers["X-FolioPath-Runtime"], identity["runtime_contract_sha256"])
            infer.return_value = {"labels": [{"name": "not_allowed", "confidence": 1}]}
            response = client.post("/api/v1/tag/predict", files={"image": ("image.png", picture())})
            self.assertEqual(response.status_code, 503)
            self.assertEqual(response.json(), {"error": "unavailable"})
