#!/usr/bin/env python3
"""Reproducibly export and verify the pinned SigLIP 2 spike candidate."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

import numpy as np
import onnx
import torch
from huggingface_hub import snapshot_download
from optimum.exporters.onnx import main_export
from transformers import AutoModel


DEFAULT_MODEL = "google/siglip2-base-patch16-224"
DEFAULT_REVISION = "75de2d55ec2d0b4efc50b3e9ad70dba96a7b2fa2"
DEFAULT_WEIGHT_BYTES = 1_500_800_904
DEFAULT_WEIGHT_SHA256 = "612923381c76ec5a9bed335d1c48827e3f2e506ac31b044b63b2031fadee6a0b"


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(8 * 1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--cache", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--reference-output", required=True, type=Path)
    parser.add_argument("--model-id", default=DEFAULT_MODEL)
    parser.add_argument("--revision", default=DEFAULT_REVISION)
    parser.add_argument("--expected-weight-bytes", type=int, default=DEFAULT_WEIGHT_BYTES)
    parser.add_argument("--expected-weight-sha256", default=DEFAULT_WEIGHT_SHA256)
    args = parser.parse_args()

    if args.output.exists() and any(args.output.iterdir()):
        raise SystemExit("output directory must be absent or empty")
    if args.reference_output.exists():
        raise SystemExit("reference output already exists")

    snapshot = Path(
        snapshot_download(
            repo_id=args.model_id,
            revision=args.revision,
            cache_dir=args.cache,
        )
    )
    weight = snapshot / "model.safetensors"
    weight_digest = sha256(weight)
    if weight.stat().st_size != args.expected_weight_bytes or weight_digest != args.expected_weight_sha256:
        raise SystemExit("source weight size or SHA-256 does not match the reviewed candidate")

    main_export(
        model_name_or_path=str(snapshot),
        output=args.output,
        task="zero-shot-image-classification",
        opset=18,
        framework="pt",
        atol=1e-4,
        local_files_only=True,
        trust_remote_code=False,
        do_validation=True,
        batch_size=1,
        sequence_length=64,
        num_channels=3,
        width=224,
        height=224,
    )

    exported = args.output / "model.onnx"
    graph = onnx.load(exported, load_external_data=False)
    onnx.checker.check_model(graph, full_check=False)

    torch.manual_seed(20260826)
    model = AutoModel.from_pretrained(snapshot, local_files_only=True, trust_remote_code=False).eval()
    input_ids = torch.randint(0, model.config.text_config.vocab_size, (2, 64), dtype=torch.long)
    pixel_values = torch.randn(1, 3, 224, 224, dtype=torch.float32)
    with torch.inference_mode():
        reference = model(input_ids=input_ids, pixel_values=pixel_values, return_dict=True)
    np.savez(
        args.reference_output,
        input_ids=input_ids.numpy(),
        pixel_values=pixel_values.numpy(),
        logits_per_image=reference.logits_per_image.numpy(),
        logits_per_text=reference.logits_per_text.numpy(),
        text_embeds=reference.text_embeds.numpy(),
        image_embeds=reference.image_embeds.numpy(),
    )

    print(
        json.dumps(
            {
                "model_id": args.model_id,
                "revision": args.revision,
                "source_weight_bytes": weight.stat().st_size,
                "source_weight_sha256": weight_digest,
                "onnx_bytes": exported.stat().st_size,
                "onnx_sha256": sha256(exported),
                "onnx_ir_version": graph.ir_version,
                "opsets": [{"domain": item.domain, "version": item.version} for item in graph.opset_import],
                "nodes": len(graph.graph.node),
                "initializers": len(graph.graph.initializer),
                "reference_sha256": sha256(args.reference_output),
            },
            indent=2,
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
