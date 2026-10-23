#!/usr/bin/env python3
"""Fetch and verify a fixed Wikimedia Commons semantic pilot."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import urllib.parse
import urllib.request
from pathlib import Path

from PIL import Image


USER_AGENT = "FolioPath-INT001-feasibility/1.0 (local development; no redistribution)"
API_HOST = "commons.wikimedia.org"
FILE_HOST = "upload.wikimedia.org"
ALLOWED_LICENSE_URLS = {
    "https://creativecommons.org/licenses/by/2.0",
    "https://creativecommons.org/licenses/by-sa/2.0",
    "https://creativecommons.org/licenses/by-sa/3.0",
    "https://creativecommons.org/licenses/by-sa/4.0",
}
SHA1 = re.compile(r"^[0-9a-f]{40}$")
SHA256 = re.compile(r"^[0-9a-f]{64}$")
FILENAME = re.compile(r"^[0-9]+\.jpg$")


def digest(path: Path, algorithm: str) -> str:
    result = hashlib.new(algorithm)
    with path.open("rb") as source:
        for chunk in iter(lambda: source.read(8 * 1024 * 1024), b""):
            result.update(chunk)
    return result.hexdigest()


def request(url: str) -> urllib.response.addinfourl:
    return urllib.request.urlopen(urllib.request.Request(url, headers={"User-Agent": USER_AGENT}), timeout=60)


def validate_manifest(manifest: dict[str, object]) -> list[dict[str, object]]:
    if manifest.get("schema_version") != 1 or manifest.get("legal_basis") != "public-license":
        raise ValueError("fixture must be a schema v1 public-license manifest")
    items = manifest.get("items")
    if not isinstance(items, list) or not 1 <= len(items) <= 100:
        raise ValueError("fixture must contain 1..100 items")
    seen_ids: set[str] = set()
    seen_pages: set[int] = set()
    for item in items:
        if not isinstance(item, dict) or not isinstance(item.get("source"), dict):
            raise ValueError("every item requires source metadata")
        source = item["source"]
        item_id = item.get("id")
        page_id = source.get("page_id")
        filename = item.get("filename")
        if not isinstance(item_id, str) or not item_id or item_id in seen_ids:
            raise ValueError("item IDs must be non-empty and unique")
        if not isinstance(page_id, int) or page_id <= 0 or page_id in seen_pages:
            raise ValueError("page IDs must be positive and unique")
        if not isinstance(filename, str) or not FILENAME.fullmatch(filename):
            raise ValueError(f"item {item_id} has an unsafe filename")
        if source.get("license_url") not in ALLOWED_LICENSE_URLS:
            raise ValueError(f"item {item_id} has an unapproved license")
        if source.get("mime") != "image/jpeg" or not SHA1.fullmatch(str(source.get("original_sha1", ""))):
            raise ValueError(f"item {item_id} has invalid media metadata")
        if not SHA256.fullmatch(str(item.get("sha256", ""))) or int(source.get("bytes", 0)) <= 0:
            raise ValueError(f"item {item_id} has invalid size or SHA-256")
        parsed = urllib.parse.urlparse(str(source.get("url", "")))
        if parsed.scheme != "https" or parsed.hostname != FILE_HOST or parsed.query or parsed.fragment:
            raise ValueError(f"item {item_id} has an unapproved source URL")
        seen_ids.add(item_id)
        seen_pages.add(page_id)
    return items


def verify_commons_metadata(items: list[dict[str, object]], api_url: str) -> None:
    page_ids = "|".join(str(item["source"]["page_id"]) for item in items)
    query = urllib.parse.urlencode(
        {
            "action": "query",
            "format": "json",
            "formatversion": "2",
            "pageids": page_ids,
            "prop": "imageinfo",
            "iiprop": "url|size|mime|sha1|timestamp|extmetadata",
        }
    )
    parsed_api = urllib.parse.urlparse(api_url)
    if parsed_api.scheme != "https" or parsed_api.hostname != API_HOST:
        raise ValueError("source_api must be the Wikimedia Commons HTTPS API")
    with request(f"{api_url}?{query}") as response:
        payload = json.load(response)
    current = {page["pageid"]: page for page in payload.get("query", {}).get("pages", [])}
    for item in items:
        source = item["source"]
        page = current.get(source["page_id"])
        if page is None or page.get("title") != source["title"] or len(page.get("imageinfo", [])) != 1:
            raise ValueError(f"Commons page identity changed for {item['id']}")
        info = page["imageinfo"][0]
        license_url = info.get("extmetadata", {}).get("LicenseUrl", {}).get("value")
        actual_url = urllib.parse.urlsplit(info.get("url", ""))._replace(query="", fragment="").geturl()
        expected = {
            "timestamp": source["revision_timestamp"],
            "mime": source["mime"],
            "width": source["width"],
            "height": source["height"],
            "size": source["bytes"],
            "sha1": source["original_sha1"],
            "url": source["url"],
            "license_url": source["license_url"],
        }
        actual = {
            "timestamp": info.get("timestamp"),
            "mime": info.get("mime"),
            "width": info.get("width"),
            "height": info.get("height"),
            "size": info.get("size"),
            "sha1": info.get("sha1"),
            "url": actual_url,
            "license_url": license_url,
        }
        if actual != expected:
            raise ValueError(f"Commons metadata changed for {item['id']}: {actual!r}")


def fetch_or_verify(item: dict[str, object], output: Path) -> None:
    source = item["source"]
    target = output / item["filename"]
    if not target.exists():
        temporary = target.with_suffix(target.suffix + ".part")
        try:
            with request(source["url"]) as response, temporary.open("wb") as destination:
                final_host = urllib.parse.urlparse(response.geturl()).hostname
                if final_host != FILE_HOST:
                    raise ValueError(f"download escaped the approved host for {item['id']}")
                remaining = int(source["bytes"])
                while remaining:
                    chunk = response.read(min(1024 * 1024, remaining + 1))
                    if not chunk:
                        break
                    remaining -= len(chunk)
                    if remaining < 0:
                        raise ValueError(f"download exceeded declared size for {item['id']}")
                    destination.write(chunk)
            os.replace(temporary, target)
        finally:
            if temporary.exists():
                temporary.unlink()
    if target.stat().st_size != source["bytes"]:
        raise ValueError(f"size mismatch for {item['id']}")
    if digest(target, "sha1") != source["original_sha1"] or digest(target, "sha256") != item["sha256"]:
        raise ValueError(f"content digest mismatch for {item['id']}")
    with Image.open(target) as image:
        if image.format != "JPEG" or image.size != (source["width"], source["height"]):
            raise ValueError(f"decoded image metadata mismatch for {item['id']}")
        image.verify()


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--manifest", required=True, type=Path)
    parser.add_argument("--output", required=True, type=Path)
    args = parser.parse_args()
    manifest = json.loads(args.manifest.read_text(encoding="utf-8"))
    items = validate_manifest(manifest)
    verify_commons_metadata(items, str(manifest["source_api"]))
    args.output.mkdir(parents=True, exist_ok=True)
    for item in items:
        fetch_or_verify(item, args.output)
    print(
        json.dumps(
            {
                "schema_version": 1,
                "dataset_id": manifest["dataset_id"],
                "verified_items": len(items),
                "verified_bytes": sum(int(item["source"]["bytes"]) for item in items),
                "network_hosts": [API_HOST, FILE_HOST],
                "redistributed_in_git": False,
            },
            indent=2,
        )
    )


if __name__ == "__main__":
    main()
