#!/usr/bin/env python3
"""Create a hash-bound float16-internal ONNX candidate for isolated evaluation."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

import onnx
from onnxruntime.transformers.float16 import convert_float_to_float16
from onnxruntime.transformers.onnx_model import OnnxModel

from semantic_onnx_tensor_smoke import sha256


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True, type=Path)
    parser.add_argument("--expected-input-bytes", required=True, type=int)
    parser.add_argument("--expected-input-sha256", required=True)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    if args.output.exists():
        raise SystemExit("float16 output already exists")
    if (
        args.input.stat().st_size != args.expected_input_bytes
        or sha256(args.input) != args.expected_input_sha256
    ):
        raise SystemExit("input size or SHA-256 mismatch")

    source = onnx.load(args.input, load_external_data=False)
    converted = convert_float_to_float16(
        source,
        keep_io_types=True,
        disable_shape_infer=False,
    )
    wrapper = OnnxModel(converted)
    wrapper.topological_sort(is_deterministic=True)
    converted = wrapper.model
    onnx.checker.check_model(converted, full_check=False)
    onnx.save(converted, args.output)
    print(
        json.dumps(
            {
                "source_bytes": args.input.stat().st_size,
                "source_sha256": sha256(args.input),
                "output_bytes": args.output.stat().st_size,
                "output_sha256": sha256(args.output),
                "keep_io_types": True,
                "nodes": len(converted.graph.node),
                "initializers": len(converted.graph.initializer),
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
