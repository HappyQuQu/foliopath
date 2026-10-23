#!/usr/bin/env python3
"""Restore/export only the three reviewed build caches on an ephemeral CI runner."""
import argparse
import os
from pathlib import Path
import re
import secrets
import subprocess
import tempfile

MOUNTS = {
    "go-mod": "/go/pkg/mod",
    "go-build": "/root/.cache/go-build",
    "npm": "/root/.npm",
}


def cache_base(dockerfile):
    # The Dockerfile remains the single owner of the pinned Debian helper image.
    match = re.search(r"^FROM (debian:[^\s]+@sha256:[a-f0-9]{64}) AS expat$",
                      dockerfile, re.MULTILINE)
    if not match:
        raise ValueError("Pinned Debian build base is required")
    return match[1]


def dancefile(base, target, mode):
    if target not in MOUNTS.values() or mode not in {"inject", "extract"}:
        raise ValueError("Unsupported cache mount or operation")
    header = f"# syntax=docker/dockerfile:1.7\nFROM {base} AS cache\nARG CACHE_NONCE\n"
    if mode == "inject":
        return header + f"""RUN --mount=type=cache,target={target} \
    --mount=type=bind,source=.,target=/restore \
    test -n "$CACHE_NONCE" && cp -a /restore/. {target}/
"""
    return header + f"""RUN --mount=type=cache,target={target} \
    test -n "$CACHE_NONCE" && mkdir /export && cp -a {target}/. /export/ \
    && find /export -type d -exec chmod u+w {{}} +
FROM scratch
COPY --from=cache /export/ /
"""


def transfer(mode, directory, dockerfile="Dockerfile", run=subprocess.run):
    root = Path(directory).resolve()
    root.mkdir(parents=True, exist_ok=True)
    base = cache_base(Path(dockerfile).read_text())
    for name, target in MOUNTS.items():
        destination = root / name
        destination.mkdir(exist_ok=True)
        with tempfile.TemporaryDirectory(prefix="foliopath-build-cache-", dir=root.parent) as scratch:
            context = destination if mode == "inject" else Path(scratch)
            command = ["docker", "buildx", "build", "--file", "-",
                       "--build-arg", "CACHE_NONCE=" + secrets.token_hex(16)]
            if mode == "extract":
                exported = Path(scratch) / "exported"
                command.extend(["--output", "type=local,dest=" + str(exported)])
            command.append(str(context))
            run(command, input=dancefile(base, target, mode), text=True, check=True, timeout=60)
            if mode == "extract":
                # Go module directories are read-only. Export to a fresh directory,
                # then replace the previous snapshot instead of overwriting it.
                previous = Path(scratch) / "previous"
                destination.rename(previous)
                exported.rename(destination)
                for current, _, _ in os.walk(previous):
                    Path(current).chmod(0o700)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("operation", choices=["inject", "extract"])
    parser.add_argument("--directory", required=True)
    args = parser.parse_args()
    transfer(args.operation, args.directory)
