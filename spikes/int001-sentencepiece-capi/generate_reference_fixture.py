#!/usr/bin/env python3
"""Generate the pinned SigLIP tokenizer conformance fixture.

Run only in a disposable environment with the exact dependencies below. The
output is deterministic and intentionally contains no timestamp or host path.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import platform
from pathlib import Path

import sentencepiece
import transformers
from transformers import SiglipTokenizer


MODEL_REVISION = "7fd15f0689c79d79e38b1c2e2e2370a7bf2761ed"
MODEL_SHA256 = "1e5036bed065526c3c212dfbe288752391797c4bb1a284aa18c9a0b23fcaf8ec"
MODEL_SIZE = 798_330
TRANSFORMERS_VERSION = "4.56.2"
SENTENCEPIECE_VERSION = "0.2.1"

CASES = [
    ("english-basic", "red armor portrait"),
    ("english-case-punctuation", "COSER in RED-GOLD armor!!!"),
    ("english-whitespace", "  blue\t hair\nportrait  "),
    ("simplified-chinese", "红色盔甲的角色"),
    ("traditional-chinese", "藍色長髮的角色肖像"),
    ("chinese-english", "角色 portrait 蓝色 hair"),
    ("japanese", "青い髪のコスプレイヤー"),
    ("korean", "빨간 갑옷을 입은 코스플레이어"),
    ("fullwidth-latin", "Ｆｕｌｌｗｉｄｔｈ 模特"),
    ("fullwidth-and-cjk-punctuation", "Ａ／Ｂ，Ｃ！ 人物（正面）"),
    ("composed-accent", "café déjà vu"),
    ("combining-accent", "Cafe\u0301 与 café"),
    ("turkish-greek-german", "İSTANBUL ΣΟΣ ẞ"),
    ("emoji", "角色🦊✨ portrait"),
    ("emoji-zwj", "摄影师👩‍💻与模特🧑🏽‍🎤"),
    ("variation-selector", "红心❤️与红心❤"),
    ("unicode-spaces", "人物\u00a0\u2003\u202f正面"),
    ("line-separators", "人物\u2028蓝色\u2029长发"),
    ("ascii-punctuation-removal", "a/b\\c_d.e,f;g:h?i!"),
    ("digits-and-units", "2 people 35mm f1.4 2026"),
    ("filesystem-like", "set_001/IMG-20260827.JPG"),
    ("quotes-brackets", "\"portrait\" [blue] {hair} <front>"),
    ("rare-cjk", "𠮷野家與龘靐齉"),
    ("arabic", "شخص يرتدي درعًا أحمر"),
    ("cyrillic", "персонаж в красных доспехах"),
    ("devanagari", "लाल कवच पहने पात्र"),
    ("zero-width", "blue\u200bhair\u2060portrait"),
    ("bidi-controls", "red\u200f armor\u202e portrait"),
    ("embedded-nul", "blue\x00hair"),
    ("repeated-emoji-truncation", "🦊" * 512),
    ("exact-rune-limit", "人" * 512),
]


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()

    model = args.model.read_bytes()
    if len(model) != MODEL_SIZE or hashlib.sha256(model).hexdigest() != MODEL_SHA256:
        raise SystemExit("model size or SHA-256 differs from the pinned contract")
    if transformers.__version__ != TRANSFORMERS_VERSION:
        raise SystemExit(f"transformers {transformers.__version__} != {TRANSFORMERS_VERSION}")
    if sentencepiece.__version__ != SENTENCEPIECE_VERSION:
        raise SystemExit(f"sentencepiece {sentencepiece.__version__} != {SENTENCEPIECE_VERSION}")

    tokenizer = SiglipTokenizer(vocab_file=str(args.model), model_max_length=64)
    cases = []
    for name, query in CASES:
        # Transformers 4.56.2 lowercases non-special text through a non-greedy
        # regex. Each match is one code point, so contextual mappings such as
        # Greek final sigma must not be produced by lowering the whole string.
        lowered = "".join(character.lower() for character in query)
        canonical = tokenizer.canonicalize_text(lowered, keep_punctuation_exact_string=None)
        encoded = tokenizer(
            query,
            padding="max_length",
            truncation=True,
            max_length=64,
            return_attention_mask=False,
        )["input_ids"]
        if len(encoded) != 64:
            raise SystemExit(f"{name}: expected 64 IDs, got {len(encoded)}")
        cases.append({"name": name, "query": query, "canonical": canonical, "input_ids": encoded})

    fixture = {
        "schema_version": 1,
        "generator": {
            "python": platform.python_version(),
            "transformers": transformers.__version__,
            "sentencepiece": sentencepiece.__version__,
        },
        "model": {
            "id": "google/siglip-base-patch16-224",
            "revision": MODEL_REVISION,
            "filename": "spiece.model",
            "size_bytes": MODEL_SIZE,
            "sha256": MODEL_SHA256,
        },
        "contract": {"sequence_length": 64, "eos_id": 1, "pad_id": 1},
        "cases": cases,
    }
    args.output.write_text(json.dumps(fixture, ensure_ascii=False, indent=2, sort_keys=True) + "\n")


if __name__ == "__main__":
    main()
