#!/usr/bin/env python3
"""Stress repeated split-encoder load, inference, close, and role switching."""

from __future__ import annotations

import argparse
import gc
import json
import platform
import time
from pathlib import Path

import numpy as np
import onnxruntime as ort

from semantic_onnx_tensor_smoke import sha256


def memory_kib(field: str) -> int | None:
    try:
        with open("/proc/self/status", encoding="utf-8") as status:
            for line in status:
                if line.startswith(field + ":"):
                    return int(line.split()[1])
    except OSError:
        return None
    return None


def verify(path: Path, expected_bytes: int, expected_sha256: str) -> None:
    if path.stat().st_size != expected_bytes or sha256(path) != expected_sha256:
        raise SystemExit(f"split ONNX size or SHA-256 mismatch: {path.name}")


def new_session(
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
    parser.add_argument("--cycles", type=int, default=10)
    parser.add_argument("--threads", type=int, default=4)
    parser.add_argument("--max-cycle-rss-growth-kib", type=int, default=131072)
    parser.add_argument("--cpu-mem-arena", choices=("enabled", "disabled"), default="enabled")
    parser.add_argument("--memory-pattern", choices=("enabled", "disabled"), default="enabled")
    args = parser.parse_args()

    if args.cycles < 2 or args.cycles > 100:
        raise SystemExit("cycles must be in [2, 100]")
    verify(args.image_model, args.image_model_bytes, args.image_model_sha256)
    verify(args.text_model, args.text_model_bytes, args.text_model_sha256)
    if sha256(args.tensors) != args.expected_tensors_sha256:
        raise SystemExit("prepared tensor SHA-256 mismatch")
    prepared = np.load(args.tensors, allow_pickle=False)
    pixel_values = prepared["pixel_values"][0:1].astype(np.float32, copy=False)
    input_ids = prepared["input_ids"][0:1].astype(np.int64, copy=False)
    cpu_mem_arena = args.cpu_mem_arena == "enabled"
    memory_pattern = args.memory_pattern == "enabled"

    cycles: list[dict[str, object]] = []
    finite = True
    for cycle in range(args.cycles):
        started = time.perf_counter()
        image_session = new_session(
            args.image_model,
            args.threads,
            cpu_mem_arena,
            memory_pattern,
        )
        image_load_seconds = time.perf_counter() - started
        started = time.perf_counter()
        image_output = image_session.run(None, {"pixel_values": pixel_values})[0]
        image_inference_seconds = time.perf_counter() - started
        finite = finite and bool(np.isfinite(image_output).all())
        del image_session
        gc.collect()
        rss_after_image_close = memory_kib("VmRSS")

        started = time.perf_counter()
        text_session = new_session(
            args.text_model,
            args.threads,
            cpu_mem_arena,
            memory_pattern,
        )
        text_load_seconds = time.perf_counter() - started
        started = time.perf_counter()
        text_output = text_session.run(None, {"input_ids": input_ids})[0]
        text_inference_seconds = time.perf_counter() - started
        finite = finite and bool(np.isfinite(text_output).all())
        del text_session
        gc.collect()
        cycles.append(
            {
                "cycle": cycle + 1,
                "image_load_seconds": image_load_seconds,
                "image_inference_seconds": image_inference_seconds,
                "text_load_seconds": text_load_seconds,
                "text_inference_seconds": text_inference_seconds,
                "rss_after_image_close_kib": rss_after_image_close,
                "rss_after_cycle_kib": memory_kib("VmRSS"),
                "peak_rss_kib": memory_kib("VmHWM"),
            }
        )

    first_rss = cycles[0]["rss_after_cycle_kib"]
    final_rss = cycles[-1]["rss_after_cycle_kib"]
    rss_growth = None if first_rss is None or final_rss is None else final_rss - first_rss
    passed = finite and (rss_growth is None or rss_growth <= args.max_cycle_rss_growth_kib)
    print(
        json.dumps(
            {
                "architecture": platform.machine(),
                "onnxruntime": ort.__version__,
                "cpu_mem_arena": cpu_mem_arena,
                "memory_pattern": memory_pattern,
                "cycles": cycles,
                "all_outputs_finite": finite,
                "rss_growth_after_first_cycle_kib": rss_growth,
                "max_cycle_rss_growth_kib": args.max_cycle_rss_growth_kib,
                "passed": passed,
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0 if passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
