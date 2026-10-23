"""Stage a distinct verified model cache, run disposable same-budget inference regression.

Existing caches and services are never written or contacted. The staged test cache
is retained for the main application's separate synthetic integration tests.
"""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import uuid


def digest(path):
    result = hashlib.sha256()
    with path.open("rb") as stream:
        while chunk := stream.read(1024 * 1024):
            result.update(chunk)
    return result.hexdigest()


def checked(path, entry):
    if path.is_symlink() or not path.is_file() or path.stat().st_size != entry["size"] or digest(path) != entry["sha256"]:
        raise ValueError(f"Unexpected model bytes: {path.name}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--image", default="foliopath-inference:automatic-tags-candidate")
    parser.add_argument("--cache", type=Path, default=Path("/Volumes/extended/foliopath/model-experiments/automatic-tags-integration-cache"))
    args = parser.parse_args()
    repo = Path(__file__).resolve().parents[2]
    image = subprocess.check_output(["docker", "image", "inspect", args.image, "--format", "{{.Id}}"], text=True, timeout=10).strip()
    manifest_code = "import json,pathlib;p=pathlib.Path('/opt/foliopath-inference');print(json.dumps([json.loads((p/n).read_text()) for n in ('cache-manifest.json','tag-manifest.json')]))"
    manifests = json.loads(subprocess.check_output(["docker", "run", "--rm", "--network", "none", "--read-only", "--entrypoint", "python", image,
                                                  "-c", manifest_code], text=True, timeout=15))
    legacy, tags = manifests
    if len(legacy) != 49 or len(tags) != 3:
        raise ValueError("Expected a face+tag candidate with 49 legacy and 3 tag files")
    if args.cache.is_symlink() or not args.cache.is_absolute() or args.cache.parent != Path("/Volumes/extended/foliopath/model-experiments"):
        raise ValueError("Staging cache must be a direct nonsymlink child of the experiment directory")
    args.cache.mkdir(exist_ok=True)
    for entry in legacy + tags:
        relative = Path(entry["path"])
        if relative.is_absolute() or ".." in relative.parts:
            raise ValueError("Invalid pinned manifest path")
        source = (Path("/Volumes/extended/foliopath/model-experiments/joytag") / relative.name
                  if entry in tags else Path("/Volumes/extended/foliopath/model-cache") / relative)
        target = args.cache / relative
        for parent in (source, *source.parents, target, *target.parents):
            if parent.is_symlink():
                raise ValueError("Symlink model cache component")
        checked(source, entry)
        if not target.exists():
            if shutil.disk_usage(args.cache).free < entry["size"] + 1024 ** 3:
                raise ValueError("Insufficient staging space")
            target.parent.mkdir(parents=True, exist_ok=True)
            # Exclusive create: never overwrite a pre-existing experiment file.
            with source.open("rb") as incoming, target.open("xb") as outgoing:
                shutil.copyfileobj(incoming, outgoing, 1024 * 1024)
        checked(target, entry)
    print(json.dumps({"staged_cache": str(args.cache), "verified_files": 52,
                      "verified_bytes": sum(entry["size"] for entry in legacy + tags), "image": image}), flush=True)
    name = "fp-tags-scheduled-" + uuid.uuid4().hex[:12]
    create = ["docker", "create", "--name", name, "--pull", "never", "--network", "none", "--read-only",
              "--cap-drop", "ALL", "--security-opt", "no-new-privileges", "--memory", "2304m", "--memory-swap", "2304m",
              "--cpus", "3", "--pids-limit", "256", "--stop-timeout", "3",
              "--tmpfs", "/verified-cache:rw,noexec,nosuid,size=1200m,mode=0700",
              "--tmpfs", "/tmp:rw,noexec,nosuid,size=128m", "--env", "PYTHONDONTWRITEBYTECODE=1"]
    for source, target in ((args.cache, "/cache"), (repo / "spikes/automatic-tags", "/probe"),
                           (repo / "prototypes/apple-redesign/assets/ai-media", "/fixtures")):
        create.extend(["--mount", f"type=bind,src={source},dst={target},readonly"])
    create.extend(["--entrypoint", "python", image, "-u", "/probe/scheduled_probe.py"])
    print(json.dumps({"container": name, "network": "none", "memory_mib": 2304, "cache_mib": 1200}), flush=True)
    try:
        subprocess.run(create, check=True, timeout=10, stdout=subprocess.DEVNULL)
        try:
            exit_code = subprocess.run(["docker", "start", "--attach", name], timeout=150).returncode
        except subprocess.TimeoutExpired:
            subprocess.run(["docker", "stop", "--time", "3", name], timeout=5, check=False)
            exit_code = 124
        state = json.loads(subprocess.check_output(["docker", "inspect", "--format", "{{json .State}}", name], text=True, timeout=5))
        print(json.dumps({"docker_state": state, "exit_code": exit_code}), flush=True)
        return exit_code
    finally:
        result = subprocess.run(["docker", "rm", "--force", name], timeout=5, capture_output=True, text=True)
        print(json.dumps({"container": name, "cleanup_exit_code": result.returncode,
                          "staged_cache_retained": str(args.cache)}), flush=True)


if __name__ == "__main__":
    raise SystemExit(main())
