"""Semantic-only protocol overlay; one HTTP listener and one shared scheduler.

The immutable legacy application still owns face/tag routes, worker execution,
identities and validation. New semantic code is covered by its separate digest.
"""
from contextlib import asynccontextmanager
import json
from pathlib import Path
import shutil
import stat
import sys

from fastapi import FastAPI, Request
from fastapi.responses import JSONResponse
from starlette.datastructures import UploadFile

import server as legacy
from multilingual_contract import MODEL, HERE, expected_identity, verified_snapshot
from scheduler import Busy

identity = None


@asynccontextmanager
async def lifespan(app):
    global identity
    # This dedicated scratch mount is not a model/source/database volume.
    # Fail closed on unknown contents; only reap our known abandoned work dirs.
    for child in Path("/verified-cache").iterdir():
        if not child.name.startswith(("verified-", "request-", "multilingual-")) or not stat.S_ISDIR(child.lstat().st_mode):
            raise ValueError("unexpected inference scratch entry")
        shutil.rmtree(child)
    async with legacy.lifespan(app):
        entries = json.loads((HERE / "multilingual-manifest.json").read_bytes())
        verified = verified_snapshot("/cache", "/verified-cache", entries)
        shutil.rmtree(verified)
        # Do not introduce another admission queue. Face/tag dispatch execs the
        # unchanged worker under the same deadline and process-group lifecycle.
        legacy.scheduler.command = (sys.executable, str(HERE / "multilingual_worker.py"))
        identity = expected_identity()
        try:
            yield
        finally:
            identity = None


app = FastAPI(lifespan=lifespan, docs_url=None, redoc_url=None, openapi_url=None)


def response(data, status=200):
    headers = {} if identity is None else {"X-FolioPath-Runtime": identity["runtime_contract_sha256"],
                                          "X-FolioPath-Cache": identity["cache_manifest_sha256"]}
    return JSONResponse(data, status_code=status, headers=headers)


@app.get("/api/v1/identity")
async def get_identity():
    return response(identity if identity else {"error": "unavailable"}, 200 if identity else 503)


@app.post("/predict")
async def predict(request: Request):
    if not identity:
        return response({"error": "unavailable"}, 503)
    try:
        async with request.form(max_files=1, max_fields=2, max_part_size=legacy.MAX_BODY) as form:
            if len(form.multi_items()) != 2 or set(form) not in ({"entries", "image"}, {"entries", "text"}):
                raise ValueError()
            visual = "image" in form
            kind = "visual" if visual else "textual"
            if json.loads(form["entries"]) != {"clip": {kind: {"modelName": MODEL}}}:
                raise ValueError()
            payload = form["image"] if visual else form["text"]
            if visual:
                if not isinstance(payload, UploadFile):
                    raise ValueError()
                payload = await payload.read(8 * 1024 * 1024 + 1)
                if not 1 <= len(payload) <= 8 * 1024 * 1024:
                    raise ValueError()
                legacy.validate_image(payload)
            elif not isinstance(payload, str) or not payload.strip() or len(payload) > 512:
                raise ValueError()
            try:
                result = await legacy.run_inference(request, kind, payload if visual else payload.encode("utf-8"))
                return response(result)
            except Busy:
                return response({"error": "busy"}, 503)
            except Exception:
                return response({"error": "unavailable"}, 503)
    except (ValueError, TypeError):
        return response({"error": "invalid_input"}, 400)


app.mount("/", legacy.application)
application = legacy.BoundedBody(app)
