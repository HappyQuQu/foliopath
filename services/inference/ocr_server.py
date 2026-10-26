"""Independent OCR overlay. Legacy routes/source identities remain immutable."""
from contextlib import asynccontextmanager
import json
from pathlib import Path
import shutil

from fastapi import FastAPI,Request
from fastapi.responses import JSONResponse
from starlette.datastructures import UploadFile
from starlette.exceptions import HTTPException

import photo_server as base
import server as legacy
from cache_snapshot import snapshot,InvalidCache
from scheduler import Busy,Unavailable
from ocr_scheduler import OCRScheduler
import ocr_contract as contract

identity=None


def response(error,status):
    body={"error":error,"identity":contract.fingerprint()}
    return JSONResponse(body,status_code=status)


@asynccontextmanager
async def lifespan(app):
    global identity
    async with base.lifespan(app):
        previous=legacy.scheduler
        if previous.pending or previous.processes or previous.requests:raise RuntimeError("scheduler already active")
        await previous.close()
        ready=False
        try:
            verified=snapshot("/ocr-cache","/verified-cache",contract.MANIFEST);shutil.rmtree(verified);ready=True
        except InvalidCache:pass
        legacy.scheduler=OCRScheduler(previous.cache,previous.tag_cache,timeout=previous.timeout,command=previous.command,
            work_root=previous.work_root,ocr_cache="/ocr-cache" if ready else None)
        identity=contract.expected_identity() if ready else None
        try:yield
        finally:identity=None


app=FastAPI(lifespan=lifespan,docs_url=None,redoc_url=None,openapi_url=None)


@app.exception_handler(HTTPException)
async def http_error(request,exc):return response("invalid_input",400)


@app.get("/api/v1/ocr/identity")
async def get_identity():return JSONResponse(identity) if identity else response("unavailable",503)


@app.post("/api/v1/ocr/predict")
async def predict(request:Request):
    if not identity:return response("unavailable",503)
    try:
        async with request.form(max_files=1,max_fields=0,max_part_size=legacy.MAX_BODY) as form:
            if len(form.multi_items())!=1 or set(form)!={"image"} or not isinstance(form["image"],UploadFile) or form["image"].filename!="input":raise contract.InvalidInput()
            payload=await form["image"].read(contract.MAX_BYTES+1)
            contract.validate_png(payload)
            try:result=await legacy.run_inference(request,"ocr",payload)
            except Busy:return response("busy",503)
            except Unavailable:return response("unavailable",503)
            if not isinstance(result,dict) or result.get("identity")!=contract.fingerprint():raise contract.InvalidOutput()
            if set(result)=={"identity","lines"}:
                clean=contract.normalize(result["lines"])
                if len(json.dumps(clean,separators=(",",":"),ensure_ascii=False).encode())>128*1024:raise contract.InvalidOutput()
                return JSONResponse(clean)
            if set(result)!={"identity","error"}:raise contract.InvalidOutput()
            code=result["error"]
            statuses={"invalid_input":400,"invalid_media":422,"input_limit":413,"output_invalid":422,"model_unavailable":503,"inference_failed":500}
            if not isinstance(code,str) or code not in statuses:raise contract.InvalidOutput()
            return response(code,statuses[code])
    except contract.InvalidInput:return response("invalid_input",400)
    except contract.InvalidMedia:return response("invalid_media",422)
    except contract.InputLimit:return response("input_limit",413)
    except contract.InvalidOutput:return response("output_invalid",422)
    except HTTPException:return response("invalid_input",400)
    except Exception:return response("inference_failed",500)


class OCRBoundary:
    """Reuse the body-limiter owner; bind its fixed OCR-only errors safely."""
    def __init__(self,inner):self.inner=inner
    async def __call__(self,scope,receive,send):
        if scope["type"]!="http" or not scope["path"].startswith("/api/v1/ocr/"):
            return await self.inner(scope,receive,send)
        start=None;chunks=bytearray()
        async def emit(message):
            nonlocal start
            if message["type"]=="http.response.start" and message["status"]>=400:
                start=message;return
            if start and message["type"]=="http.response.body":
                if len(chunks)<=1024:chunks.extend(message.get("body",b"")[:1025])
                if not message.get("more_body",False):
                    code,status="inference_failed",500
                    try:
                        body=json.loads(chunks)
                        if set(body)=={"identity","error"} and body["identity"]==contract.fingerprint():
                            allowed={(400,"invalid_input"),(413,"input_limit"),(422,"invalid_media"),(422,"output_invalid"),(503,"busy"),(503,"unavailable"),(503,"model_unavailable"),(500,"inference_failed")}
                            if (start["status"],body["error"]) in allowed:code,status=body["error"],start["status"]
                        elif body=={"error":"too_large"} and start["status"]==413:code,status="input_limit",413
                        elif body=={"error":"timeout"} and start["status"]==408:code,status="unavailable",503
                        elif not identity and body=={"error":"unavailable"}:code,status="unavailable",503
                    except (ValueError,TypeError):pass
                    await response(code,status)(scope,receive,send)
                return
            await send(message)
        await self.inner(scope,receive,emit)


class OCRApplication:
    def __init__(self):self.ocr=OCRBoundary(legacy.BoundedBody(app))
    async def __call__(self,scope,receive,send):
        if scope["type"]=="lifespan":return await app(scope,receive,send)
        if scope["type"]=="http" and scope["path"].startswith("/api/v1/ocr/"):
            return await self.ocr(scope,receive,send)
        return await base.application(scope,receive,send)


application=OCRApplication()
