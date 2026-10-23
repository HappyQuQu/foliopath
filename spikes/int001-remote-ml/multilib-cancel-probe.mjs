// Fresh local foliopath-multilib compose project only. Synthetic alpha/beta videos.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
const origin = 'http://127.0.0.1:6091';
const container = 'foliopath-multilib-machine-learning-1';
const password = process.env.FOLIOPATH_TEST_PASSWORD;
assert(password, 'Set isolated test administrator password');
const state = JSON.parse(execFileSync('docker', ['inspect', 'foliopath-multilib-app-local-1'], { encoding: 'utf8' }))[0];
assert.equal(state.Config.Labels['com.docker.compose.project'], 'foliopath-multilib');
assert(state.Mounts.some(m => m.Destination === '/library' && !m.RW && m.Source.startsWith('/tmp/foliopath-multilib.')));
let cookie = '', csrf = '';
async function request(path, method = 'GET', body, extra = {}) {
  const response = await fetch(origin + '/api/v1' + path, {
    method, signal: AbortSignal.timeout(20000),
    headers: { Origin: origin, Cookie: cookie, 'X-CSRF-Token': csrf,
      ...(body === undefined ? {} : { 'Content-Type': 'application/json' }), ...extra },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const value = await response.json();
  assert(response.ok, `${method} ${path}: ${response.status} ${JSON.stringify(value)}`);
  return { value, response };
}
async function until(label, read, predicate) {
  for (let i = 0; i < 90; i++) {
    const result = await read();
    if (predicate(result)) return result;
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
  throw new Error(label + ' timed out');
}
const resume = process.env.FOLIOPATH_TEST_RESUME === '1';
const auth = await request(resume ? '/auth/login' : '/auth/setup', 'POST', { username: 'multilib-test', password, ...(resume ? {} : { displayName: 'Multilib synthetic test' }) });
cookie = auth.response.headers.get('set-cookie').split(';')[0];
csrf = auth.value.csrfToken;
await until('model', async () => (await request('/ai/models')).value, v => v.items.some(m => m.active && m.state === 'available'));
const libraries = [];
for (const root of ['alpha', 'beta']) {
  const library = resume
    ? (await request('/libraries')).value.items.find(l => l.name === `Synthetic ${root}`)
    : (await request('/libraries', 'POST', { name: `Synthetic ${root}`, rootPath: root }, { 'Idempotency-Key': crypto.randomUUID() })).value.library;
  assert(library);
  const base = `/libraries/${library.id}`;
  const assets = await until('scan', async () => (await request(base + '/assets')).value, v => v.items.length === 1);
  assert.equal(assets.items[0].name, `${root}.mp4`);
  await request(base + '/media-processing/repair?mode=missing', 'POST');
  await until('storyboard', async () => (await request(base + '/assets')).value, v => v.items[0].storyboard.status === 'ready');
  const settings = await request(base + '/ai/semantic');
  if (!settings.value.enabled) await request(base + '/ai/semantic', 'PATCH', { enabled: true }, { 'If-Match': settings.response.headers.get('etag') });
  libraries.push({ ...library, base, assetID: assets.items[0].id });
}
const jobRead = async id => (await request(`/ai/operations/${id}`)).value;
const videoStatus = async lib => (await request(lib.base + '/ai/video-semantic')).value;
const start = async (lib, mode = 'all') => (await request(lib.base + '/ai/video-semantic/jobs', 'POST', { mode }, { 'Idempotency-Key': crypto.randomUUID() })).value;
let paused = false;
let jobs;
try {
  execFileSync('docker', ['pause', container]); paused = true;
  jobs = [await start(libraries[0])];
  await until('first running', () => jobRead(jobs[0].id), v => v.state === 'running');
  jobs.push(await start(libraries[1]));
  assert.equal((await jobRead(jobs[1].id)).state, 'queued');
  console.log('PASS two active library jobs: one running, one queued', jobs.map(j => j.id));
  const otherBefore = await jobRead(jobs[1].id);
  const first = await request(`/ai/operations/${jobs[0].id}`);
  await request(`/ai/operations/${jobs[0].id}/cancel`, 'POST', undefined, { 'If-Match': first.response.headers.get('etag') });
  const cancelled = await until('cancel without inference recovery', () => jobRead(jobs[0].id), v => v.state === 'cancelled');
  assert.equal(cancelled.completedItems, 0);
  const otherAfter = await jobRead(jobs[1].id);
  assert(['queued', 'running'].includes(otherAfter.state));
  assert.equal(otherAfter.completedItems, otherBefore.completedItems);
  assert.equal(otherAfter.errorCode, otherBefore.errorCode);
  console.log('PASS cancellation leaves other library active and unprocessed');
} finally {
  if (paused && execFileSync('docker', ['inspect', '--format', '{{.State.Paused}}', container], { encoding: 'utf8' }).trim() === 'true') {
    execFileSync('docker', ['unpause', container]);
  }
}
const second = await until('second library completion', () => jobRead(jobs[1].id), v => ['succeeded', 'failed', 'cancelled'].includes(v.state));
assert.equal(second.state, 'succeeded');
assert.equal((await videoStatus(libraries[1])).coverage.completed, 1);
assert.equal((await videoStatus(libraries[0])).coverage.completed, 0);
const retry = await start(libraries[0], 'missing');
assert.equal((await until('first retry', () => jobRead(retry.id), v => ['succeeded', 'failed', 'cancelled'].includes(v.state))).state, 'succeeded');
assert.equal((await jobRead(jobs[0].id)).state, 'cancelled');
assert.equal((await jobRead(jobs[1].id)).state, 'succeeded');
for (const lib of libraries) {
  const status = await videoStatus(lib);
  assert.equal(status.coverage.completed, 1);
  const results = (await request(`/semantic/videos?libraryId=${lib.id}&q=blue&limit=10`)).value;
  assert.equal(results.items.length, 1);
  assert.equal(results.items[0].asset.id, lib.assetID);
  console.log('PASS isolated completed coverage and search', lib.id, JSON.stringify(status));
}
console.log('PASS multilib queued scheduling, cancellation isolation, retry, scoped search');
