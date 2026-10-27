"""Bootstrap downloads stay outside the frozen scheduler and identity owners."""
import asyncio
from contextlib import asynccontextmanager
import os
import threading

import uvicorn
from starlette.responses import JSONResponse

from model_install import Cancelled, Preparation, download_opener, prepare, read_catalog

status = Preparation()
ready = asyncio.Event()


async def bootstrap(stop):
    import ocr_server
    while not stop.is_set():
        try:
            entries = read_catalog()
            opener = download_opener(os.environ.get('FOLIOPATH_MODEL_DOWNLOAD_PROXY', ''))
            await asyncio.to_thread(prepare, entries, {'cache': '/cache', 'ocr': '/ocr-cache'}, opener, stop, status)
            async with ocr_server.app.router.lifespan_context(ocr_server.app):
                ready.set()
                status.update(state='ready')
                print('AI models ready', flush=True)
                await asyncio.to_thread(stop.wait)
            return
        except Cancelled:
            return
        except Exception:
            ready.clear()
            status.update(state='failed')
            print('AI preparation failed; retrying in 60 seconds', flush=True)
            await asyncio.to_thread(stop.wait, 60)


@asynccontextmanager
async def lifespan(app):
    stop = threading.Event()
    task = asyncio.create_task(bootstrap(stop))
    try:
        yield
    finally:
        stop.set()
        await task
        ready.clear()


async def application(scope, receive, send):
    if scope['type'] == 'lifespan':
        from starlette.applications import Starlette
        await Starlette(lifespan=lifespan)(scope, receive, send)
        return
    if scope['type'] != 'http':
        return
    if scope['path'] == '/api/v1/preparation':
        await JSONResponse(status.snapshot())(scope, receive, send)
        return
    if not ready.is_set():
        await JSONResponse({'error': 'model_unavailable'}, status_code=503)(scope, receive, send)
        return
    import ocr_server
    await ocr_server.application(scope, receive, send)


if __name__ == '__main__':
    uvicorn.run(application, host='0.0.0.0', port=3003, workers=1, limit_concurrency=4, access_log=False)
