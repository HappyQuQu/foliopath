#!/usr/bin/env python3
"""Run a bounded, read-only face pipeline smoke on operator-authorized media."""

from __future__ import annotations

import argparse
import hashlib
import json
import math
import os
import platform
import time
from pathlib import Path


SUPPORTED_SUFFIXES = {".jpg", ".jpeg", ".png", ".webp"}
PLACEHOLDER_REFS = {
    "",
    "authorization-required",
    "privacy-review-required",
    "replace-me",
}


def digest(path: Path) -> str:
    result = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(1024 * 1024), b""):
            result.update(chunk)
    return result.hexdigest()


def percentile(values: list[float], quantile: float) -> float | None:
    if not values:
        return None
    ordered = sorted(values)
    index = max(0, math.ceil(quantile * len(ordered)) - 1)
    return round(ordered[index], 3)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", required=True, type=Path)
    parser.add_argument("--model-root", required=True, type=Path)
    parser.add_argument("--media-root", required=True, type=Path)
    parser.add_argument("--dataset-id", required=True)
    parser.add_argument("--authorization-ref", required=True)
    parser.add_argument("--max-per-group", type=int, default=15)
    parser.add_argument("--group-depth", type=int, default=1)
    parser.add_argument("--max-dimension", type=int, default=1600)
    parser.add_argument("--max-file-bytes", type=int, default=250 * 1024 * 1024)
    parser.add_argument("--detector-score-threshold", type=float, default=0.9)
    parser.add_argument("--pair-limit", type=int, default=100_000)
    return parser.parse_args()


def validate_args(args: argparse.Namespace) -> None:
    if not args.dataset_id.strip():
        raise SystemExit("--dataset-id must not be blank")
    if args.authorization_ref.strip().lower() in PLACEHOLDER_REFS:
        raise SystemExit("--authorization-ref must be a non-placeholder operator reference")
    if not 1 <= args.max_per_group <= 100:
        raise SystemExit("--max-per-group must be between 1 and 100")
    if not 1 <= args.group_depth <= 3:
        raise SystemExit("--group-depth must be between 1 and 3")
    if not 64 <= args.max_dimension <= 4096:
        raise SystemExit("--max-dimension must be between 64 and 4096")
    if not 1 <= args.max_file_bytes <= 1024 * 1024 * 1024:
        raise SystemExit("--max-file-bytes must be between 1 byte and 1 GiB")
    if not 0.5 <= args.detector_score_threshold <= 1.0:
        raise SystemExit("--detector-score-threshold must be between 0.5 and 1.0")
    if not 100 <= args.pair_limit <= 1_000_000:
        raise SystemExit("--pair-limit must be between 100 and 1000000")


def cosine(left: list[float], right: list[float]) -> float:
    dot = sum(a * b for a, b in zip(left, right, strict=True))
    left_norm = math.sqrt(sum(value * value for value in left))
    right_norm = math.sqrt(sum(value * value for value in right))
    return dot / (left_norm * right_norm)


