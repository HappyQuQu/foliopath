"""Pure-contract, HTTP faults and shared-scheduler tests; no real media/models."""
import asyncio
import copy
import io
import json
import os
from pathlib import Path
import struct
import sys
import tempfile
import unittest
from unittest.mock import AsyncMock,patch
from contextlib import asynccontextmanager
from types import SimpleNamespace
import zlib

from fastapi.testclient import TestClient
from PIL import Image

import ocr_contract as c
import ocr_server as http
from ocr_scheduler import OCRScheduler
from scheduler import Busy,Unavailable


def png(mode="RGB",size=(64,32)):
    data=io.BytesIO();Image.new(mode,size).save(data,format="PNG");return data.getvalue()


def line():return {"text":"中文 Cafe\u0301 <b>😀</b>","box":[.1,.1,.9,.1,.9,.3,.1,.3],"detection_score":.9,"recognition_score":.95}


class ContractTests(unittest.TestCase):
    def test_png_profile_and_crc_budget(self):
        self.assertEqual(c.validate_png(png()),(64,32))
        for data,error in [(b"",c.InvalidMedia),(b"jpeg",c.InvalidMedia),(b"x"*(c.MAX_BYTES+1),c.InputLimit),
                           (png("RGBA"),c.InvalidInput),(png(size=(64,20)),c.InputLimit),(png()+b"trailing",c.InvalidMedia),
                           (png()[:-4],c.InvalidMedia)]:
            with self.subTest(error=error),self.assertRaises(error):c.validate_png(data)
        data=bytearray(png());data[-1]^=1
        with self.assertRaises(c.InvalidMedia):c.validate_png(data)
        payload=png();chunk=b"tEXt"+b"synthetic text";chunk=struct.pack(">I",len(chunk)-4)+chunk+struct.pack(">I",zlib.crc32(chunk))
        with self.assertRaises(c.InvalidInput):c.validate_png(payload[:33]+chunk+payload[33:])

    def test_normalization_uses_generated_policy(self):
        self.assertEqual(c.normalize([])["lines"],[])
        self.assertEqual(c.normalize([line()])["lines"][0]["text"],"中文 Café <b>😀</b>")
        for mutate in [lambda v:v.update(extra=True),lambda v:v.update(text="a\x00"),lambda v:v.update(text="\ud800"),
                       lambda v:v.update(text="\u0344"*512),lambda v:v.update(recognition_score=.79),
                       lambda v:v.update(box=[0]*8),lambda v:v.update(detection_score=True)]:
            value=line();mutate(value)
            with self.assertRaises(c.InvalidOutput):c.normalize([line(),value])
        with self.assertRaises(c.InvalidOutput):c.normalize([line()]*129)

    def test_shared_go_conformance_fixtures(self):
        cases=json.loads(Path(__file__).with_name("ocr-conformance.json").read_bytes())
        for case in cases:
            with self.subTest(case=case["name"]):
                if case["valid"]:
                    self.assertEqual([v["text"] for v in c.normalize(case["lines"])["lines"]],case["text"])
                else:
                    with self.assertRaises(c.InvalidOutput):c.normalize(case["lines"])

    def test_inference_errors_are_not_mislabeled_as_invalid_media(self):
        import ocr_worker,ocr_engine
        output=io.StringIO()
        with patch.object(sys,"argv",["ocr_worker.py","ocr","/model","/work"]), patch.object(sys,"stdin",SimpleNamespace(buffer=io.BytesIO(png()))), patch.object(sys,"stdout",output), patch.dict(os.environ,dict(os.environ),clear=True), patch.object(ocr_worker,"snapshot",return_value=Path("/fake")), patch.object(ocr_engine,"infer",side_effect=ValueError("private service failure")), patch.object(Image,"MAX_IMAGE_PIXELS",Image.MAX_IMAGE_PIXELS):
            ocr_worker.main()
        self.assertEqual(json.loads(output.getvalue())["error"],"inference_failed")
        self.assertNotIn("private",output.getvalue())

    def test_fingerprint_matches_sorted_identity_json(self):
        import hashlib
        expected=hashlib.sha256(json.dumps(c.expected_identity(),sort_keys=True,separators=(",",":")).encode()).hexdigest()
        self.assertEqual(c.fingerprint(),expected)
        self.assertEqual(c.expected_identity()["input_contract_sha256"],"753de4384b85c73f55020829b8dc96d41700881f01e9064afbbb8f963df46a4e")


