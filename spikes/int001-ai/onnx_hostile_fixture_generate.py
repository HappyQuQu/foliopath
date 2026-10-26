#!/usr/bin/env python3
"""Generate tiny deterministic ONNX graphs for bounded hostile-load tests."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

import numpy as np
import onnx
from onnx import TensorProto, helper


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_external_model(path: Path, location: str) -> None:
    weight = TensorProto()
    weight.name = "weight"
    weight.data_type = TensorProto.FLOAT
    weight.dims.extend([4, 4])
    weight.data_location = TensorProto.EXTERNAL
    for key, value in (("location", location), ("offset", "0"), ("length", "64")):
        entry = weight.external_data.add()
        entry.key = key
        entry.value = value
    graph = helper.make_graph(
        [helper.make_node("MatMul", ["input", "weight"], ["output"])],
        "external-data",
        [helper.make_tensor_value_info("input", TensorProto.FLOAT, [1, 4])],
        [helper.make_tensor_value_info("output", TensorProto.FLOAT, [1, 4])],
        [weight],
    )
    model = helper.make_model(graph, opset_imports=[helper.make_opsetid("", 18)])
    path.write_bytes(model.SerializeToString(deterministic=True))


def write_embedded_model(path: Path) -> None:
    weight = helper.make_tensor(
        "weight",
        TensorProto.FLOAT,
        [4, 4],
        np.arange(16, dtype=np.float32).tolist(),
    )
    graph = helper.make_graph(
        [helper.make_node("MatMul", ["input", "weight"], ["output"])],
        "embedded-data",
        [helper.make_tensor_value_info("input", TensorProto.FLOAT, [1, 4])],
        [helper.make_tensor_value_info("output", TensorProto.FLOAT, [1, 4])],
        [weight],
    )
    model = helper.make_model(graph, opset_imports=[helper.make_opsetid("", 18)])
    path.write_bytes(model.SerializeToString(deterministic=True))


def write_unknown_operator(path: Path) -> None:
    graph = helper.make_graph(
        [helper.make_node("Unapproved", ["input"], ["output"], domain="foliopath.hostile")],
        "unknown-operator",
        [helper.make_tensor_value_info("input", TensorProto.FLOAT, [1, 4])],
        [helper.make_tensor_value_info("output", TensorProto.FLOAT, [1, 4])],
    )
    model = helper.make_model(
        graph,
        opset_imports=[
            helper.make_opsetid("", 18),
            helper.make_opsetid("foliopath.hostile", 1),
        ],
    )
    path.write_bytes(model.SerializeToString(deterministic=True))


def write_cycle(path: Path) -> None:
    graph = helper.make_graph(
        [
            helper.make_node("Add", ["input", "second"], ["first"]),
            helper.make_node("Add", ["input", "first"], ["second"]),
        ],
        "cycle",
        [helper.make_tensor_value_info("input", TensorProto.FLOAT, [1, 4])],
        [helper.make_tensor_value_info("first", TensorProto.FLOAT, [1, 4])],
    )
    model = helper.make_model(graph, opset_imports=[helper.make_opsetid("", 18)])
    path.write_bytes(model.SerializeToString(deterministic=True))


def write_oversized_allocation(path: Path) -> None:
    shape = helper.make_tensor("shape", TensorProto.INT64, [1], [1_500_000_000])
    graph = helper.make_graph(
        [helper.make_node("ConstantOfShape", ["shape"], ["output"])],
        "oversized-allocation",
        [],
        [helper.make_tensor_value_info("output", TensorProto.FLOAT, [1_500_000_000])],
        [shape],
    )
    model = helper.make_model(graph, opset_imports=[helper.make_opsetid("", 18)])
    path.write_bytes(model.SerializeToString(deterministic=True))


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=False)

    weights = np.arange(16, dtype=np.float32).reshape(4, 4)
    safe = args.output / "safe"
    escape = args.output / "escape"
    safe.mkdir()
    escape.mkdir()
    (safe / "weights.bin").write_bytes(weights.tobytes(order="C"))
    (args.output / "sentinel.bin").write_bytes(weights.tobytes(order="C"))
    write_external_model(safe / "model.onnx", "weights.bin")
    write_embedded_model(args.output / "safe-embedded.onnx")
    write_external_model(escape / "model.onnx", "../sentinel.bin")
    write_unknown_operator(args.output / "unknown-operator.onnx")
    write_cycle(args.output / "cycle.onnx")
    write_oversized_allocation(args.output / "oversized-allocation.onnx")

    files = {}
    for path in sorted(item for item in args.output.rglob("*") if item.is_file()):
        files[path.relative_to(args.output).as_posix()] = {
            "bytes": path.stat().st_size,
            "sha256": sha256(path),
        }
    manifest = {
        "generator": "onnx_hostile_fixture_generate.py",
        "onnx": onnx.__version__,
        "files": files,
    }
    print(json.dumps(manifest, indent=2, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
