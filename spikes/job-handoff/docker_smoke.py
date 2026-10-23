"""Synthetic-only legacy->candidate handoff and real SIGTERM/SIGKILL regression.

Requires explicit verified image identities/deployment in the environment. Never
mounts a real media library or an existing application data directory.
"""
import hashlib
import http.cookiejar
import json
import os
from pathlib import Path
import shutil
import socket
import subprocess
import tempfile
import time
import urllib.error
import urllib.request
import uuid


REPO = Path(__file__).resolve().parents[2]
PROJECT = "fp-handoff-" + uuid.uuid4().hex[:8]
ROOT = Path(tempfile.mkdtemp(prefix=PROJECT + "-"))
APP = PROJECT + "-foliopath-1"
DATA = ROOT / "data"
MEDIA = ROOT / "media"
DATA.mkdir()
(MEDIA / "synthetic").mkdir(parents=True)
SOURCE = Path(os.environ["HANDOFF_SYNTHETIC_PNG"])
assert "fp-localized-tags-" in str(SOURCE), "only the prior synthetic fixture is allowed"
for n in range(80):
    shutil.copyfile(SOURCE, MEDIA / "synthetic" / f"fixture-{n:03}.png")
MEDIA_HASH = {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in (MEDIA / "synthetic").iterdir()}
with socket.socket() as listener:
    listener.bind(("127.0.0.1", 0))
    PORT = listener.getsockname()[1]
BASE = f"http://localhost:{PORT}"
ENV = dict(os.environ, FOLIOPATH_PORT=str(PORT), FOLIOPATH_BIND_ADDRESS="127.0.0.1",
           FOLIOPATH_DATA_PATH=str(DATA), FOLIOPATH_LIBRARY_PATH=str(MEDIA),
           FOLIOPATH_IMAGE=os.environ["HANDOFF_LEGACY_IMAGE"])
COMPOSE = ["docker", "compose", "-p", PROJECT, "-f", str(REPO / "compose.ai.yaml"),
           "-f", str(REPO / "compose.ai.multilingual.yaml")]
TOOL = str(Path(os.environ["HANDOFF_TOOL"]).resolve())
jar = http.cookiejar.CookieJar()
client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(jar))
csrf = None


def command(args, *, data=None, ok=True):
    result = subprocess.run(args, input=data, text=True, capture_output=True, env=ENV, timeout=90)
    if ok and result.returncode:
        raise RuntimeError(f"command failed: {args[:5]}: {result.stderr[-1600:]}")
    return result


def api(path, method="GET", body=None, headers=None):
    hdr = {"Origin": BASE, "Content-Type": "application/json"}
    if csrf:
        hdr["X-CSRF-Token"] = csrf
    hdr.update(headers or {})
    request = urllib.request.Request(BASE + "/api/v1" + path,
                                    None if body is None else json.dumps(body).encode(), hdr, method=method)
    with client.open(request, timeout=15) as response:
        payload = response.read()
        return (json.loads(payload) if payload else None), response.headers


def until(fn, label, timeout=180):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            result = fn()
            if result:
                return result
        except (OSError, urllib.error.URLError):
            pass
        time.sleep(1)
    raise AssertionError("timeout: " + label)


def ready():
    def check():
        with client.open(BASE + "/health/ready", timeout=3) as response:
            return response.status == 200
    until(check, "ready")


def running_snapshot():
    result = command(["docker", "exec", APP, "/app/data/.handoff-tool", "--snapshot"])
    return json.loads(result.stdout)


def mount_tool():
    command(["docker", "cp", TOOL, APP + ":/app/data/.handoff-tool"])


def stopped():
    state = json.loads(command(["docker", "inspect", APP, "--format", "{{json .State}}"]).stdout)
    assert not state["Running"], state
    return state


