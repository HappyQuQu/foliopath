"""One bounded child; legacy face/tag exec the unchanged legacy worker."""
import asyncio
from contextlib import redirect_stdout
import io
import json
import os
from pathlib import Path
import sys

from multilingual_contract import MODEL, request_entries, verified_snapshot


def main():
    kind, source, work = sys.argv[1:]
    if kind in ("face", "tag"):
        os.execv(sys.executable, [sys.executable, str(Path(__file__).with_name("inference_worker.py")), *sys.argv[1:]])
    if kind not in ("visual", "textual"):
        raise ValueError()
    payload = sys.stdin.buffer.read(8 * 1024 * 1024 + 1)
    if not 1 <= len(payload) <= 8 * 1024 * 1024:
        raise ValueError()
    cache = str(verified_snapshot(source, work, request_entries(kind)))
    for key in list(os.environ):
        if key.startswith("MACHINE_LEARNING_"):
            del os.environ[key]
    os.environ.update({"MACHINE_LEARNING_CACHE_FOLDER": cache, "HF_HUB_OFFLINE": "1",
                       "HF_HUB_DISABLE_TELEMETRY": "1", "MACHINE_LEARNING_REQUEST_THREADS": "1",
                       "MACHINE_LEARNING_MODEL_INTRA_OP_THREADS": "2", "MACHINE_LEARNING_MODEL_INTER_OP_THREADS": "1",
                       "MACHINE_LEARNING_MODEL_TTL": "0", "MACHINE_LEARNING_ANN": "false"})
    sys.path.insert(0, "/usr/src")

    async def infer():
        from immich_ml import main as upstream
        async with upstream.lifespan(None):
            response = await upstream.predict(upstream.get_entries(json.dumps({"clip": {kind: {"modelName": MODEL}}})),
                                               image=payload if kind == "visual" else None,
                                               text=payload.decode("utf-8") if kind == "textual" else None)
            return json.loads(response.body)

    with redirect_stdout(sys.stderr):
        result = asyncio.run(infer())
    sys.stdout.write(json.dumps(result, separators=(",", ":"), allow_nan=False))
    sys.stdout.flush()


if __name__ == "__main__":
    main()
