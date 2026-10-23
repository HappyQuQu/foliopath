"""Isolated HTTP protocol probe: synthetic pixels only, no user media or files."""
import io
import json
import math
import time
import urllib.error
import urllib.request

from PIL import Image

ORIGIN = "http://machine-learning:3003"
http = urllib.request.build_opener(urllib.request.ProxyHandler({}))


def predict(entries, *, text=None, image=None):
    boundary = "foliopath-synthetic-probe"
    parts = []
    for name, value in [("entries", json.dumps(entries)), ("text", text)]:
        if value is not None:
            parts.append(f'--{boundary}\r\nContent-Disposition: form-data; name="{name}"\r\n\r\n{value}\r\n'.encode())
    if image is not None:
        parts.append(f'--{boundary}\r\nContent-Disposition: form-data; name="image"; filename="synthetic.png"\r\nContent-Type: image/png\r\n\r\n'.encode() + image + b"\r\n")
    payload = b"".join(parts) + f"--{boundary}--\r\n".encode()
    req = urllib.request.Request(ORIGIN + "/predict", payload, {"Content-Type": f"multipart/form-data; boundary={boundary}"})
    with http.open(req, timeout=60) as response:
        raw = response.read(1024 * 1024 + 1)
        assert len(raw) <= 1024 * 1024, "response limit"
        return json.loads(raw)


def vector(result):
    values = json.loads(result["clip"])
    assert len(values) == 512, "unexpected embedding dimension"
    assert all(isinstance(x, (float, int)) and math.isfinite(x) for x in values)
    assert sum(x * x for x in values) > 0
    return values


def main():
    deadline = time.monotonic() + 60
    while True:
        try:
            with http.open(ORIGIN + "/ping", timeout=2) as response:
                assert response.read(16) == b"pong"
            break
        except (urllib.error.URLError, TimeoutError):
            if time.monotonic() >= deadline:
                raise
            time.sleep(1)
    buf = io.BytesIO()
    Image.new("RGB", (224, 224), (255, 0, 0)).save(buf, format="PNG")
    visual = vector(predict({"clip": {"visual": {"modelName": "ViT-B-32__openai"}}}, image=buf.getvalue()))
    textual = vector(predict({"clip": {"textual": {"modelName": "ViT-B-32__openai"}}}, text="a red square"))
    repeated = vector(predict({"clip": {"textual": {"modelName": "ViT-B-32__openai"}}}, text="a red square"))
    assert max(abs(a - b) for a, b in zip(textual, repeated)) < 1e-5
    faces = predict({"facial-recognition": {
        "detection": {"modelName": "buffalo_l", "options": {"minScore": 0.7}},
        "recognition": {"modelName": "buffalo_l"},
    }}, image=buf.getvalue())
    assert faces["facial-recognition"] == [], "solid-color fixture unexpectedly detected a face"
    print(json.dumps({"imageDimensions": len(visual), "textDimensions": len(textual),
                      "repeatStable": True, "blankImageFaceCount": 0,
                      "scope": "offline HTTP smoke only; no recognition quality claim"}))


if __name__ == "__main__":
    main()
