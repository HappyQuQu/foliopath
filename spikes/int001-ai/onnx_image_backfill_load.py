#!/usr/bin/env python3
"""Hold the split image encoder under bounded load until a stop file appears."""

from __future__ import annotations

import argparse
import json
import platform
import statistics
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


def percentile(values: list[float], quantile: float) -> float:
    ordered = sorted(values)
    return ordered[max(0, int(np.ceil(quantile * len(ordered))) - 1)]


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True, type=Path)
    parser.add_argument("--model-bytes", required=True, type=int)
    parser.add_argument("--model-sha256", required=True)
    parser.add_argument("--tensors", required=True, type=Path)
    parser.add_argument("--expected-tensors-sha256", required=True)
    parser.add_argument("--stop-file", required=True, type=Path)
    parser.add_argument("--threads", type=int, default=2)
    parser.add_argument("--sleep-ms", type=float, default=20)
    args = parser.parse_args()

    if args.model.stat().st_size != args.model_bytes or sha256(args.model) != args.model_sha256:
        raise SystemExit("image encoder size or SHA-256 mismatch")
    if sha256(args.tensors) != args.expected_tensors_sha256:
        raise SystemExit("prepared tensor SHA-256 mismatch")
    if args.stop_file.exists():
        raise SystemExit("stop file already exists")
    if args.threads < 1 or args.threads > 4 or args.sleep_ms < 0:
        raise SystemExit("invalid bounded-load settings")

    prepared = np.load(args.tensors, allow_pickle=False)
    pixels = prepared["pixel_values"].astype(np.float32, copy=False)
    options = ort.SessionOptions()
    options.intra_op_num_threads = args.threads
    options.inter_op_num_threads = 1
    options.enable_cpu_mem_arena = False
    options.enable_mem_pattern = True
    session = ort.InferenceSession(
        args.model,
        sess_options=options,
        providers=["CPUExecutionProvider"],
    )

    latencies: list[float] = []
    finite = True
    index = 0
    started_at = time.perf_counter()
    while not args.stop_file.exists():
        started = time.perf_counter()
        output = session.run(None, {"pixel_values": pixels[index : index + 1]})[0]
        latencies.append(time.perf_counter() - started)
        finite = finite and bool(np.isfinite(output).all())
        index = (index + 1) % len(pixels)
        if args.sleep_ms:
            time.sleep(args.sleep_ms / 1000)
    elapsed = time.perf_counter() - started_at
    if not latencies:
        raise SystemExit("stop requested before the first inference")

    print(
        json.dumps(
            {
                "architecture": platform.machine(),
                "onnxruntime": ort.__version__,
                "cpu_mem_arena": False,
                "memory_pattern": True,
                "threads": args.threads,
                "sleep_ms": args.sleep_ms,
                "inferences": len(latencies),
                "elapsed_seconds": elapsed,
                "inference_p50_seconds": statistics.median(latencies),
                "inference_p95_seconds": percentile(latencies, 0.95),
                "inference_max_seconds": max(latencies),
                "all_outputs_finite": finite,
                "current_rss_kib": memory_kib("VmRSS"),
                "peak_rss_kib": memory_kib("VmHWM"),
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0 if finite else 1


if __name__ == "__main__":
    raise SystemExit(main())
