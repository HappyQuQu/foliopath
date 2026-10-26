"""Preparation faults use tiny synthetic bytes; never download real models."""
import hashlib
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch
import urllib.request

import fetch_models as f


class FetchTests(unittest.TestCase):
    def run_fetch(self, root, data=b"synthetic", **changes):
        entry = {"path": "ocr/test/model.onnx", "url": "https://www.modelscope.cn/fixed",
                 "sha256": hashlib.sha256(b"synthetic").hexdigest(), "maxBytes": 16, "size": 9}
        entry.update(changes)
        manifest = root / "manifest.json"
        manifest.write_text(json.dumps({"files": [entry]}))
        opener = Mock()
        opener.open.return_value = io.BytesIO(data)
        with patch.object(f, "MANIFEST", manifest), patch.object(f.urllib.request, "build_opener", return_value=opener) as build:
            result = f.fetch(root / "cache")
            self.assertEqual(build.call_args.args[0].proxies, {})
        return result, opener

    def test_pinned_atomic_publish_and_existing_verification_without_network(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            result, _ = self.run_fetch(root)
            target = root / "cache/ocr/test/model.onnx"
            self.assertEqual(target.read_bytes(), b"synthetic")
            self.assertEqual(target.stat().st_mode & 0o777, 0o444)
            again, opener = self.run_fetch(root)
            self.assertEqual(result, again)
            opener.open.assert_not_called()

    def test_bad_hash_size_and_overflow_never_publish_or_leave_partial(self):
        for data in (b"incorrect", b"short", b"x" * 17, b""):
            with self.subTest(data=data), tempfile.TemporaryDirectory() as folder:
                root = Path(folder)
                with self.assertRaises(ValueError): self.run_fetch(root, data)
                self.assertEqual(list((root / "cache/ocr/test").iterdir()), [])

    def test_existing_tamper_is_preserved_and_rejected(self):
        with tempfile.TemporaryDirectory() as folder:
            root = Path(folder)
            target = root / "cache/ocr/test/model.onnx"
            target.parent.mkdir(parents=True)
            target.write_bytes(b"incorrect")
            with self.assertRaises(ValueError): self.run_fetch(root)
            self.assertEqual(target.read_bytes(), b"incorrect")

    def test_redirects_and_proxy_credentials_fail_closed(self):
        redirect = f.FixedRedirect()
        request = urllib.request.Request("https://www.modelscope.cn/fixed")
        for url in ("http://www.modelscope.cn/file", "https://example.com/file", "https://user@modelscope.cn/file"):
            with self.subTest(url=url), self.assertRaises(ValueError):
                redirect.redirect_request(request, None, 302, "", {}, url)
        for proxy in ("http://user:secret@localhost:7890", "http://localhost/path", "ftp://localhost"):
            with self.subTest(proxy=proxy), self.assertRaises(ValueError): f.fetch("unused", proxy)


if __name__ == "__main__": unittest.main()
