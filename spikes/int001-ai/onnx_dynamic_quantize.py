#!/usr/bin/env python3
"""Create a hash-bound dynamic-int8 ONNX candidate for isolated evaluation."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

import onnx
from onnxruntime.quantization import QuantType, quantize_dynamic

from semantic_onnx_tensor_smoke import sha256


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--input", required=True, type=Path)
    parser.add_argument("--expected-input-bytes", required=True, type=int)
    parser.add_argument("--expected-input-sha256", required=True)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    if args.output.exists():
        raise SystemExit("quantized output already exists")
    if (
        args.input.stat().st_size != args.expected_input_bytes
        or sha256(args.input) != args.expected_input_sha256
    ):
        raise SystemExit("input size or SHA-256 mismatch")

    quantize_dynamic(
        model_input=str(args.input),
        model_output=str(args.output),
        op_types_to_quantize=["MatMul", "Gemm"],
        per_channel=False,
        reduce_range=False,
        weight_type=QuantType.QInt8,
    )
    graph = onnx.load(args.output, load_external_data=False)
    onnx.checker.check_model(graph, full_check=False)
    print(
        json.dumps(
            {
                "source_bytes": args.input.stat().st_size,
                "source_sha256": sha256(args.input),
                "output_bytes": args.output.stat().st_size,
                "output_sha256": sha256(args.output),
                "weight_type": "QInt8",
                "op_types": ["MatMul", "Gemm"],
                "per_channel": False,
                "reduce_range": False,
                "nodes": len(graph.graph.node),
                "initializers": len(graph.graph.initializer),
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
