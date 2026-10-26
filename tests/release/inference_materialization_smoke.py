#!/usr/bin/env python3
"""Build the canonical native AI profile and compare its actual runtime files."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import uuid


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def manifest(root):
    return {p.name: {"sha256": sha(p), "bytes": p.stat().st_size}
            for p in sorted(root.iterdir()) if p.is_file() and p.suffix in (".py", ".json")}


def verify(repo, expected, output):
    recipe = repo / "services/inference/Dockerfile.release"
    copies = [json.loads(line[5:]) for line in recipe.read_text().splitlines()
              if line.startswith("COPY ")]
    output.mkdir(parents=True)
    context = output / "context"
    context.mkdir()
    source_map = []
    for source, destination in copies:
        target = context / source
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(repo / source, target)
        source_map.append({"source": source, "target": Path(destination).name,
                           "sha256": sha(target), "bytes": target.stat().st_size})
    shutil.copyfile(recipe, context / "Dockerfile")
    expected_manifest = manifest(expected)
    generated = {"cache-manifest.json", "tag-manifest.json", "ocr-identity.json"}
    if {x["target"] for x in source_map} | generated != set(expected_manifest):
        raise ValueError("Recipe input/output set differs from the reference runtime")
    for item in source_map:
        if {k: item[k] for k in ("sha256", "bytes")} != expected_manifest[item["target"]]:
            raise ValueError("Canonical source differs from the reference runtime: " + item["source"])
    architecture = subprocess.check_output(
        ["docker", "info", "--format", "{{.Architecture}}"], text=True).strip()
    arch = {"aarch64": "arm64", "arm64": "arm64", "x86_64": "amd64", "amd64": "amd64"}[architecture]
    command = ["docker", "build", "--pull=false", "--network=none", "--platform=linux/" + arch,
               "--iidfile", str(output / "image-id.txt"), "--file", str(context / "Dockerfile"), str(context)]
    with (output / "build.log").open("w") as log:
        subprocess.run(command, check=True, stdout=log, stderr=subprocess.STDOUT)
    image = (output / "image-id.txt").read_text().strip()
    inspected = json.loads(subprocess.check_output(["docker", "image", "inspect", image], text=True))[0]
    if inspected["Architecture"] != arch:
        raise ValueError("Built image is not native to the owning Docker daemon")
    name = "fp-inference-materialization-" + uuid.uuid4().hex[:12]
    subprocess.run(["docker", "create", "--name", name, "--network", "none", "--read-only",
                    "--entrypoint", "/bin/true", image], check=True, stdout=subprocess.DEVNULL)
    try:
        actual_root = output / "runtime"
        actual_root.mkdir()
        subprocess.run(["docker", "cp", name + ":/opt/foliopath-inference/.", str(actual_root)], check=True)
    finally:
        subprocess.run(["docker", "rm", name], check=True, stdout=subprocess.DEVNULL)
    actual = manifest(actual_root)
    if actual != expected_manifest:
        raise ValueError("Built runtime bytes differ from the frozen deployed reference")
    identity_command = ["docker", "run", "--rm", "--network", "none", "--read-only",
                        "--cpus", "1", "--memory", "256m", "--memory-swap", "256m",
                        "--env", "PYTHONDONTWRITEBYTECODE=1", "--entrypoint", "python",
                        image, "-m", "ocr_contract"]
    identity = json.loads(subprocess.check_output(identity_command, text=True))
    if identity != json.loads((expected / "ocr-identity.json").read_text()):
        raise ValueError("Built OCR compatibility identity changed")
    result = {"generatedBy": "tests/release/inference_materialization_smoke.py",
              "nativeDockerArchitecture": architecture, "platform": "linux/" + arch,
              "image": image, "recipeSha256": sha(recipe), "buildCommand": command,
              "generatedCommands": [line[4:] for line in recipe.read_text().splitlines() if line.startswith("RUN ")],
              "sourceMap": source_map, "runtimeFiles": actual, "files": len(actual),
              "pythonFiles": sum(name.endswith(".py") for name in actual),
              "jsonFiles": sum(name.endswith(".json") for name in actual),
              "referenceBytesEqual": True, "ocrIdentityEqual": True,
              "ocrIdentity": identity, "modelsOrMediaMounted": False,
              "httpServiceStarted": False, "canonicalSourceChanged": False}
    (output / "verification.json").write_text(json.dumps(result, indent=2) + "\n")
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", type=Path, default=Path(__file__).resolve().parents[2])
    parser.add_argument("--expected-runtime", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    result = verify(args.repo.resolve(), args.expected_runtime.resolve(), args.output.resolve())
    print(json.dumps({key: result[key] for key in ("image", "platform", "files", "referenceBytesEqual", "ocrIdentityEqual")}))


if __name__ == "__main__":
    main()
