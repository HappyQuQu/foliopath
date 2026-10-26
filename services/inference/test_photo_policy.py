import io
import unittest
from unittest.mock import AsyncMock, patch

from fastapi.testclient import TestClient
from PIL import Image
import photo_policy as policy
import photo_server as service
import server as legacy
import tag_contract
from scheduler import Busy


class PhotoPolicyTests(unittest.TestCase):
    def test_catalog_matches_model_classes_and_expands_legacy(self):
        # Model metadata only; no original library or user media access.
        import os
        source = os.environ.get("FOLIOPATH_TEST_TAG_LABELS")
        if source:
            from pathlib import Path
            self.assertTrue(policy.SAFE_LABELS <= set(Path(source).read_text().splitlines()))
        self.assertGreater(len(policy.SAFE_LABELS), 92)
        self.assertLessEqual(len(policy.SAFE_LABELS), 256)
        self.assertTrue(tag_contract.SAFE_LABELS < policy.SAFE_LABELS)
        for key in policy.SAFE_LABELS:
            self.assertRegex(key, r"^[a-z]+(_[a-z]+)*$")
        self.assertEqual(policy.THRESHOLD, tag_contract.THRESHOLD)
        self.assertEqual(policy.MAX_LABELS, tag_contract.MAX_LABELS)
        before = legacy.expected_tag_identity(tag_contract.MANIFEST)
        after = policy.expected_identity()
        for key in ("model_sha256", "cache_manifest_sha256"):
            self.assertEqual(before[key], after[key])
        for key in ("policy_sha256", "runtime_contract_sha256"):
            self.assertNotEqual(before[key], after[key])
        self.assertEqual(legacy.runtime_digest(), "be1d1baaccb2f8532d9df872483b0453e66e1221d5d1768432357dc76c1cfc8a")

    def test_whole_response_is_fail_closed(self):
        good = {"labels": [{"name": "boots", "confidence": .9}]}
        self.assertEqual(policy.normalize(good), good)
        for score in (True, "0.8", None, .39, 1.1, float("nan"), float("inf")):
            with self.assertRaises(ValueError):
                policy.normalize({"labels": [{"name": "boots", "confidence": score}]})
        for value in ({"labels": good["labels"]*2}, {"labels": good["labels"]*21},
                      {"labels": [{"name": "unknown", "confidence": 1}]},
                      {"labels": [], "extra": True}):
            with self.assertRaises(ValueError): policy.normalize(value)

    def test_http_limits_headers_shared_inference_and_legacy_identity(self):
        image = io.BytesIO()
        Image.new("RGB", (32, 24), "blue").save(image, format="PNG")
        client = TestClient(service.application)
        self.assertEqual(client.get("/api/v1/tag/identity").status_code, 503)
        identity = policy.expected_identity()
        good = {"labels": [{"name": "boots", "confidence": .9}]}
        with patch.object(service, "identity", identity), patch.object(legacy,"run_inference", AsyncMock(return_value=good)) as inference:
            self.assertEqual(client.get("/api/v1/tag/identity").json(), identity)
            response = client.post("/api/v1/tag/predict", files={"image": ("i.png", image.getvalue())})
            self.assertEqual(response.json(), good)
            self.assertEqual(response.headers["X-FolioPath-Tag-Policy"], identity["policy_sha256"])
            self.assertEqual(inference.call_args.args[1], "tag")
            self.assertEqual(client.post("/api/v1/tag/predict", files={"image": ("i", image.getvalue())},data={"labels":"boots"}).status_code,400)
            self.assertEqual(client.post("/api/v1/tag/predict", files={"image": ("i",b"bad")}).status_code,400)
            oversized=client.post("/api/v1/tag/predict",content=b"x"*(legacy.MAX_BODY+1))
            self.assertEqual(oversized.status_code,413)
            self.assertEqual(oversized.headers["X-FolioPath-Runtime"],identity["runtime_contract_sha256"])
            inference.return_value={"labels":[{"name":"unreviewed", "confidence":1}]}
            self.assertEqual(client.post("/api/v1/tag/predict",files={"image":("i",image.getvalue())}).status_code,503)
            inference.side_effect=Busy()
            self.assertEqual(client.post("/api/v1/tag/predict",files={"image":("i",image.getvalue())}).json(),{"error":"busy"})

    def test_real_tcp_disconnect_through_new_overlay_reuses_reaping_contract(self):
        # Reuse the existing real-socket/process test against this actual ASGI
        # stack. Mocking Request.is_disconnected alone is insufficient evidence.
        from test_tcp_disconnect import TCPDisconnectTests
        case = TCPDisconnectTests("test_disconnected_socket_reaps_running_child_and_allows_next_request")
        with patch.object(legacy, "application", service.application), patch.object(service, "identity", policy.expected_identity()):
            case.test_disconnected_socket_reaps_running_child_and_allows_next_request()


if __name__ == "__main__": unittest.main()
