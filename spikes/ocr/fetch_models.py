"""Explicit preparation only; no application import or automatic runtime downloads."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import tempfile
import urllib.parse
import urllib.request

MANIFEST = Path(__file__).with_name("model-manifest.json")

class FixedRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        parsed = urllib.parse.urlsplit(newurl)
        if parsed.scheme != "https" or parsed.hostname not in ("www.modelscope.cn", "modelscope.cn", "cdn-lfs-cn-1.modelscope.cn") or parsed.username or parsed.password:
            raise ValueError(f"Unapproved model redirect host: {parsed.hostname}")
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def fetch(destination, proxy=None):
    if proxy:
        parsed = urllib.parse.urlsplit(proxy)
        if parsed.scheme not in ("http", "https") or not parsed.hostname or parsed.username or parsed.password or parsed.path not in ("", "/") or parsed.query or parsed.fragment:
            raise ValueError("Proxy must be an explicit origin without credentials")
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({"https": proxy} if proxy else {}), FixedRedirect())
    manifest = json.loads(MANIFEST.read_text())
    receipts = []
    destination = Path(destination)
    for entry in manifest["files"]:
        target = destination / entry["path"]
        target.parent.mkdir(parents=True, exist_ok=True)
        if target.exists():
            if target.is_symlink() or not target.is_file() or target.stat().st_size > entry["maxBytes"]:
                raise ValueError("Existing model is not the pinned artifact")
            with target.open("rb") as existing:
                if hashlib.file_digest(existing, "sha256").hexdigest() != entry["sha256"] or ("size" in entry and target.stat().st_size != entry["size"]):
                    raise ValueError("Existing model is not the pinned artifact")
        else:
            fd, partial = tempfile.mkstemp(prefix=".ocr-download-", dir=target.parent)
            try:
                digest = hashlib.sha256()
                size = 0
                with os.fdopen(fd, "wb") as output, opener.open(entry["url"], timeout=60) as response:
                    while chunk := response.read(1024 * 1024):
                        size += len(chunk)
                        if size > entry["maxBytes"]:
                            raise ValueError("Model exceeds preparation budget")
                        digest.update(chunk)
                        output.write(chunk)
                if not size or digest.hexdigest() != entry["sha256"] or ("size" in entry and size != entry["size"]):
                    raise ValueError("Model hash or size mismatch")
                os.chmod(partial, 0o444)
                # Preparation never overwrites an existing artifact.
                os.link(partial, target)
            finally:
                os.unlink(partial)
        receipts.append({"path": entry["path"], "size": target.stat().st_size, "sha256": entry["sha256"]})
    return receipts

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("destination")
    parser.add_argument("--proxy")
    args = parser.parse_args()
    print(json.dumps(fetch(args.destination, args.proxy), sort_keys=True))
