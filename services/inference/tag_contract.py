"""Fixed JoyTag output policy. Not prompts, a user vocabulary, or calibrated probabilities."""
import hashlib
import json
import math

MODEL = "JoyTag"
MODEL_SHA256 = "f85b7130e6e549b5b0822537007b7482e8c4c8e754c8d9a5bee08e27050e1097"
THRESHOLD = 0.4
MAX_LABELS = 20
SAFE_LABELS = frozenset(
    "airplane apple backpack bag beach bicycle bird blurry_background boat book "
    "bottle bread bridge building butterfly camera car castle cat chair cherry_blossoms "
    "city cloud computer cup day depth_of_field dog door dress fence fish flower "
    "flower_field food forest fruit glasses grass hat indoors jacket lake "
    "light_particles monochrome moon motorcycle mountain night ocean outdoors phone "
    "plant pool rain river road rock running sand scenery shirt ship sitting skirt "
    "sky snow standing stairs street sunrise sunset swimming sword table train tree "
    "umbrella walking water waterfall window wooden_floor blue_background "
    "red_background green_background yellow_background white_background "
    "black_background pink_background purple_background grey_background".split()
)
POLICY = {"version": 1, "labels": sorted(SAFE_LABELS), "name_mapping": "canonical-english-class-id-v1",
          "threshold": THRESHOLD, "max_labels": MAX_LABELS, "score": "sigmoid-logits",
          "preprocessing": "exif-rgb:white-square-center:bicubic448:clip-mean-std:nchw-float32-v1"}
POLICY_SHA256 = hashlib.sha256(json.dumps(POLICY, sort_keys=True, separators=(",", ":")).encode()).hexdigest()
MANIFEST = [
    {"path": "tagging/JoyTag/model.onnx", "size": 366116154, "sha256": MODEL_SHA256},
    {"path": "tagging/JoyTag/config.json", "size": 330,
     "sha256": "b53ee29a8f0d6353331f455fd640c70ef45e5bd979b656f7b9983cffa3724562"},
    {"path": "tagging/JoyTag/top_tags.txt", "size": 76752,
     "sha256": "32b1963a234af848643b2bbf47d8eff1f1c7889406810c57b980f41b2b9e01d0"},
]


def normalize(result):
    """Fail closed on the entire child response, never expose an unfiltered class."""
    if not isinstance(result, dict) or set(result) != {"labels"}:
        raise ValueError()
    labels = result["labels"]
    if not isinstance(labels, list) or len(labels) > MAX_LABELS:
        raise ValueError()
    seen = set()
    previous = None
    for item in labels:
        if not isinstance(item, dict) or set(item) != {"name", "confidence"}:
            raise ValueError()
        name, score = item["name"], item["confidence"]
        if not isinstance(name, str) or name not in SAFE_LABELS or name in seen or type(score) not in (float, int):
            raise ValueError()
        if not math.isfinite(score) or not THRESHOLD <= score <= 1:
            raise ValueError()
        order = (-score, name)
        if previous is not None and order < previous:
            raise ValueError()
        previous = order
        seen.add(name)
    return result
