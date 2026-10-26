#!/usr/bin/env python3
"""Compatibility probe with optional isolated conversion; never authorizes installation."""

import argparse
import hashlib
import json
from pathlib import Path

import onnx
import sentencepiece


PINNED = {
    "image.onnx": (371674292, "0981d03d194baa6153106d41b5a5ca73faa514e8d9a007a3332eecb63be2c74a"),
    "text.onnx": (441295409, "3983f96348daba5024f5e3085d2ea8cc67c11ce553e4061f193e8ea37aabd792"),
    "spiece.model": (798330, "1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec"),
    "tokenizer.json": (2421038, "ec6fe74c37f2a65be3857ef5e8fbca7662ebb15fcfa569a845a59958db969f76"),
}


def digest(path):
    with path.open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def tensor(value):
    spec = value.type.tensor_type
    return {"name": value.name, "dtype": spec.elem_type,
            "shape": [d.dim_value if d.HasField("dim_value") else d.dim_param for d in spec.shape.dim]}


def convert_and_compare(graph, name, output):
    """Only reshape the external ABI; retain every upstream weight unchanged."""
    import numpy as np
    import onnxruntime as ort

    text = name == "text.onnx"
    old_input = "text" if text else "image"
    new_input = "input_ids" if text else "pixel_values"
    new_output = "text_embeds" if text else "image_embeds"
    options = ort.SessionOptions()
    options.intra_op_num_threads = 2
    options.inter_op_num_threads = 1
    options.enable_cpu_mem_arena = False
    session = ort.InferenceSession(graph.SerializeToString(), sess_options=options, providers=["CPUExecutionProvider"])
    if text:
        sample = np.ones((1, 64), dtype=np.int32)
        sample[0, :3] = [1226, 14431, 9391]
    else:
        sample = np.random.default_rng(20260910).uniform(-1, 1, (1, 3, 224, 224)).astype(np.float32)
    before = session.run(None, {old_input: sample})[0]
    del session
    for node in graph.graph.node:
        for index, value in enumerate(node.input):
            if not text and value == old_input:
                node.input[index] = new_input
            if value == "embedding":
                node.input[index] = new_output
        for index, value in enumerate(node.output):
            if value == "embedding":
                node.output[index] = new_output
    graph.graph.input[0].name = new_input
    graph.graph.output[0].name = new_output
    if text:
        graph.graph.input[0].type.tensor_type.elem_type = onnx.TensorProto.INT64
        graph.graph.node.insert(0, onnx.helper.make_node("Cast", [new_input], [old_input], to=onnx.TensorProto.INT32))
    onnx.checker.check_model(graph)
    destination = output / ("text_encoder.onnx" if text else "image_encoder.onnx")
    if destination.exists():
        raise SystemExit("Refusing to overwrite a candidate")
    onnx.save(graph, destination)
    session = ort.InferenceSession(str(destination), sess_options=options, providers=["CPUExecutionProvider"])
    after = session.run(None, {new_input: sample.astype(np.int64) if text else sample})[0]
    equivalent = bool(np.isfinite(before).all() and np.isfinite(after).all() and np.allclose(before, after, atol=1e-6, rtol=1e-6))
    if not equivalent:
        raise SystemExit("Converted graph failed synthetic equivalence check")
    return {"file": destination.name, "bytes": destination.stat().st_size, "sha256": digest(destination),
            "synthetic_outputs_equivalent": equivalent, "max_abs_error": float(np.abs(before - after).max()),
            "runtime_version": ort.__version__, "release_ready": False}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", type=Path, required=True)
    parser.add_argument("--candidate-output", type=Path, help="Optional NEW directory for development-only ABI conversion")
    parser.add_argument("--token-reference", type=Path, help="Optional repository canonical-token reference fixture")
    args = parser.parse_args()
    # Verify every large input before any native parser is called.
    for name, (size, checksum) in PINNED.items():
        path = args.source / name
        if path.is_symlink() or not path.is_file() or path.stat().st_size != size or digest(path) != checksum:
            raise SystemExit(f"Pinned input verification failed: {name}")
    result = {"status": "candidate-probe-only", "verified_files": list(PINNED), "graphs": {}}
    if args.candidate_output:
        args.candidate_output.mkdir(parents=False, exist_ok=False)
        result["conversion"] = {}
    for name in ("image.onnx", "text.onnx"):
        graph = onnx.load(args.source / name, load_external_data=False)
        if any(t.data_location == onnx.TensorProto.EXTERNAL or t.external_data for t in graph.graph.initializer):
            raise SystemExit("External model data is not allowed")
        onnx.checker.check_model(graph)
        result["graphs"][name] = {
            "inputs": [tensor(v) for v in graph.graph.input],
            "outputs": [tensor(v) for v in graph.graph.output],
            "opsets": [{"domain": v.domain, "version": v.version} for v in graph.opset_import],
        }
        if args.candidate_output:
            result["conversion"][name] = convert_and_compare(graph, name, args.candidate_output)
    tokenizer_path = args.source / "tokenizer.json"
    if tokenizer_path.is_symlink() or tokenizer_path.stat().st_size > 4 * 1024 * 1024:
        raise SystemExit("Invalid tokenizer JSON")
    tokenizer = json.loads(tokenizer_path.read_text())
    sp = sentencepiece.SentencePieceProcessor(model_file=str(args.source / "spiece.model"))
    vocabulary = tokenizer["model"]["vocab"]
    # Matching pieces is necessary, but NOT proof of equivalent normalization/encoding.
    mismatches = [i for i in range(min(len(vocabulary), sp.get_piece_size()))
                  if vocabulary[i][0] != sp.id_to_piece(i)]
    result["tokenizer"] = {
        "json_sha256": digest(tokenizer_path), "json_model_type": tokenizer["model"]["type"],
        "json_vocabulary_size": len(vocabulary), "sentencepiece_vocabulary_size": sp.get_piece_size(),
        "piece_id_mismatches": len(mismatches), "first_mismatched_ids": mismatches[:10],
        "encoding_equivalence_verified": False,
    }
    if args.token_reference:
        cases = json.loads(args.token_reference.read_text())["cases"]
        failures = [i for i, case in enumerate(cases)
                    if (sp.encode(case["canonical"])[:63] + [1] * 64)[:64] != case["input_ids"]]
        result["canonical_token_reference"] = {"cases": len(cases), "failed_indices": failures}
        if failures:
            raise SystemExit("Canonical token reference mismatch")
    result["production_compatible"] = False
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
