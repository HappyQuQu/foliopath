#!/usr/bin/env python3
"""Exercise bounded semantic ONNX runtime failure and recovery behavior."""

from __future__ import annotations

import argparse
import json
import platform
import threading
import time

import numpy as np
import onnxruntime as ort


def memory_kib(field: str) -> int | None:
    try:
        with open("/proc/self/status", encoding="utf-8") as status:
            for line in status:
                if line.startswith(field + ":"):
                    return int(line.split()[1])
    except OSError:
        return None
    return None


def rejected(session: ort.InferenceSession, inputs: dict[str, np.ndarray]) -> bool:
    try:
        session.run(None, inputs)
    except Exception:  # Runtime-specific invalid-input subclasses vary by wheel.
        return True
    return False


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True)
    parser.add_argument("--reference", required=True)
    parser.add_argument("--threads", type=int, default=4)
    parser.add_argument("--iterations", type=int, default=100)
    parser.add_argument("--max-rss-growth-kib", type=int, default=65536)
    args = parser.parse_args()

    options = ort.SessionOptions()
    options.intra_op_num_threads = args.threads
    options.inter_op_num_threads = 1
    session = ort.InferenceSession(
        args.model,
        sess_options=options,
        providers=["CPUExecutionProvider"],
    )
    fixture = np.load(args.reference, allow_pickle=False)
    normal = {
        "input_ids": fixture["input_ids"],
        "pixel_values": fixture["pixel_values"],
    }

    warm = session.run(None, normal)
    finite = all(np.isfinite(value).all() for value in warm)
    rss_after_warmup = memory_kib("VmRSS")
    started = time.perf_counter()
    for _ in range(args.iterations):
        output = session.run(None, normal)
        finite = finite and all(np.isfinite(value).all() for value in output)
    loop_seconds = time.perf_counter() - started
    rss_after_loop = memory_kib("VmRSS")
    rss_growth = (
        None
        if rss_after_warmup is None or rss_after_loop is None
        else rss_after_loop - rss_after_warmup
    )

    malformed = {
        "missing_pixel_values": rejected(session, {"input_ids": normal["input_ids"]}),
        "input_ids_float32": rejected(
            session,
            {
                "input_ids": normal["input_ids"].astype(np.float32),
                "pixel_values": normal["pixel_values"],
            },
        ),
        "pixel_channels_four": rejected(
            session,
            {
                "input_ids": normal["input_ids"],
                "pixel_values": np.zeros((1, 4, 224, 224), dtype=np.float32),
            },
        ),
        "sequence_length_65": rejected(
            session,
            {
                "input_ids": np.zeros((1, 65), dtype=np.int64),
                "pixel_values": normal["pixel_values"],
            },
        ),
    }

    cancel_inputs = {
        "input_ids": np.tile(normal["input_ids"], (8, 1)),
        "pixel_values": np.tile(normal["pixel_values"], (4, 1, 1, 1)),
    }
    run_options = ort.RunOptions()
    cancellation: dict[str, object] = {"thread_stopped": False, "reported_error": False}

    def cancellable_run() -> None:
        try:
            session.run(None, cancel_inputs, run_options)
        except Exception as error:  # The runtime has version-specific exception classes.
            cancellation["reported_error"] = True
            cancellation["error_type"] = type(error).__name__

    worker = threading.Thread(target=cancellable_run)
    worker.start()
    time.sleep(0.01)
    run_options.terminate = True
    worker.join(timeout=10)
    cancellation["thread_stopped"] = not worker.is_alive()
    cancellation["subsequent_inference_finite"] = all(
        np.isfinite(value).all() for value in session.run(None, normal)
    )

    memory_passed = rss_growth is None or rss_growth <= args.max_rss_growth_kib
    passed = (
        finite
        and all(malformed.values())
        and bool(cancellation["thread_stopped"])
        and bool(cancellation["reported_error"])
        and bool(cancellation["subsequent_inference_finite"])
        and memory_passed
    )
    print(
        json.dumps(
            {
                "architecture": platform.machine(),
                "onnxruntime": ort.__version__,
                "iterations": args.iterations,
                "loop_seconds": loop_seconds,
                "all_outputs_finite": finite,
                "rss_after_warmup_kib": rss_after_warmup,
                "rss_after_loop_kib": rss_after_loop,
                "rss_growth_kib": rss_growth,
                "max_rss_growth_kib": args.max_rss_growth_kib,
                "peak_rss_kib": memory_kib("VmHWM"),
                "malformed_inputs_rejected": malformed,
                "cancellation": cancellation,
                "passed": passed,
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0 if passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
