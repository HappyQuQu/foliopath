#!/usr/bin/env python3
"""Exercise YuNet detection, landmark alignment, and SFace embedding."""

from __future__ import annotations

import argparse
import hashlib
import json
import statistics
import time
from pathlib import Path

import cv2
import numpy as np


def digest(path: Path) -> str:
    result = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            result.update(chunk)
    return result.hexdigest()


def percentile(values: list[float], quantile: float) -> float:
    ordered = sorted(values)
    return ordered[max(0, int(np.ceil(quantile * len(ordered))) - 1)]


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", required=True, type=Path)
    parser.add_argument("--fixture-manifest", required=True, type=Path)
    parser.add_argument("--model-root", required=True, type=Path)
    parser.add_argument("--iterations", type=int, default=20)
    args = parser.parse_args()

    catalog = json.loads(args.catalog.read_text(encoding="utf-8"))
    fixture = json.loads(args.fixture_manifest.read_text(encoding="utf-8"))
    by_purpose = {entry["purpose"]: entry for entry in catalog["models"]}
    detector_entry = by_purpose["face_detection"]
    embedder_entry = by_purpose["face_embedding"]
    detector_path = args.model_root / detector_entry["filename"]
    embedder_path = args.model_root / embedder_entry["filename"]
    image_path = args.model_root / fixture["filename"]

    for path, expected in (
        (detector_path, detector_entry),
        (embedder_path, embedder_entry),
        (image_path, fixture),
    ):
        if path.stat().st_size != expected["size_bytes"] or digest(path) != expected["sha256"]:
            raise SystemExit(f"artifact verification failed: {path.name}")

    image = cv2.imread(str(image_path), cv2.IMREAD_COLOR)
    if image is None:
        raise SystemExit("fixture decode failed")
    height, width = image.shape[:2]
    detector = cv2.FaceDetectorYN.create(
        str(detector_path), "", (width, height), 0.9, 0.3, 5000
    )
    recognizer = cv2.FaceRecognizerSF.create(str(embedder_path), "")

    detector.detect(image)
    latencies: list[float] = []
    faces: np.ndarray | None = None
    for _ in range(args.iterations):
        started = time.perf_counter()
        _, faces = detector.detect(image)
        latencies.append((time.perf_counter() - started) * 1000)
    if faces is None or len(faces) == 0:
        raise SystemExit("pipeline smoke found no face candidates")

    embeddings: list[np.ndarray] = []
    for face in faces:
        aligned = recognizer.alignCrop(image, face)
        feature = recognizer.feature(aligned)
        if feature.shape != (1, 128) or not np.isfinite(feature).all():
            raise SystemExit("invalid SFace embedding")
        embeddings.append(feature)

    print(
        json.dumps(
            {
                "schema_version": 1,
                "evidence_class": "pipeline-smoke-only",
                "runtime": {
                    "opencv": cv2.__version__,
                    "backend": "OpenCV DNN CPU",
                },
                "fixture": {
                    "id": fixture["id"],
                    "sha256": fixture["sha256"],
                    "image_shape": list(image.shape),
                    "consent_or_identity_ground_truth": False,
                },
                "iterations": args.iterations,
                "detected_candidates": len(faces),
                "detector_p50_ms": round(statistics.median(latencies), 3),
                "detector_p95_ms": round(percentile(latencies, 0.95), 3),
                "detector_p99_ms": round(percentile(latencies, 0.99), 3),
                "embedding_count": len(embeddings),
                "embedding_shape": list(embeddings[0].shape),
                "caveats": [
                    "The fixture has no reviewed identity/consent ground truth.",
                    "Candidate count is not detector recall and must not be used as quality evidence.",
                    "OpenCV DNN smoke does not approve the planned ONNX Runtime C API adapter.",
                ],
            },
            indent=2,
        )
    )


if __name__ == "__main__":
    main()
