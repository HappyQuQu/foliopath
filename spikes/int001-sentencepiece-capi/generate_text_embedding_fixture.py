#!/usr/bin/env python3
"""Generate fixed SigLIP 1 text-encoder outputs from tokenizer reference IDs."""

from __future__ import annotations

import argparse
import base64
import hashlib
import json
import platform
from pathlib import Path

import numpy as np
import onnxruntime as ort


GRAPH_SHA256 = "16eef12730b862a0c4f75926213d86749d9c6a5ec79b37b6feebc20f826fd664"
GRAPH_SIZE = 441_217_411
TOKEN_FIXTURE_SHA256 = "fa12da1f146659256d0607b548b7375cb49af7fc933b0395ad9a32344fb85d0b"
ORT_VERSION = "1.29.0"
NUMPY_VERSION = "2.5.2"


def digest(value: bytes) -> str:
    return hashlib.sha256(value).hexdigest()


def file_digest(path: Path) -> str:
    value = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(8 * 1024 * 1024), b""):
            value.update(chunk)
    return value.hexdigest()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--graph", required=True, type=Path)
    parser.add_argument("--token-fixture", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    token_bytes = args.token_fixture.read_bytes()
    if args.graph.stat().st_size != GRAPH_SIZE or file_digest(args.graph) != GRAPH_SHA256:
        raise SystemExit("text graph differs from pinned contract")
    if digest(token_bytes) != TOKEN_FIXTURE_SHA256:
        raise SystemExit("token fixture differs from pinned contract")
    if ort.__version__ != ORT_VERSION or np.__version__ != NUMPY_VERSION:
        raise SystemExit("reference runtime version differs from pinned contract")

    tokens = json.loads(token_bytes)
    options = ort.SessionOptions()
    options.intra_op_num_threads = 2
    options.inter_op_num_threads = 1
    options.enable_cpu_mem_arena = False
    session = ort.InferenceSession(str(args.graph), sess_options=options, providers=["CPUExecutionProvider"])
    if [(item.name, item.type, item.shape) for item in session.get_inputs()] != [("input_ids", "tensor(int64)", [1, 64])]:
        raise SystemExit("unexpected text graph input ABI")
    if [(item.name, item.type, item.shape) for item in session.get_outputs()] != [("text_embeds", "tensor(float)", [1, 768])]:
        raise SystemExit("unexpected text graph output ABI")

    cases = []
    for case in tokens["cases"]:
        ids = np.asarray([case["input_ids"]], dtype=np.int64)
        output = np.asarray(session.run(["text_embeds"], {"input_ids": ids})[0][0], dtype="<f4")
        if output.shape != (768,) or not np.isfinite(output).all():
            raise SystemExit(f"{case['name']}: invalid output")
        raw = output.tobytes(order="C")
        cases.append(
            {
                "name": case["name"],
                "float32_le_base64": base64.b64encode(raw).decode("ascii"),
                "sha256": digest(raw),
                "l2_norm": float(np.linalg.norm(output.astype(np.float64))),
            }
        )

    result = {
        "schema_version": 1,
        "generator": {"python": platform.python_version(), "onnxruntime": ort.__version__, "numpy": np.__version__},
        "graph": {"filename": "text_encoder.onnx", "size_bytes": GRAPH_SIZE, "sha256": GRAPH_SHA256},
        "token_fixture": {"filename": args.token_fixture.name, "sha256": digest(token_bytes)},
        "contract": {"input_name": "input_ids", "input_shape": [1, 64], "output_name": "text_embeds", "output_shape": [1, 768]},
        "cases": cases,
    }
    args.output.write_text(json.dumps(result, indent=2, sort_keys=True) + "\n")


if __name__ == "__main__":
    main()
