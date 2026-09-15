import io
import json
import unittest
from unittest.mock import AsyncMock, patch

from fastapi.testclient import TestClient
from fastapi.responses import JSONResponse
from PIL import Image

import face_contract
import server


def picture(size=(32, 24)):
    stream = io.BytesIO()
    Image.new("RGB", size, (25, 70, 130)).save(stream, format="PNG")
    return stream.getvalue()


def result():
    return {"imageWidth": 32, "imageHeight": 24, "facial-recognition": [{
        "boundingBox": {"x1": -1, "y1": 0, "x2": 16, "y2": 12},
        "score": 0.9, "embedding": json.dumps([2.0] + [0.0] * 511)}]}


class FaceContractTests(unittest.TestCase):
    def test_normalizes_and_clips(self):
        face = face_contract.normalize(result(), 32, 24)["faces"][0]
        self.assertEqual(face["box"], {"x": 0, "y": 0, "width": 0.5, "height": 0.5})
        self.assertEqual(face["embedding"], [1.0] + [0.0] * 511)
        self.assertNotIn("quality", face)

    def test_invalid_vectors(self):
        for vector in ([0.0] * 512, [1.0], [None] * 512, [True] * 512, [float("nan")] * 512):
            with self.subTest(vector=vector[:1]):
                data = result()
                data["facial-recognition"][0]["embedding"] = json.dumps(vector)
                with self.assertRaises(ValueError):
                    face_contract.normalize(data, 32, 24)

    def test_overflow_and_invalid_boxes(self):
        data = result()
        data["facial-recognition"] *= 65
        with self.assertRaises(ValueError):
            face_contract.normalize(data, 32, 24)
        for value in (float("inf"), True, -2, 0):
            data = result()
            data["facial-recognition"][0]["boundingBox"]["x2"] = value
            with self.assertRaises(ValueError):
                face_contract.normalize(data, 32, 24)

    def test_empty(self):
        data = result()
        data["facial-recognition"] = []
        self.assertEqual(face_contract.normalize(data, 32, 24), {"faces": []})

    def test_http_contract(self):
        client = TestClient(server.application)
        self.assertEqual(client.get("/api/v1/face/identity").status_code, 503)
        self.assertEqual(client.post("/api/v1/face/predict").status_code, 503)
        with patch.object(server, "face_identity", {"protocol": "foliopath-face-v1"}), patch.object(server, "upstream") as upstream:
            upstream.predict = AsyncMock(return_value=JSONResponse(result()))
            response = client.post("/api/v1/face/predict", files={"image": ("image.png", picture())})
            self.assertEqual(response.status_code, 200)
            self.assertEqual(response.json()["faces"][0]["embedding"][0], 1)
            self.assertEqual(json.loads(upstream.get_entries.call_args.args[0]), face_contract.ENTRIES)
            for payload in (b"corrupt", picture((1601, 1))):
                self.assertEqual(client.post("/api/v1/face/predict", files={"image": ("image.png", payload)}).status_code, 400)
            upstream.predict.side_effect = RuntimeError("private path")
            response = client.post("/api/v1/face/predict", files={"image": ("image.png", picture())})
            self.assertEqual(response.json(), {"error": "unavailable"})
            self.assertEqual(response.status_code, 503)
        self.assertEqual(server.active, 0)

    def test_shared_admission(self):
        client = TestClient(server.application)
        with patch.object(server, "face_identity", {"version": 1}), patch.object(server, "active", 2):
            self.assertEqual(client.post("/api/v1/face/predict").json(), {"error": "busy"})


if __name__ == "__main__":
    unittest.main()
