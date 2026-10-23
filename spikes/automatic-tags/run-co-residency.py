"""Host launcher: bounded isolated container, inspect OOM status, always clean up."""

import json
import subprocess
import sys
import uuid
from pathlib import Path


IMAGE = "sha256:31484348fd97a6cfefd01a83dfe3104f22ba9e4d23e92c1e1000d62774c30689"


def main():
    repo = Path(__file__).resolve().parents[2]
    name = "fp-tag-residency-spike-" + uuid.uuid4().hex[:12]
    args = ["docker", "create", "--name", name, "--pull", "never", "--network", "none",
            "--read-only", "--cap-drop", "ALL", "--security-opt", "no-new-privileges",
            "--memory", "2304m", "--memory-swap", "2304m", "--cpus", "3",
            "--pids-limit", "256", "--stop-timeout", "3",
            "--tmpfs", "/verified-cache:rw,noexec,nosuid,size=1200m,mode=0700",
            "--tmpfs", "/tmp:rw,noexec,nosuid,size=128m",
            "--env", "PYTHONDONTWRITEBYTECODE=1", "--env", "HF_HUB_OFFLINE=1",
            "--env", "HF_HUB_DISABLE_TELEMETRY=1"]
    for source, target in [
        (Path("/Volumes/extended/foliopath/model-cache"), "/legacy-cache"),
        (Path("/Volumes/extended/foliopath/model-experiments/joytag"), "/joytag-source"),
        (repo / "spikes/automatic-tags", "/probe"),
        (repo / "prototypes/apple-redesign/assets/ai-media", "/fixtures"),
    ]:
        if source.is_symlink() or not source.is_dir():
            raise ValueError(f"Expected existing nonsymlink experiment source: {source}")
        args += ["--mount", f"type=bind,src={source},dst={target},readonly"]
    args += ["--entrypoint", "python", IMAGE, "-u", "/probe/co_residency.py"]
    print(json.dumps({"container": name, "image": IMAGE, "deadline_seconds": 150,
                      "memory_mib": 2304, "verified_cache_mib": 1200}), flush=True)
    # Mark the exact random experiment name before create: a timed-out Docker CLI
    # may still have created the container, so attempt bounded cleanup either way.
    attempted = False
    try:
        attempted = True
        subprocess.run(args, check=True, timeout=10, stdout=subprocess.DEVNULL)
        try:
            finished = subprocess.run(["docker", "start", "--attach", name], timeout=150)
            exit_code = finished.returncode
        except subprocess.TimeoutExpired:
            print(json.dumps({"launcher_result": "timeout"}), flush=True)
            subprocess.run(["docker", "stop", "--time", "3", name], timeout=5, check=False)
            exit_code = 124
        state = subprocess.run(["docker", "inspect", "--format", "{{json .State}}", name],
                               check=True, capture_output=True, text=True, timeout=5)
        print(json.dumps({"docker_state": json.loads(state.stdout), "launcher_exit_code": exit_code}), flush=True)
        return exit_code
    finally:
        if attempted:
            cleanup = subprocess.run(["docker", "rm", "--force", name], timeout=5,
                                     capture_output=True, text=True)
            print(json.dumps({"cleanup": "removed isolated container and its tmpfs"
                              if cleanup.returncode == 0 else "container absent or cleanup failed",
                              "container": name, "cleanup_exit_code": cleanup.returncode}), flush=True)


if __name__ == "__main__":
    sys.exit(main())
