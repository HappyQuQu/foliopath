#!/usr/bin/env python3
"""Offline SigLIP semantic retrieval and resource smoke."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import platform
import statistics
import time
from pathlib import Path

import numpy as np
import psutil
import torch
from PIL import Image
from transformers import AutoModel, AutoProcessor


EXPECTED_MODEL_SHA256 = "612923381c76ec5a9bed335d1c48827e3f2e506ac31b044b63b2031fadee6a0b"
EXPECTED_MODEL_BYTES = 1_500_800_904


def file_sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(8 * 1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def percentile(values: list[float], quantile: float) -> float:
    ordered = sorted(values)
    return ordered[max(0, int(np.ceil(quantile * len(ordered))) - 1)]


def normalized(value: torch.Tensor) -> np.ndarray:
    value = value.float()
    value = value / value.norm(p=2, dim=-1, keepdim=True).clamp_min(1e-12)
    result = value.detach().cpu().numpy().copy()
    if not np.isfinite(result).all():
        raise RuntimeError("model produced a non-finite embedding")
    return result


def language_metrics(results: list[dict[str, object]], language: str) -> dict[str, float]:
    selected = [result for result in results if result["language"] == language]
    reciprocal = [1.0 / int(result["first_relevant_rank"]) for result in selected]
    return {
        "queries": len(selected),
        "recall_at_1": sum(int(result["first_relevant_rank"]) <= 1 for result in selected) / len(selected),
        "recall_at_3": sum(int(result["first_relevant_rank"]) <= 3 for result in selected) / len(selected),
        "mean_relevant_recall_at_3": sum(float(result["relevant_recall_at_3"]) for result in selected) / len(selected),
        "mrr": sum(reciprocal) / len(reciprocal),
    }


def ranked_results(
    queries: list[dict[str, object]],
    candidate_ids: list[str],
    similarities: np.ndarray,
) -> list[dict[str, object]]:
    results: list[dict[str, object]] = []
    for query_index, query in enumerate(queries):
        order = sorted(
            range(len(candidate_ids)),
            key=lambda index: (-float(similarities[query_index, index]), candidate_ids[index]),
        )
        ranked = [candidate_ids[index] for index in order]
        relevant = set(query["relevant"])
        first_rank = min(index + 1 for index, item_id in enumerate(ranked) if item_id in relevant)
        relevant_in_top3 = sum(item_id in relevant for item_id in ranked[:3])
        results.append(
            {
                "id": query["id"],
                "pair_id": query["pair_id"],
                "language": query["language"],
                "text": query["text"],
                "first_relevant_rank": first_rank,
                "relevant_count": len(relevant),
                "relevant_in_top3": relevant_in_top3,
                "relevant_recall_at_3": relevant_in_top3 / len(relevant),
                "top3": ranked[:3],
                "top3_scores": [round(float(similarities[query_index, index]), 6) for index in order[:3]],
            }
        )
    return results


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True, type=Path)
    parser.add_argument("--model-id", default="google/siglip2-base-patch16-224")
    parser.add_argument(
        "--revision",
        default="75de2d55ec2d0b4efc50b3e9ad70dba96a7b2fa2",
    )
    parser.add_argument("--expected-weight-bytes", type=int, default=EXPECTED_MODEL_BYTES)
    parser.add_argument("--expected-weight-sha256", default=EXPECTED_MODEL_SHA256)
    parser.add_argument("--fixture", required=True, type=Path)
    parser.add_argument("--images", required=True, type=Path)
    parser.add_argument("--threads", type=int, default=4)
    args = parser.parse_args()

    os.environ["HF_HUB_OFFLINE"] = "1"
    os.environ["TRANSFORMERS_OFFLINE"] = "1"
    torch.set_num_threads(args.threads)
    torch.set_num_interop_threads(1)
    model_file = args.model / "model.safetensors"
    if not model_file.is_file():
        raise SystemExit(f"SigLIP model artifact is missing: {model_file}")
    if (
        model_file.stat().st_size != args.expected_weight_bytes
        or file_sha256(model_file) != args.expected_weight_sha256
    ):
        raise SystemExit("SigLIP model artifact verification failed")
    fixture = json.loads(args.fixture.read_text(encoding="utf-8"))
    process = psutil.Process()
    rss_before = process.memory_info().rss
    load_started = time.perf_counter()
    processor = AutoProcessor.from_pretrained(args.model, local_files_only=True)
    model = AutoModel.from_pretrained(args.model, local_files_only=True).eval()
    load_ms = (time.perf_counter() - load_started) * 1000
    rss_after_load = process.memory_info().rss

    item_ids: list[str] = []
    embeddings: list[np.ndarray] = []
    image_latencies: list[float] = []
    with torch.inference_mode():
        for item in fixture["items"]:
            image_path = args.images / item["filename"]
            if item.get("sha256") and file_sha256(image_path) != item["sha256"]:
                raise RuntimeError(f"fixture image SHA-256 mismatch: {item['id']}")
            with Image.open(image_path) as source:
                image = source.convert("RGB")
            inputs = processor(images=[image], return_tensors="pt")
            started = time.perf_counter()
            embedding = model.get_image_features(**inputs)
            image_latencies.append((time.perf_counter() - started) * 1000)
            embeddings.append(normalized(embedding)[0])
            item_ids.append(item["id"])
    image_matrix = np.stack(embeddings)
    rss_after_images = process.memory_info().rss

    texts = [query["text"] for query in fixture["queries"]]
    text_inputs = processor(
        text=texts,
        padding="max_length",
        max_length=64,
        truncation=True,
        return_tensors="pt",
    )
    with torch.inference_mode():
        started = time.perf_counter()
        text_matrix = normalized(model.get_text_features(**text_inputs))
        text_ms = (time.perf_counter() - started) * 1000
    similarities = (torch.from_numpy(text_matrix) @ torch.from_numpy(image_matrix).T).numpy()
    results = ranked_results(fixture["queries"], item_ids, similarities)
    paired: dict[str, list[dict[str, object]]] = {}
    for result in results:
        paired.setdefault(str(result["pair_id"]), []).append(result)
    top1_agreement = sum(len(values) == 2 and values[0]["top3"][0] == values[1]["top3"][0] for values in paired.values()) / len(paired)

    video_storyboard_proxy = None
    if fixture.get("videos") and fixture.get("video_queries"):
        item_index = {item_id: index for index, item_id in enumerate(item_ids)}
        video_ids = [video["id"] for video in fixture["videos"]]
        video_frames = [
            np.stack([image_matrix[item_index[item_id]] for item_id in video["frame_item_ids"]])
            for video in fixture["videos"]
        ]
        video_mean_matrix = np.stack([frames.mean(axis=0) for frames in video_frames])
        video_mean_matrix /= np.linalg.norm(video_mean_matrix, axis=1, keepdims=True).clip(min=1e-12)
        video_texts = [query["text"] for query in fixture["video_queries"]]
        video_text_inputs = processor(
            text=video_texts,
            padding="max_length",
            max_length=64,
            truncation=True,
            return_tensors="pt",
        )
        with torch.inference_mode():
            video_text_matrix = normalized(model.get_text_features(**video_text_inputs))
        video_mean_similarities = (torch.from_numpy(video_text_matrix) @ torch.from_numpy(video_mean_matrix).T).numpy()
        video_max_similarities = np.stack(
            [
                np.asarray(
                    [float((torch.from_numpy(text) @ torch.from_numpy(frames).T).max()) for frames in video_frames],
                    dtype=np.float32,
                )
                for text in video_text_matrix
            ]
        )
        video_mean_results = ranked_results(fixture["video_queries"], video_ids, video_mean_similarities)
        video_max_results = ranked_results(fixture["video_queries"], video_ids, video_max_similarities)
        video_storyboard_proxy = {
            "frames_per_video": 4,
            "mean_pooling": {
                "zh-CN": language_metrics(video_mean_results, "zh-CN"),
                "en": language_metrics(video_mean_results, "en"),
                "results": video_mean_results,
            },
            "max_frame_score": {
                "zh-CN": language_metrics(video_max_results, "zh-CN"),
                "en": language_metrics(video_max_results, "en"),
                "results": video_max_results,
            },
        }

    fixture_caveats = fixture.get("caveats") or [
        "Eight programmatically drawn images are pipeline smoke, not representative person/cosplay quality evidence."
    ]
    if video_storyboard_proxy is not None:
        fixture_caveats.append(
            "Video results reuse still-image embeddings as a storyboard proxy; they do not test FFmpeg frame admission or real video distributions."
        )
    caveats = fixture_caveats + [
        "PyTorch macOS evidence does not approve an ONNX Linux runtime.",
        "Model resource acceptance requires the complete FolioPath process under a 4 GiB limit with concurrent browsing.",
    ]

    report = {
                "schema_version": 1,
                "evidence_class": fixture.get("evidence_class", "synthetic-development-only"),
                "input_pipeline": fixture.get("input_pipeline", "fixture-defined"),
                "environment": {
                    "os": platform.system().lower(),
                    "machine": platform.machine(),
                    "python": platform.python_version(),
                    "torch": torch.__version__,
                    "transformers": __import__("transformers").__version__,
                    "threads": args.threads,
                    "network_mode": "offline/local_files_only",
                },
                "model": {
                    "id": args.model_id,
                    "revision": args.revision,
                    "license": "Apache-2.0",
                    "weight_bytes": args.expected_weight_bytes,
                    "weight_sha256": args.expected_weight_sha256,
                    "embedding_dimensions": int(image_matrix.shape[1]),
                },
                "load_ms": round(load_ms, 3),
                "image_inference_p50_ms": round(statistics.median(image_latencies), 3),
                "image_inference_p95_ms": round(percentile(image_latencies, 0.95), 3),
                "text_batch_queries": len(texts),
                "text_batch_ms": round(text_ms, 3),
                "rss_before_bytes": rss_before,
                "rss_after_load_bytes": rss_after_load,
                "rss_after_images_bytes": rss_after_images,
                "metrics": {
                    "zh-CN": language_metrics(results, "zh-CN"),
                    "en": language_metrics(results, "en"),
                    "paired_top1_agreement": top1_agreement,
                },
                "results": results,
                "caveats": caveats,
            }
    if video_storyboard_proxy is not None:
        report["video_storyboard_proxy"] = video_storyboard_proxy
    print(json.dumps(report, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
