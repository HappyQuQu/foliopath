"""Pinned-byte cache snapshot; no original-media or database access."""
import hashlib
import json
import os
from pathlib import Path
import shutil
import stat
import tempfile


class InvalidCache(Exception):
    pass


def canonical_manifest(entries):
    if not isinstance(entries, list) or not 1 <= len(entries) <= 1000:
        raise InvalidCache()
    seen = set()
    total = 0
    for entry in entries:
        if not isinstance(entry, dict) or set(entry) != {"path", "size", "sha256"}:
            raise InvalidCache()
        path, size, digest = entry["path"], entry["size"], entry["sha256"]
        if not isinstance(path, str) or len(path) > 1024 or "\x00" in path:
            raise InvalidCache()
        parts = path.split("/")
        if len(parts) > 16 or any(p in ("", ".", "..") for p in parts) or path in seen:
            raise InvalidCache()
        if type(size) is not int or not 0 <= size <= 2**30:
            raise InvalidCache()
        if not isinstance(digest, str) or len(digest) != 64 or any(c not in "0123456789abcdef" for c in digest):
            raise InvalidCache()
        seen.add(path)
        total += size
        if total > 2**31:
            raise InvalidCache()
    return json.dumps(sorted(entries, key=lambda e: e["path"]), sort_keys=True, separators=(",", ":")).encode()


def open_pinned(root_fd, path):
    directory = os.dup(root_fd)
    try:
        parts = path.split("/")
        for part in parts[:-1]:
            child = os.open(part, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW, dir_fd=directory)
            os.close(directory)
            directory = child
        return os.open(parts[-1], os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK, dir_fd=directory)
    finally:
        os.close(directory)


def snapshot(source, parent, entries):
    canonical_manifest(entries)
    target = Path(tempfile.mkdtemp(prefix="verified-", dir=parent))
    root = None
    try:
        root = os.open(source, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
        for entry in entries:
            fd = open_pinned(root, entry["path"])
            with os.fdopen(fd, "rb") as stream:
                before = os.fstat(stream.fileno())
                if not stat.S_ISREG(before.st_mode) or before.st_size != entry["size"]:
                    raise InvalidCache()
                output = target / entry["path"]
                output.parent.mkdir(parents=True, exist_ok=True)
                digest = hashlib.sha256()
                count = 0
                with output.open("xb") as destination:
                    while chunk := stream.read(1024 * 1024):
                        count += len(chunk)
                        if count > entry["size"]:
                            raise InvalidCache()
                        digest.update(chunk)
                        destination.write(chunk)
                after = os.fstat(stream.fileno())
                if count != entry["size"] or digest.hexdigest() != entry["sha256"] or (
                    before.st_mtime_ns, before.st_ctime_ns, before.st_size
                ) != (after.st_mtime_ns, after.st_ctime_ns, after.st_size):
                    raise InvalidCache()
                output.chmod(0o444)
        return target
    except Exception:
        shutil.rmtree(target)
        raise InvalidCache() from None
    finally:
        if root is not None:
            os.close(root)
