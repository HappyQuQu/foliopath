"""Run inside the face candidate, with verified model cache and no network."""
import io

from fastapi.testclient import TestClient
from PIL import Image

import server

stream = io.BytesIO()
Image.new("RGB", (320, 240), (25, 70, 130)).save(stream, format="PNG")
with TestClient(server.application) as client:
    identity = client.get("/api/v1/face/identity")
    assert identity.status_code == 200, identity.text
    assert identity.json()["protocol"] == "foliopath-face-v1"
    print(identity.json(), flush=True)
    response = client.post("/api/v1/face/predict", files={"image": ("synthetic.png", stream.getvalue())})
    assert response.status_code == 200 and response.json() == {"faces": []}, response.text
    assert server.scheduler.completed == 1 and not server.scheduler.processes
    print("REAL_OFFLINE_FACE_EMPTY_PASS", flush=True)
