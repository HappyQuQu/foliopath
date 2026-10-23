"""Private face-v1 contract. Never expose embeddings through the public API."""
import json
import math

MODEL = "buffalo_l"
MAX_FACES = 64
DIMENSION = 512
ENTRIES = {"facial-recognition": {
    "detection": {"modelName": MODEL, "options": {"minScore": 0.7}},
    "recognition": {"modelName": MODEL},
}}


def normalize(result, width, height):
    """Validate the entire result; overflow/malformed output is not partial success."""
    if result.get("imageWidth") != width or result.get("imageHeight") != height:
        raise ValueError()
    faces = result.get("facial-recognition")
    if not isinstance(faces, list) or len(faces) > MAX_FACES:
        raise ValueError()
    output = []
    for face in faces:
        box = face["boundingBox"]
        coords = [box[key] for key in ("x1", "y1", "x2", "y2")]
        score = face["score"]
        if any(type(v) not in (int, float) or not math.isfinite(v) for v in [*coords, score]):
            raise ValueError()
        x1, y1, x2, y2 = coords
        if not 0.7 <= score <= 1 or x2 <= x1 or y2 <= y1:
            raise ValueError()
        # Detectors can report boxes crossing image edges. Clip, then reject empty boxes.
        x1, y1, x2, y2 = max(0, x1), max(0, y1), min(width, x2), min(height, y2)
        if x2 <= x1 or y2 <= y1:
            raise ValueError()
        vector = json.loads(face["embedding"])
        if not isinstance(vector, list) or len(vector) != DIMENSION or any(
            type(v) not in (int, float) or not math.isfinite(v) for v in vector
        ):
            raise ValueError()
        norm = math.hypot(*vector)
        if not math.isfinite(norm) or norm <= 1e-12:
            raise ValueError()
        output.append({"box": {"x": x1 / width, "y": y1 / height,
                               "width": (x2 - x1) / width, "height": (y2 - y1) / height},
                       "detection": score, "embedding": [v / norm for v in vector]})
    return {"faces": output}
