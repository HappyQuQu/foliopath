#!/usr/bin/env python3
"""Export fixed-shape image and text SigLIP encoders as separate ONNX graphs."""

from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path

import numpy as np
import onnx
import onnxruntime as ort
import torch
from transformers import AutoModel


EXPECTED_WEIGHT_BYTES = 1_500_800_904
EXPECTED_WEIGHT_SHA256 = "612923381c76ec5a9bed335d1c48827e3f2e506ac31b044b63b2031fadee6a0b"


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(8 * 1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


class ImageEncoder(torch.nn.Module):
    def __init__(self, model: torch.nn.Module) -> None:
        super().__init__()
        self.encoder = model.vision_model

    def forward(self, pixel_values: torch.Tensor) -> torch.Tensor:
        return self.encoder(pixel_values=pixel_values, interpolate_pos_encoding=False).pooler_output


class TextEncoder(torch.nn.Module):
    def __init__(self, model: torch.nn.Module) -> None:
        super().__init__()
        self.encoder = model.text_model

    def forward(self, input_ids: torch.Tensor) -> torch.Tensor:
        return self.encoder(input_ids=input_ids).pooler_output


def export(
    module: torch.nn.Module,
    value: torch.Tensor,
    path: Path,
    input_name: str,
    output_name: str,
) -> None:
    torch.onnx.export(
        module,
        (value,),
        path,
        input_names=[input_name],
        output_names=[output_name],
        opset_version=18,
        dynamo=False,
        external_data=False,
        do_constant_folding=True,
    )
    graph = onnx.load(path, load_external_data=False)
    onnx.checker.check_model(graph, full_check=False)


def compare(path: Path, input_name: str, value: torch.Tensor, expected: np.ndarray) -> dict[str, object]:
    options = ort.SessionOptions()
    options.intra_op_num_threads = 4
    options.inter_op_num_threads = 1
    session = ort.InferenceSession(path, sess_options=options, providers=["CPUExecutionProvider"])
    actual = session.run(None, {input_name: value.numpy()})[0]
    difference = np.abs(actual - expected)
    return {
        "finite": bool(np.isfinite(actual).all()),
        "max_abs": float(difference.max()),
        "mean_abs": float(difference.mean()),
        "allclose_1e_4": bool(np.allclose(actual, expected, atol=1e-4, rtol=1e-4)),
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    parser.add_argument("--expected-weight-bytes", type=int, default=EXPECTED_WEIGHT_BYTES)
    parser.add_argument("--expected-weight-sha256", default=EXPECTED_WEIGHT_SHA256)
    args = parser.parse_args()

    if args.output.exists() and any(args.output.iterdir()):
        raise SystemExit("output directory must be absent or empty")
    weight = args.source / "model.safetensors"
    if weight.stat().st_size != args.expected_weight_bytes or sha256(weight) != args.expected_weight_sha256:
        raise SystemExit("source weight size or SHA-256 mismatch")
    args.output.mkdir(parents=True, exist_ok=True)

    torch.manual_seed(20260826)
    model = AutoModel.from_pretrained(args.source, local_files_only=True, trust_remote_code=False).eval()
    pixel_values = torch.randn(1, 3, 224, 224, dtype=torch.float32)
    input_ids = torch.randint(0, model.config.text_config.vocab_size, (1, 64), dtype=torch.long)
    with torch.inference_mode():
        expected_image = model.get_image_features(pixel_values=pixel_values).numpy()
        expected_text = model.get_text_features(input_ids=input_ids).numpy()

    image_path = args.output / "image_encoder.onnx"
    text_path = args.output / "text_encoder.onnx"
    export(ImageEncoder(model).eval(), pixel_values, image_path, "pixel_values", "image_embeds")
    export(TextEncoder(model).eval(), input_ids, text_path, "input_ids", "text_embeds")

    result = {
        "fixed_shapes": {"pixel_values": [1, 3, 224, 224], "input_ids": [1, 64]},
        "opset": 18,
        "image_encoder": {
            "bytes": image_path.stat().st_size,
            "sha256": sha256(image_path),
            "comparison": compare(image_path, "pixel_values", pixel_values, expected_image),
        },
        "text_encoder": {
            "bytes": text_path.stat().st_size,
            "sha256": sha256(text_path),
            "comparison": compare(text_path, "input_ids", input_ids, expected_text),
        },
    }
    print(json.dumps(result, indent=2, sort_keys=True))
    passed = all(
        result[name]["comparison"]["finite"] and result[name]["comparison"]["allclose_1e_4"]
        for name in ("image_encoder", "text_encoder")
    )
    return 0 if passed else 1


if __name__ == "__main__":
    raise SystemExit(main())