class ContourBudgetTests(unittest.TestCase):
    def test_texture_contours_are_filtered_before_the_line_budget(self):
        import numpy as np
        from ocr_engine import BoundedPostProcess
        bitmap=np.zeros((128,128),dtype=np.uint8)
        bitmap[4:124:10,4:124:10]=1
        processor=BoundedPostProcess(max_candidates=c.OUTPUT["max_contours"])
        boxes,scores=processor.boxes_from_bitmap(bitmap.astype(np.float32),bitmap,128,128)
        self.assertEqual(len(boxes),0)
        self.assertEqual(len(scores),0)

    def test_raw_contour_work_remains_bounded(self):
        import numpy as np
        from ocr_engine import BoundedPostProcess
        bitmap=np.zeros((132,132),dtype=np.uint8)
        bitmap[::2,::2]=1
        processor=BoundedPostProcess(max_candidates=c.OUTPUT["max_contours"])
        with self.assertRaises(c.InputLimit):
            processor.boxes_from_bitmap(bitmap.astype(np.float32),bitmap,132,132)


class HTTPTests(unittest.TestCase):
    def setUp(self):
        self.identity=patch.object(http,"identity",c.expected_identity());self.identity.start();self.addCleanup(self.identity.stop)
        self.infer=AsyncMock(return_value=c.normalize([line()]))
        self.mock=patch.object(http.legacy,"run_inference",self.infer);self.mock.start();self.addCleanup(self.mock.stop)
        self.client=TestClient(http.application);self.addCleanup(self.client.close)

    def post(self,data=None,**kwargs):
        return self.client.post("/api/v1/ocr/predict",files={"image":("input",png() if data is None else data)},**kwargs)

    def test_identity_success_and_empty(self):
        self.assertEqual(self.client.get("/api/v1/ocr/identity").json(),c.expected_identity())
        result=self.post();self.assertEqual(result.status_code,200);self.assertEqual(result.json(),c.normalize([line()]))
        self.assertEqual(self.infer.call_args.args[1],"ocr")
        self.infer.return_value=c.normalize([]);self.assertEqual(self.post().json()["lines"],[])

    def test_invalid_inputs_never_start_child(self):
        for result,status in [(self.post(b"bad"),422),(self.post(png("RGBA")),400),(self.post(png(size=(64,20))),413),
            (self.client.post("/api/v1/ocr/predict",files=[("image",("input",png())),("image",("input",png()))]),400),
            (self.client.post("/api/v1/ocr/predict",files={"image":("arbitrary",png())}),400),
            (self.client.post("/api/v1/ocr/predict",content=b"x"*(9*1024*1024+1)),413)]:
            self.assertEqual(result.status_code,status);self.assertEqual(result.json()["identity"],c.fingerprint())
        self.infer.assert_not_called()

    def test_classified_errors_and_output_identity_fail_closed(self):
        for code,status in [("model_unavailable",503),("inference_failed",500),("invalid_media",422),("input_limit",413),("output_invalid",422)]:
            self.infer.return_value={"identity":c.fingerprint(),"error":code};response=self.post()
            self.assertEqual(response.status_code,status);self.assertEqual(response.json()["error"],code)
        for body in [{"identity":"0"*64,"lines":[]},{"identity":c.fingerprint(),"lines":[{"text":"partial"}]},{"identity":c.fingerprint(),"error":{}}, {"identity":c.fingerprint(),"error":"/private/secret"}]:
            self.infer.return_value=body;response=self.post();self.assertEqual(response.status_code,422);self.assertEqual(response.json()["error"],"output_invalid")
            self.assertNotIn("secret",response.text)
        for exc,code in [(Busy(),"busy"),(Unavailable(),"unavailable"),(RuntimeError("/secret"),"inference_failed")]:
            self.infer.side_effect=exc;response=self.post();self.assertEqual(response.json()["error"],code);self.assertNotIn("secret",response.text)

    def test_missing_ocr_identity_is_unavailable(self):
        with patch.object(http,"identity",None):
            self.assertEqual(self.client.get("/api/v1/ocr/identity").status_code,503)
            self.assertEqual(self.post().status_code,503);self.assertEqual(self.post().json()["identity"],c.fingerprint());self.infer.assert_not_called()


OK=(sys.executable,"-c","import sys,json;sys.stdin.buffer.read();print(json.dumps({'kind':sys.argv[1],'cache':sys.argv[2]}))")
SLEEP=(sys.executable,"-c","import sys,time;sys.stdin.buffer.read();time.sleep(60)")


