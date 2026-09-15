"""FolioPath-only HTTP surface over the fixed upstream inference runtime."""
import asyncio
from contextlib import asynccontextmanager
import hashlib
import json
import io
import os
from pathlib import Path
import shutil

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse, PlainTextResponse
from starlette.datastructures import UploadFile
from starlette.exceptions import HTTPException

from cache_snapshot import canonical_manifest, snapshot
import face_contract

MODEL = "ViT-B-32__openai"
BASE = "5a0839dc5303cd7215bcd2180a26aed3af41675aefb3e75e5157e9f10ad16e6e"
HERE = Path(__file__).parent
MAX_BODY = 9 * 1024 * 1024
identity = None
face_identity = None
upstream = None
active = 0


def expected_identity(entries):
    runtime = hashlib.sha256(BASE.encode() + (HERE / "server.py").read_bytes() + (HERE / "cache_snapshot.py").read_bytes() + (HERE / "face_contract.py").read_bytes()).hexdigest()
    return {"version": 1, "protocol": "foliopath-ml-v1", "model": MODEL, "dimension": 512,
            "cache_manifest_sha256": hashlib.sha256(canonical_manifest(entries)).hexdigest(),
            "runtime_contract_sha256": runtime}


@asynccontextmanager
async def lifespan(app):
    global identity, face_identity, upstream
    entries = json.loads((HERE / "cache-manifest.json").read_bytes())
    cache = snapshot("/cache", "/verified-cache", entries)
    try:
        # Override ambient upstream settings: only the verified private snapshot is used.
        for key in list(os.environ):
            if key.startswith("MACHINE_LEARNING_"):
                del os.environ[key]
        os.environ.update({"MACHINE_LEARNING_CACHE_FOLDER": str(cache), "HF_HUB_OFFLINE": "1",
                           "HF_HUB_DISABLE_TELEMETRY": "1", "MACHINE_LEARNING_REQUEST_THREADS": "1",
                           "MACHINE_LEARNING_MODEL_INTRA_OP_THREADS": "2", "MACHINE_LEARNING_MODEL_INTER_OP_THREADS": "1",
                           "MACHINE_LEARNING_MODEL_TTL": "0", "MACHINE_LEARNING_ANN": "false"})
        # Avoid upstream rewriting the pinned recognition graph to add a batch axis.
        os.environ["MACHINE_LEARNING_MAX_BATCH_SIZE__FACIAL_RECOGNITION"] = "1"
        from immich_ml import main
        upstream = main
        async with main.lifespan(app):
            identity = expected_identity(entries)
            required = {f"facial-recognition/buffalo_l/{kind}/model.onnx" for kind in ("detection", "recognition")}
            if required <= {entry["path"] for entry in entries}:
                face_identity = {**identity, "protocol": "foliopath-face-v1", "model": face_contract.MODEL}
            yield
    finally:
        identity = None
        face_identity = None
        upstream = None
        shutil.rmtree(cache)


app = FastAPI(lifespan=lifespan, docs_url=None, redoc_url=None, openapi_url=None)


@app.middleware("http")
async def identity_headers(request, call_next):
    response = await call_next(request)
    if identity:
        response.headers["X-FolioPath-Runtime"] = identity["runtime_contract_sha256"]
        response.headers["X-FolioPath-Cache"] = identity["cache_manifest_sha256"]
    return response


@app.exception_handler(HTTPException)
async def http_error(request, exc):
    return JSONResponse({"error": "invalid_input"}, status_code=exc.status_code)


@app.get("/api/v1/identity")
async def get_identity():
    return JSONResponse(identity) if identity else JSONResponse({"error": "unavailable"}, status_code=503)


@app.get("/ping")
async def ping():
    return PlainTextResponse("pong") if identity else JSONResponse({"error": "unavailable"}, status_code=503)


@app.get("/api/v1/face/identity")
async def get_face_identity():
    return JSONResponse(face_identity) if face_identity else JSONResponse({"error": "unavailable"}, status_code=503)


