"""Deployment-only fixed source fetcher; no media, API URLs, or production jobs.

Downloads outside the repository, verifies pinned upstream Git/LFS identities,
then mechanically generates the reviewed SHA256 manifest used by the image.
"""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
import json
from pathlib import Path
import subprocess
import urllib.request

REPO = "immich-app/XLM-Roberta-Large-Vit-B-16Plus"
REVISION = "f8857d09bd30188c40214543d2181f732e64eab4"
PREFIX = "clip/XLM-Roberta-Large-Vit-B-16Plus/"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--cache", type=Path, required=True)
    parser.add_argument("--manifest", type=Path, required=True)
    parser.add_argument("--proxy")
    args = parser.parse_args()
    handlers = [urllib.request.ProxyHandler({"https": args.proxy} if args.proxy else {})]
    opener = urllib.request.build_opener(*handlers)
    entries = []
    for suffix in ("", "/textual", "/visual"):
        url = f"https://huggingface.co/api/models/{REPO}/tree/{REVISION}{suffix}"
        with opener.open(url, timeout=30) as response:
            rows = json.load(response)
        for row in rows:
            path = row["path"]
            if row["type"] != "file" or (not suffix and path != "config.json"):
                continue
            if path.startswith("/") or any(p in ("", ".", "..") for p in path.split("/")):
                raise ValueError("invalid fixed source path")
            if not 0 < row["size"] <= 2**30:
                raise ValueError("file budget")
            entries.append(row)
    if not 3 <= len(entries) <= 512 or sum(e["size"] for e in entries) > 4 * 2**30:
        raise ValueError("manifest budget")
    print(f"fixed revision {REVISION}: {len(entries)} files / {sum(e['size'] for e in entries)} bytes", flush=True)

    def fetch(row):
        path = args.cache / PREFIX / row["path"]
        path.parent.mkdir(parents=True, exist_ok=True)
        if not path.exists():
            partial = path.with_name(path.name + ".partial")
            command = ["curl", "--fail", "--silent", "--show-error", "--location", "--proto", "=https",
                       "--proto-redir", "=https", "--max-redirs", "5", "--retry", "5", "--retry-all-errors", "--connect-timeout", "15",
                       "--max-time", "900", "--output", str(partial)]
            command += ["--proxy", args.proxy] if args.proxy else ["--noproxy", "*"]
            command += [f"https://huggingface.co/{REPO}/resolve/{REVISION}/{row['path']}"]
            subprocess.run(command, check=True)
            partial.rename(path)
        digest = hashlib.sha256()
        git = hashlib.sha1(f"blob {row['size']}\0".encode())
        size = 0
        with path.open("rb") as source:
            while chunk := source.read(1024 * 1024):
                size += len(chunk)
                digest.update(chunk)
                git.update(chunk)
        if size != row["size"] or (digest.hexdigest() != row["lfs"]["oid"] if "lfs" in row else git.hexdigest() != row["oid"]):
            raise ValueError(f"fixed source hash mismatch: {row['path']}")
        print("verified " + row["path"], flush=True)
        return {"path": PREFIX + row["path"], "size": size, "sha256": digest.hexdigest()}

    with ThreadPoolExecutor(max_workers=4) as pool:
        manifest = sorted(pool.map(fetch, entries), key=lambda e: e["path"])
    args.manifest.write_text(json.dumps(manifest, indent=2) + "\n")
    print("verified manifest generated", flush=True)


if __name__ == "__main__":
    main()
