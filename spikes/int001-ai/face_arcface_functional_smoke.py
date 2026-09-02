#!/usr/bin/env python3
"""Run a bounded YuNet + ArcFace replacement-candidate functional smoke."""

from __future__ import annotations

import argparse
import json
import platform
import time
from pathlib import Path

from face_functional_smoke import (
    PLACEHOLDER_REFS,
    digest,
    percentile,
    score_directory_group_pairs,
    select_images,
    validate_args,
)


ARCFACE_REFERENCE = (
    (38.2946, 51.6963),
    (73.5318, 51.5014),
    (56.0252, 71.7366),
    (41.5493, 92.3655),
    (70.7299, 92.2041),
)

EMBEDDER_CONTRACTS = {
    "open-model-zoo-arcface-resnet100-8": {
        "input": ("data", (1, 3, 112, 112), "tensor(float)"),
        "output": ("fc1", (1, 512), "tensor(float)"),
        "preprocess": "open-model-zoo-rgb-raw-v1",
    },
    "fal-auraface-v1-glintr100": {
        "input": ("data", ("dynamic", 3, 112, 112), "tensor(float)"),
        "output": ("1333", (1, 512), "tensor(float)"),
        "preprocess": "insightface-rgb-minus-127.5-div-127.5-v1",
    },
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--catalog", required=True, type=Path)
    parser.add_argument("--model-root", required=True, type=Path)
    parser.add_argument("--normalized-embedder", type=Path)
    parser.add_argument("--normalized-embedder-sha256")
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


def load_models(args: argparse.Namespace) -> tuple[dict[str, object], dict[str, object]]:
    catalog = json.loads(args.catalog.read_text(encoding="utf-8"))
    by_purpose = {entry["purpose"]: entry for entry in catalog["models"]}
    detector = by_purpose["face_detection"]
    embedder = by_purpose["face_embedding"]
    for entry in (detector, embedder):
        path = args.model_root / str(entry["filename"])
        if path.is_symlink() or not path.is_file():
            raise SystemExit(f"model artifact is not a regular file: {path.name}")
        if path.stat().st_size != entry["size_bytes"] or digest(path) != entry["sha256"]:
            raise SystemExit(f"model artifact verification failed: {path.name}")
    return detector, embedder


def align_face(cv2, np, image, face):
    source = np.asarray(face[4:14], dtype=np.float32).reshape(5, 2)
    target = np.asarray(ARCFACE_REFERENCE, dtype=np.float32)
    transform, _ = cv2.estimateAffinePartial2D(source, target, method=cv2.LMEDS)
    if transform is None or transform.shape != (2, 3) or not np.isfinite(transform).all():
        return None
    return cv2.warpAffine(
        image,
        transform,
        (112, 112),
        flags=cv2.INTER_LINEAR,
        borderMode=cv2.BORDER_CONSTANT,
        borderValue=0,
    )


def normalize_shape(shape):
    return tuple("dynamic" if value is None or isinstance(value, str) else value for value in shape)


def validate_embedder_contract(entry, session):
    contract = EMBEDDER_CONTRACTS.get(str(entry["id"]))
    if contract is None:
        raise SystemExit("face embedder has no frozen tensor contract")
    inputs = session.get_inputs()
    outputs = session.get_outputs()
    actual_input = None if len(inputs) != 1 else (inputs[0].name, normalize_shape(inputs[0].shape), inputs[0].type)
    actual_output = None if len(outputs) != 1 else (outputs[0].name, normalize_shape(outputs[0].shape), outputs[0].type)
    if actual_input != contract["input"] or actual_output != contract["output"]:
        raise SystemExit("face embedder graph does not match the frozen tensor contract")
    return contract


def arcface_tensor(cv2, np, aligned, preprocess):
    rgb = cv2.cvtColor(aligned, cv2.COLOR_BGR2RGB).astype(np.float32)
    if preprocess == "insightface-rgb-minus-127.5-div-127.5-v1":
        rgb = (rgb - np.float32(127.5)) / np.float32(127.5)
    elif preprocess != "open-model-zoo-rgb-raw-v1":
        raise SystemExit("unsupported face embedder preprocess contract")
    return rgb.transpose(2, 0, 1)[None, ...]


def main() -> None:
    import cv2
    import numpy as np
    import onnxruntime as ort

    args = parse_args()
    validate_args(args)
    if args.authorization_ref.strip().lower() in PLACEHOLDER_REFS:
        raise SystemExit("--authorization-ref must be a non-placeholder operator reference")
    images, group_count, skipped_oversized = select_images(args)
    detector_entry, embedder_entry = load_models(args)
    detector_path = args.model_root / str(detector_entry["filename"])
    embedder_path = args.model_root / str(embedder_entry["filename"])
    normalized_digest = None
    if (args.normalized_embedder is None) != (args.normalized_embedder_sha256 is None):
        raise SystemExit("normalized embedder path and SHA-256 must be provided together")
    if args.normalized_embedder is not None:
        if args.normalized_embedder.is_symlink() or not args.normalized_embedder.is_file():
            raise SystemExit("normalized embedder is not a regular file")
        normalized_digest = digest(args.normalized_embedder)
        if normalized_digest != args.normalized_embedder_sha256:
            raise SystemExit("normalized embedder SHA-256 mismatch")
        embedder_path = args.normalized_embedder
    detector = cv2.FaceDetectorYN.create(
        str(detector_path), "", (320, 320), args.detector_score_threshold, 0.3, 5000
    )
    options = ort.SessionOptions()
    options.intra_op_num_threads = 2
    options.inter_op_num_threads = 1
    options.enable_cpu_mem_arena = False
    session = ort.InferenceSession(
        str(embedder_path), sess_options=options, providers=["CPUExecutionProvider"]
    )
    embedder_contract = validate_embedder_contract(embedder_entry, session)

    report: dict[str, object] = {
        "schema_version": 2,
        "evidence_class": "replacement-candidate-local-functional-only",
        "dataset_id": args.dataset_id,
        "authorization_ref": args.authorization_ref,
        "generated_at": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "runtime": {
            "opencv": cv2.__version__,
            "onnxruntime": ort.__version__,
            "machine": platform.machine(),
            "backend": "OpenCV YuNet + ONNX Runtime CPU",
        },
        "models": [
            {"id": entry["id"], "version": entry["version"], "sha256": entry["sha256"]}
            for entry in (detector_entry, embedder_entry)
        ],
        "embedder_contract": {
            "input_name": embedder_contract["input"][0],
            "input_shape": list(embedder_contract["input"][1]),
            "output_name": embedder_contract["output"][0],
            "output_shape": list(embedder_contract["output"][1]),
            "preprocess": embedder_contract["preprocess"],
        },
        "derived_graph": None
        if normalized_digest is None
        else {
            "sha256": normalized_digest,
            "transform": "remove-154-legacy-batchnormalization-spatial-zero-attributes-v1",
            "production_approved": False,
        },
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
        report["max_candidates_per_image"] = max(int(report["max_candidates_per_image"]), count)
        report["images_with_candidates"] = int(report["images_with_candidates"]) + int(count > 0)
        if faces is None:
            continue
        for face in faces:
            started = time.perf_counter()
            aligned = align_face(cv2, np, image, face)
            if aligned is None:
                report["invalid_embeddings"] = int(report["invalid_embeddings"]) + 1
                continue
            try:
                feature = session.run(
                    [embedder_contract["output"][0]],
                    {
                        embedder_contract["input"][0]: arcface_tensor(
                            cv2, np, aligned, embedder_contract["preprocess"]
                        )
                    },
                )[0]
            except Exception:
                raise SystemExit(
                    "ArcFace graph failed the frozen ONNX Runtime execution contract"
                ) from None
            embed_ms.append((time.perf_counter() - started) * 1000)
            valid = feature.shape == (1, 512) and np.isfinite(feature).all() and float(np.linalg.norm(feature)) > 0
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
                "This is a replacement-candidate compatibility test, not a model selection or approval.",
                "Directory grouping is not face-level identity ground truth.",
                "The large graph and its exact source/training lineage still require capacity and compliance review.",
            ],
        }
    )
    print(json.dumps(report, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
