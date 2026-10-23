"""One request, one model family, one process. No HTTP listener or durable queue."""
import asyncio
from contextlib import redirect_stdout
import io
import json
import os
from pathlib import Path
import sys

import face_contract
import tag_contract
from cache_snapshot import snapshot
from runtime_manifest import request_entries

MAX_INPUT = 8 * 1024 * 1024


def prepare_tag_image(payload):
    import numpy as np
    from PIL import Image, ImageOps
    with Image.open(io.BytesIO(payload)) as opened:
        image = ImageOps.exif_transpose(opened).convert("RGB")
        edge = max(image.size)
        square = Image.new("RGB", (edge, edge), (255, 255, 255))
        square.paste(image, ((edge - image.width) // 2, (edge - image.height) // 2))
        pixels = np.asarray(square.resize((448, 448), Image.Resampling.BICUBIC), dtype=np.float32) / np.float32(255)
    mean = np.array([0.48145466, 0.4578275, 0.40821073], dtype=np.float32)
    std = np.array([0.26862954, 0.26130258, 0.27577711], dtype=np.float32)
    return np.ascontiguousarray(((pixels - mean) / std).transpose(2, 0, 1)[None])


def infer_tags(cache, payload):
    import numpy as np
    import immich_ml.models  # Canonical upstream ORT initialization order/options.
    from immich_ml.sessions.ort import OrtSession
    model = Path(cache) / "tagging/JoyTag"
    config = json.loads((model / "config.json").read_text())
    labels = (model / "top_tags.txt").read_text().splitlines()
    if config["image_size"] != 448 or config["n_tags"] != 5813 or len(labels) != 5813 or len(set(labels)) != 5813:
        raise ValueError()
    session = OrtSession(model / "model.onnx", providers=["CPUExecutionProvider"])
    if len(session.get_inputs()) != 1 or session.get_inputs()[0].shape[1:] != [3, 448, 448]:
        raise ValueError()
    output = session.run(None, {session.get_inputs()[0].name: prepare_tag_image(payload)})[0]
    values = np.asarray(output, dtype=np.float32)
    if values.shape != (1, len(labels)) or not np.isfinite(values).all():
        raise ValueError()
    scores = 1 / (1 + np.exp(-np.clip(values[0], -80, 80)))
    results = sorted(((name, float(scores[index])) for index, name in enumerate(labels)
                      if name in tag_contract.SAFE_LABELS and scores[index] >= tag_contract.THRESHOLD),
                     key=lambda item: (-item[1], item[0]))[:tag_contract.MAX_LABELS]
    return tag_contract.normalize({"labels": [{"name": name, "confidence": score} for name, score in results]})


async def infer_upstream(kind, cache, payload):
    from immich_ml import main
    async with main.lifespan(None):
        entries = face_contract.ENTRIES if kind == "face" else {"clip": {kind: {"modelName": "ViT-B-32__openai"}}}
        response = await main.predict(main.get_entries(json.dumps(entries)),
                                      image=None if kind == "textual" else payload,
                                      text=payload.decode("utf-8") if kind == "textual" else None)
        result = json.loads(response.body)
        if kind == "face":
            from PIL import Image
            with Image.open(io.BytesIO(payload)) as image:
                result = face_contract.normalize(result, *image.size)
        return result


def main():
    kind, source, work = sys.argv[1:]
    if kind not in ("visual", "textual", "face", "tag"):
        raise ValueError()
    payload = sys.stdin.buffer.read(MAX_INPUT + 1)
    if not 1 <= len(payload) <= MAX_INPUT:
        raise ValueError()
    # Copy/hash and inference share the same killable 26s child lifetime. The
    # parent owns this work directory and deletes it only after reaping us.
    cache = str(snapshot(source, work, request_entries(kind)))
    for key in list(os.environ):
        if key.startswith("MACHINE_LEARNING_"):
            del os.environ[key]
    os.environ.update({"MACHINE_LEARNING_CACHE_FOLDER": cache, "HF_HUB_OFFLINE": "1",
                       "HF_HUB_DISABLE_TELEMETRY": "1", "MACHINE_LEARNING_REQUEST_THREADS": "1",
                       "MACHINE_LEARNING_MODEL_INTRA_OP_THREADS": "2", "MACHINE_LEARNING_MODEL_INTER_OP_THREADS": "1",
                       "MACHINE_LEARNING_MODEL_TTL": "0", "MACHINE_LEARNING_ANN": "false",
                       "MACHINE_LEARNING_MAX_BATCH_SIZE__FACIAL_RECOGNITION": "1"})
    sys.path.insert(0, "/usr/src")
    # Upstream logs cannot contaminate the bounded stdout protocol or leak to HTTP.
    with redirect_stdout(sys.stderr):
        result = infer_tags(cache, payload) if kind == "tag" else asyncio.run(infer_upstream(kind, cache, payload))
    sys.stdout.write(json.dumps(result, separators=(",", ":"), allow_nan=False))
    sys.stdout.flush()


if __name__ == "__main__":
    main()
