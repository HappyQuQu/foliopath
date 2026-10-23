#!/usr/bin/env python3
"""Score fixed, externally prepared semantic tensors with ONNX Runtime."""

from __future__ import annotations

import argparse
import hashlib
import json
import time
from pathlib import Path

import numpy as np
import onnxruntime as ort


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(8 * 1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def normalize(values: np.ndarray) -> np.ndarray:
    values = values.astype(np.float32, copy=False)
    return values / np.linalg.norm(values, axis=1, keepdims=True).clip(min=1e-12)


def results(
    queries: list[dict[str, object]], item_ids: list[str], similarities: np.ndarray
) -> list[dict[str, object]]:
    scored: list[dict[str, object]] = []
    for query_index, query in enumerate(queries):
        order = sorted(
            range(len(item_ids)),
            key=lambda index: (-float(similarities[query_index, index]), item_ids[index]),
        )
        ranked = [item_ids[index] for index in order]
        relevant = set(query["relevant"])
        first = min(index + 1 for index, item_id in enumerate(ranked) if item_id in relevant)
        relevant_top3 = sum(item_id in relevant for item_id in ranked[:3])
        scored.append(
            {
                "id": query["id"],
                "language": query["language"],
                "first_relevant_rank": first,
                "relevant_recall_at_3": relevant_top3 / len(relevant),
                "top3": ranked[:3],
            }
        )
    return scored


def metrics(scored: list[dict[str, object]], language: str) -> dict[str, float]:
    selected = [item for item in scored if item["language"] == language]
    return {
        "queries": len(selected),
        "recall_at_1": sum(item["first_relevant_rank"] <= 1 for item in selected) / len(selected),
        "recall_at_3": sum(item["first_relevant_rank"] <= 3 for item in selected) / len(selected),
        "mean_relevant_recall_at_3": sum(item["relevant_recall_at_3"] for item in selected)
        / len(selected),
        "mrr": sum(1.0 / item["first_relevant_rank"] for item in selected) / len(selected),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True, type=Path)
    parser.add_argument("--tensors", required=True, type=Path)
    parser.add_argument("--fixture", required=True, type=Path)
    parser.add_argument("--expected-model-bytes", required=True, type=int)
    parser.add_argument("--expected-model-sha256", required=True)
    parser.add_argument("--expected-tensors-sha256", required=True)
    parser.add_argument("--threads", type=int, default=4)
    parser.add_argument("--text-batch-size", type=int, default=0)
    args = parser.parse_args()

    if args.model.stat().st_size != args.expected_model_bytes or sha256(args.model) != args.expected_model_sha256:
        raise SystemExit("ONNX size or SHA-256 mismatch")
    if sha256(args.tensors) != args.expected_tensors_sha256:
        raise SystemExit("prepared tensor SHA-256 mismatch")
    fixture = json.loads(args.fixture.read_text(encoding="utf-8"))
    prepared = np.load(args.tensors, allow_pickle=False)
    if str(prepared["fixture_sha256"]) != sha256(args.fixture):
        raise SystemExit("prepared tensors do not bind to this fixture")
    item_ids = [str(item) for item in prepared["item_ids"]]
    if item_ids != [item["id"] for item in fixture["items"]]:
        raise SystemExit("prepared item order does not match fixture")

    options = ort.SessionOptions()
    options.intra_op_num_threads = args.threads
    options.inter_op_num_threads = 1
    session = ort.InferenceSession(
        args.model, sess_options=options, providers=["CPUExecutionProvider"]
    )
    names = [output.name for output in session.get_outputs()]
    image_index = names.index("image_embeds")
    text_index = names.index("text_embeds")
    dummy_ids = np.zeros((1, 64), dtype=np.int64)
    pixels = prepared["pixel_values"].astype(np.float32, copy=False)
    embeddings: list[np.ndarray] = []
    latencies: list[float] = []
    for pixel in pixels:
        started = time.perf_counter()
        output = session.run(None, {"input_ids": dummy_ids, "pixel_values": pixel[None, ...]})
        latencies.append(time.perf_counter() - started)
        embeddings.append(output[image_index][0])
    all_input_ids = prepared["input_ids"].astype(np.int64, copy=False)
    text_batch_size = args.text_batch_size or len(all_input_ids)
    if text_batch_size < 1:
        raise SystemExit("text batch size must be positive")
    text_latencies: list[float] = []
    text_embeddings: list[np.ndarray] = []
    for start in range(0, len(all_input_ids), text_batch_size):
        started = time.perf_counter()
        output = session.run(
            None,
            {
                "input_ids": all_input_ids[start : start + text_batch_size],
                "pixel_values": pixels[0][None, ...],
            },
        )
        text_latencies.append(time.perf_counter() - started)
        text_embeddings.append(output[text_index])

    image_matrix = normalize(np.stack(embeddings))
    text_matrix = normalize(np.concatenate(text_embeddings))
    float32 = results(fixture["queries"], item_ids, text_matrix @ image_matrix.T)
    float16_images = normalize(image_matrix.astype(np.float16).astype(np.float32))
    float16 = results(fixture["queries"], item_ids, text_matrix @ float16_images.T)
    print(
        json.dumps(
            {
                "architecture": __import__("platform").machine(),
                "onnxruntime": ort.__version__,
                "tensor_sha256": args.expected_tensors_sha256,
                "image_inference_seconds": latencies,
                "text_batch_size": text_batch_size,
                "text_inference_seconds": text_latencies,
                "text_total_inference_seconds": sum(text_latencies),
                "float32": {
                    "zh-CN": metrics(float32, "zh-CN"),
                    "en": metrics(float32, "en"),
                    "results": float32,
                },
                "float16_image_storage": {
                    "zh-CN": metrics(float16, "zh-CN"),
                    "en": metrics(float16, "en"),
                    "top3_identical_queries": sum(
                        left["top3"] == right["top3"]
                        for left, right in zip(float32, float16, strict=True)
                    ),
                    "results": float16,
                },
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
