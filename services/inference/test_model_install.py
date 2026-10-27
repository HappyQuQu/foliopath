import hashlib
import http.server
import json
from pathlib import Path
import tempfile
import threading
import unittest
import urllib.request

from model_install import (Cancelled, InvalidDownload, Preparation, destination,
                           download_opener, prepare, read_catalog, validate_url)


class InstallTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        self.payload = b'model-bytes' * 4096
        self.served = self.payload
        self.requests = 0
        owner = self
        class Handler(http.server.BaseHTTPRequestHandler):
            def do_GET(self):
                owner.requests += 1
                self.send_response(200)
                self.send_header('Content-Length', str(len(owner.served)))
                self.end_headers()
                self.wfile.write(owner.served)
            def log_message(self, *args): pass
        self.server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), Handler)
        self.thread = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.thread.start()
        self.entry = dict(path='family/model.onnx', root='cache', size=len(self.payload),
                          sha256=hashlib.sha256(self.payload).hexdigest(),
                          url=f'http://127.0.0.1:{self.server.server_port}/model')
        self.opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
        self.stop = threading.Event()
        self.status = Preparation()

    def tearDown(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()
        self.temp.cleanup()

    def prepare(self):
        prepare([self.entry], {'cache': self.root}, self.opener, self.stop, self.status)

    def test_download_restart_and_corrupt_cache_recovery(self):
        self.prepare()
        self.assertEqual(self.status.snapshot()['state'], 'loading')
        target = self.root / self.entry['path']
        self.assertEqual(target.read_bytes(), self.payload)
        self.prepare()
        self.assertEqual(self.requests, 1)
        target.write_bytes(b'x' * len(self.payload))
        self.prepare()
        self.assertEqual(self.requests, 2)
        self.assertEqual(target.read_bytes(), self.payload)

    def test_same_size_corrupt_download_never_replaces_cache(self):
        target = destination(self.root, self.entry['path'])
        target.write_bytes(b'previous-cache')
        self.served = b'x' * len(self.payload)
        with self.assertRaises(InvalidDownload): self.prepare()
        self.assertEqual(target.read_bytes(), b'previous-cache')
        self.assertEqual(list(target.parent.glob('.download-*')), [])
        self.served = self.payload
        self.prepare()
        self.assertEqual(target.read_bytes(), self.payload)

    def test_truncated_and_oversized_downloads(self):
        for payload in (b'truncated', self.payload + b'extra'):
            self.served = payload
            with self.assertRaises(InvalidDownload): self.prepare()
            self.assertFalse((self.root / self.entry['path']).exists())

    def test_cancelled_install_preserves_previous_and_retries(self):
        target = destination(self.root, self.entry['path'])
        target.write_bytes(b'previous')
        class CancelOnProgress(Preparation):
            def update(inner, **values):
                super().update(**values)
                if values.get('completedBytes', 0) > 0: self.stop.set()
        self.status = CancelOnProgress()
        with self.assertRaises(Cancelled): self.prepare()
        self.assertEqual(target.read_bytes(), b'previous')
        self.assertEqual(list(target.parent.glob('.download-*')), [])
        self.stop.clear()
        self.status = Preparation()
        self.prepare()
        self.assertEqual(target.read_bytes(), self.payload)

    def test_symlink_cache_is_rejected_without_writes(self):
        outside = self.root / 'outside'
        outside.mkdir()
        (self.root / 'family').symlink_to(outside, target_is_directory=True)
        with self.assertRaises(InvalidDownload): self.prepare()
        self.assertEqual(list(outside.iterdir()), [])

    def test_image_catalog_is_fixed_bounded_and_public_https(self):
        entries = read_catalog()
        self.assertEqual(len(entries), 186)
        self.assertLess(sum(e['size'] for e in entries), 4 * 2**30)
        for entry in entries: validate_url(entry['url'])
        self.assertFalse(any('.cache/' in e['path'] for e in entries))
        self.assertFalse(any('ViT-B-32' in e['path'] for e in entries))
        for url in ['http://huggingface.co/a', 'https://127.0.0.1/a', 'https://huggingface.co.evil/a',
                    'https://huggingface.co:444/a', 'https://user:secret@huggingface.co/a']:
            with self.assertRaises(InvalidDownload): validate_url(url)
        with self.assertRaises(InvalidDownload): download_opener('http://user:secret@proxy:7890')


if __name__ == '__main__': unittest.main()
