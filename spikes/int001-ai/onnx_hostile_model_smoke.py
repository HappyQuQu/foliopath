#!/usr/bin/env python3
"""Run hostile ONNX fixtures in bounded child processes."""

from __future__ import annotations

import argparse
import json
import platform
import subprocess
import sys
from pathlib import Path

import numpy as np
import onnxruntime as ort


def session(path: Path) -> ort.InferenceSession:
    options = ort.SessionOptions()
    options.intra_op_num_threads = 1
    options.inter_op_num_threads = 1
    options.enable_cpu_mem_arena = False
    return ort.InferenceSession(path, sess_options=options, providers=["CPUExecutionProvider"])


def child(path: Path, expect_inference: bool) -> int:
    try:
        loaded = session(path)
        if expect_inference:
            result = loaded.run(
                None,
                {"input": np.ones((1, 4), dtype=np.float32)},
            )[0]
            return 0 if result.shape == (1, 4) and np.isfinite(result).all() else 2
    except Exception:
        return 1 if expect_inference else 0
    return 3


def bounded(path: Path, expect_inference: bool, timeout: float) -> dict[str, object]:
    command = [sys.executable, str(Path(__file__).resolve()), "--child", str(path)]
    if expect_inference:
        command.append("--expect-inference")
    try:
        completed = subprocess.run(
            command,
            check=False,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        return {
            "passed": completed.returncode == 0,
            "exit_code": completed.returncode,
            "timed_out": False,
            "terminated_by_signal": completed.returncode < 0,
        }
    except subprocess.TimeoutExpired:
        return {
            "passed": False,
            "exit_code": None,
            "timed_out": True,
            "terminated_by_signal": False,
        }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--fixtures", type=Path)
    parser.add_argument("--timeout-seconds", type=float, default=15.0)
    parser.add_argument("--child", type=Path)
    parser.add_argument("--expect-inference", action="store_true")
    args = parser.parse_args()
    if args.child is not None:
        return child(args.child, args.expect_inference)
    if args.fixtures is None:
        raise SystemExit("--fixtures is required")
    if args.timeout_seconds <= 0 or args.timeout_seconds > 60:
        raise SystemExit("timeout must be in (0, 60]")

    cases = {
        "safe_external_data_control": bounded(
            args.fixtures / "safe" / "model.onnx", True, args.timeout_seconds
        ),
        "parent_traversal_external_data_rejected": bounded(
            args.fixtures / "escape" / "model.onnx", False, args.timeout_seconds
        ),
        "unknown_operator_rejected": bounded(
            args.fixtures / "unknown-operator.onnx", False, args.timeout_seconds
        ),
        "cyclic_graph_rejected": bounded(
            args.fixtures / "cycle.onnx", False, args.timeout_seconds
        ),
    }
    passed = all(
        result["passed"] and not result["timed_out"] and not result["terminated_by_signal"]
        for result in cases.values()
    )
    print(
        json.dumps(
            {
                "architecture": platform.machine(),
                "onnxruntime": ort.__version__,
                "timeout_seconds": args.timeout_seconds,
                "cases": cases,
                "passed": passed,
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0 if passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
