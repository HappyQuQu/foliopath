#!/usr/bin/env python3
"""Measure two bounded resident split-encoder sessions under alternating load."""

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


def proc_memory_kib(field: str) -> int | None:
    try:
        with open("/proc/self/status", encoding="utf-8") as status:
            for line in status:
                if line.startswith(field + ":"):
                    return int(line.split()[1])
    except OSError:
        return None
    return None


def cgroup_value(name: str) -> int | None:
    try:
        return int(Path("/sys/fs/cgroup", name).read_text(encoding="utf-8").strip())
    except (OSError, ValueError):
        return None


def cgroup_stat() -> dict[str, int]:
    result: dict[str, int] = {}
    try:
        for line in Path("/sys/fs/cgroup/memory.stat").read_text(encoding="utf-8").splitlines():
            key, value = line.split()
            if key in {"anon", "file", "kernel", "pagetables"}:
                result[key] = int(value)
    except (OSError, ValueError):
        return {}
    return result


def snapshot(stage: str) -> dict[str, object]:
    return {
        "stage": stage,
        "process_rss_kib": proc_memory_kib("VmRSS"),
        "process_peak_rss_kib": proc_memory_kib("VmHWM"),
        "cgroup_current_bytes": cgroup_value("memory.current"),
        "cgroup_peak_bytes": cgroup_value("memory.peak"),
        "cgroup_stat_bytes": cgroup_stat(),
    }


def verify(path: Path, expected_bytes: int, expected_sha256: str) -> None:
    if path.stat().st_size != expected_bytes or sha256(path) != expected_sha256:
        raise SystemExit(f"split ONNX size or SHA-256 mismatch: {path.name}")


def new_session(path: Path, threads: int) -> ort.InferenceSession:
    options = ort.SessionOptions()
    options.intra_op_num_threads = threads
    options.inter_op_num_threads = 1
    options.enable_cpu_mem_arena = False
    options.enable_mem_pattern = True
    return ort.InferenceSession(path, sess_options=options, providers=["CPUExecutionProvider"])


def percentile(values: list[float], quantile: float) -> float:
    ordered = sorted(values)
    return ordered[max(0, int(np.ceil(quantile * len(ordered))) - 1)]


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
    parser.add_argument("--cycles", type=int, default=100)
    parser.add_argument("--threads", type=int, default=2)
    parser.add_argument("--stop-file", type=Path)
    args = parser.parse_args()

    if args.cycles < 2 or args.cycles > 1000 or args.threads < 1 or args.threads > 4:
        raise SystemExit("invalid bounded resident-session settings")
    if args.stop_file is not None and args.stop_file.exists():
        raise SystemExit("stop file already exists")
    verify(args.image_model, args.image_model_bytes, args.image_model_sha256)
    verify(args.text_model, args.text_model_bytes, args.text_model_sha256)
    if sha256(args.tensors) != args.expected_tensors_sha256:
        raise SystemExit("prepared tensor SHA-256 mismatch")

    prepared = np.load(args.tensors, allow_pickle=False)
    pixel_values = prepared["pixel_values"][0:1].astype(np.float32, copy=False)
    input_ids = prepared["input_ids"][0:1].astype(np.int64, copy=False)
    snapshots = [snapshot("before_load")]

    started = time.perf_counter()
    image_session = new_session(args.image_model, args.threads)
    image_load_seconds = time.perf_counter() - started
    snapshots.append(snapshot("after_image_load"))

    started = time.perf_counter()
    text_session = new_session(args.text_model, args.threads)
    text_load_seconds = time.perf_counter() - started
    snapshots.append(snapshot("after_text_load"))

    image_latencies: list[float] = []
    text_latencies: list[float] = []
    finite = True
    for cycle in range(args.cycles):
        if cycle >= 2 and args.stop_file is not None and args.stop_file.exists():
            break
        started = time.perf_counter()
        image_output = image_session.run(None, {"pixel_values": pixel_values})[0]
        image_latencies.append(time.perf_counter() - started)
        started = time.perf_counter()
        text_output = text_session.run(None, {"input_ids": input_ids})[0]
        text_latencies.append(time.perf_counter() - started)
        finite = finite and bool(np.isfinite(image_output).all() and np.isfinite(text_output).all())
        if cycle in {0, 9, 29, args.cycles - 1}:
            snapshots.append(snapshot(f"after_cycle_{cycle + 1}"))

    completed_cycles = len(image_latencies)
    final_stage = f"after_cycle_{completed_cycles}"
    if snapshots[-1]["stage"] != final_stage:
        snapshots.append(snapshot(final_stage))

    print(
        json.dumps(
            {
                "architecture": platform.machine(),
                "onnxruntime": ort.__version__,
                "cpu_mem_arena": False,
                "memory_pattern": True,
                "threads": args.threads,
                "maximum_cycles": args.cycles,
                "completed_cycles": completed_cycles,
                "image_load_seconds": image_load_seconds,
                "text_load_seconds": text_load_seconds,
                "image_inference_p50_seconds": statistics.median(image_latencies),
                "image_inference_p95_seconds": percentile(image_latencies, 0.95),
                "text_inference_p50_seconds": statistics.median(text_latencies),
                "text_inference_p95_seconds": percentile(text_latencies, 0.95),
                "all_outputs_finite": finite,
                "snapshots": snapshots,
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0 if finite else 1


if __name__ == "__main__":
    raise SystemExit(main())
