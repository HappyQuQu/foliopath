"""Offline feasibility probe; not an application adapter or a quality benchmark.

Only verified model data and an explicit list of repository synthetic PNGs are read.
No model-authored Python, production media, databases, or HTTP clients are used.
"""

import argparse
import hashlib
import json
import platform
import resource
import statistics
import time
from pathlib import Path

import numpy as np
import onnxruntime as ort
from PIL import Image, ImageOps


# A spike-only output filter, not text prompts or user-supplied CLIP candidates.
# The trained head still independently predicts all 5813 built-in classes.
# Only objective objects/scenes/actions/colors are emitted; all other labels,
# including character identities, intimate anatomy and sensitive traits, are omitted.
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

SYNTHETIC_FIXTURES = tuple(
    f"prototypes/apple-redesign/assets/ai-media/{series}-{index:02d}.png"
    for series in ("azure", "crimson", "forest")
    for index in range(1, 4)
) + (
    "tests/fixtures/ai-faces/synthetic-pair.png",
    "tests/fixtures/media/viewer-blue-violet.png",
)


def sha256_file(path):
    digest = hashlib.sha256()
    with path.open("rb") as source:
        while chunk := source.read(1024 * 1024):
            digest.update(chunk)
    return digest.hexdigest()


def verify_model(directory, manifest):
    for item in manifest["files"]:
        path = directory / item["name"]
        if path.is_symlink() or not path.is_file():
            raise ValueError(f"Model file missing or symlink: {item['name']}")
        if path.stat().st_size != item["bytes"] or sha256_file(path) != item["sha256"]:
            raise ValueError(f"Model hash/size mismatch: {item['name']}")


def prepare_image(path, image_size=448):
    if path.is_symlink() or path.stat().st_size > 20 * 1024 * 1024:
        raise ValueError("Synthetic fixture must be a regular bounded file")
    with Image.open(path) as opened:
        if opened.width * opened.height > 10_000_000:
            raise ValueError("Fixture pixel limit exceeded")
        image = ImageOps.exif_transpose(opened).convert("RGB")
        edge = max(image.size)
        square = Image.new("RGB", (edge, edge), (255, 255, 255))
        square.paste(image, ((edge - image.width) // 2, (edge - image.height) // 2))
        square = square.resize((image_size, image_size), Image.Resampling.BICUBIC)
        pixels = np.asarray(square, dtype=np.float32) / np.float32(255.0)
    mean = np.array([0.48145466, 0.4578275, 0.40821073], dtype=np.float32)
    std = np.array([0.26862954, 0.26130258, 0.27577711], dtype=np.float32)
    return np.ascontiguousarray(((pixels - mean) / std).transpose(2, 0, 1)[None])


def select_safe_tags(logits, labels, threshold=0.4, limit=10):
    values = np.asarray(logits, dtype=np.float32)
    if values.shape != (1, len(labels)) or not np.isfinite(values).all():
        raise ValueError("Unexpected or non-finite model output")
    # Pinned graph ends in Gemm, not Sigmoid (opset 17, pytorch 2.2.1).
    # The author applies sigmoid to logits; do not apply CLIP cosine conversion.
    scores = 1 / (1 + np.exp(-np.clip(values[0], -80, 80)))
    results = sorted(
        ((label, float(scores[index])) for index, label in enumerate(labels)
         if label in SAFE_LABELS and scores[index] >= threshold),
        key=lambda item: (-item[1], item[0]),
    )[:limit]
    return [{"tag": label, "model_score": round(score, 6)} for label, score in results]


def cgroup_number(name):
    try:
        return int((Path("/sys/fs/cgroup") / name).read_text().strip())
    except (OSError, ValueError):
        return None


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", type=Path, default=Path("/model"))
    parser.add_argument("--repo", type=Path, default=Path("/repo"))
    parser.add_argument("--repeats", type=int, default=3)
    args = parser.parse_args()
    if not 1 <= args.repeats <= 5:
        raise ValueError("Repeats must be 1..5")
    manifest = json.loads(Path(__file__).with_name("model-manifest.json").read_text())
    verify_model(args.model, manifest)
    config = json.loads((args.model / "config.json").read_text())
    labels = (args.model / "top_tags.txt").read_text().splitlines()
    if config["image_size"] != 448 or config["n_tags"] != 5813 or len(labels) != 5813:
        raise ValueError("Pinned label count/image shape changed")

    options = ort.SessionOptions()
    options.intra_op_num_threads = 2
    options.inter_op_num_threads = 1
    options.execution_mode = ort.ExecutionMode.ORT_SEQUENTIAL
    started = time.perf_counter()
    session = ort.InferenceSession(str(args.model / "model.onnx"), options,
                                   providers=["CPUExecutionProvider"])
    load_ms = (time.perf_counter() - started) * 1000
    inputs, outputs = session.get_inputs(), session.get_outputs()
    if len(inputs) != 1 or inputs[0].shape[1:] != [3, 448, 448] or len(outputs) != 1:
        raise ValueError("Unexpected ONNX input/output contract")

    # First inference is separately reported; retained rows use the warm session.
    warm_tensor = prepare_image(args.repo / SYNTHETIC_FIXTURES[0])
    started = time.perf_counter()
    session.run(None, {inputs[0].name: warm_tensor})
    first_inference_ms = (time.perf_counter() - started) * 1000
    rows, all_timings = [], []
    for relative in SYNTHETIC_FIXTURES:
        path = args.repo / relative
        tensor = prepare_image(path)
        timings = []
        for _ in range(args.repeats):
            started = time.perf_counter()
            predictions = session.run(None, {inputs[0].name: tensor})[0]
            timings.append((time.perf_counter() - started) * 1000)
        all_timings.extend(timings)
        rows.append({"fixture": relative, "sha256": sha256_file(path),
                     "synthetic": True, "inference_ms": [round(t, 3) for t in timings],
                     "tags": select_safe_tags(predictions, labels)})

    print(json.dumps({
        "purpose": "isolated feasibility only; not generalization or release approval",
        "model_repository": manifest["repository"], "model_revision": manifest["revision"],
        "model_sha256": manifest["files"][0]["sha256"],
        "model_label_count": len(labels), "reported_allowlist_count": len(SAFE_LABELS),
        "threshold": 0.4, "threshold_is_calibrated": False,
        "score_semantics": "sigmoid of trained multi-label logits; not CLIP candidate ranking",
        "platform": platform.platform(), "architecture": platform.machine(),
        "python": platform.python_version(), "onnxruntime": ort.__version__,
        "providers": session.get_providers(), "intra_op_threads": 2, "inter_op_threads": 1,
        "cold_session_load_ms": round(load_ms, 3),
        "first_inference_ms": round(first_inference_ms, 3),
        "inference_p50_ms": round(statistics.median(all_timings), 3),
        "inference_p95_ms": round(float(np.percentile(all_timings, 95)), 3),
        "process_peak_rss_kib": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
        "container_memory_peak_bytes": cgroup_number("memory.peak"),
        "container_memory_current_bytes": cgroup_number("memory.current"),
        "container_memory_limit_bytes": cgroup_number("memory.max"),
        "notes": ["No CLIP/face sessions loaded; combined residency is untested.",
                  "Model read via readonly bind; production tmpfs copy is not simulated.",
                  "Only 11 pre-existing synthetic fixtures; no vehicle/landscape quality set.",
                  "Excluded labels are not printed or persisted."],
        "fixtures": rows,
    }, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