@app.post("/api/v1/face/predict")
async def predict_faces(request: Request):
    global active
    if not face_identity:
        return JSONResponse({"error": "unavailable"}, status_code=503)
    if active >= 2:
        return JSONResponse({"error": "busy"}, status_code=503)
    active += 1
    try:
        async with request.form(max_files=1, max_fields=0, max_part_size=MAX_BODY) as form:
            if len(form.multi_items()) != 1 or set(form) != {"image"} or not isinstance(form["image"], UploadFile):
                raise ValueError()
            payload = await form["image"].read(8 * 1024 * 1024 + 1)
            if not 1 <= len(payload) <= 8 * 1024 * 1024:
                raise ValueError()
            from PIL import Image
            try:
                with Image.open(io.BytesIO(payload)) as decoded:
                    width, height = decoded.size
                    if decoded.format not in ("JPEG", "PNG", "WEBP") or getattr(decoded, "n_frames", 1) != 1 or not (
                        0 < width <= 1600 and 0 < height <= 1600
                    ):
                        raise ValueError()
                    decoded.verify()
            except Exception:
                raise ValueError() from None
            try:
                entries = upstream.get_entries(json.dumps(face_contract.ENTRIES))
                response = await upstream.predict(entries, image=payload, text=None)
                return JSONResponse(face_contract.normalize(json.loads(response.body), width, height))
            except Exception:
                return JSONResponse({"error": "unavailable"}, status_code=503)
    except (ValueError, TypeError):
        return JSONResponse({"error": "invalid_input"}, status_code=400)
    finally:
        active -= 1


@app.post("/predict")
async def predict(request: Request):
    global active
    if not identity:
        return JSONResponse({"error": "unavailable"}, status_code=503)
    if active >= 2:
        return JSONResponse({"error": "busy"}, status_code=503)
    active += 1
    try:
        async with request.form(max_files=1, max_fields=2, max_part_size=MAX_BODY) as form:
            if len(form.multi_items()) != 2 or set(form) not in ({"entries", "image"}, {"entries", "text"}):
                raise ValueError()
            visual = "image" in form
            kind = "visual" if visual else "textual"
            expected = {"clip": {kind: {"modelName": MODEL}}}
            if json.loads(form["entries"]) != expected:
                raise ValueError()
            payload = form["image"] if visual else form["text"]
            if visual:
                if not isinstance(payload, UploadFile):
                    raise ValueError()
                payload = await payload.read(8 * 1024 * 1024 + 1)
                if not 1 <= len(payload) <= 8 * 1024 * 1024:
                    raise ValueError()
            elif not isinstance(payload, str) or not payload.strip() or len(payload) > 512:
                raise ValueError()
            entries = upstream.get_entries(json.dumps(expected))
            try:
                result = await upstream.predict(entries, image=payload if visual else None, text=None if visual else payload)
                return result
            except Exception:
                return JSONResponse({"error": "unavailable"}, status_code=503)
    except (ValueError, TypeError):
        return JSONResponse({"error": "invalid_input"}, status_code=400)
    finally:
        active -= 1


class BoundedBody:
    def __init__(self, inner):
        self.inner = inner

    async def __call__(self, scope, receive, send):
        if scope["type"] != "http" or scope["method"] != "POST":
            return await self.inner(scope, receive, send)
        # Bound the whole request before multipart parsing; never spill arbitrary files.
        body = bytearray()
        try:
            async with asyncio.timeout(30):
                while True:
                    event = await receive()
                    if event["type"] == "http.disconnect":
                        return
                    body.extend(event.get("body", b""))
                    if len(body) > MAX_BODY:
                        return await JSONResponse({"error": "too_large"}, status_code=413)(scope, receive, send)
                    if not event.get("more_body", False):
                        break
        except TimeoutError:
            return await JSONResponse({"error": "timeout"}, status_code=408)(scope, receive, send)
        async def replay():
            return {"type": "http.request", "body": bytes(body), "more_body": False}
        await self.inner(scope, replay, send)


application = BoundedBody(app)
