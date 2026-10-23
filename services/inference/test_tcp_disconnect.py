"""Real uvicorn TCP disconnect, not a mocked Request.is_disconnected signal."""
import asyncio
import io
import json
import os
from pathlib import Path
import socket
import sys
import threading
import time
import unittest
from unittest.mock import patch

from PIL import Image
import uvicorn

import server
from scheduler import InferenceScheduler
import tag_contract


class TCPDisconnectTests(unittest.TestCase):
    def test_disconnected_socket_reaps_running_child_and_allows_next_request(self):
        # No model/database/media source is mounted. A real native sleeping process
        # makes cancellation observable independent of fast model completion.
        owner = InferenceScheduler("/unused", "/unused", work_root="/tmp",
            command=(sys.executable, "-c", "import sys,time;sys.stdin.buffer.read();time.sleep(60)"))
        listener = socket.socket()
        listener.bind(("127.0.0.1", 0))
        port = listener.getsockname()[1]
        runtime = uvicorn.Server(uvicorn.Config(server.application, host="127.0.0.1", port=port,
            lifespan="off", log_level="error", access_log=False, timeout_graceful_shutdown=1))
        loops = []

        async def serve():
            loops.append(asyncio.get_running_loop())
            await runtime.serve(sockets=[listener])

        thread = threading.Thread(target=lambda: asyncio.run(serve()), daemon=True)
        image = io.BytesIO()
        Image.new("RGB", (32, 24), (20, 50, 80)).save(image, format="PNG")
        body = (b'--boundary\r\nContent-Disposition: form-data; name="image"; filename="test.png"\r\n'
                b'Content-Type: image/png\r\n\r\n' + image.getvalue() + b'\r\n--boundary--\r\n')
        wire = (f'POST /api/v1/tag/predict HTTP/1.1\r\nHost: 127.0.0.1\r\n'
                f'Content-Type: multipart/form-data; boundary=boundary\r\nContent-Length: {len(body)}\r\n'
                'Connection: close\r\n\r\n').encode() + body
        tag_identity = server.expected_tag_identity(tag_contract.MANIFEST)
        with patch.object(server, "scheduler", owner), patch.object(server, "tag_identity", tag_identity):
            try:
                thread.start()
                self.until(lambda: runtime.started)
                with socket.create_connection(("127.0.0.1", port), timeout=3) as client:
                    client.sendall(wire)
                    self.until(lambda: len(owner.processes) == 1)
                    pid = owner.last_pid
                    client.shutdown(socket.SHUT_RDWR)
                self.until(lambda: not owner.processes and owner.pending == 0, timeout=1)
                with self.assertRaises(ProcessLookupError):
                    os.kill(pid, 0)
                owner.command = (sys.executable, "-c", 'import sys;sys.stdin.buffer.read();print(\'{"labels":[]}\')')
                with socket.create_connection(("127.0.0.1", port), timeout=3) as client:
                    client.sendall(wire)
                    response = bytearray()
                    while chunk := client.recv(4096):
                        response.extend(chunk)
                self.assertIn(b"200 OK", response)
                self.assertIn(b'"labels":[]', response)
                self.assertEqual(owner.completed, 1)
                self.assertFalse(owner.processes)
            finally:
                if loops and thread.is_alive():
                    asyncio.run_coroutine_threadsafe(owner.close(), loops[0]).result(timeout=3)
                runtime.should_exit = True
                thread.join(timeout=3)
                listener.close()
        self.assertFalse(thread.is_alive())

    def until(self, predicate, timeout=3):
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if predicate():
                return
            time.sleep(0.005)
        self.fail("real TCP inference lifecycle did not reach expected state")
