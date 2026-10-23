"""Real three-family HTTP regression at the unchanged AI container budget."""
import json
import io
import math
import os
from pathlib import Path
import resource
import time
from concurrent.futures import ThreadPoolExecutor

from fastapi.testclient import TestClient
from PIL import Image

import server
import tag_contract


def memory():
    root = Path("/sys/fs/cgroup")
    def number(name):
        return int((root / name).read_text().strip())
    events = {key: int(value) for key, value in (line.split() for line in (root / "memory.events").read_text().splitlines())}
    return {"memory_current_bytes": number("memory.current"), "memory_peak_bytes": number("memory.peak"),
            "memory_limit_bytes": number("memory.max"),
            "parent_peak_rss_kib": resource.getrusage(resource.RUSAGE_SELF).ru_maxrss,
            "children_peak_rss_kib": resource.getrusage(resource.RUSAGE_CHILDREN).ru_maxrss,
            "memory_events": events}


def report(stage, **fields):
    print(json.dumps({"stage": stage, **memory(), **fields}, sort_keys=True), flush=True)


def vector(response):
    values = json.loads(response["clip"])
    assert len(values) == 512 and all(type(v) in (float, int) and math.isfinite(v) for v in values)
    assert abs(math.hypot(*values) - 1) < 0.01


def main():
    payload = Path("/fixtures/azure-01.png").read_bytes()
    pids, rows = [], []
    report("before_startup")
    with TestClient(server.application) as client:
        expected = server.expected_tag_identity(tag_contract.MANIFEST)
        assert client.get("/api/v1/tag/identity").json() == expected
        assert client.get("/api/v1/identity").json() == server.identity
        assert client.get("/api/v1/face/identity").json() == server.face_identity
        assert memory()["memory_limit_bytes"] == 2304 * 1024 * 1024
        report("verified_ready", tag_identity=expected, search_identity=server.identity,
               verified_source_bytes=1168346001, verified_cache_budget_mib=1200)
        assert not list(Path("/verified-cache").iterdir())
        for iteration in range(3):
            for kind in ("textual", "visual", "face", "tag"):
                started = time.perf_counter()
                if kind in ("textual", "visual"):
                    fields = {"entries": json.dumps({"clip": {kind: {"modelName": server.MODEL}}})}
                    if kind == "textual":
                        fields["text"] = "a flower in a forest"
                        response = client.post("/predict", data=fields)
                    else:
                        response = client.post("/predict", data=fields, files={"image": ("synthetic.png", payload)})
                else:
                    response = client.post(f"/api/v1/{kind}/predict", files={"image": ("synthetic.png", payload)})
                elapsed = round((time.perf_counter() - started) * 1000, 3)
                assert response.status_code == 200, (kind, response.status_code, response.text)
                result = response.json()
                details = {}
                if kind in ("textual", "visual"):
                    vector(result)
                    details["dimension"] = 512
                elif kind == "face":
                    details["face_count"] = len(result["faces"])
                    for face in result["faces"]:
                        assert len(face["embedding"]) == 512 and abs(math.hypot(*face["embedding"]) - 1) < 0.001
                else:
                    tag_contract.normalize(result)
                    assert result["labels"], "Synthetic fixture unexpectedly produced no allowed labels"
                    details["labels"] = result["labels"]
                    for header, field in (("X-FolioPath-Tag-Model", "model_sha256"), ("X-FolioPath-Tag-Policy", "policy_sha256"),
                                          ("X-FolioPath-Tag-Cache", "cache_manifest_sha256"), ("X-FolioPath-Runtime", "runtime_contract_sha256")):
                        assert response.headers[header] == expected[field]
                owner = server.scheduler
                assert not owner.processes and owner.pending == 0
                assert not list(Path("/verified-cache").iterdir())
                pid = owner.last_pid
                try:
                    os.kill(pid, 0)
                except ProcessLookupError:
                    pass
                else:
                    raise AssertionError("Inference child still exists after HTTP response")
                pids.append(pid)
                row = {"iteration": iteration + 1, "kind": kind, "elapsed_ms": elapsed,
                       "child_pid": pid, "child_reaped": True, **details}
                rows.append(row)
                report("inference_completed", **row)
        assert len(set(pids)) == 12 and server.scheduler.completed == 12
        # Largest accepted edge, from an existing synthetic fixture, never real media.
        stream = io.BytesIO()
        with Image.open(io.BytesIO(payload)) as fixture:
            fixture.convert("RGB").resize((1600, 1600), Image.Resampling.BICUBIC).save(stream, format="PNG")
        large = stream.getvalue()
        for kind in ("textual", "visual", "face", "tag"):
            started = time.perf_counter()
            if kind == "textual":
                response = client.post("/predict", data={"entries": json.dumps({"clip": {kind: {"modelName": server.MODEL}}}), "text": "a flower in a forest"})
            elif kind == "visual":
                response = client.post("/predict", data={"entries": json.dumps({"clip": {kind: {"modelName": server.MODEL}}})}, files={"image": ("synthetic-max.png", large)})
            else:
                response = client.post(f"/api/v1/{kind}/predict", files={"image": ("synthetic-max.png", large)})
            assert response.status_code == 200, (kind, response.status_code, response.text)
            assert server.scheduler.pending == 0 and not server.scheduler.processes
            assert not list(Path("/verified-cache").iterdir())
            report("max_input_completed", kind=kind, width=1600, height=1600, image_bytes=len(large),
                   elapsed_ms=round((time.perf_counter()-started)*1000, 3), child_reaped=True)
        # Real inference overlap: the second family waits, a third is rejected.
        with ThreadPoolExecutor(2) as pool:
            first = pool.submit(client.post, "/api/v1/face/predict", files={"image": ("max.png", large)})
            second = pool.submit(client.post, "/api/v1/tag/predict", files={"image": ("max.png", large)})
            deadline = time.monotonic() + 5
            while server.scheduler.pending != 2 and time.monotonic() < deadline:
                time.sleep(0.005)
            assert server.scheduler.pending == 2 and len(server.scheduler.processes) == 1
            overloaded = client.post("/predict", data={"entries": json.dumps({"clip": {"textual": {"modelName": server.MODEL}}}), "text": "flower"})
            assert overloaded.status_code == 503 and overloaded.json() == {"error": "busy"}
            assert first.result(timeout=30).status_code == 200
            assert second.result(timeout=30).status_code == 200
        assert server.scheduler.pending == 0 and not server.scheduler.processes
        report("overload_completed", active_limit=1, queued_limit=1, third_status=503, all_children_reaped=True)
        assert memory()["memory_events"]["oom"] == 0 and memory()["memory_events"]["oom_kill"] == 0
        report("completed", request_count=18, result="PASS", no_resident_children=True,
               caveat="Synthetic fixtures, CPU ARM64 only; not accuracy, throughput or release acceptance")
    assert server.scheduler is None
    assert not list(Path("/verified-cache").iterdir())
    report("shutdown_cleaned")


if __name__ == "__main__":
    main()
