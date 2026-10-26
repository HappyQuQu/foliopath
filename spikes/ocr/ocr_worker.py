"""Private spike worker; never imported by the application or production AI service."""
from contextlib import redirect_stdout
import io
import json
import math
import os
from pathlib import Path
import resource
import sys
import time

from ocr_contract import (MODEL, MAX_BYTES, MAX_PIXELS, MAX_EDGE, MAX_REGIONS, MAX_CROP_PIXELS,
                          MIN_DETECTION, MIN_RECOGNITION, InputLimit, InvalidOutput, entries, identity, normalize, align_detections, reading_order)
from cache_snapshot import snapshot, InvalidCache

sys.path.insert(0, "/usr/src")

class InvalidMedia(Exception):
    pass


def prepare_image(payload):
    from PIL import Image, ImageOps
    if not payload:
        raise ValueError()
    if len(payload) > MAX_BYTES:
        raise InputLimit()
    Image.MAX_IMAGE_PIXELS = MAX_PIXELS
    with Image.open(io.BytesIO(payload)) as source:
        if source.format not in ("JPEG", "PNG", "WEBP", "GIF"):
            raise ValueError()
        if source.width * source.height > MAX_PIXELS or min(source.size) < 32:
            raise InputLimit()
        source_size = list(source.size)
        image = ImageOps.exif_transpose(source).convert("RGB")
        image.thumbnail((MAX_EDGE, MAX_EDGE), Image.Resampling.LANCZOS)
        return image, source_size


def check_contour_budget(bitmap):
    import cv2
    import numpy as np
    count = len(cv2.findContours((bitmap * 255).astype(np.uint8), cv2.RETR_LIST, cv2.CHAIN_APPROX_SIMPLE)[-2])
    if count > MAX_REGIONS:
        raise InputLimit()
    return count


def check_crop_budget(boxes):
    crop_pixels = 0
    for box in boxes:
        width = max(math.dist(box[0], box[1]), math.dist(box[2], box[3]))
        height = max(math.dist(box[0], box[3]), math.dist(box[1], box[2]))
        if not math.isfinite(width * height) or width <= 0 or height <= 0:
            raise InvalidOutput()
        crop_pixels += math.ceil(width) * math.ceil(height)
    if crop_pixels > MAX_CROP_PIXELS:
        raise InputLimit()
    return crop_pixels


def infer(cache, image):
    import cv2
    import numpy as np
    import immich_ml.models  # Preserve upstream ORT initialization and threading.
    from immich_ml.models.ocr.detection import TextDetector
    from immich_ml.models.ocr.recognition import TextRecognizer
    from rapidocr.ch_ppocr_det.utils import DBPostProcess

    class BoundedPostProcess(DBPostProcess):
        def boxes_from_bitmap(self, pred, bitmap, dest_width, dest_height):
            # Reject overflow before the upstream max_candidates truncation.
            check_contour_budget(bitmap)
            return super().boxes_from_bitmap(pred, bitmap, dest_width, dest_height)

    def no_download(*args, **kwargs):
        raise InvalidCache()
    TextDetector._download = no_download
    TextRecognizer._download = no_download
    detector = TextDetector(MODEL, cache_dir=Path(cache) / "ocr" / MODEL, min_score=MIN_DETECTION)
    detector.postprocess = BoundedPostProcess(thresh=0.3, box_thresh=MIN_DETECTION,
        max_candidates=MAX_REGIONS, unclip_ratio=1.6, use_dilation=True, score_mode="fast")
    # Retain raw box/score pairing and apply a tested horizontal row order,
    # rather than inheriting upstream array ordering as a layout guarantee.
    detector.sorted_boxes = lambda boxes: boxes
    detected = detector.predict(image)
    boxes = detected["boxes"]
    if len(boxes):
        ordered = boxes[reading_order(boxes.tolist())]
        scores = align_detections(boxes.tolist(), detected["scores"].tolist(), ordered.tolist())
        detected = {"boxes": ordered, "scores": np.asarray(scores, dtype=np.float32)}
        boxes = ordered
    if len(boxes) > MAX_REGIONS:
        raise InputLimit()
    if not len(boxes):
        return {"lines": []}
    check_crop_budget(boxes)
    recognizer = TextRecognizer(MODEL, cache_dir=Path(cache) / "ocr" / MODEL, min_score=MIN_RECOGNITION)
    result = recognizer.predict(image, detected)
    return normalize({"text": result["text"], "boxes": np.asarray(result["box"]).reshape(-1, 8).tolist(),
                      "detection": result["boxScore"].tolist(), "recognition": result["textScore"].tolist()})


def main():
    # In this isolated experiment only, the existing scheduler's 'visual' transport
    # test seam executes a fixed OCR child command. This is NOT a CLIP endpoint,
    # a production capability registration, or an embedding identity.
    kind, source, work = sys.argv[1:]
    if kind != "visual":
        raise ValueError()
    started = time.monotonic()
    payload = sys.stdin.buffer.read(MAX_BYTES + 1)
    result = {"identity": identity()}
    try:
        try:
            from PIL import Image
            try:
                image, source_size = prepare_image(payload)
            except Image.DecompressionBombError:
                raise InputLimit() from None
        except (ValueError, OSError):
            raise InvalidMedia() from None
        cache = str(snapshot(source, work, entries()))
        for key in list(os.environ):
            if key.startswith("MACHINE_LEARNING_"):
                del os.environ[key]
        os.environ.update({"MACHINE_LEARNING_CACHE_FOLDER": cache, "HF_HUB_OFFLINE": "1",
            "HF_HUB_DISABLE_TELEMETRY": "1", "MACHINE_LEARNING_REQUEST_THREADS": "1",
            "MACHINE_LEARNING_MODEL_INTRA_OP_THREADS": "2", "MACHINE_LEARNING_MODEL_INTER_OP_THREADS": "1",
            "MACHINE_LEARNING_MAX_BATCH_SIZE__OCR": "1", "MPLCONFIGDIR": "/tmp/matplotlib"})
        with redirect_stdout(sys.stderr):
            value = infer(cache, image)
        result.update({"status": "ready", "result": value, "sourceSize": source_size, "processedSize": list(image.size)})
    except InputLimit:
        result.update({"status": "failed", "error": "input_limit"})
    except InvalidCache:
        result.update({"status": "failed", "error": "model_unavailable"})
    except InvalidOutput:
        result.update({"status": "failed", "error": "output_invalid"})
    except InvalidMedia:
        result.update({"status": "failed", "error": "invalid_media"})
    except Exception:
        result.update({"status": "failed", "error": "inference_failed"})
    result["metrics"] = {"seconds": round(time.monotonic() - started, 4),
                         "peakRSSKiB": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss}
    sys.stdout.write(json.dumps(result, separators=(",", ":"), allow_nan=False))
    sys.stdout.flush()

if __name__ == "__main__":
    main()
