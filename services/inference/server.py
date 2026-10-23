"""FolioPath-only HTTP surface over the fixed upstream inference runtime."""
import asyncio
from contextlib import asynccontextmanager
import hashlib
import json
import io
from pathlib import Path
import shutil

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse, PlainTextResponse
from starlette.datastructures import UploadFile
from starlette.exceptions import HTTPException

from cache_snapshot import canonical_manifest, snapshot
import face_contract
import tag_contract
from scheduler import Busy, InferenceScheduler, Unavailable

MODEL = "ViT-B-32__openai"
BASE = "5a0839dc5303cd7215bcd2180a26aed3af41675aefb3e75e5157e9f10ad16e6e"
HERE = Path(__file__).parent
MAX_BODY = 9 * 1024 * 1024
identity = None
face_identity = None
tag_identity = None
scheduler = None


def runtime_digest():
    sources = ("server.py", "cache_snapshot.py", "face_contract.py", "tag_contract.py",
               "scheduler.py", "inference_worker.py", "runtime_manifest.py")
    digest = hashlib.sha256(BASE.encode())
    for name in sources:
        digest.update(name.encode() + b"\x00" + (HERE / name).read_bytes())
    return digest.hexdigest()


def expected_identity(entries):
    return {"version": 1, "protocol": "foliopath-ml-v1", "model": MODEL, "dimension": 512,
            "cache_manifest_sha256": hashlib.sha256(canonical_manifest(entries)).hexdigest(),
            "runtime_contract_sha256": runtime_digest()}


def expected_tag_identity(entries):
    if entries != tag_contract.MANIFEST:
        raise ValueError("Unexpected tag manifest")
    return {"version": 1, "protocol": "foliopath-tag-v1", "model": tag_contract.MODEL,
            "model_sha256": tag_contract.MODEL_SHA256, "policy_sha256": tag_contract.POLICY_SHA256,
            "cache_manifest_sha256": hashlib.sha256(canonical_manifest(entries)).hexdigest(),
            "runtime_contract_sha256": runtime_digest()}


@asynccontextmanager
async def lifespan(app):
    global identity, face_identity, tag_identity, scheduler
    entries = json.loads((HERE / "cache-manifest.json").read_bytes())
    # Verify every accepted byte before readiness, but do not hold all model
    # snapshots resident. Each child independently re-verifies its complete subset.
    cache = snapshot("/cache", "/verified-cache", entries)
    shutil.rmtree(cache)
    try:
        tag_entries = json.loads((HERE / "tag-manifest.json").read_bytes())
        if tag_entries:
            candidate = expected_tag_identity(tag_entries)
            tag_cache = snapshot("/cache", "/verified-cache", tag_entries)
            shutil.rmtree(tag_cache)
            tag_identity = candidate
        scheduler = InferenceScheduler("/cache", "/cache" if tag_entries else None)
        identity = expected_identity(entries)
        required = {f"facial-recognition/buffalo_l/{kind}/model.onnx" for kind in ("detection", "recognition")}
        if required <= {entry["path"] for entry in entries}:
            face_identity = {**identity, "protocol": "foliopath-face-v1", "model": face_contract.MODEL}
        yield
    finally:
        identity = None
        face_identity = None
        tag_identity = None
        if scheduler:
            await scheduler.close()
        scheduler = None


app = FastAPI(lifespan=lifespan, docs_url=None, redoc_url=None, openapi_url=None)


def identity_headers(path):
    headers = {}
    if identity:
        headers["X-FolioPath-Runtime"] = identity["runtime_contract_sha256"]
        headers["X-FolioPath-Cache"] = identity["cache_manifest_sha256"]
    if tag_identity and path.startswith("/api/v1/tag/"):
        headers["X-FolioPath-Runtime"] = tag_identity["runtime_contract_sha256"]
        headers["X-FolioPath-Tag-Model"] = tag_identity["model_sha256"]
        headers["X-FolioPath-Tag-Policy"] = tag_identity["policy_sha256"]
        headers["X-FolioPath-Tag-Cache"] = tag_identity["cache_manifest_sha256"]
    return headers


class IdentityHeaders:
    """Pure ASGI: do not interpose BaseHTTPMiddleware's receive task/channel.

    Request.is_disconnected must see the transport's actual disconnect event.
    Wrap the body limiter too so ready-service 413/408 carry the same identity.
    """
    def __init__(self, inner):
        self.inner = inner

    async def __call__(self, scope, receive, send):
        async def send_headers(message):
            if scope["type"] == "http" and message["type"] == "http.response.start":
                message = {**message, "headers": [*message.get("headers", []),
                    *((name.lower().encode("ascii"), value.encode("ascii"))
                      for name, value in identity_headers(scope["path"]).items())]}
            await send(message)
        await self.inner(scope, receive, send_headers)


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


@app.get("/api/v1/tag/identity")
async def get_tag_identity():
    return JSONResponse(tag_identity) if tag_identity else JSONResponse({"error": "unavailable"}, status_code=503)


def validate_image(payload):
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


async def run_inference(request, kind, payload):
    if scheduler is None:
        raise Unavailable()
    inference = asyncio.create_task(scheduler.infer(kind, payload))
    try:
        while not inference.done():
            done, _ = await asyncio.wait((inference,), timeout=0.05)
            if done:
                break
            if await request.is_disconnected():
                raise Unavailable()
        return await inference
    finally:
        if not inference.done():
            inference.cancel()
        await asyncio.gather(inference, return_exceptions=True)


@app.post("/api/v1/face/predict")
async def predict_faces(request: Request):
    return await predict_single_image(request, "face", face_identity)


@app.post("/api/v1/tag/predict")
async def predict_tags(request: Request):
    return await predict_single_image(request, "tag", tag_identity)


async def predict_single_image(request, kind, ready):
    if not ready:
        return JSONResponse({"error": "unavailable"}, status_code=503)
    try:
        async with request.form(max_files=1, max_fields=0, max_part_size=MAX_BODY) as form:
            if len(form.multi_items()) != 1 or set(form) != {"image"} or not isinstance(form["image"], UploadFile):
                raise ValueError()
            payload = await form["image"].read(8 * 1024 * 1024 + 1)
            if not 1 <= len(payload) <= 8 * 1024 * 1024:
                raise ValueError()
            validate_image(payload)
            try:
                result = await run_inference(request, kind, payload)
                return JSONResponse(tag_contract.normalize(result) if kind == "tag" else result)
            except Busy:
                return JSONResponse({"error": "busy"}, status_code=503)
            except Exception:
                return JSONResponse({"error": "unavailable"}, status_code=503)
    except (ValueError, TypeError):
        return JSONResponse({"error": "invalid_input"}, status_code=400)


@app.post("/predict")
async def predict(request: Request):
    if not identity:
        return JSONResponse({"error": "unavailable"}, status_code=503)
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
                validate_image(payload)
            elif not isinstance(payload, str) or not payload.strip() or len(payload) > 512:
                raise ValueError()
            try:
                result = await run_inference(request, kind, payload if visual else payload.encode("utf-8"))
                return JSONResponse(result)
            except Busy:
                return JSONResponse({"error": "busy"}, status_code=503)
            except Exception:
                return JSONResponse({"error": "unavailable"}, status_code=503)
    except (ValueError, TypeError):
        return JSONResponse({"error": "invalid_input"}, status_code=400)


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
        delivered = False
        async def replay():
            nonlocal delivered
            if delivered:
                return await receive()
            delivered = True
            return {"type": "http.request", "body": bytes(body), "more_body": False}
        await self.inner(scope, replay, send)


application = IdentityHeaders(BoundedBody(app))
