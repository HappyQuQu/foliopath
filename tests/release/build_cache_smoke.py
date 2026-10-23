#!/usr/bin/env python3
"""Prove cache mounts survive builder export/import without touching other builders."""
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import uuid

ROOT = Path(__file__).resolve().parents[2]
sys.path.insert(0, str(ROOT / "scripts"))
from buildkit_cache_mounts import MOUNTS, transfer

builder = "foliopath-cache-smoke-" + uuid.uuid4().hex[:12]
env = dict(os.environ, BUILDX_BUILDER=builder)


def run(command, **kwargs):
    return subprocess.run(command, env=env, **kwargs)


try:
    subprocess.run(["docker", "buildx", "create", "--name", builder,
                    "--driver", "docker-container", "--bootstrap"], check=True)
    with tempfile.TemporaryDirectory(prefix="foliopath-cache-roundtrip-") as directory:
        root = Path(directory)
        for name in MOUNTS:
            (root / name).mkdir()
            (root / name / "foliopath-roundtrip").write_text("before")
            # Go module directories are 0555. A non-root Linux local exporter
            # must still be able to create their children in a fresh snapshot.
            module = root / name / "readonly-module" / "nested"
            module.mkdir(parents=True)
            (module / "source.go").write_text("package fixture\n")
            module.chmod(0o555)
            module.parent.chmod(0o555)
        transfer("inject", root, ROOT / "Dockerfile", run)
        for name in MOUNTS:
            (root / name / "foliopath-roundtrip").write_text("after")
        transfer("extract", root, ROOT / "Dockerfile", run)
        for name in MOUNTS:
            assert (root / name / "foliopath-roundtrip").read_text() == "before"
            module = root / name / "readonly-module" / "nested"
            assert (module / "source.go").read_text() == "package fixture\n"
            assert module.stat().st_mode & 0o200
        print("Three real BuildKit cache mounts restored successfully on an isolated builder")
finally:
    subprocess.run(["docker", "buildx", "rm", builder], check=False)
