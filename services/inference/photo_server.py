"""Independent tag-only overlay; legacy semantic/face code remains unchanged."""
from contextlib import asynccontextmanager
from pathlib import Path
import sys

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from starlette.datastructures import UploadFile

import server as legacy
import photo_policy
from scheduler import Busy

if Path(__file__).with_name("multilingual-manifest.json").is_file():
    import multilingual_server as base
else:
    base = legacy

identity = None


@asynccontextmanager
async def lifespan(app):
    global identity
    async with base.lifespan(app):
        if not legacy.tag_identity:
            raise ValueError("tag cache unavailable")
        identity = photo_policy.expected_identity()
        legacy.scheduler.command = (sys.executable, str(Path(__file__).with_name("photo_worker.py")))
        try:
            yield
        finally:
            identity = None


app = FastAPI(lifespan=lifespan, docs_url=None, redoc_url=None, openapi_url=None)


@app.get("/api/v1/tag/identity")
async def get_identity():
    return JSONResponse(identity if identity else {"error": "unavailable"}, status_code=200 if identity else 503)


@app.post("/api/v1/tag/predict")
async def predict(request: Request):
    if not identity:
        return JSONResponse({"error": "unavailable"}, status_code=503)
    try:
        async with request.form(max_files=1, max_fields=0, max_part_size=legacy.MAX_BODY) as form:
            if len(form.multi_items()) != 1 or set(form) != {"image"} or not isinstance(form["image"], UploadFile):
                raise ValueError()
            payload = await form["image"].read(8 * 1024 * 1024 + 1)
            if not 1 <= len(payload) <= 8 * 1024 * 1024:
                raise ValueError()
            legacy.validate_image(payload)
            try:
                return JSONResponse(photo_policy.normalize(await legacy.run_inference(request, "tag", payload)))
            except Busy:
                return JSONResponse({"error": "busy"}, status_code=503)
            except Exception:
                return JSONResponse({"error": "unavailable"}, status_code=503)
    except (ValueError, TypeError):
        return JSONResponse({"error": "invalid_input"}, status_code=400)


class TagHeaders:
    def __init__(self, inner):
        self.inner = inner

    async def __call__(self, scope, receive, send):
        async def emit(message):
            if scope["type"] == "http" and scope["path"].startswith("/api/v1/tag/") and identity and message["type"] == "http.response.start":
                headers = {"X-FolioPath-Runtime": identity["runtime_contract_sha256"],
                           "X-FolioPath-Tag-Model": identity["model_sha256"],
                           "X-FolioPath-Tag-Policy": identity["policy_sha256"],
                           "X-FolioPath-Tag-Cache": identity["cache_manifest_sha256"]}
                message = {**message, "headers": [*message.get("headers", []), *((k.lower().encode(), v.encode()) for k, v in headers.items())]}
            await send(message)
        await self.inner(scope, receive, emit)


app.mount("/", base.application)
application = TagHeaders(legacy.BoundedBody(app))
