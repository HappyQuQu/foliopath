"""Bootstrap downloads stay outside the frozen scheduler and identity owners."""
import asyncio
from contextlib import asynccontextmanager
import os
import logging
import threading
import urllib.parse

import uvicorn
from starlette.responses import JSONResponse

from model_install import Cancelled, Preparation, download_opener, failure_reason, prepare, read_catalog

status = Preparation()
ready = asyncio.Event()
logger = logging.getLogger('foliopath.models')


async def bootstrap(stop):
    import ocr_server
    while not stop.is_set():
        try:
            entries = read_catalog()
            proxy = os.environ.get('FOLIOPATH_MODEL_DOWNLOAD_PROXY', '')
            opener = download_opener(proxy)
            logger.info('Preparing AI models: profile=standard-v1 download_mode=%s proxy_host=%s',
                        'proxy' if proxy else 'direct', urllib.parse.urlsplit(proxy).hostname or 'none')
            await asyncio.to_thread(prepare, entries, {'cache': '/cache', 'ocr': '/ocr-cache'}, opener, stop, status)
            async with ocr_server.app.router.lifespan_context(ocr_server.app):
                ready.set()
                status.update(state='ready')
                logger.info('AI models ready')
                await asyncio.to_thread(stop.wait)
            return
        except Cancelled:
            return
        except Exception as error:
            ready.clear()
            stage = status.snapshot()['state']
            status.update(state='failed')
            logger.error('AI preparation failed: stage=%s reason=%s; retrying in 60 seconds',
                         stage, failure_reason(error))
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
    logging.basicConfig(level=logging.INFO, format='%(asctime)s %(levelname)s %(name)s: %(message)s')
    uvicorn.run(application, host='0.0.0.0', port=3003, workers=1, limit_concurrency=4, access_log=False)