def balanced_quotas(capacities: list[int], limit: int) -> list[int]:
    quotas = [0] * len(capacities)
    remaining = min(limit, sum(capacities))
    active = [index for index, capacity in enumerate(capacities) if capacity > 0]
    while remaining > 0 and active:
        share = max(1, remaining // len(active))
        next_active: list[int] = []
        for index in active:
            available = capacities[index] - quotas[index]
            added = min(share, available, remaining)
            quotas[index] += added
            remaining -= added
            if quotas[index] < capacities[index]:
                next_active.append(index)
            if remaining == 0:
                next_active.extend(item for item in active if item > index)
                break
        active = next_active
    return quotas


def evenly_spaced_indices(total: int, count: int) -> set[int]:
    if count >= total:
        return set(range(total))
    return {min(total - 1, ((2 * index + 1) * total) // (2 * count)) for index in range(count)}


def score_directory_group_pairs(
    observations: list[tuple[str, list[float]]], pair_limit: int
) -> dict[str, object]:
    by_group: dict[str, list[list[float]]] = {}
    for group, vector in observations:
        by_group.setdefault(group, []).append(vector)
    groups = sorted(by_group)
    within_group_capacities = [
        len(by_group[group]) * (len(by_group[group]) - 1) // 2 for group in groups
    ]
    within_group_quotas = balanced_quotas(within_group_capacities, pair_limit)
    within_group: list[float] = []
    for group, quota, capacity in zip(
        groups, within_group_quotas, within_group_capacities, strict=True
    ):
        selected = evenly_spaced_indices(capacity, quota)
        pair_index = 0
        vectors = by_group[group]
        for left in range(len(vectors)):
            for right in range(left + 1, len(vectors)):
                if pair_index in selected:
                    within_group.append(cosine(vectors[left], vectors[right]))
                pair_index += 1
    group_pairs = [
        (left_group, right_group)
        for left_group in range(len(groups))
        for right_group in range(left_group + 1, len(groups))
    ]
    cross_group_capacities = [
        len(by_group[groups[left]]) * len(by_group[groups[right]])
        for left, right in group_pairs
    ]
    cross_group_quotas = balanced_quotas(cross_group_capacities, pair_limit)
    cross_group: list[float] = []
    for (left_group, right_group), quota, capacity in zip(
        group_pairs, cross_group_quotas, cross_group_capacities, strict=True
    ):
        selected = evenly_spaced_indices(capacity, quota)
        pair_index = 0
        for left in by_group[groups[left_group]]:
            for right in by_group[groups[right_group]]:
                if pair_index in selected:
                    cross_group.append(cosine(left, right))
                pair_index += 1
    thresholds = (0.363, 0.5, 0.6, 0.7, 0.8)
    metrics: list[dict[str, object]] = []
    for threshold in thresholds:
        within_group_accepted = sum(value >= threshold for value in within_group)
        cross_group_accepted = sum(value >= threshold for value in cross_group)
        metrics.append(
            {
                "threshold": threshold,
                "within_group_accept_rate": round(
                    within_group_accepted / len(within_group), 6
                )
                if within_group
                else None,
                "cross_group_accept_rate": round(
                    cross_group_accepted / len(cross_group), 6
                )
                if cross_group
                else None,
                "cross_group_accepted_pairs": cross_group_accepted,
            }
        )
    return {
        "functional_groups": len(groups),
        "embedded_observations": len(observations),
        "within_group_pairs": len(within_group),
        "cross_group_pairs": len(cross_group),
        "pair_limit": pair_limit,
        "sampling": "deterministic-balanced-across-group-pairs",
        "group_semantics": "directory-group-only-not-identity-ground-truth",
        "threshold_metrics": metrics,
        "identity_labels_persisted": False,
        "embeddings_persisted": False,
        "quality_gate": False,
    }


def regular_images(group: Path, max_file_bytes: int) -> tuple[list[Path], int]:
    images: list[Path] = []
    skipped_oversized = 0
    for current, directories, filenames in os.walk(group, followlinks=False):
        current_path = Path(current)
        directories[:] = sorted(
            name
            for name in directories
            if not (current_path / name).is_symlink()
        )
        for filename in sorted(filenames):
            path = current_path / filename
            if path.suffix.lower() not in SUPPORTED_SUFFIXES or path.is_symlink():
                continue
            stat = path.stat()
            if not path.is_file():
                continue
            if stat.st_size > max_file_bytes:
                skipped_oversized += 1
                continue
            images.append(path)
    return images, skipped_oversized


def select_images(args: argparse.Namespace) -> tuple[list[Path], int, int]:
    if args.media_root.is_symlink():
        raise SystemExit("--media-root must not be a symlink")
    media_root = args.media_root.resolve(strict=True)
    if not media_root.is_dir():
        raise SystemExit("--media-root must be an ordinary directory")
    groups: list[Path] = []
    for current, directories, _ in os.walk(media_root, followlinks=False):
        current_path = Path(current)
        directories[:] = sorted(
            name for name in directories if not (current_path / name).is_symlink()
        )
        depth = len(current_path.relative_to(media_root).parts)
        if depth == args.group_depth:
            groups.append(current_path)
            directories[:] = []
    if not groups:
        groups = [media_root]
    if len(groups) > 500:
        raise SystemExit("media root exceeds the 500-group functional-smoke bound")
    selected: list[Path] = []
    skipped_oversized = 0
    for group in groups:
        candidates, oversized = regular_images(group, args.max_file_bytes)
        selected.extend(candidates[: args.max_per_group])
        skipped_oversized += oversized
        if len(selected) > 5000:
            raise SystemExit("functional smoke exceeds the 5000-image bound")
    if not selected:
        raise SystemExit("no supported regular images found")
    return selected, len(groups), skipped_oversized


def load_models(args: argparse.Namespace) -> tuple[dict[str, object], dict[str, object]]:
    catalog = json.loads(args.catalog.read_text(encoding="utf-8"))
    by_purpose = {entry["purpose"]: entry for entry in catalog["models"]}
    detector_entry = by_purpose["face_detection"]
    embedder_entry = by_purpose["face_embedding"]
    for entry in (detector_entry, embedder_entry):
        path = args.model_root / str(entry["filename"])
        if path.is_symlink() or not path.is_file():
            raise SystemExit(f"model artifact is not a regular file: {path.name}")
        if path.stat().st_size != entry["size_bytes"] or digest(path) != entry["sha256"]:
            raise SystemExit(f"model artifact verification failed: {path.name}")
    return detector_entry, embedder_entry


def main() -> None:
    import cv2
    import numpy as np

    args = parse_args()
    validate_args(args)
    images, group_count, skipped_oversized = select_images(args)
    detector_entry, embedder_entry = load_models(args)
    detector_path = args.model_root / str(detector_entry["filename"])
    embedder_path = args.model_root / str(embedder_entry["filename"])
    detector = cv2.FaceDetectorYN.create(
        str(detector_path),
        "",
        (320, 320),
        args.detector_score_threshold,
        0.3,
        5000,
    )
    recognizer = cv2.FaceRecognizerSF.create(str(embedder_path), "")
    report: dict[str, object] = {
        "schema_version": 2,
        "evidence_class": "operator-authorized-local-functional-only",
        "dataset_id": args.dataset_id,
        "authorization_ref": args.authorization_ref,
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "runtime": {
            "opencv": cv2.__version__,
            "machine": platform.machine(),
            "backend": "OpenCV DNN CPU",
        },
        "models": [
            {"id": entry["id"], "version": entry["version"], "sha256": entry["sha256"]}
            for entry in (detector_entry, embedder_entry)
        ],
        "sampling": {
            "groups": group_count,
            "group_depth": args.group_depth,
            "max_per_group": args.max_per_group,
            "sample_files": len(images),
            "max_dimension": args.max_dimension,
            "max_file_bytes": args.max_file_bytes,
            "skipped_oversized": skipped_oversized,
        },
        "decoded": 0,
        "decode_failures": 0,
        "images_with_candidates": 0,
        "detected_candidates": 0,
        "max_candidates_per_image": 0,
        "valid_embeddings": 0,
        "invalid_embeddings": 0,
    }
    detect_ms: list[float] = []
    embed_ms: list[float] = []
    group_embeddings: list[tuple[str, list[float]]] = []
    media_root = args.media_root.resolve(strict=True)
    for path in images:
        image = cv2.imread(str(path), cv2.IMREAD_COLOR)
        if image is None:
            report["decode_failures"] = int(report["decode_failures"]) + 1
            continue
        report["decoded"] = int(report["decoded"]) + 1
        height, width = image.shape[:2]
        scale = min(1.0, args.max_dimension / max(height, width))
        if scale < 1.0:
            image = cv2.resize(
                image,
                (round(width * scale), round(height * scale)),
                interpolation=cv2.INTER_AREA,
            )
            height, width = image.shape[:2]
        detector.setInputSize((width, height))
        started = time.perf_counter()
        _, faces = detector.detect(image)
        detect_ms.append((time.perf_counter() - started) * 1000)
        count = 0 if faces is None else len(faces)
        report["detected_candidates"] = int(report["detected_candidates"]) + count
        report["max_candidates_per_image"] = max(
            int(report["max_candidates_per_image"]), count
        )
        report["images_with_candidates"] = int(report["images_with_candidates"]) + int(
            count > 0
        )
        if faces is None:
            continue
        for face in faces:
            started = time.perf_counter()
            aligned = recognizer.alignCrop(image, face)
            feature = recognizer.feature(aligned)
            embed_ms.append((time.perf_counter() - started) * 1000)
            valid = (
                feature.shape == (1, 128)
                and np.isfinite(feature).all()
                and float(np.linalg.norm(feature)) > 0
            )
            key = "valid_embeddings" if valid else "invalid_embeddings"
            report[key] = int(report[key]) + 1
            if valid:
                relative = path.relative_to(media_root)
                group = "/".join(relative.parts[: args.group_depth])
                if len(relative.parts) <= args.group_depth:
                    group = "root"
                group_embeddings.append((group, feature.reshape(-1).tolist()))
    report.update(
        {
            "detector_p50_ms": percentile(detect_ms, 0.50),
            "detector_p95_ms": percentile(detect_ms, 0.95),
            "embedding_p50_ms": percentile(embed_ms, 0.50),
            "embedding_p95_ms": percentile(embed_ms, 0.95),
            "raw_images_copied": False,
            "face_crops_persisted": False,
            "embeddings_persisted": False,
            "identity_ground_truth": False,
            "quality_or_release_gate": False,
            "group_pair_functional": score_directory_group_pairs(
                group_embeddings, args.pair_limit
            ),
            "caveats": [
                "This smoke proves bounded local decode, detection, alignment, and embedding only.",
                "Candidate counts are not detector recall or clustering-quality evidence.",
                "No identity inference, training, model publication, or production face composition occurs.",
            ],
        }
    )
    print(json.dumps(report, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
