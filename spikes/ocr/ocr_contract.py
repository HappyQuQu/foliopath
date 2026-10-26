"""Feasibility-only OCR bounds and strict private output normalization."""
import hashlib
import json
import math
from pathlib import Path
import unicodedata

MODEL = "PP-OCRv5_server"
MAX_BYTES = 8 * 1024 * 1024
MAX_PIXELS = 4 * 1024 * 1024
MAX_EDGE = 2048
MAX_REGIONS = 128
MAX_CROP_PIXELS = 16 * 1024 * 1024
MAX_TEXT = 8192
MAX_LINE = 512
MIN_DETECTION = 0.5
MIN_RECOGNITION = 0.8

class InputLimit(Exception):
    pass
class InvalidOutput(Exception):
    pass


def entries():
    source = json.loads(Path(__file__).with_name("model-manifest.json").read_bytes())
    return [{key: item[key] for key in ("path", "size", "sha256")} for item in source["files"]]


def identity():
    spec = {"protocol": "foliopath-ocr-feasibility-v1", "model": MODEL,
            "runtimeImage": json.loads(Path(__file__).with_name("model-manifest.json").read_bytes())["runtimeImage"],
            "weights": entries(), "profile": {"rgb": True, "exifTranspose": True, "maxBytes": MAX_BYTES,
                "maxPixels": MAX_PIXELS, "maxEdge": MAX_EDGE, "detectorResolution": 736,
                "maxContoursAndRegions": MAX_REGIONS, "maxCropPixels": MAX_CROP_PIXELS,
                "maxText": MAX_TEXT, "maxLine": MAX_LINE, "minDetection": MIN_DETECTION,
                "minRecognition": MIN_RECOGNITION, "recognitionBatch": 1, "readingOrder": "horizontal-row-left-v1"},
            "sourceHashes": {name: hashlib.sha256(Path(__file__).with_name(name).read_bytes()).hexdigest()
                             for name in ("ocr_contract.py", "ocr_worker.py")}}
    return hashlib.sha256(json.dumps(spec, sort_keys=True, separators=(",", ":")).encode()).hexdigest()


def normalize(raw):
    if not isinstance(raw, dict) or set(raw) != {"text", "boxes", "detection", "recognition"}:
        raise InvalidOutput()
    values = [raw[key] for key in ("text", "boxes", "detection", "recognition")]
    if any(not isinstance(value, list) for value in values) or len({len(value) for value in values}) != 1 or len(values[0]) > MAX_REGIONS:
        raise InvalidOutput()
    lines = []
    characters = 0
    for text, box, detected, recognized in zip(*values):
        if not isinstance(text, str) or not text.strip() or len(text) > MAX_LINE or any(unicodedata.category(c) in ("Cc", "Cs") for c in text):
            raise InvalidOutput()
        text = unicodedata.normalize("NFC", text)
        if len(text) > MAX_LINE:
            raise InvalidOutput()
        characters += len(text)
        if characters > MAX_TEXT or not isinstance(box, list) or len(box) != 8:
            raise InvalidOutput()
        if any(type(value) not in (int, float) or not math.isfinite(value) or not 0 <= value <= 1 for value in [*box, detected, recognized]):
            raise InvalidOutput()
        if detected < MIN_DETECTION or recognized < MIN_RECOGNITION:
            raise InvalidOutput()
        points = list(zip(box[::2], box[1::2]))
        crosses = []
        for i in range(4):
            a, b, c = points[i], points[(i + 1) % 4], points[(i + 2) % 4]
            crosses.append((b[0] - a[0]) * (c[1] - b[1]) - (b[1] - a[1]) * (c[0] - b[0]))
        if not (all(c > 0 for c in crosses) or all(c < 0 for c in crosses)):
            raise InvalidOutput()
        lines.append({"text": text, "box": box, "detectionScore": detected, "recognitionScore": recognized})
    return {"lines": lines}


def align_detections(boxes, scores, ordered):
    """Keep the detector score paired with its exact quadrilateral after sorting."""
    from collections import defaultdict, deque
    if len(boxes) != len(scores) or len(boxes) != len(ordered):
        raise InvalidOutput()
    positions = defaultdict(deque)
    def key(box):
        if not isinstance(box, list) or len(box) != 4 or any(not isinstance(point, list) or len(point) != 2 for point in box):
            raise InvalidOutput()
        values = tuple(value for point in box for value in point)
        if any(type(v) not in (int, float) or not math.isfinite(v) for v in values):
            raise InvalidOutput()
        return values
    for index, box in enumerate(boxes):
        positions[key(box)].append(index)
    reordered = []
    for box in ordered:
        indices = positions.get(key(box))
        if not indices:
            raise InvalidOutput()
        reordered.append(scores[indices.popleft()])
    if any(type(s) not in (int, float) or not math.isfinite(s) or not MIN_DETECTION <= s <= 1 for s in reordered):
        raise InvalidOutput()
    return reordered


def reading_order(boxes):
    """Deterministic horizontal row-major experiment, not document-layout analysis."""
    bounds = []
    for index, box in enumerate(boxes):
        if len(box) != 4 or any(len(point) != 2 for point in box):
            raise InvalidOutput()
        values = [value for point in box for value in point]
        if any(type(v) not in (int, float) or not math.isfinite(v) for v in values):
            raise InvalidOutput()
        left, top = min(values[::2]), min(values[1::2])
        right, bottom = max(values[::2]), max(values[1::2])
        if right <= left or bottom <= top:
            raise InvalidOutput()
        bounds.append((index, left, top, bottom, (top + bottom) / 2))
    rows = []
    for item in sorted(bounds, key=lambda item: (item[2], item[1], item[0])):
        if rows and abs(item[4] - rows[-1][0][4]) <= min(item[3] - item[2], rows[-1][0][3] - rows[-1][0][2]) / 2:
            rows[-1].append(item)
        else:
            rows.append([item])
    return [item[0] for row in rows for item in sorted(row, key=lambda item: (item[1], item[2], item[0]))]
