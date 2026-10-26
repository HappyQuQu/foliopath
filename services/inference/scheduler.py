"""Single owner of request admission and native inference process lifetime (ADR-0021)."""
import asyncio
import json
import os
from pathlib import Path
import signal
import shutil
import sys
import tempfile

MAX_RESULT = 2 * 1024 * 1024


class Busy(Exception):
    pass


class Unavailable(Exception):
    pass


class InferenceScheduler:
    def __init__(self, cache, tag_cache=None, *, timeout=26, command=None, work_root="/verified-cache"):
        self.cache = str(cache)
        self.tag_cache = str(tag_cache) if tag_cache is not None else None
        self.timeout = timeout
        self.command = command or (sys.executable, str(Path(__file__).with_name("inference_worker.py")))
        self.work_root = work_root
        self.lock = asyncio.Lock()
        self.pending = 0
        self.processes = set()
        self.requests = set()
        self.completed = 0
        self.last_pid = None
        self.closed = False

    async def infer(self, kind, payload):
        if self.closed or kind not in ("visual", "textual", "face", "tag") or (kind == "tag" and not self.tag_cache):
            raise Unavailable()
        if self.pending >= 2:
            raise Busy()
        self.pending += 1
        task = asyncio.current_task()
        self.requests.add(task)
        try:
            # The deadline includes waiting; no second full timeout after admission.
            async with asyncio.timeout(self.timeout):
                async with self.lock:
                    if self.closed:
                        raise Unavailable()
                    return await self._run(kind, payload)
        except (TimeoutError, OSError, ValueError, json.JSONDecodeError) as error:
            raise Unavailable() from error
        finally:
            self.pending -= 1
            self.requests.discard(task)

    async def _run(self, kind, payload):
        proc = None
        io_tasks = []
        work = tempfile.mkdtemp(prefix="request-", dir=self.work_root)
        try:
            cache = self.tag_cache if kind == "tag" else self.cache
            proc = await asyncio.create_subprocess_exec(
                *self.command, kind, cache, work, stdin=asyncio.subprocess.PIPE,
                stdout=asyncio.subprocess.PIPE, stderr=asyncio.subprocess.DEVNULL,
                start_new_session=True, limit=64 * 1024,
            )
            self.processes.add(proc)
            self.last_pid = proc.pid

            async def write():
                proc.stdin.write(payload)
                await proc.stdin.drain()
                proc.stdin.close()
                await proc.stdin.wait_closed()

            async def read():
                chunks, size = [], 0
                while chunk := await proc.stdout.read(64 * 1024):
                    size += len(chunk)
                    if size > MAX_RESULT:
                        raise Unavailable()
                    chunks.append(chunk)
                return b"".join(chunks)

            io_tasks = [asyncio.create_task(write()), asyncio.create_task(read())]
            _, output = await asyncio.gather(*io_tasks)
            if await proc.wait() != 0:
                raise Unavailable()
            result = json.loads(output)
            if not isinstance(result, dict):
                raise Unavailable()
            self.completed += 1
            return result
        finally:
            for task in io_tasks:
                if not task.done():
                    task.cancel()
            await asyncio.gather(*io_tasks, return_exceptions=True)
            try:
                if proc is not None:
                    # Hold the admission lock until OS-level reclamation is complete.
                    cleanup = asyncio.create_task(self._reap(proc))
                    try:
                        await asyncio.shield(cleanup)
                    except asyncio.CancelledError:
                        await cleanup
                        raise
            finally:
                shutil.rmtree(work)

    async def _reap(self, proc):
        if proc.returncode is None:
            try:
                os.killpg(proc.pid, signal.SIGKILL)
            except ProcessLookupError:
                pass
        if proc.stdin is not None:
            proc.stdin.close()
        # A killed child can leave its stdout transport paused at the buffer high
        # waterline. Drain in bounded chunks, otherwise Process.wait can deadlock.
        if proc.stdout is not None:
            while await proc.stdout.read(64 * 1024):
                pass
        await proc.wait()
        self.processes.discard(proc)

    async def close(self):
        self.closed = True
        tasks = tuple(self.requests)
        for task in tasks:
            task.cancel()
        await asyncio.gather(*tasks, return_exceptions=True)
        await asyncio.gather(*(self._reap(proc) for proc in tuple(self.processes)))
