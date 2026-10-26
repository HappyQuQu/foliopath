"""Route adapter only; frozen scheduler still owns every admission and child."""
from contextvars import ContextVar
from pathlib import Path
import sys

from scheduler import InferenceScheduler, Unavailable


class OCRScheduler(InferenceScheduler):
    def __init__(self, *args, ocr_cache=None, ocr_command=None, **kwargs):
        super().__init__(*args, **kwargs)
        self.ocr_cache = str(ocr_cache) if ocr_cache is not None else None
        self.ocr_command = ocr_command or (sys.executable, str(Path(__file__).with_name("ocr_worker.py")))
        self._route = ContextVar("ocr_route", default=None)

    async def infer(self, kind, payload):
        if kind == "ocr" and self.ocr_cache is None:
            raise Unavailable()
        token = self._route.set(kind)
        try:
            # Admission compatibility category only. No CLIP inference/protocol
            # sees this alias: the child gets explicit 'ocr' below.
            return await super().infer("visual" if kind == "ocr" else kind, payload)
        finally:
            self._route.reset(token)

    async def _run(self, kind, payload):
        if self._route.get() != "ocr":
            return await super()._run(kind, payload)
        # The original infer has already acquired its one lock. Restore routing
        # only after original _run's finally reaps the child and its snapshot.
        old_cache, old_command = self.cache, self.command
        self.cache, self.command = self.ocr_cache, self.ocr_command
        try:
            return await super()._run("ocr", payload)
        finally:
            self.cache, self.command = old_cache, old_command
