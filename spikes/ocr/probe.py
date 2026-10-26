"""Network-disabled, synthetic-only OCR feasibility probe. No HTTP/API or database."""
import asyncio
import hashlib
import json
import math
import os
import socket
from pathlib import Path
import tempfile

from ocr_contract import identity, entries
from scheduler import InferenceScheduler, Busy, Unavailable

FIXTURES = Path("/fixtures")
WORK = Path("/work")
CACHE = Path("/ocr-cache")
COMMAND = ("python", "/opt/ocr-spike/ocr_worker.py")
EXPECTED = ["FolioPath OCR 2026", "中文文字识别测试", "订单编号 123456", "LOCAL TEST ONLY"]


def canonical(text):
    return "".join(c for c in text.casefold() if c.isalnum())


def current_anon():
    data = dict(line.split() for line in Path("/sys/fs/cgroup/memory.stat").read_text().splitlines())
    return int(data["anon"])


def source_hashes():
    result = {}
    for entry in entries():
        with (CACHE / entry["path"]).open("rb") as stream:
            result[entry["path"]] = hashlib.file_digest(stream, "sha256").hexdigest()
    return result


def reaped(owner):
    assert owner.pending == 0 and not owner.processes
    assert not list(WORK.iterdir())
    if owner.last_pid:
        try:
            os.kill(owner.last_pid, 0)
        except ProcessLookupError:
            pass
        else:
            raise AssertionError("child still exists")


async def native_loaded(owner, task):
    async def poll():
        while True:
            if task.done():
                raise AssertionError("worker completed before native-memory observation")
            if owner.processes:
                pid = next(iter(owner.processes)).pid
                try:
                    lines = Path(f"/proc/{pid}/status").read_text().splitlines()
                    rss = next(int(line.split()[1]) for line in lines if line.startswith("VmRSS:"))
                    if rss > 768 * 1024:
                        return pid, rss
                except (FileNotFoundError, StopIteration):
                    pass
            await asyncio.sleep(.02)
    return await asyncio.wait_for(poll(), 15)


