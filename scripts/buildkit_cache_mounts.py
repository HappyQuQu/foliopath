#!/usr/bin/env python3
"""Restore/export only the three reviewed build caches on an ephemeral CI runner."""
import argparse
import os
from pathlib import Path, PurePosixPath
import re
import secrets
import shutil
import subprocess
import tarfile
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


def unpack_snapshot(archive, destination):
    """Unpack only ordinary cache files/directories, including on Python 3.9."""
    destination.mkdir()
    with tarfile.open(archive) as snapshot:
        for member in snapshot:
            relative = PurePosixPath(member.name)
            if (relative.is_absolute() or ".." in relative.parts or "\\" in member.name or
                    member.type not in {tarfile.REGTYPE, tarfile.AREGTYPE, tarfile.DIRTYPE}):
                raise ValueError("Unsafe cache archive member")
            target = destination.joinpath(*relative.parts)
            if member.isdir():
                target.mkdir(parents=True, exist_ok=True)
            else:
                target.parent.mkdir(parents=True, exist_ok=True)
                with snapshot.extractfile(member) as source, target.open("wb") as output:
                    shutil.copyfileobj(source, output)
                target.chmod((member.mode & 0o777) | 0o600)
                os.utime(target, (member.mtime, member.mtime))


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
                archive = Path(scratch) / "cache.tar"
                # One tar stream avoids the local exporter's per-file RPCs,
                # which stalled with the full module cache on a native ARM runner.
                command.extend(["--output", "type=tar,dest=" + str(archive)])
            command.append(str(context))
            run(command, input=dancefile(base, target, mode), text=True, check=True, timeout=60)
            if mode == "extract":
                unpack_snapshot(archive, exported)
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
