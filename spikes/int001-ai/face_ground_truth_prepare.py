#!/usr/bin/env python3
"""Prepare a private, human-reviewable face evaluation set from authorized media.

The source tree is opened read-only.  Derived thumbnails, embeddings and the review
index are written only below an explicit output directory which must not be inside
the source tree.  Candidate identities are hints; the output never claims ground
truth until a reviewer replaces ``pending`` in review.csv.
"""

from __future__ import annotations

import argparse
import csv
import hashlib
import json
import os
from collections import Counter
from pathlib import Path

from face_arcface_functional_smoke import (
    EMBEDDER_CONTRACTS,
    align_face,
    arcface_tensor,
    normalize_shape,
)
from face_functional_smoke import PLACEHOLDER_REFS, digest


SUPPORTED_SUFFIXES = {".jpg", ".jpeg", ".png", ".webp"}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", required=True, type=Path)
    parser.add_argument("--model-root", required=True, type=Path)
    parser.add_argument("--media-root", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--authorization-ref", required=True)
    parser.add_argument("--max-per-group", type=int, default=60)
    parser.add_argument("--max-images", type=int, default=1000)
    parser.add_argument("--max-dimension", type=int, default=1600)
    parser.add_argument("--detector-score-threshold", type=float, default=0.9)
    parser.add_argument("--cluster-threshold", type=float, default=0.6)
    return parser.parse_args()


def validate_paths(args: argparse.Namespace) -> tuple[Path, Path]:
    if args.authorization_ref.strip().lower() in PLACEHOLDER_REFS:
        raise SystemExit("--authorization-ref must be a non-placeholder reference")
    if not 20 <= args.max_per_group <= 500 or not 20 <= args.max_images <= 5000:
        raise SystemExit("invalid sampling bound")
    if not 64 <= args.max_dimension <= 4096:
        raise SystemExit("invalid --max-dimension")
    if not 0.5 <= args.detector_score_threshold <= 1.0:
        raise SystemExit("invalid detector threshold")
    if not 0.3 <= args.cluster_threshold <= 0.95:
        raise SystemExit("invalid cluster threshold")
    if args.media_root.is_symlink():
        raise SystemExit("--media-root must not be a symlink")
    media_root = args.media_root.resolve(strict=True)
    output = args.output.resolve(strict=False)
    if not media_root.is_dir():
        raise SystemExit("--media-root must be a directory")
    if output == media_root or media_root in output.parents:
        raise SystemExit("--output must be outside the source media tree")
    if output.exists() and any(output.iterdir()):
        raise SystemExit("--output must be absent or empty")
    output.mkdir(parents=True, exist_ok=True)
    return media_root, output


def catalog_models(catalog_path: Path, model_root: Path):
    catalog = json.loads(catalog_path.read_text(encoding="utf-8"))
    by_purpose = {entry["purpose"]: entry for entry in catalog["models"]}
    detector_entry = by_purpose["face_detection"]
    embedder_entry = by_purpose["face_embedding"]
    paths = []
    for entry in (detector_entry, embedder_entry):
        path = model_root / entry["filename"]
        if path.is_symlink() or not path.is_file():
            raise SystemExit(f"model is not a regular file: {path.name}")
        if path.stat().st_size != entry["size_bytes"] or digest(path) != entry["sha256"]:
            raise SystemExit(f"model verification failed: {path.name}")
        paths.append(path)
    return detector_entry, embedder_entry, paths[0], paths[1]


def regular_images(group: Path) -> list[Path]:
    result: list[Path] = []
    for current, directories, filenames in os.walk(group, followlinks=False):
        current_path = Path(current)
        directories[:] = sorted(
            name for name in directories if not (current_path / name).is_symlink()
        )
        for filename in sorted(filenames):
            path = current_path / filename
            if path.suffix.lower() in SUPPORTED_SUFFIXES and not path.is_symlink() and path.is_file():
                result.append(path)
    return result


def evenly_sample(values: list[Path], count: int) -> list[Path]:
    if len(values) <= count:
        return values
    return [values[min(len(values) - 1, ((2 * index + 1) * len(values)) // (2 * count))] for index in range(count)]


def select_images(media_root: Path, max_per_group: int, max_images: int):
    groups = [path for path in sorted(media_root.iterdir()) if path.is_dir() and not path.is_symlink()]
    selected: list[tuple[str, Path]] = []
    for group in groups:
        for path in evenly_sample(regular_images(group), max_per_group):
            selected.append((group.name, path))
    if len(selected) > max_images:
        indices = evenly_sample(list(range(len(selected))), max_images)
        selected = [selected[index] for index in indices]
    return groups, selected


def opaque_id(relative_path: Path, face_index: int) -> str:
    payload = f"{relative_path.as_posix()}\0{face_index}".encode()
    return "face-" + hashlib.sha256(payload).hexdigest()[:16]


def connected_components(np, embeddings, threshold: float, keys=None) -> list[int]:
    if len(embeddings) == 0:
        return []
    # Some reviewed ArcFace graphs expose finite activations large enough for a
    # float32 sum-of-squares to overflow.  Normalize in float64 so candidate
    # clustering cannot silently split identities because of NaN similarities.
    matrix = np.asarray(embeddings, dtype=np.float64)
    if keys is None:
        order = list(range(len(matrix)))
    else:
        if len(keys) != len(matrix) or len(set(keys)) != len(keys):
            raise ValueError("candidate cluster keys must be unique")
        order = sorted(range(len(matrix)), key=lambda index: keys[index])
        matrix = matrix[order]
    matrix /= np.maximum(np.linalg.norm(matrix, axis=1, keepdims=True), 1e-12)
    parent = list(range(len(matrix)))
    component_members = [[index] for index in range(len(matrix))]

    def find(index):
        while parent[index] != index:
            parent[index] = parent[parent[index]]
            index = parent[index]
        return index

    def union(left, right):
        left, right = find(left), find(right)
        if left == right:
            return
        if left > right:
            left, right = right, left
        moving = matrix[component_members[right]]
        anchor_scores = np.einsum("ik,k->i", moving, matrix[left], optimize=False)
        if np.any(anchor_scores < threshold):
            return
        parent[right] = left
        component_members[left].extend(component_members[right])
        component_members[right] = []

    for start in range(0, len(matrix), 256):
        # Avoid platform BLAS implementations which have emitted spurious
        # floating-point exceptions for finite, unit-normalized face vectors.
        scores = np.einsum(
            "ik,jk->ij", matrix[start : start + 256], matrix, optimize=False
        )
        rows, columns = np.where(scores >= threshold)
        for row, column in zip(rows.tolist(), columns.tolist(), strict=True):
            absolute_row = start + row
            if column > absolute_row:
                union(absolute_row, column)
    roots = {}
    ordered_clusters = [roots.setdefault(find(index), len(roots) + 1) for index in range(len(matrix))]
    result = [0] * len(matrix)
    for ordered_index, original_index in enumerate(order):
        result[original_index] = ordered_clusters[ordered_index]
    return result


def write_review(output: Path, records: list[dict], embeddings, clusters) -> None:
    fields = [
        "item_id", "candidate_cluster", "source_group", "review_status", "identity_id",
        "skin_tone", "age", "lighting", "occlusion", "people_count", "relative_path", "thumbnail",
    ]
    with (output / "review.csv").open("w", newline="", encoding="utf-8") as target:
        # Keep the private review file friendly to the Unix tooling used by the
        # documented local workflow while still opening it with newline="" as
        # required by Python's csv module.
        writer = csv.DictWriter(target, fieldnames=fields, lineterminator="\n")
        writer.writeheader()
        for record, cluster in zip(records, clusters, strict=True):
            writer.writerow({
                "item_id": record["item_id"],
                "candidate_cluster": f"candidate-{cluster:03d}",
                "source_group": record["source_group"],
                "review_status": "pending",
                "identity_id": "",
                "skin_tone": "", "age": "", "lighting": "", "occlusion": "", "people_count": "",
                "relative_path": record["relative_path"],
                "thumbnail": record["thumbnail"],
            })
    import numpy as np
    np.save(output / "candidate-embeddings.npy", np.asarray(embeddings, dtype=np.float32), allow_pickle=False)
    (output / "README.txt").write_text(
        "PRIVATE REVIEW MATERIAL — DO NOT COMMIT\n\n"
        "Open contact-sheet.html, then edit review.csv. Candidate clusters are model hints only.\n"
        "For each accepted row set review_status=accepted, assign an opaque identity_id, and label all five slices.\n"
        "Reject secondary people, false detections, duplicates, and uncertain identities.\n",
        encoding="utf-8",
    )
    cards = []
    for record, cluster in zip(records, clusters, strict=True):
        cards.append(
            f'<figure data-cluster="{cluster}"><img src="{record["thumbnail"]}" loading="lazy">'
            f'<figcaption>{record["item_id"]}<br>candidate-{cluster:03d}<br>{record["source_group"]}</figcaption></figure>'
        )
    (output / "contact-sheet.html").write_text(
        "<!doctype html><meta charset=utf-8><title>FolioPath private face review</title>"
        "<style>body{font:13px system-ui;background:#111;color:#eee}main{display:grid;grid-template-columns:repeat(auto-fill,minmax(150px,1fr));gap:10px}figure{margin:0;padding:6px;background:#222}img{width:100%;aspect-ratio:1;object-fit:cover}figcaption{overflow-wrap:anywhere}</style>"
        "<h1>Private candidate review</h1><p>Clusters are hints, not identity ground truth.</p><main>" + "".join(cards) + "</main>",
        encoding="utf-8",
    )


def main() -> None:
    import cv2
    import numpy as np
    import onnxruntime as ort

    args = parse_args()
    media_root, output = validate_paths(args)
    detector_entry, embedder_entry, detector_path, embedder_path = catalog_models(args.catalog, args.model_root)
    contract = EMBEDDER_CONTRACTS.get(embedder_entry["id"])
    if contract is None:
        raise SystemExit("embedder has no frozen contract")
    detector = cv2.FaceDetectorYN.create(str(detector_path), "", (320, 320), args.detector_score_threshold, 0.3, 5000)
    options = ort.SessionOptions()
    options.intra_op_num_threads = 2
    options.inter_op_num_threads = 1
    options.enable_cpu_mem_arena = False
    session = ort.InferenceSession(str(embedder_path), sess_options=options, providers=["CPUExecutionProvider"])
    actual_input = session.get_inputs()[0]
    actual_output = session.get_outputs()[0]
    if (actual_input.name, normalize_shape(actual_input.shape), actual_input.type) != contract["input"] or (actual_output.name, normalize_shape(actual_output.shape), actual_output.type) != contract["output"]:
        raise SystemExit("embedder graph does not match frozen contract")

    groups, images = select_images(media_root, args.max_per_group, args.max_images)
    thumbnails = output / "thumbnails"
    thumbnails.mkdir()
    records, embeddings = [], []
    decoded = images_with_faces = 0
    for source_group, path in images:
        image = cv2.imread(str(path), cv2.IMREAD_COLOR)
        if image is None:
            continue
        decoded += 1
        height, width = image.shape[:2]
        scale = min(1.0, args.max_dimension / max(height, width))
        if scale < 1:
            image = cv2.resize(image, (round(width * scale), round(height * scale)), interpolation=cv2.INTER_AREA)
            height, width = image.shape[:2]
        detector.setInputSize((width, height))
        _, faces = detector.detect(image)
        if faces is None:
            continue
        images_with_faces += 1
        relative = path.relative_to(media_root)
        for face_index, face in enumerate(faces):
            aligned = align_face(cv2, np, image, face)
            if aligned is None:
                continue
            tensor = arcface_tensor(cv2, np, aligned, contract["preprocess"])
            feature = session.run([contract["output"][0]], {contract["input"][0]: tensor})[0].reshape(-1)
            if feature.shape != (512,) or not np.isfinite(feature).all() or float(np.linalg.norm(feature)) == 0:
                continue
            item_id = opaque_id(relative, face_index)
            thumbnail = f"thumbnails/{item_id}.jpg"
            if not cv2.imwrite(str(output / thumbnail), aligned, [cv2.IMWRITE_JPEG_QUALITY, 88]):
                raise SystemExit("failed to write review thumbnail")
            records.append({"item_id": item_id, "source_group": source_group, "relative_path": relative.as_posix(), "thumbnail": thumbnail})
            embeddings.append(feature)
    clusters = connected_components(
        np, embeddings, args.cluster_threshold, [record["item_id"] for record in records]
    )
    write_review(output, records, embeddings, clusters)
    summary = {
        "schema_version": 1,
        "authorization_ref": args.authorization_ref,
        "source_read_only": True,
        "source_groups": len(groups),
        "selected_images": len(images),
        "decoded_images": decoded,
        "images_with_faces": images_with_faces,
        "candidate_faces": len(records),
        "candidate_clusters": len(set(clusters)),
        "candidate_clusters_at_least_20": sum(
            count >= 20 for count in Counter(clusters).values()
        ),
        "largest_candidate_cluster": max(Counter(clusters).values(), default=0),
        "cluster_threshold": args.cluster_threshold,
        "review_complete": False,
        "identity_ground_truth": False,
        "models": [{"id": entry["id"], "sha256": entry["sha256"]} for entry in (detector_entry, embedder_entry)],
    }
    (output / "summary.json").write_text(json.dumps(summary, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(summary, indent=2))


if __name__ == "__main__":
    main()