async def main():
    # Docker Desktop exposes unconfigured kernel tunnel templates even with
    # NetworkMode=none. Require no routes and independently fail an egress socket;
    # the host runner also verifies Docker's actual network mode and no ports.
    assert len(Path("/proc/net/route").read_text().splitlines()) == 1
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as connection:
        connection.settimeout(1)
        try:
            connection.connect(("203.0.113.1", 443))
        except OSError:
            pass
        else:
            raise AssertionError("unexpected network egress")
    assert os.statvfs(CACHE).f_flag & os.ST_RDONLY, "model source must be read-only"
    assert os.statvfs(FIXTURES).f_flag & os.ST_RDONLY, "fixtures must be read-only"
    before = source_hashes()
    assert before == {e["path"]: e["sha256"] for e in entries()}
    fixture_hashes = {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(FIXTURES.glob("*"))}
    owner = InferenceScheduler(CACHE, command=COMMAND, work_root=str(WORK))
    cases = {}
    profile = identity()
    try:
        for name, wanted in [("bilingual.png", "ready"), ("blank.png", "ready"), ("wide.png", "ready"),
                             ("exif-oriented.jpg", "ready"), ("vertical.png", "ready"),
                             ("corrupt.png", "invalid_media"), ("over-pixels.png", "input_limit")]:
            value = await owner.infer("visual", (FIXTURES / name).read_bytes())
            assert value["identity"] == profile
            assert value["status"] == "ready" if wanted == "ready" else value.get("error") == wanted, (name, value)
            if wanted == "ready":
                lines = value["result"]["lines"]
                if name in ("bilingual.png", "exif-oriented.jpg"):
                    assert [canonical(line["text"]) for line in lines] == [canonical(text) for text in EXPECTED], value
                if name == "blank.png": assert lines == []
                if name == "wide.png":
                    assert value["processedSize"][0] == 2048
                    assert "wideimageocr2048" in canonical(" ".join(line["text"] for line in lines)), value
                if name == "vertical.png":
                    value["syntheticMatchingLines"] = sum(canonical(text) in [canonical(line["text"]) for line in lines] for text in EXPECTED)
                    value["syntheticExpectedLines"] = 4
                    # Observation only: this stage does not promise rotated-text accuracy.
            cases[name] = value
            reaped(owner)
        for name, payload, error in [("empty", b"", "invalid_media"), ("over-bytes", b"x" * (8*1024*1024+1), "input_limit")]:
            value = await owner.infer("visual", payload)
            assert value.get("error") == error, value
            cases[name] = value
            reaped(owner)

        # Two real OCR requests share the existing scheduler's one-child gate.
        first = asyncio.create_task(owner.infer("visual", (FIXTURES / "bilingual.png").read_bytes()))
        await native_loaded(owner, first)
        second = asyncio.create_task(owner.infer("visual", (FIXTURES / "blank.png").read_bytes()))
        while owner.pending < 2: await asyncio.sleep(.005)
        assert len(owner.processes) == 1
        try:
            await owner.infer("visual", (FIXTURES / "blank.png").read_bytes())
        except Busy:
            pass
        else:
            raise AssertionError("unbounded queue accepted")
        repeated, blank = await asyncio.gather(first, second)
        assert repeated["result"] == cases["bilingual.png"]["result"] and blank["result"]["lines"] == []
        reaped(owner)
        cases["sharedAdmission"] = {"status": "passed", "pendingBound": 2, "maxChildrenObserved": 1, "thirdRejected": True}

        # Cancellation occurs after observing substantial native model memory, then
        # must reap the child and delete its private snapshot before next admission.
        anon_before = current_anon()
        cancelled = asyncio.create_task(owner.infer("visual", (FIXTURES / "bilingual.png").read_bytes()))
        pid, rss = await native_loaded(owner, cancelled)
        cancelled.cancel()
        try: await cancelled
        except asyncio.CancelledError: pass
        else: raise AssertionError("cancellation did not propagate")
        reaped(owner)
        anon_after = current_anon()
        assert anon_after <= anon_before + 128 * 1024**2, (anon_before, anon_after)
        recovered = await owner.infer("visual", (FIXTURES / "bilingual.png").read_bytes())
        assert recovered["result"] == cases["bilingual.png"]["result"]
        reaped(owner)
        cases["nativeCancellation"] = {"status": "passed", "observedChildRSSKiB": rss,
            "anonymousBefore": anon_before, "anonymousAfterReap": anon_after, "nextRequestReady": True}
    finally:
        await owner.close()
        reaped(owner)

    with tempfile.TemporaryDirectory(prefix="ocr-fault-", dir="/tmp") as faults:
        root = Path(faults)
        for mode in ("missing", "tampered", "symlink"):
            bad = root / mode
            bad.mkdir()
            for entry in entries():
                target = bad / entry["path"]
                target.parent.mkdir(parents=True, exist_ok=True)
                if mode == "tampered":
                    # Same-size sparse file, wrong bytes: size alone must not pass.
                    with target.open("wb") as stream: stream.truncate(entry["size"])
                if mode == "symlink": target.symlink_to(CACHE / entry["path"])
            fault_owner = InferenceScheduler(bad, command=COMMAND, work_root=str(WORK))
            try:
                value = await fault_owner.infer("visual", (FIXTURES / "bilingual.png").read_bytes())
                assert value.get("error") == "model_unavailable", value
                cases[mode+"Model"] = value
            finally:
                await fault_owner.close()
                reaped(fault_owner)

    timed = InferenceScheduler(CACHE, command=COMMAND, timeout=.25, work_root=str(WORK))
    try:
        try: await timed.infer("visual", (FIXTURES / "bilingual.png").read_bytes())
        except Unavailable: pass
        else: raise AssertionError("cache/preprocessing deadline did not expire")
        reaped(timed)
        cases["boundedDeadline"] = {"status": "passed", "seconds": .25, "phase": "worker startup/cache-preprocessing", "childReaped": True}
    finally: await timed.close()

    assert source_hashes() == before
    assert {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(FIXTURES.glob("*"))} == fixture_hashes
    events = dict(line.split() for line in Path("/sys/fs/cgroup/memory.events").read_text().splitlines())
    assert int(events["oom"]) == 0 and int(events["oom_kill"]) == 0
    print(json.dumps({"status": "passed", "scope": "S0 experiment; not production OCR or semantic search",
        "knownLimitations": ["90-degree fixture matched 0/4 lines; no rotation recovery", "synthetic horizontal samples are not a quality benchmark", "no OCR persistence, jobs, API or search integration"],
        "identity": profile, "cases": cases, "caseCount": len(cases), "sourceHashesPreserved": True,
        "fixtureHashes": fixture_hashes, "network": "none", "readOnlyMounts": True,
        "memoryPeakBytes": int(Path("/sys/fs/cgroup/memory.peak").read_text()), "memoryEvents": events,
        "scheduler": "existing command injection seam; visual is experiment transport only, no production registration"}, ensure_ascii=False, sort_keys=True))

if __name__ == "__main__": asyncio.run(main())
