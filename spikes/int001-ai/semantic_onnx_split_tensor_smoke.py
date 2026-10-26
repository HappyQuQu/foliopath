#!/usr/bin/env python3
"""Score prepared semantic tensors using fixed image/text encoder graphs."""

from __future__ import annotations

import argparse
import gc
import json
import platform
import time
from pathlib import Path

import numpy as np
import onnxruntime as ort

from semantic_onnx_tensor_smoke import metrics, normalize, results, sha256


def memory_kib(field: str) -> int | None:
    try:
        with open("/proc/self/status", encoding="utf-8") as status:
            for line in status:
                if line.startswith(field + ":"):
                    return int(line.split()[1])
    except OSError:
        return None
    return None


def session(
    path: Path,
    threads: int,
    cpu_mem_arena: bool,
    memory_pattern: bool,
) -> ort.InferenceSession:
    options = ort.SessionOptions()
    options.intra_op_num_threads = threads
    options.inter_op_num_threads = 1
    options.enable_cpu_mem_arena = cpu_mem_arena
    options.enable_mem_pattern = memory_pattern
    return ort.InferenceSession(path, sess_options=options, providers=["CPUExecutionProvider"])


def verify(path: Path, expected_bytes: int, expected_sha256: str) -> None:
    if path.stat().st_size != expected_bytes or sha256(path) != expected_sha256:
        raise SystemExit(f"split ONNX size or SHA-256 mismatch: {path.name}")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--image-model", required=True, type=Path)
    parser.add_argument("--text-model", required=True, type=Path)
    parser.add_argument("--image-model-bytes", required=True, type=int)
    parser.add_argument("--image-model-sha256", required=True)
    parser.add_argument("--text-model-bytes", required=True, type=int)
    parser.add_argument("--text-model-sha256", required=True)
    parser.add_argument("--tensors", required=True, type=Path)
    parser.add_argument("--expected-tensors-sha256", required=True)
    parser.add_argument("--fixture", required=True, type=Path)
    parser.add_argument("--threads", type=int, default=4)
    parser.add_argument("--cpu-mem-arena", choices=("enabled", "disabled"), default="enabled")
    parser.add_argument("--memory-pattern", choices=("enabled", "disabled"), default="enabled")
    args = parser.parse_args()

    verify(args.image_model, args.image_model_bytes, args.image_model_sha256)
    verify(args.text_model, args.text_model_bytes, args.text_model_sha256)
    if sha256(args.tensors) != args.expected_tensors_sha256:
        raise SystemExit("prepared tensor SHA-256 mismatch")
    fixture = json.loads(args.fixture.read_text(encoding="utf-8"))
    prepared = np.load(args.tensors, allow_pickle=False)
    if str(prepared["fixture_sha256"]) != sha256(args.fixture):
        raise SystemExit("prepared tensors do not bind to this fixture")
    item_ids = [str(item) for item in prepared["item_ids"]]
    if item_ids != [item["id"] for item in fixture["items"]]:
        raise SystemExit("prepared item order does not match fixture")

    cpu_mem_arena = args.cpu_mem_arena == "enabled"
    memory_pattern = args.memory_pattern == "enabled"
    image_session = session(args.image_model, args.threads, cpu_mem_arena, memory_pattern)
    image_latencies: list[float] = []
    image_embeddings: list[np.ndarray] = []
    for pixel in prepared["pixel_values"].astype(np.float32, copy=False):
        started = time.perf_counter()
        output = image_session.run(None, {"pixel_values": pixel[None, ...]})[0]
        image_latencies.append(time.perf_counter() - started)
        image_embeddings.append(output[0])
    rss_after_image_kib = memory_kib("VmRSS")
    del image_session
    gc.collect()
    rss_after_image_close_kib = memory_kib("VmRSS")

    text_session = session(args.text_model, args.threads, cpu_mem_arena, memory_pattern)
    text_latencies: list[float] = []
    text_embeddings: list[np.ndarray] = []
    for input_ids in prepared["input_ids"].astype(np.int64, copy=False):
        started = time.perf_counter()
        output = text_session.run(None, {"input_ids": input_ids[None, ...]})[0]
        text_latencies.append(time.perf_counter() - started)
        text_embeddings.append(output[0])
    rss_after_text_kib = memory_kib("VmRSS")

    image_matrix = normalize(np.stack(image_embeddings))
    text_matrix = normalize(np.stack(text_embeddings))
    float32 = results(fixture["queries"], item_ids, text_matrix @ image_matrix.T)
    float16_images = normalize(image_matrix.astype(np.float16).astype(np.float32))
    float16 = results(fixture["queries"], item_ids, text_matrix @ float16_images.T)
    print(
        json.dumps(
            {
                "architecture": platform.machine(),
                "onnxruntime": ort.__version__,
                "cpu_mem_arena": cpu_mem_arena,
                "memory_pattern": memory_pattern,
                "image_inference_seconds": image_latencies,
                "text_inference_seconds": text_latencies,
                "rss_after_image_kib": rss_after_image_kib,
                "rss_after_image_close_kib": rss_after_image_close_kib,
                "rss_after_text_kib": rss_after_text_kib,
                "peak_rss_kib": memory_kib("VmHWM"),
                "float32": {
                    "zh-CN": metrics(float32, "zh-CN"),
                    "en": metrics(float32, "en"),
                    "results": float32,
                },
                "float16_image_storage": {
                    "zh-CN": metrics(float16, "zh-CN"),
                    "en": metrics(float16, "en"),
                    "top3_identical_queries": sum(
                        left["top3"] == right["top3"]
                        for left, right in zip(float32, float16, strict=True)
                    ),
                    "results": float16,
                },
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
