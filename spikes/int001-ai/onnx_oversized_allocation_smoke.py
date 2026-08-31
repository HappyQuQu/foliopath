#!/usr/bin/env python3
"""Bound an oversized ONNX allocation in a child process on Unix."""

from __future__ import annotations

import argparse
import json
import platform
import resource
import subprocess
import sys
from pathlib import Path

import numpy as np
import onnxruntime as ort


def limit_address_space(limit_bytes: int) -> None:
    resource.setrlimit(resource.RLIMIT_AS, (limit_bytes, limit_bytes))


def session(path: Path) -> ort.InferenceSession:
    options = ort.SessionOptions()
    options.intra_op_num_threads = 1
    options.inter_op_num_threads = 1
    options.enable_cpu_mem_arena = False
    return ort.InferenceSession(path, sess_options=options, providers=["CPUExecutionProvider"])


def child(path: Path, control: bool) -> int:
    try:
        loaded = session(path)
        if control:
            output = loaded.run(None, {"input": np.ones((1, 4), dtype=np.float32)})[0]
            print(json.dumps({"outcome": "finite", "error_type": None}))
            return 0 if output.shape == (1, 4) and np.isfinite(output).all() else 3
        loaded.run(None, {})
    except Exception as error:
        print(json.dumps({"outcome": "rejected", "error_type": type(error).__name__}))
        return 0 if not control else 2
    print(json.dumps({"outcome": "allocated", "error_type": None}))
    return 4


def bounded(
    path: Path, control: bool, timeout: float, address_space_bytes: int
) -> dict[str, object]:
    command = [sys.executable, str(Path(__file__).resolve()), "--child", str(path)]
    command.extend(["--child-address-space-bytes", str(address_space_bytes)])
    if control:
        command.append("--control")
    try:
        completed = subprocess.run(
            command,
            check=False,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        detail: dict[str, object] = {}
        for line in reversed(completed.stdout.splitlines()):
            try:
                candidate = json.loads(line)
            except json.JSONDecodeError:
                continue
            if isinstance(candidate, dict):
                detail = candidate
                break
        return {
            "passed": completed.returncode == 0,
            "exit_code": completed.returncode,
            "timed_out": False,
            "terminated_by_signal": completed.returncode < 0,
            "outcome": detail.get("outcome"),
            "error_type": detail.get("error_type"),
        }
    except subprocess.TimeoutExpired:
        return {
            "passed": False,
            "exit_code": None,
            "timed_out": True,
            "terminated_by_signal": False,
            "outcome": None,
            "error_type": None,
        }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--fixtures", type=Path)
    parser.add_argument("--timeout-seconds", type=float, default=15.0)
    parser.add_argument("--address-space-bytes", type=int, default=1073741824)
    parser.add_argument("--child", type=Path)
    parser.add_argument("--child-address-space-bytes", type=int)
    parser.add_argument("--control", action="store_true")
    args = parser.parse_args()
    if args.child is not None:
        if args.child_address_space_bytes is None:
            raise SystemExit("child address-space bound is required")
        limit_address_space(args.child_address_space_bytes)
        return child(args.child, args.control)
    if args.fixtures is None:
        raise SystemExit("--fixtures is required")
    if args.timeout_seconds <= 0 or args.timeout_seconds > 60:
        raise SystemExit("timeout must be in (0, 60]")
    if args.address_space_bytes < 268435456 or args.address_space_bytes > 2147483648:
        raise SystemExit("address-space bound must be between 256 MiB and 2 GiB")

    control = bounded(
        args.fixtures / "safe" / "model.onnx",
        True,
        args.timeout_seconds,
        args.address_space_bytes,
    )
    oversized = bounded(
        args.fixtures / "oversized-allocation.onnx",
        False,
        args.timeout_seconds,
        args.address_space_bytes,
    )
    passed = all(
        item["passed"] and not item["timed_out"] and not item["terminated_by_signal"]
        for item in (control, oversized)
    )
    print(
        json.dumps(
            {
                "architecture": platform.machine(),
                "onnxruntime": ort.__version__,
                "timeout_seconds": args.timeout_seconds,
                "address_space_bytes": args.address_space_bytes,
                "control": control,
                "oversized_allocation_rejected": oversized,
                "passed": passed,
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0 if passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
