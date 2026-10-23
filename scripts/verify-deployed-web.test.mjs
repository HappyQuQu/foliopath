import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { createServer } from 'node:http';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';
import { verifyDeployedWeb } from './verify-deployed-web.mjs';

async function fixture(t, stale = false) {
  const root = await mkdtemp(join(tmpdir(), 'foliopath-deployed-web-test-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  await mkdir(join(root, 'assets'));
  const index = '<script src="/assets/app.js"></script>';
  const script = 'console.log("fresh AI frontend")';
  await writeFile(join(root, 'index.html'), index);
  await writeFile(join(root, 'assets/app.js'), script);
  await writeFile(join(root, '.gitkeep'), '');
  const server = createServer((request, response) => {
    if (request.url === '/index.html') { response.writeHead(301, { Location: '/' }).end(); return; }
    if (request.url === '/missing') { response.writeHead(404).end(); return; }
    response.end(request.url === '/assets/app.js' ? (stale ? 'old frontend' : script) : index);
  });
  await new Promise((done) => server.listen(0, '127.0.0.1', done));
  t.after(() => new Promise((done) => { server.close(done); server.closeAllConnections(); }));
  return { root, base: `http://127.0.0.1:${server.address().port}` };
}

test('verifies exact built files and SPA routes, ignoring the embed placeholder', async (t) => {
  const { root, base } = await fixture(t);
  const checked = await verifyDeployedWeb(base, root, ['/settings/ai', '/intelligence']);
  assert.equal(checked.length, 4);
  assert.ok(checked.some((entry) => entry.path === '/'));
  assert.ok(checked.every((entry) => /^[a-f0-9]{64}$/.test(entry.sha256)));
});

test('rejects a healthy server serving stale JavaScript', async (t) => {
  const { root, base } = await fixture(t, true);
  await assert.rejects(verifyDeployedWeb(base, root), /deployed bytes differ/);
});

test('rejects missing routes and non-origin targets', async (t) => {
  const { root, base } = await fixture(t);
  await assert.rejects(verifyDeployedWeb(base, root, ['/missing']), /HTTP 404/);
  await assert.rejects(verifyDeployedWeb(`${base}/somewhere`, root), /Expected an HTTP/);
  await assert.rejects(verifyDeployedWeb(base, root, ['//example.com']), /application route/);
});
