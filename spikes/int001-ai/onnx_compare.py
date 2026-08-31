#!/usr/bin/env python3
"""Compare a semantic ONNX export with a precomputed reference fixture."""

from __future__ import annotations

import argparse
import json
import os
import platform
import time

import numpy as np
import onnxruntime as ort


def peak_rss_kib() -> int | None:
    try:
        with open("/proc/self/status", encoding="utf-8") as status:
            for line in status:
                if line.startswith("VmHWM:"):
                    return int(line.split()[1])
    except OSError:
        return None
    return None


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True)
    parser.add_argument("--reference", required=True)
    parser.add_argument("--threads", type=int, default=4)
    parser.add_argument("--atol", type=float, default=1e-4)
    parser.add_argument("--rtol", type=float, default=1e-4)
    args = parser.parse_args()

    reference = np.load(args.reference, allow_pickle=False)
    options = ort.SessionOptions()
    options.intra_op_num_threads = args.threads
    options.inter_op_num_threads = 1

    started = time.perf_counter()
    session = ort.InferenceSession(
        args.model,
        sess_options=options,
        providers=["CPUExecutionProvider"],
    )
    load_seconds = time.perf_counter() - started

    inputs = {
        "input_ids": reference["input_ids"],
        "pixel_values": reference["pixel_values"],
    }
    started = time.perf_counter()
    outputs = session.run(None, inputs)
    inference_seconds = time.perf_counter() - started

    comparisons: dict[str, object] = {}
    passed = True
    for metadata, actual in zip(session.get_outputs(), outputs, strict=True):
        expected = reference[metadata.name]
        difference = np.abs(actual - expected)
        close = bool(np.allclose(actual, expected, atol=args.atol, rtol=args.rtol))
        finite = bool(np.isfinite(actual).all())
        passed = passed and close and finite
        comparisons[metadata.name] = {
            "shape": list(actual.shape),
            "max_abs": float(difference.max()),
            "mean_abs": float(difference.mean()),
            "allclose": close,
            "finite": finite,
        }

    print(
        json.dumps(
            {
                "architecture": platform.machine(),
                "onnxruntime": ort.__version__,
                "providers": session.get_providers(),
                "network_disabled_by_caller": os.environ.get("INT001_NETWORK_DISABLED")
                == "1",
                "threads": args.threads,
                "atol": args.atol,
                "rtol": args.rtol,
                "session_load_seconds": load_seconds,
                "inference_seconds": inference_seconds,
                "peak_rss_kib": peak_rss_kib(),
                "comparisons": comparisons,
                "passed": passed,
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0 if passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
