"""Synthetic libvips -> fixed experimental OCR bridge; never a product service."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import uuid

ROOT = Path(__file__).resolve().parents[2]
FIXTURES = ROOT / "tests/fixtures/ocr"
S0 = json.loads((ROOT / "docs/evidence/ocr/2026-10-25-ocr-feasibility-local.json").read_text())
NAMES = ("bilingual.png", "exif-oriented.jpg", "wide.png")
EXPECTED = ("FolioPath OCR 2026", "中文文字识别测试", "订单编号123456", "LOCAL TEST ONLY")


def sha(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def canonical(text):
    return "".join(c for c in text.casefold() if c.isalnum())


def run(image, arguments, output, label, payload=None, mounts=(), memory="1g", entrypoint=None):
    name = "fp-ocr-input-" + uuid.uuid4().hex[:10]
    command = ["docker", "run", "--name", name, "--network", "none", "--read-only",
               "--memory", memory, "--cpus", "2", "--cap-drop", "ALL",
               "--security-opt", "no-new-privileges", "--tmpfs", "/tmp:size=256m",
               "--tmpfs", "/work:size=400m"]
    if payload is not None:
        command += ["-i"]
    for source, target, readonly in mounts:
        command += ["--mount", f"type=bind,src={source},dst={target}" + (",readonly" if readonly else "")]
    if entrypoint:
        command += ["--entrypoint", entrypoint]
    command += [image, *arguments]
    try:
        response = subprocess.run(command, input=payload, capture_output=True, timeout=35)
        (output / (label + ".stdout")).write_bytes(response.stdout)
        (output / (label + ".stderr")).write_bytes(response.stderr)
        inspected = json.loads(subprocess.check_output(["docker", "inspect", name], timeout=10))[0]
        assert inspected["HostConfig"]["NetworkMode"] == "none"
        assert not inspected["HostConfig"]["PortBindings"]
        assert inspected["HostConfig"]["ReadonlyRootfs"]
        assert {(m["Destination"], not m["RW"]) for m in inspected["Mounts"]} == {(m[1], m[2]) for m in mounts}
        assert response.returncode == 0 and not inspected["State"]["OOMKilled"], "bridge process failed"
        value = json.loads(response.stdout)
        return value, {"image": inspected["Image"], "exitCode": 0, "oomKilled": False,
                       "network": "none", "memoryBytes": inspected["HostConfig"]["Memory"]}
    finally:
        subprocess.run(["docker", "rm", "-f", name], stdout=subprocess.DEVNULL,
                       stderr=subprocess.DEVNULL, timeout=15)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--image", required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--cache", type=Path, default=Path("/Volumes/extended/foliopath/ocr-feasibility-cache"))
    args = parser.parse_args()
    for name in NAMES:
        path = FIXTURES / name
        assert not path.is_symlink() and sha(path) == S0["fixtureHashes"][str(path.relative_to(ROOT))]
    assert args.output.is_absolute() and not args.output.is_symlink()
    assert args.cache.is_absolute() and args.cache.is_dir() and not args.cache.is_symlink()
    manifest = json.loads((ROOT / "spikes/ocr/model-manifest.json").read_text())
    model_hashes = {}
    for item in manifest["files"]:
        path = args.cache / item["path"]
        assert not path.is_symlink() and sha(path) == item["sha256"]
        model_hashes[item["path"]] = item["sha256"]
    args.output.mkdir(parents=True, exist_ok=True)
    prepared = Path(tempfile.mkdtemp(prefix="prepared-", dir=args.output))
    native_image = subprocess.check_output(["docker", "image", "inspect", args.image, "--format", "{{.Id}}"], text=True, timeout=10).strip()
    native, native_state = run(native_image, [], args.output, "native", mounts=((FIXTURES, "/fixtures", True), (prepared, "/output", False)))
    assert native["status"] == "passed"
    cases = {}
    for item in native["cases"]:
        name = item["fixture"]
        assert name in NAMES and item["sourceUnchanged"]
        path = prepared / name
        assert sha(path) == item["pngSHA256"]
        result, state = run(S0["candidate"], ["/opt/ocr-spike/ocr_worker.py", "visual", "/ocr-cache", "/work"], args.output, name,
                            payload=path.read_bytes(), memory="4g", entrypoint="python", mounts=((args.cache, "/ocr-cache", True),))
        assert result["identity"] == S0["probe"]["identity"] and result["status"] == "ready"
        lines = [canonical(line["text"]) for line in result["result"]["lines"]]
        if name == "wide.png":
            assert "wideimageocr2048" in "".join(lines)
        else:
            assert lines == [canonical(text) for text in EXPECTED]
        cases[name] = {"prepared": item, "actualOCR": result, "container": state}
    for name in NAMES:
        assert sha(FIXTURES / name) == S0["fixtureHashes"][str((FIXTURES / name).relative_to(ROOT))]
    for path, digest in model_hashes.items():
        assert sha(args.cache / path) == digest
    evidence = {"status": "passed", "scope": "synthetic imagevips input + existing S0 OCR worker, not a production OCR endpoint/index",
                "native": native, "nativeContainer": native_state, "cases": cases,
                "modelHashesPreserved": True, "syntheticSourceHashesPreserved": True,
                "containersRemoved": True, "producerImplemented": False}
    (args.output / "result.json").write_text(json.dumps(evidence, ensure_ascii=False, indent=2) + "\n")
    print(json.dumps({"status": "passed", "cases": len(cases), "image": native_image}))


if __name__ == "__main__":
    main()
