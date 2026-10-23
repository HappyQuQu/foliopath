#!/usr/bin/env python3
"""Build bounded WebP semantic inputs from an already verified pilot.

This is a Pillow/JPEG shrink-on-load surrogate for the production libvips
thumbnail path. It does not claim libvips equivalence or release evidence.
"""

from __future__ import annotations

import argparse
import copy
import hashlib
import json
import os
from pathlib import Path

from PIL import Image, ImageOps


MAX_SOURCE_BYTES = 256 << 20
MAX_DECODED_PIXELS = 100_000_000
MAX_DIMENSION = 32_768
OUTPUT_DIMENSION = 512
OUTPUT_QUALITY = 82


def file_sha256(path: Path) -> str:
    result = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(8 * 1024 * 1024), b""):
            result.update(chunk)
    return result.hexdigest()


def prepare_image(source_path: Path, target_path: Path) -> tuple[int, int, int, str]:
    size = source_path.stat().st_size
    if size <= 0 or size > MAX_SOURCE_BYTES:
        raise ValueError(f"source byte limit failed: {source_path.name}")
    with Image.open(source_path) as source:
        if source.format != "JPEG":
            raise ValueError(f"pilot source is not JPEG: {source_path.name}")
        width, height = source.size
        if (
            width <= 0
            or height <= 0
            or width > MAX_DIMENSION
            or height > MAX_DIMENSION
            or width * height > MAX_DECODED_PIXELS
        ):
            raise ValueError(f"source dimension limit failed: {source_path.name}")
        source.draft("RGB", (OUTPUT_DIMENSION, OUTPUT_DIMENSION))
        image = ImageOps.exif_transpose(source)
        image.thumbnail(
            (OUTPUT_DIMENSION, OUTPUT_DIMENSION),
            Image.Resampling.LANCZOS,
            reducing_gap=3.0,
        )
        image = image.convert("RGB")
        temporary = target_path.with_suffix(target_path.suffix + ".part")
        try:
            image.save(
                temporary,
                format="WEBP",
                quality=OUTPUT_QUALITY,
                method=4,
                exif=b"",
                icc_profile=None,
            )
            os.replace(temporary, target_path)
        finally:
            if temporary.exists():
                temporary.unlink()
    with Image.open(target_path) as output:
        output.verify()
    return target_path.stat().st_size, image.width, image.height, file_sha256(target_path)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source-manifest", required=True, type=Path)
    parser.add_argument("--source-images", required=True, type=Path)
    parser.add_argument("--output-manifest", required=True, type=Path)
    parser.add_argument("--output-images", required=True, type=Path)
    args = parser.parse_args()

    manifest = json.loads(args.source_manifest.read_text(encoding="utf-8"))
    items = manifest.get("items")
    if manifest.get("evidence_class") != "public-license-pilot" or not isinstance(items, list):
        raise ValueError("expected a public-license pilot manifest")
    args.output_images.mkdir(parents=True, exist_ok=True)
    expected_outputs: set[str] = set()
    output_bytes = 0
    prepared = copy.deepcopy(manifest)
    prepared["input_pipeline"] = "pillow-jpeg-draft-512-webp-libvips-surrogate"
    prepared["input_transform"] = {
        "maximum_source_bytes": MAX_SOURCE_BYTES,
        "maximum_decoded_pixels": MAX_DECODED_PIXELS,
        "maximum_dimension": MAX_DIMENSION,
        "output_maximum_dimension": OUTPUT_DIMENSION,
        "output_format": "webp",
        "output_quality": OUTPUT_QUALITY,
        "production_equivalent": False,
    }
    for item in prepared["items"]:
        source_path = args.source_images / item["filename"]
        if file_sha256(source_path) != item["sha256"]:
            raise ValueError(f"source SHA-256 mismatch: {item['id']}")
        filename = f"{item['source']['page_id']}.webp"
        expected_outputs.add(filename)
        target_path = args.output_images / filename
        byte_size, width, height, sha256 = prepare_image(source_path, target_path)
        output_bytes += byte_size
        item["filename"] = filename
        item["sha256"] = sha256
        item["prepared"] = {
            "bytes": byte_size,
            "width": width,
            "height": height,
            "format": "image/webp",
        }
    extras = {
        path.name
        for path in args.output_images.iterdir()
        if path.is_file() and path.name not in expected_outputs
    }
    if extras:
        raise ValueError(f"unexpected prepared files: {sorted(extras)!r}")
    temporary_manifest = args.output_manifest.with_suffix(args.output_manifest.suffix + ".part")
    try:
        temporary_manifest.write_text(
            json.dumps(prepared, ensure_ascii=False, indent=2) + "\n",
            encoding="utf-8",
        )
        os.replace(temporary_manifest, args.output_manifest)
    finally:
        if temporary_manifest.exists():
            temporary_manifest.unlink()
    print(
        json.dumps(
            {
                "schema_version": 1,
                "items": len(prepared["items"]),
                "source_bytes": sum(int(item["source"]["bytes"]) for item in prepared["items"]),
                "prepared_bytes": output_bytes,
                "input_pipeline": prepared["input_pipeline"],
                "production_equivalent": False,
            },
            indent=2,
        )
    )


if __name__ == "__main__":
    main()
