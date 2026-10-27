import asyncio
import os
import sys
import tempfile
import unittest
from unittest.mock import AsyncMock, patch

from scheduler import Busy, InferenceScheduler, Unavailable
import server


def command(body):
    return (sys.executable, "-c", body)


SLEEP = command("import sys,time; sys.stdin.buffer.read(); time.sleep(60)")
OK = command("import sys; sys.stdin.buffer.read(); print('{\"ok\":true}')")


class SchedulerTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)

    def owner(self, *args, **kwargs):
        return InferenceScheduler(*args, **kwargs, work_root=self.temp.name)

    async def wait_process(self, owner):
        async def ready():
            while not owner.processes:
                await asyncio.sleep(0.005)
        await asyncio.wait_for(ready(), 2)
        return owner.last_pid

    def assert_reaped(self, owner, pid):
        self.assertEqual(owner.pending, 0)
        self.assertFalse(owner.processes)
        self.assertEqual(os.listdir(self.temp.name), [])
        with self.assertRaises(ProcessLookupError):
            os.kill(pid, 0)

    async def test_success_reaps_and_allows_next_family(self):
        owner = self.owner("/cache", "/tags", command=OK)
        for kind in ("textual", "visual", "face", "tag"):
            self.assertEqual(await owner.infer(kind, b"input"), {"ok": True})
            self.assert_reaped(owner, owner.last_pid)
        self.assertEqual(owner.completed, 4)

    async def test_cancel_reaps_native_process_before_slot_release(self):
        owner = self.owner("/cache", command=SLEEP)
        task = asyncio.create_task(owner.infer("visual", b"input"))
        pid = await self.wait_process(owner)
        task.cancel()
        with self.assertRaises(asyncio.CancelledError):
            await task
        self.assert_reaped(owner, pid)
        owner.command = OK
        self.assertEqual(await owner.infer("textual", b"hello"), {"ok": True})

    async def test_timeout_reaps(self):
        owner = self.owner("/cache", command=SLEEP, timeout=0.15)
        task = asyncio.create_task(owner.infer("face", b"input"))
        pid = await self.wait_process(owner)
        with self.assertRaises(Unavailable):
            await task
        self.assert_reaped(owner, pid)

    async def test_one_active_one_waiter_and_overload(self):
        owner = self.owner("/cache", "/tags", command=SLEEP)
        running = asyncio.create_task(owner.infer("visual", b"input"))
        pid = await self.wait_process(owner)
        waiting = asyncio.create_task(owner.infer("tag", b"input"))
        await asyncio.sleep(0.01)
        self.assertEqual(owner.pending, 2)
        self.assertEqual(len(owner.processes), 1)
        with self.assertRaises(Busy):
            await owner.infer("face", b"input")
        waiting.cancel()
        with self.assertRaises(asyncio.CancelledError):
            await waiting
        self.assertEqual(owner.last_pid, pid)
        await owner.close()
        with self.assertRaises(asyncio.CancelledError):
            await running
        self.assert_reaped(owner, pid)

    async def test_deadline_includes_queue_wait(self):
        owner = self.owner("/cache", command=SLEEP, timeout=0.2)
        async with owner.lock:
            with self.assertRaises(Unavailable):
                await owner.infer("textual", b"input")
        self.assertIsNone(owner.last_pid)
        self.assertEqual(owner.pending, 0)

    async def test_bad_or_oversized_child_output_fails_closed_and_reaps(self):
        for script in ("import sys; sys.stdin.buffer.read(); print('invalid')",
                       "import sys; sys.stdin.buffer.read(); print('x' * (3*1024*1024))",
                       "import sys; sys.stdin.buffer.read(); sys.exit(1)"):
            with self.subTest(script=script):
                owner = self.owner("/cache", command=command(script))
                with self.assertRaises(Unavailable):
                    await owner.infer("visual", b"input")
                self.assert_reaped(owner, owner.last_pid)

    async def test_closed_or_missing_tag_cache_is_unavailable(self):
        owner = self.owner("/cache", command=OK)
        with self.assertRaises(Unavailable):
            await owner.infer("tag", b"input")
        await owner.close()
        with self.assertRaises(Unavailable):
            await owner.infer("visual", b"input")

    async def test_http_disconnect_cancels_and_reaps(self):
        owner = self.owner("/cache", command=SLEEP)
        request = AsyncMock()
        request.is_disconnected.return_value = True
        with patch.object(server, "scheduler", owner):
            with self.assertRaises(Unavailable):
                await server.run_inference(request, "visual", b"input")
        self.assert_reaped(owner, owner.last_pid)

    async def test_real_worker_missing_source_cache_fails_without_residue(self):
        owner = self.owner(self.temp.name, timeout=3)
        with self.assertRaises(Unavailable):
            await owner.infer("textual", b"hello")
        self.assert_reaped(owner, owner.last_pid)
