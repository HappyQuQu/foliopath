"""One serial installer for the image-owned catalog, before inference admission."""
import hashlib
import http.client
import ipaddress
import json
import logging
import os
from pathlib import Path
import socket
import ssl
import stat
import threading
import time
import urllib.error
import urllib.parse
import urllib.request

HERE = Path(__file__).parent
ALLOWED_HOSTS = ('huggingface.co', 'hf.co', 'modelscope.cn')
logger = logging.getLogger('foliopath.models')


class Cancelled(Exception): pass
class InvalidDownload(Exception): pass


def failure_reason(error):
    if isinstance(error, urllib.error.HTTPError):
        return f'HTTP {error.code}'
    if isinstance(error, urllib.error.URLError):
        return failure_reason(error.reason)
    if isinstance(error, TimeoutError):
        return 'connection/read timeout'
    if isinstance(error, socket.gaierror):
        return 'DNS resolution failed'
    if isinstance(error, ssl.SSLError):
        return 'TLS connection failed'
    if isinstance(error, ConnectionError):
        return 'connection failed'
    if isinstance(error, InvalidDownload):
        return str(error)
    if isinstance(error, OSError):
        return f'{type(error).__name__} (errno={error.errno})'
    return type(error).__name__


def validate_url(url):
    parsed = urllib.parse.urlsplit(url)
    host = parsed.hostname or ''
    if (parsed.scheme != 'https' or parsed.username or parsed.password or parsed.fragment or
            parsed.port not in (None, 443) or not any(host == h or host.endswith('.' + h) for h in ALLOWED_HOSTS)):
        raise InvalidDownload('download origin rejected')
    return parsed


class Redirects(urllib.request.HTTPRedirectHandler):
    max_redirections = 5
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        validate_url(newurl)
        return super().redirect_request(req, fp, code, msg, headers, newurl)


class PublicHTTPS(http.client.HTTPSConnection):
    def connect(self):
        addresses = socket.getaddrinfo(self.host, self.port, type=socket.SOCK_STREAM)
        if not addresses or any(not ipaddress.ip_address(a[4][0]).is_global for a in addresses):
            raise InvalidDownload('download address rejected')
        address = addresses[0][4]
        connection = socket.create_connection(address[:2], self.timeout, self.source_address)
        try:
            self.sock = self._context.wrap_socket(connection, server_hostname=self.host)
        except BaseException:
            connection.close()
            raise


class HTTPS(urllib.request.HTTPSHandler):
    def https_open(self, req):
        return self.do_open(PublicHTTPS, req, context=ssl.create_default_context())


def download_opener(proxy):
    if proxy:
        value = urllib.parse.urlsplit(proxy)
        if (value.scheme != 'http' or not value.hostname or value.username or value.password or
                value.path not in ('', '/') or value.query or value.fragment):
            raise InvalidDownload('invalid explicit proxy')
        # The explicitly trusted proxy owns destination DNS/address enforcement.
        return urllib.request.build_opener(urllib.request.ProxyHandler({'https': proxy}), Redirects())
    return urllib.request.build_opener(urllib.request.ProxyHandler({}), HTTPS(), Redirects())


def read_catalog():
    entries = json.loads((HERE / 'managed-catalog.generated.json').read_text())
    if not isinstance(entries, list) or not 1 <= len(entries) <= 512:
        raise InvalidDownload('invalid catalog')
    seen = set()
    for entry in entries:
        path = entry['path']
        if (entry['root'] not in ('cache', 'ocr') or not isinstance(path, str) or '\0' in path or
                any(p in ('', '.', '..') for p in path.split('/')) or (entry['root'], path) in seen or
                type(entry['size']) is not int or not 0 < entry['size'] <= 2**30 or
                len(entry['sha256']) != 64 or any(c not in '0123456789abcdef' for c in entry['sha256'])):
            raise InvalidDownload('invalid catalog entry')
        validate_url(entry['url'])
        seen.add((entry['root'], path))
    return entries


def destination(root, relative):
    root = Path(root)
    root.mkdir(parents=True, exist_ok=True)
    parent = root
    if not stat.S_ISDIR(parent.lstat().st_mode):
        raise InvalidDownload('invalid cache directory')
    for part in relative.split('/')[:-1]:
        parent /= part
        parent.mkdir(exist_ok=True)
        if not stat.S_ISDIR(parent.lstat().st_mode):
            raise InvalidDownload('invalid cache directory')
    target = parent / Path(relative).name
    if target.is_symlink():
        raise InvalidDownload('invalid cache file')
    return target


