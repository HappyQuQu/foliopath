#!/usr/bin/env python3
"""Bounded failure-closed smoke for fixed-shape split ONNX encoders."""

from __future__ import annotations

import argparse
import gc
import hashlib
import json
import platform
import subprocess
import sys
import tempfile
from pathlib import Path

import numpy as np
import onnxruntime as ort


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def verify(path: Path, expected_bytes: int, expected_sha256: str) -> None:
    if path.stat().st_size != expected_bytes or sha256(path) != expected_sha256:
        raise SystemExit(f"model size or SHA-256 mismatch: {path.name}")


def options(threads: int) -> ort.SessionOptions:
    result = ort.SessionOptions()
    result.intra_op_num_threads = threads
    result.inter_op_num_threads = 1
    result.enable_cpu_mem_arena = False
    result.enable_mem_pattern = True
    return result


def rejected(session: ort.InferenceSession, inputs: dict[str, np.ndarray]) -> bool:
    try:
        session.run(None, inputs)
    except Exception:  # Runtime exception subclasses differ between releases.
        return True
    return False


def finite(session: ort.InferenceSession, inputs: dict[str, np.ndarray]) -> bool:
    return all(np.isfinite(value).all() for value in session.run(None, inputs))


def load_only(path: Path, threads: int) -> int:
    try:
        ort.InferenceSession(
            path,
            sess_options=options(threads),
            providers=["CPUExecutionProvider"],
        )
    except Exception:
        return 0
    return 1


def rejected_in_subprocess(path: Path, threads: int, timeout: float) -> dict[str, object]:
    command = [
        sys.executable,
        str(Path(__file__).resolve()),
        "--load-only",
        str(path),
        "--threads",
        str(threads),
    ]
    try:
        completed = subprocess.run(
            command,
            check=False,
            capture_output=True,
            text=True,
            timeout=timeout,
        )
        return {
            "rejected": completed.returncode == 0,
            "exit_code": completed.returncode,
            "timed_out": False,
            "terminated_by_signal": completed.returncode < 0,
        }
    except subprocess.TimeoutExpired:
        return {
            "rejected": False,
            "exit_code": None,
            "timed_out": True,
            "terminated_by_signal": False,
        }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--load-only", type=Path)
    parser.add_argument("--image-model", type=Path)
    parser.add_argument("--image-model-bytes", type=int)
    parser.add_argument("--image-model-sha256")
    parser.add_argument("--text-model", type=Path)
    parser.add_argument("--text-model-bytes", type=int)
    parser.add_argument("--text-model-sha256")
    parser.add_argument("--tensors", type=Path)
    parser.add_argument("--expected-tensors-sha256")
    parser.add_argument("--threads", type=int, default=2)
    parser.add_argument("--load-timeout-seconds", type=float, default=15.0)
    args = parser.parse_args()

    if args.threads < 1 or args.threads > 4:
        raise SystemExit("threads must be between 1 and 4")
    if args.load_only is not None:
        return load_only(args.load_only, args.threads)

    required = (
        args.image_model,
        args.image_model_bytes,
        args.image_model_sha256,
        args.text_model,
        args.text_model_bytes,
        args.text_model_sha256,
        args.tensors,
        args.expected_tensors_sha256,
    )
    if any(value is None for value in required):
        raise SystemExit("all model, tensor, size and SHA-256 arguments are required")
    if args.load_timeout_seconds <= 0 or args.load_timeout_seconds > 60:
        raise SystemExit("load timeout must be in (0, 60]")

    verify(args.image_model, args.image_model_bytes, args.image_model_sha256)
    verify(args.text_model, args.text_model_bytes, args.text_model_sha256)
    if sha256(args.tensors) != args.expected_tensors_sha256:
        raise SystemExit("prepared tensor SHA-256 mismatch")

    with tempfile.TemporaryDirectory(prefix="foliopath-onnx-failure-") as directory:
        root = Path(directory)
        empty = root / "empty.onnx"
        empty.write_bytes(b"")
        random_payload = root / "random.onnx"
        random_payload.write_bytes(bytes((index * 131 + 17) % 256 for index in range(65536)))
        truncated = root / "truncated.onnx"
        with args.image_model.open("rb") as source:
            truncated.write_bytes(source.read(65536))
        corruptions = {
            "empty": rejected_in_subprocess(empty, args.threads, args.load_timeout_seconds),
            "deterministic_random_64k": rejected_in_subprocess(
                random_payload, args.threads, args.load_timeout_seconds
            ),
            "truncated_real_graph_64k": rejected_in_subprocess(
                truncated, args.threads, args.load_timeout_seconds
            ),
        }

    fixture = np.load(args.tensors, allow_pickle=False)
    pixel_values = fixture["pixel_values"][0:1].astype(np.float32, copy=False)
    input_ids = fixture["input_ids"][0:1].astype(np.int64, copy=False)

    image = ort.InferenceSession(
        args.image_model,
        sess_options=options(args.threads),
        providers=["CPUExecutionProvider"],
    )
    image_normal = {"pixel_values": pixel_values}
    image_cases = {
        "missing_input": rejected(image, {}),
        "wrong_dtype": rejected(image, {"pixel_values": pixel_values.astype(np.float64)}),
        "wrong_channels": rejected(
            image, {"pixel_values": np.zeros((1, 4, 224, 224), dtype=np.float32)}
        ),
        "wrong_height": rejected(
            image, {"pixel_values": np.zeros((1, 3, 225, 224), dtype=np.float32)}
        ),
    }
    image_recovered = finite(image, image_normal)
    del image
    gc.collect()

    text_session = ort.InferenceSession(
        args.text_model,
        sess_options=options(args.threads),
        providers=["CPUExecutionProvider"],
    )
    text_normal = {"input_ids": input_ids}
    text_cases = {
        "missing_input": rejected(text_session, {}),
        "wrong_dtype": rejected(text_session, {"input_ids": input_ids.astype(np.float32)}),
        "wrong_sequence_length": rejected(
            text_session, {"input_ids": np.zeros((1, 65), dtype=np.int64)}
        ),
        "wrong_batch": rejected(
            text_session, {"input_ids": np.zeros((2, 64), dtype=np.int64)}
        ),
    }
    text_recovered = finite(text_session, text_normal)

    corruptions_passed = all(
        item["rejected"] and not item["timed_out"] and not item["terminated_by_signal"]
        for item in corruptions.values()
    )
    passed = (
        corruptions_passed
        and all(image_cases.values())
        and all(text_cases.values())
        and image_recovered
        and text_recovered
    )
    print(
        json.dumps(
            {
                "architecture": platform.machine(),
                "onnxruntime": ort.__version__,
                "cpu_mem_arena": False,
                "threads": args.threads,
                "corrupt_model_loads": corruptions,
                "image_malformed_inputs_rejected": image_cases,
                "text_malformed_inputs_rejected": text_cases,
                "image_recovered_after_errors": image_recovered,
                "text_recovered_after_errors": text_recovered,
                "passed": passed,
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0 if passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
