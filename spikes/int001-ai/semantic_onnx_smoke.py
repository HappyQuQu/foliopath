#!/usr/bin/env python3
"""Run the licensed semantic pilot through an exported SigLIP ONNX graph."""

from __future__ import annotations

import argparse
import hashlib
import json
import time
from pathlib import Path

import numpy as np
import onnxruntime as ort
from PIL import Image
from transformers import AutoProcessor


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(8 * 1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def normalize(values: np.ndarray) -> np.ndarray:
    values = values.astype(np.float32, copy=False)
    norms = np.linalg.norm(values, axis=1, keepdims=True).clip(min=1e-12)
    result = values / norms
    if not np.isfinite(result).all():
        raise RuntimeError("model produced non-finite embeddings")
    return result


def rank(
    queries: list[dict[str, object]],
    item_ids: list[str],
    similarities: np.ndarray,
) -> list[dict[str, object]]:
    results: list[dict[str, object]] = []
    for query_index, query in enumerate(queries):
        order = sorted(
            range(len(item_ids)),
            key=lambda index: (-float(similarities[query_index, index]), item_ids[index]),
        )
        ranked = [item_ids[index] for index in order]
        relevant = set(query["relevant"])
        first = min(index + 1 for index, item_id in enumerate(ranked) if item_id in relevant)
        relevant_top3 = sum(item_id in relevant for item_id in ranked[:3])
        results.append(
            {
                "id": query["id"],
                "language": query["language"],
                "first_relevant_rank": first,
                "relevant_recall_at_3": relevant_top3 / len(relevant),
                "top3": ranked[:3],
            }
        )
    return results


def metrics(results: list[dict[str, object]], language: str) -> dict[str, float]:
    selected = [result for result in results if result["language"] == language]
    return {
        "queries": len(selected),
        "recall_at_1": sum(result["first_relevant_rank"] <= 1 for result in selected) / len(selected),
        "recall_at_3": sum(result["first_relevant_rank"] <= 3 for result in selected) / len(selected),
        "mean_relevant_recall_at_3": sum(result["relevant_recall_at_3"] for result in selected)
        / len(selected),
        "mrr": sum(1.0 / result["first_relevant_rank"] for result in selected) / len(selected),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True, type=Path)
    parser.add_argument("--processor", required=True, type=Path)
    parser.add_argument("--fixture", required=True, type=Path)
    parser.add_argument("--images", required=True, type=Path)
    parser.add_argument("--expected-model-bytes", required=True, type=int)
    parser.add_argument("--expected-model-sha256", required=True)
    parser.add_argument("--prepared-output", type=Path)
    parser.add_argument("--threads", type=int, default=4)
    args = parser.parse_args()

    if args.model.stat().st_size != args.expected_model_bytes or sha256(args.model) != args.expected_model_sha256:
        raise SystemExit("ONNX size or SHA-256 mismatch")

    fixture = json.loads(args.fixture.read_text(encoding="utf-8"))
    processor = AutoProcessor.from_pretrained(args.processor, local_files_only=True, use_fast=False)
    options = ort.SessionOptions()
    options.intra_op_num_threads = args.threads
    options.inter_op_num_threads = 1
    session = ort.InferenceSession(
        args.model,
        sess_options=options,
        providers=["CPUExecutionProvider"],
    )
    output_names = [output.name for output in session.get_outputs()]
    image_output = output_names.index("image_embeds")
    text_output = output_names.index("text_embeds")
    dummy_ids = np.zeros((1, 64), dtype=np.int64)

    item_ids: list[str] = []
    image_embeddings: list[np.ndarray] = []
    prepared_pixels: list[np.ndarray] = []
    image_seconds: list[float] = []
    dummy_pixels: np.ndarray | None = None
    for item in fixture["items"]:
        image_path = args.images / item["filename"]
        if sha256(image_path) != item["sha256"]:
            raise RuntimeError(f"fixture image hash mismatch: {item['id']}")
        with Image.open(image_path) as source:
            prepared = processor(images=[source.convert("RGB")], return_tensors="np")
        pixels = prepared["pixel_values"].astype(np.float32, copy=False)
        prepared_pixels.append(pixels[0])
        dummy_pixels = pixels
        started = time.perf_counter()
        outputs = session.run(None, {"input_ids": dummy_ids, "pixel_values": pixels})
        image_seconds.append(time.perf_counter() - started)
        image_embeddings.append(outputs[image_output][0])
        item_ids.append(item["id"])
    assert dummy_pixels is not None

    texts = [query["text"] for query in fixture["queries"]]
    text_inputs = processor(
        text=texts,
        padding="max_length",
        truncation=True,
        max_length=64,
        return_tensors="np",
    )
    if args.prepared_output:
        if args.prepared_output.exists():
            raise SystemExit("prepared tensor output already exists")
        np.savez(
            args.prepared_output,
            pixel_values=np.stack(prepared_pixels),
            input_ids=text_inputs["input_ids"].astype(np.int64, copy=False),
            item_ids=np.asarray(item_ids),
            fixture_sha256=np.asarray(sha256(args.fixture)),
        )
    started = time.perf_counter()
    outputs = session.run(
        None,
        {
            "input_ids": text_inputs["input_ids"].astype(np.int64, copy=False),
            "pixel_values": dummy_pixels,
        },
    )
    text_seconds = time.perf_counter() - started

    image_matrix = normalize(np.stack(image_embeddings))
    text_matrix = normalize(outputs[text_output])
    float32_results = rank(fixture["queries"], item_ids, text_matrix @ image_matrix.T)
    float16_images = normalize(image_matrix.astype(np.float16).astype(np.float32))
    float16_results = rank(fixture["queries"], item_ids, text_matrix @ float16_images.T)
    top3_identical = sum(
        left["top3"] == right["top3"] for left, right in zip(float32_results, float16_results, strict=True)
    )

    print(
        json.dumps(
            {
                "model_bytes": args.model.stat().st_size,
                "model_sha256": args.expected_model_sha256,
                "onnxruntime": ort.__version__,
                "items": len(item_ids),
                "queries": len(texts),
                "image_inference_seconds": image_seconds,
                "text_batch_inference_seconds": text_seconds,
                "monolithic_graph_caveat": "Every call executes both image and text branches.",
                "float32": {
                    "zh-CN": metrics(float32_results, "zh-CN"),
                    "en": metrics(float32_results, "en"),
                    "results": float32_results,
                },
                "float16_image_storage": {
                    "zh-CN": metrics(float16_results, "zh-CN"),
                    "en": metrics(float16_results, "en"),
                    "top3_identical_queries": top3_identical,
                    "results": float16_results,
                },
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
