#!/usr/bin/env python3
"""Prepare deterministic processor tensors without loading an inference graph."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

import numpy as np
from PIL import Image
from transformers import AutoProcessor

from semantic_onnx_tensor_smoke import sha256


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--processor", required=True, type=Path)
    parser.add_argument("--fixture", required=True, type=Path)
    parser.add_argument("--images", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    if args.output.exists():
        raise SystemExit("prepared tensor output already exists")
    fixture = json.loads(args.fixture.read_text(encoding="utf-8"))
    processor = AutoProcessor.from_pretrained(args.processor, local_files_only=True, use_fast=False)
    item_ids: list[str] = []
    pixels: list[np.ndarray] = []
    for item in fixture["items"]:
        image_path = args.images / item["filename"]
        if sha256(image_path) != item["sha256"]:
            raise SystemExit(f"fixture image hash mismatch: {item['id']}")
        with Image.open(image_path) as source:
            prepared = processor(images=[source.convert("RGB")], return_tensors="np")
        pixels.append(prepared["pixel_values"][0].astype(np.float32, copy=False))
        item_ids.append(item["id"])

    text = processor(
        text=[query["text"] for query in fixture["queries"]],
        padding="max_length",
        truncation=True,
        max_length=64,
        return_tensors="np",
    )
    np.savez(
        args.output,
        pixel_values=np.stack(pixels),
        input_ids=text["input_ids"].astype(np.int64, copy=False),
        item_ids=np.asarray(item_ids),
        fixture_sha256=np.asarray(sha256(args.fixture)),
    )
    print(
        json.dumps(
            {
                "output_sha256": sha256(args.output),
                "items": len(item_ids),
                "queries": len(fixture["queries"]),
                "pixel_shape": list(np.stack(pixels).shape),
                "input_ids_shape": list(text["input_ids"].shape),
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
