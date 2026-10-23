"""All five CPU sessions under unchanged Compose memory/tmpfs budgets.

Uses the existing image's canonical snapshot and ORT configuration implementations.
It does not start the HTTP service, modify source caches, or access media libraries.
"""

import json
import os
import resource
import signal
import sys
import time
from pathlib import Path

from probe import cgroup_number, prepare_image, select_safe_tags


def report(stage, **fields):
    events = {}
    try:
        for line in Path("/sys/fs/cgroup/memory.events").read_text().splitlines():
            key, value = line.split()
            events[key] = int(value)
    except OSError:
        pass
    print(json.dumps({"stage": stage,
                      "process_peak_rss_kib": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
                      "cgroup_memory_current_bytes": cgroup_number("memory.current"),
                      "cgroup_memory_peak_bytes": cgroup_number("memory.peak"),
                      "cgroup_memory_limit_bytes": cgroup_number("memory.max"),
                      "memory_events": events, **fields}, sort_keys=True), flush=True)


def alarm_handler(_signal, _frame):
    raise TimeoutError("Co-residency probe exceeded its 140 second internal deadline")


def main():
    signal.signal(signal.SIGALRM, alarm_handler)
    signal.alarm(140)
    for key in list(os.environ):
        if key.startswith("MACHINE_LEARNING_"):
            del os.environ[key]
    os.environ.update({"MACHINE_LEARNING_MODEL_INTRA_OP_THREADS": "2",
                       "MACHINE_LEARNING_MODEL_INTER_OP_THREADS": "1",
                       "MACHINE_LEARNING_MODEL_TTL": "0",
                       "MACHINE_LEARNING_ANN": "false",
                       "HF_HUB_OFFLINE": "1", "HF_HUB_DISABLE_TELEMETRY": "1"})

    # Canonical implementations already inside the verified local image.
    # The upstream uvicorn entry point adds its /usr/src app directory; a script
    # invoked from /probe does not, so add that trusted installed source explicitly.
    sys.path.insert(0, "/usr/src")
    from cache_snapshot import snapshot
    import immich_ml.models  # Preserve upstream package initialization order.
    from immich_ml.sessions.ort import OrtSession

    image_manifest = Path("/opt/foliopath-inference/cache-manifest.json")
    legacy_entries = json.loads(image_manifest.read_text())
    manifest = json.loads(Path(__file__).with_name("model-manifest.json").read_text())
    joy_entries = [{"path": item["name"], "size": item["bytes"], "sha256": item["sha256"]}
                   for item in manifest["files"]]
    expected_models = [
        "clip/ViT-B-32__openai/visual/model.onnx",
        "clip/ViT-B-32__openai/textual/model.onnx",
        "facial-recognition/buffalo_l/detection/model.onnx",
        "facial-recognition/buffalo_l/recognition/model.onnx",
    ]
    if not set(expected_models).issubset({entry["path"] for entry in legacy_entries}):
        raise ValueError("Existing image lacks the expected verified CLIP/face manifest")
    size = sum(entry["size"] for entry in legacy_entries + joy_entries)
    filesystem = os.statvfs("/verified-cache")
    available = filesystem.f_bavail * filesystem.f_frsize
    report("preflight", legacy_file_count=len(legacy_entries),
           legacy_snapshot_bytes=sum(entry["size"] for entry in legacy_entries),
           joytag_snapshot_bytes=sum(entry["size"] for entry in joy_entries),
           combined_snapshot_bytes=size, tmpfs_available_bytes=available,
           tmpfs_capacity_bytes=filesystem.f_blocks * filesystem.f_frsize)
    if size > available:
        report("budget_failed", reason="verified-cache snapshot capacity")
        return 2

    started = time.perf_counter()
    legacy = snapshot("/legacy-cache", "/verified-cache", legacy_entries)
    report("legacy_snapshot_verified", elapsed_ms=round((time.perf_counter() - started) * 1000, 3))
    joytag = snapshot("/joytag-source", "/verified-cache", joy_entries)
    report("both_snapshots_verified", elapsed_ms=round((time.perf_counter() - started) * 1000, 3))

    sessions = []
    for label, path in [(name, legacy / name) for name in expected_models] + [("joytag", joytag / "model.onnx")]:
        report("before_session_load", model=label, resident_session_count=len(sessions))
        started = time.perf_counter()
        session = OrtSession(path, providers=["CPUExecutionProvider"])
        sessions.append(session)  # Keep every session resident; no unloading to make it pass.
        report("session_loaded", model=label, resident_session_count=len(sessions),
               elapsed_ms=round((time.perf_counter() - started) * 1000, 3),
               providers=session.providers, provider_options=session.provider_options,
               intra_op_threads=session.sess_options.intra_op_num_threads,
               inter_op_threads=session.sess_options.inter_op_num_threads,
               enable_cpu_mem_arena=session.sess_options.enable_cpu_mem_arena,
               execution_mode=session.sess_options.execution_mode.name,
               graph_optimization_level=session.sess_options.graph_optimization_level.name)

    tensor = prepare_image(Path("/fixtures/azure-01.png"))
    labels = (joytag / "top_tags.txt").read_text().splitlines()
    for iteration in range(3):
        started = time.perf_counter()
        output = sessions[-1].run(None, {sessions[-1].get_inputs()[0].name: tensor})[0]
        report("joytag_inference", iteration=iteration + 1,
               resident_session_count=len(sessions),
               elapsed_ms=round((time.perf_counter() - started) * 1000, 3),
               safe_tags=select_safe_tags(output, labels))
    report("completed", result="five sessions resident and three JoyTag inferences completed",
           caveat="Other model inference workspaces and concurrent requests not exercised")
    signal.alarm(0)
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except Exception as error:
        report("failed", error_type=type(error).__name__, error=str(error))
        raise