class SchedulerTests(unittest.IsolatedAsyncioTestCase):
    def setUp(self):
        self.temp=tempfile.TemporaryDirectory();self.addCleanup(self.temp.cleanup)

    def owner(self,**kwargs):
        return OCRScheduler("/legacy","/tags",command=OK,ocr_cache="/ocr",ocr_command=OK,work_root=self.temp.name,**kwargs)

    async def wait_child(self,owner):
        async def ready():
            while not owner.processes:await asyncio.sleep(.005)
        await asyncio.wait_for(ready(),2);return owner.last_pid

    def reaped(self,owner,pid):
        self.assertEqual(owner.pending,0);self.assertFalse(owner.processes);self.assertEqual(os.listdir(self.temp.name),[])
        with self.assertRaises(ProcessLookupError):os.kill(pid,0)

    async def test_actual_ocr_kind_and_legacy_routing_are_distinct(self):
        owner=self.owner()
        for kind,cache in [("ocr","/ocr"),("visual","/legacy"),("textual","/legacy"),("face","/legacy"),("tag","/tags")]:
            self.assertEqual(await owner.infer(kind,b"input"),{"kind":kind,"cache":cache});self.reaped(owner,owner.last_pid)
            self.assertEqual(owner.cache,"/legacy");self.assertEqual(owner.command,OK)
        await owner.close()

    async def test_cross_family_bound_and_cancellation_restores_route(self):
        owner=self.owner();owner.ocr_command=SLEEP
        running=asyncio.create_task(owner.infer("ocr",b"input"));pid=await self.wait_child(owner)
        waiting=asyncio.create_task(owner.infer("tag",b"input"));await asyncio.sleep(.01)
        self.assertEqual(owner.pending,2);self.assertEqual(len(owner.processes),1)
        with self.assertRaises(Busy):await owner.infer("face",b"input")
        running.cancel()
        with self.assertRaises(asyncio.CancelledError):await running
        self.assertEqual(await waiting,{"kind":"tag","cache":"/tags"});self.reaped(owner,owner.last_pid)
        with self.assertRaises(ProcessLookupError):os.kill(pid,0)
        owner.ocr_command=OK;self.assertEqual((await owner.infer("ocr",b"input"))["kind"],"ocr")
        await owner.close()

    async def test_timeout_queue_closed_and_unknown_are_core_owned(self):
        owner=self.owner(timeout=.15);owner.ocr_command=SLEEP
        task=asyncio.create_task(owner.infer("ocr",b"input"));pid=await self.wait_child(owner)
        with self.assertRaises(Unavailable):await task
        self.reaped(owner,pid);self.assertEqual(owner.cache,"/legacy")
        async with owner.lock:
            with self.assertRaises(Unavailable):await owner.infer("ocr",b"input")
        self.assertEqual(owner.pending,0)
        with self.assertRaises(Unavailable):await owner.infer("other",b"input")
        await owner.close()
        with self.assertRaises(Unavailable):await owner.infer("ocr",b"input")

    async def test_disconnect_closes_real_child_and_legacy_recovers(self):
        owner=self.owner();owner.ocr_command=SLEEP
        request=AsyncMock();request.is_disconnected.return_value=True
        with patch.object(http.legacy,"scheduler",owner):
            with self.assertRaises(Unavailable):await http.legacy.run_inference(request,"ocr",b"input")
        self.reaped(owner,owner.last_pid)
        self.assertEqual((await owner.infer("visual",b"input"))["cache"],"/legacy");await owner.close()


class StartupTests(unittest.IsolatedAsyncioTestCase):
    async def test_missing_ocr_cache_does_not_disable_legacy_owner(self):
        from cache_snapshot import InvalidCache
        from scheduler import InferenceScheduler
        with tempfile.TemporaryDirectory() as folder:
            old=InferenceScheduler("/legacy","/tags",command=OK,work_root=folder)
            @asynccontextmanager
            async def base_lifespan(app):
                http.legacy.scheduler=old
                try:yield
                finally:await http.legacy.scheduler.close()
            previous=http.legacy.scheduler
            try:
                with patch.object(http.base,"lifespan",base_lifespan),patch.object(http,"snapshot",side_effect=InvalidCache()):
                    async with http.lifespan(None):
                        self.assertTrue(old.closed);self.assertIsNone(http.identity)
                        self.assertIsNone(http.legacy.scheduler.ocr_cache)
                        self.assertEqual(await http.legacy.scheduler.infer("visual",b"input"),{"kind":"visual","cache":"/legacy"})
                        with self.assertRaises(Unavailable):await http.legacy.scheduler.infer("ocr",b"input")
                    self.assertTrue(http.legacy.scheduler.closed)
            finally:http.legacy.scheduler=previous
