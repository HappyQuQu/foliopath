"""Explicit real-weight native cancellation/cache faults, no media/DB/network."""
import asyncio
import io
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

from PIL import Image

import ocr_contract as c
from ocr_scheduler import OCRScheduler


OK=(sys.executable,"-c","import sys;sys.stdin.buffer.read();print('{\"ok\":true}')")


def payload():
    buffer=io.BytesIO();Image.new("RGB",(1000,700),"white").save(buffer,format="PNG");return buffer.getvalue()


def anonymous():
    return int(dict(line.split() for line in Path("/sys/fs/cgroup/memory.stat").read_text().splitlines())["anon"])


class NativeTests(unittest.IsolatedAsyncioTestCase):
    def owner(self,cache="/ocr-cache"):
        return OCRScheduler("/legacy",command=OK,ocr_cache=cache,work_root="/verified-cache")

    def reaped(self,owner,pid):
        self.assertFalse(owner.processes);self.assertEqual(owner.pending,0);self.assertEqual(list(Path("/verified-cache").iterdir()),[])
        with self.assertRaises(ProcessLookupError):os.kill(pid,0)

    async def test_native_cancel_reclaims_memory_and_next_families_recover(self):
        owner=self.owner();before=anonymous();task=asyncio.create_task(owner.infer("ocr",payload()))
        async def loaded():
            while not task.done():
                if owner.processes:
                    pid=next(iter(owner.processes)).pid
                    try:
                        rss=next(int(v.split()[1]) for v in Path(f"/proc/{pid}/status").read_text().splitlines() if v.startswith("VmRSS:"))
                        if rss>768*1024:return pid,rss
                    except (FileNotFoundError,StopIteration):pass
                await asyncio.sleep(.02)
            self.fail("worker finished before native memory observation")
        try:
            pid,rss=await asyncio.wait_for(loaded(),20);task.cancel()
            with self.assertRaises(asyncio.CancelledError):await task
            self.reaped(owner,pid);after=anonymous();self.assertLessEqual(after,before+128*1024**2)
            self.assertEqual(await owner.infer("visual",b"synthetic"),{"ok":True})
            result=await owner.infer("ocr",payload());self.assertEqual(result,c.normalize([]));self.reaped(owner,owner.last_pid)
            print(json.dumps({"nativeCancelRSSKiB":rss,"anonymousBefore":before,"anonymousAfterReap":after,"legacyAndOCRRecovered":True}))
        finally:await owner.close()

    async def test_invalid_model_caches_are_not_source_errors(self):
        with tempfile.TemporaryDirectory(dir="/tmp") as folder:
            for mode in ("missing","tampered","symlink"):
                bad=Path(folder)/mode;bad.mkdir()
                for entry in c.MANIFEST:
                    path=bad/entry["path"];path.parent.mkdir(parents=True,exist_ok=True)
                    if mode=="tampered":
                        with path.open("wb") as stream:stream.truncate(entry["size"])
                    if mode=="symlink":path.symlink_to(Path("/ocr-cache")/entry["path"])
                owner=self.owner(str(bad))
                try:
                    result=await owner.infer("ocr",payload());self.assertEqual(result,{"identity":c.fingerprint(),"error":"model_unavailable"})
                    self.reaped(owner,owner.last_pid)
                finally:await owner.close()


if __name__=="__main__":
    assert sorted(line.split()[0] for line in Path("/proc/net/route").read_text().splitlines()[1:])==[]
    unittest.main()
