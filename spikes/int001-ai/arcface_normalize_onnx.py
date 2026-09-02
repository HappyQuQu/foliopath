#!/usr/bin/env python3
"""Deterministically normalize one pinned legacy ArcFace ONNX graph."""

from __future__ import annotations

import argparse
import hashlib
import os
import stat
import tempfile
from pathlib import Path


SOURCE_BYTES = 261_036_388
SOURCE_SHA256 = "f0a2e278b430372d308fef67c1aea308c2baf37f32e8908d9bfce035c26a3fb4"
OUTPUT_BYTES = 261_033_924
OUTPUT_SHA256 = "345e28fd93dc48fd7bfb3552c58434ca7e279f85ee2132c810b26945d4550844"
EXPECTED_NODE_COUNT = 412
EXPECTED_INITIALIZER_COUNT = 823
EXPECTED_BATCH_NORMALIZATION_COUNT = 154


def digest(path: Path) -> str:
    result = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            result.update(chunk)
    return result.hexdigest()


def require_regular(path: Path, expected_bytes: int, expected_sha256: str) -> None:
    metadata = path.lstat()
    if not stat.S_ISREG(metadata.st_mode) or metadata.st_size != expected_bytes:
        raise SystemExit("model artifact does not match the frozen regular-file contract")
    with path.open("rb") as source:
        opened = os.fstat(source.fileno())
        if not stat.S_ISREG(opened.st_mode) or (metadata.st_dev, metadata.st_ino) != (
            opened.st_dev,
            opened.st_ino,
        ):
            raise SystemExit("model artifact identity changed while opening")
    if digest(path) != expected_sha256:
        raise SystemExit("model artifact does not match the frozen SHA-256")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    if args.source == args.output or args.output.exists() or args.output.is_symlink():
        raise SystemExit("output must be a new path distinct from the source")
    require_regular(args.source, SOURCE_BYTES, SOURCE_SHA256)

    import onnx

    if onnx.__version__ != "1.22.0":
        raise SystemExit("the deterministic normalizer requires onnx 1.22.0")
    model = onnx.load(args.source, load_external_data=False)
    if [(item.domain, item.version) for item in model.opset_import] != [("", 8)]:
        raise SystemExit("source graph has an unexpected opset contract")
    if len(model.graph.node) != EXPECTED_NODE_COUNT or len(model.graph.initializer) != EXPECTED_INITIALIZER_COUNT:
        raise SystemExit("source graph structure does not match the frozen contract")
    changed = 0
    for node in model.graph.node:
        if node.op_type != "BatchNormalization":
            continue
        spatial = [attribute for attribute in node.attribute if attribute.name == "spatial"]
        if len(spatial) != 1 or spatial[0].i != 0:
            raise SystemExit("BatchNormalization does not match the legacy spatial contract")
        retained = [attribute for attribute in node.attribute if attribute.name != "spatial"]
        del node.attribute[:]
        node.attribute.extend(retained)
        changed += 1
    if changed != EXPECTED_BATCH_NORMALIZATION_COUNT:
        raise SystemExit("source graph has an unexpected BatchNormalization count")
    onnx.checker.check_model(model)
    encoded = model.SerializeToString(deterministic=True)
    if len(encoded) != OUTPUT_BYTES or hashlib.sha256(encoded).hexdigest() != OUTPUT_SHA256:
        raise SystemExit("normalized graph is not byte-identical to the frozen transform")

    args.output.parent.mkdir(parents=True, exist_ok=True)
    descriptor, temporary_name = tempfile.mkstemp(prefix=".arcface-normalized-", dir=args.output.parent)
    try:
        with os.fdopen(descriptor, "wb") as target:
            target.write(encoded)
            target.flush()
            os.fsync(target.fileno())
        os.link(temporary_name, args.output)
    finally:
        try:
            os.unlink(temporary_name)
        except FileNotFoundError:
            pass
    require_regular(args.output, OUTPUT_BYTES, OUTPUT_SHA256)
    print(f"normalized_sha256={OUTPUT_SHA256}")


if __name__ == "__main__":
    main()
