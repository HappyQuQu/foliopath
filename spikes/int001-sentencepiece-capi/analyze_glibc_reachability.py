#!/usr/bin/env python3
"""Collect preliminary direct-import evidence for the current glibc findings.

This utility deliberately does not emit VEX. Absence from ELF undefined-symbol
tables cannot exclude glibc-internal paths, dynamic lookup, or future code.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import re
import subprocess
import sys


SCANF = re.compile(r"^(?:[A-Za-z0-9_]*scanf)$")
AFFECTED = {
    "CVE-2026-5450": lambda symbol: bool(SCANF.fullmatch(symbol)),
    "CVE-2026-5928": lambda symbol: symbol in {"ungetwc", "ungetwc_unlocked"},
    "CVE-2026-5435": lambda symbol: symbol in {"ns_printrrf", "ns_printrr", "fp_nquery"},
}
EXACT_STRING_TOKENS = (
    b"ungetwc",
    b"ungetwc_unlocked",
    b"ns_printrrf",
    b"ns_printrr",
    b"fp_nquery",
)


def parse_undefined_symbols(output: str) -> list[str]:
    symbols = []
    for line in output.splitlines():
        if "*UND*" not in line:
            continue
        fields = line.split()
        if fields:
            symbols.append(fields[-1].split("@", 1)[0])
    return sorted(set(symbols))


def parse_needed(output: str) -> list[str]:
    needed = []
    for line in output.splitlines():
        match = re.match(r"^\s*NEEDED\s+(\S+)\s*$", line)
        if match:
            needed.append(match.group(1))
    return sorted(set(needed))


def direct_matches(symbols: list[str]) -> dict[str, list[str]]:
    return {
        cve: sorted(symbol for symbol in symbols if matcher(symbol))
        for cve, matcher in AFFECTED.items()
    }


def string_matches(content: bytes) -> dict[str, list[str]]:
    printable = re.findall(rb"[\x20-\x7e]{4,}", content)
    words = set()
    for value in printable:
        words.update(re.findall(rb"[A-Za-z_][A-Za-z0-9_]*", value))
    decoded = {word.decode("ascii") for word in words}
    matches = direct_matches(sorted(decoded))
    for token in EXACT_STRING_TOKENS:
        text = token.decode("ascii")
        if token in content:
            for cve, matcher in AFFECTED.items():
                if matcher(text) and text not in matches[cve]:
                    matches[cve].append(text)
                    matches[cve].sort()
    return matches


def run_objdump(objdump: str, option: str, path: pathlib.Path) -> str:
    completed = subprocess.run(
        [objdump, option, str(path)],
        check=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )
    return completed.stdout


def analyze(objdump: str, architecture: str, binaries: list[tuple[str, pathlib.Path]]) -> dict:
    expected_format = {"arm64": "elf64-littleaarch64", "amd64": "elf64-x86-64"}.get(
        architecture
    )
    if expected_format is None:
        raise ValueError("architecture must be arm64 or amd64")

    records = []
    all_symbols = set()
    aggregate_direct = {cve: set() for cve in AFFECTED}
    aggregate_strings = {cve: set() for cve in AFFECTED}
    for label, path in binaries:
        if not path.is_file():
            raise ValueError(f"{label}: not a regular file: {path}")
        dynamic = run_objdump(objdump, "-T", path)
        if expected_format not in dynamic:
            raise ValueError(f"{label}: expected {expected_format}")
        program = run_objdump(objdump, "-p", path)
        symbols = parse_undefined_symbols(dynamic)
        symbol_bytes = ("\n".join(symbols) + "\n").encode("utf-8")
        content = path.read_bytes()
        found_direct = direct_matches(symbols)
        found_strings = string_matches(content)
        for cve in AFFECTED:
            aggregate_direct[cve].update(found_direct[cve])
            aggregate_strings[cve].update(found_strings[cve])
        all_symbols.update(symbols)
        records.append(
            {
                "label": label,
                "size_bytes": len(content),
                "sha256": hashlib.sha256(content).hexdigest(),
                "needed": parse_needed(program),
                "undefined_symbol_count": len(symbols),
                "undefined_symbols_sha256": hashlib.sha256(symbol_bytes).hexdigest(),
                "affected_direct_imports": found_direct,
                "affected_exact_name_strings": found_strings,
            }
        )

    combined_bytes = ("\n".join(sorted(all_symbols)) + "\n").encode("utf-8")
    return {
        "schema_version": 1,
        "status": "preliminary_reachability_input_only",
        "architecture": architecture,
        "scope": "external non-glibc ELF closure",
        "binaries": records,
        "combined_unique_undefined_symbol_count": len(all_symbols),
        "combined_undefined_symbols_sha256": hashlib.sha256(combined_bytes).hexdigest(),
        "affected_direct_imports": {
            cve: sorted(values) for cve, values in aggregate_direct.items()
        },
        "affected_exact_name_strings": {
            cve: sorted(values) for cve, values in aggregate_strings.items()
        },
        "vex_decision": "not_made",
        "limitations": [
            "absence of a direct import does not exclude glibc-internal reachability",
            "runtime symbol lookup and future production composition are not covered",
            "the amd64 result, when used, is package evidence from a QEMU-built image",
        ],
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--objdump", default="objdump")
    parser.add_argument("--architecture", required=True)
    parser.add_argument("--binary", action="append", required=True, metavar="LABEL=PATH")
    parser.add_argument("--output", type=pathlib.Path, required=True)
    args = parser.parse_args()
    try:
        binaries = []
        for value in args.binary:
            label, separator, raw_path = value.partition("=")
            if not separator or not label or not raw_path:
                raise ValueError("--binary must be LABEL=PATH")
            binaries.append((label, pathlib.Path(raw_path)))
        result = analyze(args.objdump, args.architecture, binaries)
        encoded = json.dumps(result, indent=2, sort_keys=True) + "\n"
        args.output.write_text(encoded, encoding="utf-8")
    except (OSError, ValueError, subprocess.CalledProcessError) as error:
        print(error, file=sys.stderr)
        return 1
    print(
        json.dumps(
            {
                "architecture": result["architecture"],
                "binary_count": len(result["binaries"]),
                "direct_match_count": sum(
                    len(items) for items in result["affected_direct_imports"].values()
                ),
                "output_sha256": hashlib.sha256(encoded.encode("utf-8")).hexdigest(),
                "vex_decision": result["vex_decision"],
            },
            sort_keys=True,
        )
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