def db_read(sql, args=()):
    stopped()
    result = command(["docker", "run", "--rm", "--network", "none", "--read-only",
                      "--mount", f"type=bind,src={DATA},dst=/app/data", "--entrypoint", "python",
                      ENV["FOLIOPATH_AI_IMAGE"], "-c",
                      "import sqlite3,json,sys; c=sqlite3.connect('file:/app/data/foliopath.db?mode=ro',uri=True); c.execute('PRAGMA query_only=1'); print(json.dumps(c.execute(sys.argv[1],json.loads(sys.argv[2])).fetchall())); c.close()",
                      sql, json.dumps(args)])
    return json.loads(result.stdout)


def offline_tool(*flags, claims=None, ok=True):
    stopped()
    return command(["docker", "run", "--rm", "-i", "--network", "none", "--read-only",
                    "--mount", f"type=bind,src={DATA},dst=/app/data",
                    "--mount", f"type=bind,src={TOOL},dst=/maintenance-handoff,readonly",
                    "--entrypoint", "/maintenance-handoff", os.environ["HANDOFF_CANDIDATE_IMAGE"], *flags],
                   data=None if claims is None else json.dumps(claims), ok=ok)


def legacy_crash_and_expire():
    # Simulate elapsed lease time only in this stopped synthetic fixture. The
    # actual crash and production recovery/attempt transitions are not mocked.
    command(["docker", "kill", "--signal", "KILL", APP])
    assert stopped()["ExitCode"] == 137
    command(["docker", "run", "--rm", "--network", "none", "--read-only",
             "--mount", f"type=bind,src={DATA},dst=/app/data", "--entrypoint", "python",
             ENV["FOLIOPATH_AI_IMAGE"], "-c",
             "import sqlite3,time; c=sqlite3.connect('/app/data/foliopath.db'); c.execute(\"UPDATE autotag_jobs SET lease_expires_ms=? WHERE state='running'\",(int(time.time()*1000)-1,)); c.commit(); c.close()"])
    command(["docker", "start", APP])
    ready()


def auto_claim():
    return next((c for c in running_snapshot() if c["kind"] == "automatic_tags"), None)


