#!/usr/bin/env python3
"""Synthetic-only ONNX load and inference smoke for INT-001 candidates."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import platform
import statistics
import time
from pathlib import Path

import numpy as np
import onnxruntime as ort
import psutil


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", required=True, type=Path)
    parser.add_argument("--model-root", required=True, type=Path)
    parser.add_argument("--iterations", type=int, default=100)
    return parser.parse_args()


def percentile(values: list[float], quantile: float) -> float:
    ordered = sorted(values)
    return ordered[max(0, int(np.ceil(quantile * len(ordered))) - 1)]


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def synthetic_input(purpose: str, seed: int) -> np.ndarray:
    rng = np.random.default_rng(seed)
    if purpose == "face_detection":
        # Capacity smoke only. This is not a face-quality fixture.
        image = rng.uniform(-1.0, 1.0, (1, 3, 640, 640))
    elif purpose == "face_embedding":
        image = rng.uniform(-1.0, 1.0, (1, 3, 112, 112))
    else:
        raise ValueError(f"unsupported purpose: {purpose}")
    return image.astype(np.float32)


def smoke_model(entry: dict[str, object], root: Path, iterations: int) -> dict[str, object]:
    path = root / str(entry["filename"])
    size = path.stat().st_size
    actual_hash = sha256_file(path)
    if size != entry["size_bytes"] or actual_hash != entry["sha256"]:
        raise RuntimeError(f"artifact verification failed for {path.name}")

    options = ort.SessionOptions()
    options.intra_op_num_threads = 1
    options.inter_op_num_threads = 1
    options.execution_mode = ort.ExecutionMode.ORT_SEQUENTIAL
    options.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
    options.log_severity_level = 3

    process = psutil.Process()
    rss_before = process.memory_info().rss
    loaded_at = time.perf_counter()
    session = ort.InferenceSession(
        str(path), sess_options=options, providers=["CPUExecutionProvider"]
    )
    load_ms = (time.perf_counter() - loaded_at) * 1000
    rss_after_load = process.memory_info().rss

    public_inputs = session.get_inputs()
    if len(public_inputs) != 1:
        raise RuntimeError(f"expected one runtime input, got {len(public_inputs)}")
    tensor = synthetic_input(str(entry["purpose"]), 20260825)
    session.run(None, {public_inputs[0].name: tensor})
    # Separate one-time allocator/session warmup growth from steady-loop growth.
    rss_after_warmup = process.memory_info().rss

    latencies: list[float] = []
    last_outputs: list[np.ndarray] = []
    rss_samples: list[dict[str, int]] = []
    sample_every = max(1, iterations // 10)
    for index in range(iterations):
        started = time.perf_counter()
        last_outputs = session.run(None, {public_inputs[0].name: tensor})
        latencies.append((time.perf_counter() - started) * 1000)
        if (index + 1) % sample_every == 0 or index + 1 == iterations:
            rss_samples.append(
                {"iteration": index + 1, "rss_bytes": process.memory_info().rss}
            )
    if not last_outputs or not all(np.isfinite(value).all() for value in last_outputs):
        raise RuntimeError("model emitted empty or non-finite output")

    rss_after_loop = process.memory_info().rss
    return {
        "id": entry["id"],
        "status": entry["status"],
        "purpose": entry["purpose"],
        "filename": path.name,
        "size_bytes": size,
        "sha256": actual_hash,
        "load_ms": round(load_ms, 3),
        "inference_p50_ms": round(statistics.median(latencies), 3),
        "inference_p95_ms": round(percentile(latencies, 0.95), 3),
        "inference_p99_ms": round(percentile(latencies, 0.99), 3),
        "rss_before_bytes": rss_before,
        "rss_after_load_bytes": rss_after_load,
        "rss_after_warmup_bytes": rss_after_warmup,
        "rss_after_loop_bytes": rss_after_loop,
        "rss_load_delta_bytes": rss_after_load - rss_before,
        "rss_loop_delta_bytes": rss_after_loop - rss_after_warmup,
        "rss_samples": rss_samples,
        "input": {
            "name": public_inputs[0].name,
            "shape": public_inputs[0].shape,
            "type": public_inputs[0].type,
        },
        "outputs": [
            {"name": meta.name, "shape": list(value.shape), "type": meta.type}
            for meta, value in zip(session.get_outputs(), last_outputs, strict=True)
        ],
    }


def main() -> None:
    args = parse_args()
    if args.iterations <= 0:
        raise SystemExit("--iterations must be positive")
    catalog = json.loads(args.catalog.read_text(encoding="utf-8"))
    models = [
        smoke_model(entry, args.model_root, args.iterations)
        for entry in catalog["models"]
        if entry["status"] in {"candidate", "approved"}
    ]
    report = {
        "schema_version": 1,
        "evidence_class": "synthetic-development-only",
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "environment": {
            "os": platform.system().lower(),
            "machine": platform.machine(),
            "python": platform.python_version(),
            "onnxruntime": ort.__version__,
            "numpy": np.__version__,
            "cpu_count": os.cpu_count(),
            "providers": ort.get_available_providers(),
            "intra_op_threads": 1,
            "inter_op_threads": 1,
        },
        "iterations": args.iterations,
        "models": models,
        "caveats": [
            "Random tensors prove load/inference compatibility and capacity only.",
            "No detector recall, embedding ROC, clustering precision, or demographic quality is measured.",
            "macOS arm64 Python wheels are not evidence for the planned Go/C API Linux runtime.",
        ],
    }
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    main()
