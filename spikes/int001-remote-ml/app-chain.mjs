// Local candidate only: compose.app.yaml mounts synthetic fixtures, never real media.
import assert from 'node:assert/strict';
const origin = 'http://127.0.0.1:6091';
const password = process.env.FOLIOPATH_TEST_PASSWORD;
assert(password, 'Set FOLIOPATH_TEST_PASSWORD for the isolated candidate administrator');
let cookie = '', csrf = '';
const runKey = crypto.randomUUID();
async function request(path, method = 'GET', body, headers = {}) {
  const response = await fetch(origin + '/api/v1' + path, {
    method, signal: AbortSignal.timeout(40000),
    headers: { Origin: origin, Cookie: cookie, 'X-CSRF-Token': csrf,
      ...(body === undefined ? {} : { 'Content-Type': 'application/json' }), ...headers },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const value = await response.json();
  assert(response.ok, `${method} ${path}: ${response.status} ${JSON.stringify(value)}`);
  return { value, response };
}
const login = await request('/auth/login', 'POST', { username: 'candidate-test', password });
cookie = login.response.headers.get('set-cookie').split(';')[0];
csrf = login.value.csrfToken;
const { value: libraries } = await request('/libraries');
let library = libraries.items.find(item => item.name === 'Synthetic candidate chain');
if (!library) {
  const created = await request('/libraries', 'POST', { name: 'Synthetic candidate chain', rootPath: '' },
    { 'Idempotency-Key': 'candidate-synthetic-library-v1' });
  library = created.value.library;
}
assert(library?.id, 'library ID missing');
const base = `/libraries/${library.id}`;
async function until(label, action, ready) {
  for (let i = 0; i < 60; i++) {
    const value = await action();
    if (ready(value)) return value;
    await new Promise(resolve => setTimeout(resolve, 2000));
  }
  throw new Error(`${label} timed out`);
}
console.log('library', library.id);
const assets = await until('scan', async () => (await request(base + '/assets')).value,
  value => value.items.length >= 2);
console.log('scanned assets', assets.items.length);
await until('model activation', async () => (await request('/ai/models')).value,
  value => value.items.some(model => model.active && model.state === 'available'));
const settings = await request(base + '/ai/semantic');
if (!settings.value.enabled) await request(base + '/ai/semantic', 'PATCH', { enabled: true },
  { 'If-Match': settings.response.headers.get('etag') });
const job = await request(base + '/ai/semantic/jobs', 'POST', { mode: 'missing' },
  { 'Idempotency-Key': 'candidate-synthetic-backfill-' + runKey });
console.log('job', job.value);
const completed = await until('backfill', async () => (await request(`/ai/operations/${job.value.id}`)).value,
  value => ['succeeded', 'failed', 'cancelled'].includes(value.state));
assert.equal(completed.state, 'succeeded', JSON.stringify(completed));
const coverage = (await request(base + '/ai/semantic')).value;
assert.equal(coverage.coverage.complete, true, JSON.stringify(coverage));
assert(coverage.coverage.completed >= 1);
console.log('coverage', coverage);
const results = (await request(`/semantic/assets?libraryId=${library.id}&q=blue&limit=10`)).value;
assert(results.items.length >= 1, JSON.stringify(results));
console.log('search', JSON.stringify(results));
console.log('PASS candidate main-process image chain');
await request(base + '/media-processing/repair?mode=missing', 'POST');
await until('storyboard', async () => (await request(base + '/assets')).value,
  value => value.items.some(asset => asset.kind === 'video' && asset.storyboard.status === 'ready'));
const videoJob = await request(base + '/ai/video-semantic/jobs', 'POST', { mode: 'missing' },
  { 'Idempotency-Key': 'candidate-synthetic-video-' + runKey });
const videoCompleted = await until('video backfill', async () => (await request(`/ai/operations/${videoJob.value.id}`)).value,
  value => ['succeeded', 'failed', 'cancelled'].includes(value.state));
assert.equal(videoCompleted.state, 'succeeded', JSON.stringify(videoCompleted));
const videos = (await request(`/semantic/videos?libraryId=${library.id}&q=blue&limit=10`)).value;
assert(videos.items.length >= 1, JSON.stringify(videos));
assert(videos.coverage.completed >= 1, JSON.stringify(videos));
// The retained one-second Range fixture intentionally has no storyboard.
assert.equal(videos.coverage.degraded, 1, JSON.stringify(videos));
assert.equal(videos.coverage.complete, false, JSON.stringify(videos));
assert([4, 10].includes(videos.items[0].matchedFrame.planSize));
console.log('video search', JSON.stringify(videos));
console.log('PASS candidate main-process image/video chain');
const videoStatus = (await request(base + '/ai/video-semantic')).value;
assert.equal(videoStatus.libraryId, library.id);
assert.equal(videoStatus.enabled, true);
assert.equal(videoStatus.offline, false);
assert(videoStatus.activeGenerationId);
for (const field of ['eligible', 'completed', 'degraded', 'failed', 'stale', 'complete']) {
  assert.equal(videoStatus.coverage[field], videos.coverage[field], `video status mismatch: ${field}`);
}
console.log('video status', JSON.stringify(videoStatus));
console.log('PASS candidate video status matches real search coverage');
if (process.env.FOLIOPATH_TEST_VIDEO_REMOVAL === '1') {
  const { lstat, mkdtemp, rename } = await import('node:fs/promises');
  const root = process.env.FOLIOPATH_TEST_SYNTHETIC_MEDIA;
  assert(/^\/tmp\/foliopath-app-fixtures\.[A-Za-z0-9]+$/.test(root ?? ''));
  assert(!(await lstat(root)).isSymbolicLink());
  const source = root + '/storyboard-test.mp4';
  assert((await lstat(source)).isFile() && !(await lstat(source)).isSymbolicLink());
  const holding = await mkdtemp('/tmp/foliopath-held-video.');
  const held = holding + '/storyboard-test.mp4';
  await rename(source, held);
  try {
    await request(base + '/scans', 'POST');
    await until('video removed', async () => (await request(base + '/assets')).value,
      value => !value.items.some(asset => asset.name === 'storyboard-test.mp4'));
    const pending = (await request(`/semantic/videos?libraryId=${library.id}&q=blue&limit=10`)).value;
    assert.equal(pending.items.length, 0);
    assert.equal(pending.coverage.eligible, 1);
    assert.equal(pending.coverage.completed, 0);
    assert.equal(pending.coverage.degraded, 0);
    assert.equal(pending.coverage.failed, 0);
    assert.equal(pending.coverage.complete, false);
    const recheck = await request(base + '/ai/video-semantic/jobs', 'POST', { mode: 'missing' },
      { 'Idempotency-Key': 'candidate-video-removed-' + runKey });
    const terminal = await until('remaining video reclassification', async () => (await request(`/ai/operations/${recheck.value.id}`)).value,
      value => ['succeeded', 'failed', 'cancelled'].includes(value.state));
    assert.equal(terminal.state, 'succeeded');
    const classified = (await request(`/semantic/videos?libraryId=${library.id}&q=blue&limit=10`)).value;
    assert.equal(classified.coverage.degraded, 1);
    assert.equal(classified.coverage.completed, 0);
    assert.equal(classified.coverage.complete, false);
    const historical = (await request(`/ai/operations/${videoJob.value.id}`)).value;
    assert.equal(historical.state, videoCompleted.state);
    assert.equal(historical.revision, videoCompleted.revision);
    console.log('PASS removed video -> pending -> degraded reclassification; history unchanged');
  } finally {
    await rename(held, source);
    await request(base + '/scans', 'POST');
  }
  console.log('Synthetic video restored; rescan requested');
}
if (process.env.FOLIOPATH_TEST_INVALIDATION === '1') {
  const { lstat, utimes } = await import('node:fs/promises');
  const root = process.env.FOLIOPATH_TEST_SYNTHETIC_MEDIA;
  assert(/^\/tmp\/foliopath-app-fixtures\.[A-Za-z0-9]+$/.test(root ?? ''), 'Only generated temporary fixtures may be mutated');
  assert(!(await lstat(root)).isSymbolicLink());
  const path = root + '/viewer-blue-violet.png';
  const before = await lstat(path);
  assert(before.isFile() && !before.isSymbolicLink());
  await utimes(path, before.atime, new Date(before.mtimeMs + 1000));
  await request(base + '/scans', 'POST');
  const stale = await until('stale coverage', async () => (await request(base + '/ai/semantic')).value,
    value => value.coverage.stale === 1);
  assert.equal(stale.coverage.completed, 0);
  assert.equal(stale.coverage.complete, false);
  assert.equal(stale.state, 'degraded');
  const excluded = (await request(`/semantic/assets?libraryId=${library.id}&q=blue&limit=10`)).value;
  assert.equal(excluded.items.length, 0);
  assert.equal(excluded.coverage.stale, 1);
  assert.equal(excluded.coverage.complete, false);
  const refreshed = await request(base + '/ai/semantic/jobs', 'POST', { mode: 'missing' },
    { 'Idempotency-Key': 'candidate-source-change-' + crypto.randomUUID() });
  const terminal = await until('changed source backfill', async () => (await request(`/ai/operations/${refreshed.value.id}`)).value,
    value => ['succeeded', 'failed', 'cancelled'].includes(value.state));
  assert.equal(terminal.state, 'succeeded');
  const restored = (await request(base + '/ai/semantic')).value;
  assert.equal(restored.coverage.complete, true);
  assert.equal(restored.coverage.stale, 0);
  assert.equal(restored.activeGenerationId, coverage.activeGenerationId);
  assert.equal((await request(`/semantic/assets?libraryId=${library.id}&q=blue&limit=10`)).value.items.length, 1);
  console.log('PASS scanned source change: complete -> stale -> rebuilt, same generation');
}