try:
    print(json.dumps({"project": PROJECT, "root": str(ROOT), "port": PORT}), flush=True)
    command(COMPOSE + ["up", "-d"])
    command(["docker", "update", "--restart=no", APP])
    ready()
    mount_tool()
    session, _ = api("/auth/setup", "POST", {"username": "handoff-synthetic", "displayName": "Synthetic handoff", "password": "Synthetic-Handoff-2026!"})
    csrf = session["csrfToken"]
    library, _ = api("/libraries", "POST", {"name": "synthetic", "rootPath": "synthetic"}, {"Idempotency-Key": str(uuid.uuid4())})
    library_id = library["library"]["id"]
    until(lambda: api("/libraries/" + library_id)[0]["status"] == "ready", "scan")
    photo = api("/libraries/" + library_id + "/assets")[0]["items"][0]
    manual, _ = api("/tags", "POST", {"name": "人工升级保留"})
    current, headers = api("/assets/" + photo["id"] + "/curation")
    api("/assets/" + photo["id"] + "/tags/" + manual["id"], "PUT", headers={"If-Match": headers["ETag"]})
    api("/assets/" + photo["id"] + "/favorite", "PUT", {"favorite": True})
    settings_path = "/libraries/" + library_id + "/ai/automatic-tags"
    settings, headers = api(settings_path)
    api(settings_path, "PUT", {"enabled": True}, {"If-Match": headers["ETag"]})
    first = until(lambda: (c if (c := auto_claim()) and c["completedItems"] >= 1 else None), "first committed tag")
    for attempt in (2, 3):
        legacy_crash_and_expire()
        claim = until(lambda: (c if (c := auto_claim()) and c["attemptCount"] == attempt else None), "legacy claim " + str(attempt))
        assert claim["operationId"] == first["operationId"]
    until(lambda: (c if (c := auto_claim()) and c["completedItems"] > first["completedItems"] else None), "legacy progress")
    command(["docker", "stop", "--time", "30", APP])
    assert stopped()["ExitCode"] == 0
    shutil.copytree(DATA, ROOT / "legacy-backup")
    claims = json.loads(offline_tool("--snapshot").stdout)
    before = next(c for c in claims if c["kind"] == "automatic_tags")
    assert before["attemptCount"] == 3, before
    wrong = json.loads(json.dumps(claims))
    wrong[0]["checkpointId"] += 1
    assert offline_tool("--confirm-owner-stopped", claims=wrong, ok=False).returncode != 0
    offline_tool("--confirm-owner-stopped", claims=claims)
    assert offline_tool("--confirm-owner-stopped", claims=claims, ok=False).returncode != 0
    preserved = db_read("SELECT checkpoint_id,attempt_count,state FROM autotag_jobs WHERE operation_id=?", (before["operationId"],))[0]
    assert preserved == [before["checkpointId"], 2, "queued"], preserved
    ENV["FOLIOPATH_IMAGE"] = os.environ["HANDOFF_CANDIDATE_IMAGE"]
    command(COMPOSE + ["up", "-d", "--no-deps", "foliopath"])
    command(["docker", "update", "--restart=no", APP])
    ready()
    # Same bind data persists the read-only diagnostic executable, not product code.
    for turn in range(3):
        resumed = until(lambda: (c if (c := auto_claim()) and c["attemptCount"] == 3 and c["completedItems"] > before["completedItems"] else None), "candidate resumed progress")
        assert resumed["operationId"] == before["operationId"]
        command(["docker", "stop", "--time", "30", APP])
        assert stopped()["ExitCode"] == 0
        row = db_read("SELECT j.state,j.attempt_count,j.checkpoint_id,o.completed_items FROM autotag_jobs j JOIN ai_model_operations o ON o.id=j.operation_id WHERE j.operation_id=?", (before["operationId"],))[0]
        assert row[0] == "queued" and row[1] == 2 and row[2] >= resumed["checkpointId"] and row[3] >= resumed["completedItems"], row
        receipts = db_read("SELECT attempt_before,checkpoint_id,completed_items FROM analysis_job_handoffs WHERE operation_id=?", (before["operationId"],))
        assert len(receipts) == turn + 2 and all(r[0] == 3 for r in receipts), receipts
        before = resumed
        command(["docker", "start", APP])
        ready()
    # A real final SIGKILL is not granted an orderly handoff. Advance only the
    # synthetic lease clock, then verify the original third-attempt crash limit.
    until(auto_claim, "running before final crash")
    legacy_crash_and_expire()
    operation = until(lambda: (o if (o := api("/ai/operations/" + before["operationId"])[0])["state"] == "failed" else None), "crash limit retained")
    assert operation["errorCode"] == "operation_interrupted", operation
    current, _ = api("/assets/" + photo["id"] + "/curation")
    assert current["favorite"] is True and any(t["id"] == manual["id"] and t["name"] == "人工升级保留" and t["source"] == "manual" for t in current["tags"]), current
    command(["docker", "exec", APP, "/app/foliopath", "database-check"])
    assert MEDIA_HASH == {p.name: hashlib.sha256(p.read_bytes()).hexdigest() for p in (MEDIA / "synthetic").iterdir()}
    print(json.dumps({"passed": True, "project": PROJECT, "root": str(ROOT), "port": PORT,
                      "operation": before["operationId"], "lastCheckpoint": before["checkpointId"],
                      "completed": before["completedItems"], "plannedCandidateRestarts": 3,
                      "legacyOfflineHandoff": True, "crashLimitRetained": True,
                      "sourceHashesUnchanged": True, "manualAndFavoritePreserved": True}), flush=True)
except BaseException:
    print(command(["docker", "logs", "--tail", "25", APP], ok=False).stdout, flush=True)
    raise