def verified(path, entry, stop):
    try:
        fd = os.open(path, os.O_RDONLY | os.O_NOFOLLOW | os.O_NONBLOCK)
    except FileNotFoundError:
        return False
    with os.fdopen(fd, 'rb') as stream:
        info = os.fstat(stream.fileno())
        if not stat.S_ISREG(info.st_mode) or info.st_size != entry['size']:
            return False
        digest = hashlib.sha256()
        for chunk in iter(lambda: stream.read(1024 * 1024), b''):
            if stop.is_set(): raise Cancelled()
            digest.update(chunk)
        return digest.hexdigest() == entry['sha256']


def install(entry, target, opener, stop, progress):
    name = target.with_name('.download-' + target.name)
    descriptor = os.open(name, os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o600)
    try:
        digest, count = hashlib.sha256(), 0
        deadline = time.monotonic() + 900
        with os.fdopen(descriptor, 'wb') as output:
            with opener.open(entry['url'], timeout=15) as response:
                length = response.headers.get('Content-Length')
                if length is not None and int(length) != entry['size']:
                    raise InvalidDownload('download size mismatch')
                while True:
                    if stop.is_set(): raise Cancelled()
                    if time.monotonic() >= deadline: raise TimeoutError()
                    chunk = response.read(1024 * 1024)
                    if not chunk: break
                    count += len(chunk)
                    if count > entry['size']: raise InvalidDownload('download size exceeded')
                    digest.update(chunk)
                    output.write(chunk)
                    progress(count)
            if count != entry['size'] or digest.hexdigest() != entry['sha256']:
                raise InvalidDownload('download checksum mismatch')
            output.flush()
            os.fsync(output.fileno())
        os.replace(name, target)
    finally:
        Path(name).unlink(missing_ok=True)


class Preparation:
    def __init__(self):
        self.lock = threading.Lock()
        self.value = dict(state='preparing', completedFiles=0, totalFiles=0, completedBytes=0, totalBytes=0)

    def update(self, **values):
        with self.lock:
            self.value.update(values)

    def snapshot(self):
        with self.lock:
            return dict(self.value)


def prepare(entries, roots, opener, stop, status):
    total, completed = sum(e['size'] for e in entries), 0
    reused = 0
    logger.info('Checking model cache: files=%d total_bytes=%d', len(entries), total)
    status.update(state='preparing', totalFiles=len(entries), totalBytes=total, completedFiles=0, completedBytes=0)
    for index, entry in enumerate(entries):
        if stop.is_set(): raise Cancelled()
        try:
            target = destination(roots[entry['root']], entry['path'])
            if verified(target, entry, stop):
                reused += 1
            else:
                status.update(state='downloading')
                host = urllib.parse.urlsplit(entry['url']).hostname
                logger.info('Downloading model file %d/%d: file=%s source=%s bytes=%d',
                            index + 1, len(entries), entry['path'], host, entry['size'])
                started = last_report = time.monotonic()

                def progress(count):
                    nonlocal last_report
                    status.update(completedBytes=completed + count)
                    now = time.monotonic()
                    if now - last_report >= 15:
                        logger.info('Model download progress: file=%s bytes=%d/%d total_bytes=%d/%d elapsed_seconds=%.1f',
                                    entry['path'], count, entry['size'], completed + count, total, now - started)
                        last_report = now

                install(entry, target, opener, stop, progress)
                logger.info('Model file verified: file=%s bytes=%d elapsed_seconds=%.1f',
                            entry['path'], entry['size'], time.monotonic() - started)
        except Cancelled:
            logger.info('Model preparation cancelled: file=%s', entry['path'])
            raise
        except Exception as error:
            logger.error('Model preparation failed: file=%s source=%s reason=%s',
                         entry['path'], urllib.parse.urlsplit(entry['url']).hostname, failure_reason(error))
            raise
        completed += entry['size']
        status.update(completedFiles=index + 1, completedBytes=completed)
    status.update(state='loading')
    logger.info('Model cache verified: files=%d reused_files=%d total_bytes=%d; loading inference runtime',
                len(entries), reused, total)
