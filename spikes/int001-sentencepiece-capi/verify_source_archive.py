#!/usr/bin/env python3
"""Verify a SentencePiece source archive against a pinned Git tree response."""

from __future__ import annotations

import argparse
import hashlib
import json
import pathlib
import tarfile


EXPECTED_COMMIT = "31646a467d2051eb904e0b45de3a73e91fe1c1e3"
EXPECTED_PREFIX = "sentencepiece-0.2.1/"
EXPECTED_ARCHIVE_SHA256 = "c1a59e9259c9653ad0ade653dadff074cd31f0a6ff2a11316f67bee4189a8f1b"
MAXIMUM_ARCHIVE_BYTES = 64 << 20
MAXIMUM_TREE_BYTES = 4 << 20
MAXIMUM_MEMBER_BYTES = 32 << 20
MAXIMUM_EXPANDED_BYTES = 128 << 20


def git_blob_digest(data: bytes) -> str:
    header = f"blob {len(data)}\0".encode()
    return hashlib.sha1(header + data, usedforsecurity=False).hexdigest()


def file_sha256(file_path: pathlib.Path) -> str:
    digest = hashlib.sha256()
    with file_path.open("rb") as source:
        while chunk := source.read(1 << 20):
            digest.update(chunk)
    return digest.hexdigest()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--archive", required=True, type=pathlib.Path)
    parser.add_argument("--tree", required=True, type=pathlib.Path)
    arguments = parser.parse_args()

    if not arguments.archive.is_file() or arguments.archive.stat().st_size > MAXIMUM_ARCHIVE_BYTES:
        raise SystemExit("archive is absent or exceeds the verification bound")
    archive_sha256 = file_sha256(arguments.archive)
    if archive_sha256 != EXPECTED_ARCHIVE_SHA256:
        raise SystemExit("archive digest differs from the pinned source input")
    if not arguments.tree.is_file() or arguments.tree.stat().st_size > MAXIMUM_TREE_BYTES:
        raise SystemExit("tree response is absent or exceeds the verification bound")
    tree_document = json.loads(arguments.tree.read_text())
    if tree_document.get("sha") != EXPECTED_COMMIT or tree_document.get("truncated") is not False:
        raise SystemExit("tree response is not the complete pinned commit")
    expected = {
        entry["path"]: entry
        for entry in tree_document.get("tree", [])
        if entry.get("type") == "blob"
    }
    if not expected:
        raise SystemExit("tree response contains no blobs")

    exact = 0
    newline_normalized = 0
    archive_modes: dict[str, bool] = {}
    seen: set[str] = set()
    expanded_bytes = 0
    with tarfile.open(arguments.archive, "r:gz") as archive:
        for member in archive.getmembers():
            if member.name == EXPECTED_PREFIX.rstrip("/") and member.isdir():
                continue
            if not member.name.startswith(EXPECTED_PREFIX):
                raise SystemExit(f"unexpected archive prefix: {member.name}")
            relative = member.name.removeprefix(EXPECTED_PREFIX)
            if not relative:
                continue
            relative_path = pathlib.PurePosixPath(relative)
            if relative_path.is_absolute() or ".." in relative_path.parts:
                raise SystemExit(f"unsafe archive path: {relative}")
            if member.isdir():
                continue
            if not member.isfile():
                raise SystemExit(f"non-regular archive entry: {relative}")
            expanded_bytes += member.size
            if member.size > MAXIMUM_MEMBER_BYTES or expanded_bytes > MAXIMUM_EXPANDED_BYTES:
                raise SystemExit("archive content exceeds the verification bound")
            entry = expected.get(relative)
            if entry is None:
                raise SystemExit(f"archive file absent from Git tree: {relative}")
            extracted = archive.extractfile(member)
            if extracted is None:
                raise SystemExit(f"cannot read archive file: {relative}")
            data = extracted.read()
            if len(data) != entry.get("size"):
                raise SystemExit(f"size differs: {relative}")
            digest = git_blob_digest(data)
            if digest == entry.get("sha"):
                exact += 1
            elif b"\r\n" in data and git_blob_digest(data.replace(b"\r\n", b"\n")) == entry.get("sha"):
                newline_normalized += 1
            else:
                raise SystemExit(f"content differs: {relative}")
            archive_modes[relative] = bool(member.mode & 0o111)
            seen.add(relative)

    missing = sorted(set(expected) - seen)
    if missing:
        raise SystemExit(f"archive is missing {len(missing)} Git blobs: {missing[:3]}")
    for relative, entry in expected.items():
        executable = entry.get("mode") == "100755"
        if archive_modes[relative] != executable:
            raise SystemExit(f"executable mode differs: {relative}")

    result = {
        "schema_version": 1,
        "commit": EXPECTED_COMMIT,
        "archive_sha256": archive_sha256,
        "tree_response_sha256": file_sha256(arguments.tree),
        "blob_count": len(expected),
        "exact_blob_count": exact,
        "newline_normalized_blob_count": newline_normalized,
        "executable_blob_count": sum(entry.get("mode") == "100755" for entry in expected.values()),
        "result": "equivalent",
    }
    print(json.dumps(result, indent=2, sort_keys=True))


if __name__ == "__main__":
    main()
