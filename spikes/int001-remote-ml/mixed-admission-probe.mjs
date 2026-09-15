// Local synthetic foliopath-multilib only; requires alpha video and gamma image.
import assert from 'node:assert/strict';
import { execFileSync } from 'node:child_process';
const origin = 'http://127.0.0.1:6091';
const ai = 'foliopath-multilib-machine-learning-1';
const inspect = JSON.parse(execFileSync('docker', ['inspect', 'foliopath-multilib-app-local-1'], { encoding: 'utf8' }))[0];
assert.equal(inspect.Config.Labels['com.docker.compose.project'], 'foliopath-multilib');
// Docker Desktop may report the canonical macOS VM-shared path.
assert(inspect.Mounts.some(m => m.Destination === '/library' && !m.RW &&
  /^(?:\/host_mnt)?\/(?:private\/)?tmp\/foliopath-multilib\.[A-Za-z0-9]+$/.test(m.Source)));
assert(process.env.FOLIOPATH_TEST_PASSWORD);
let cookie = '', csrf = '';
async function request(path, method = 'GET', body, extra = {}) {
  const response = await fetch(origin + '/api/v1' + path, { method, signal: AbortSignal.timeout(20000),
    headers: { Origin: origin, Cookie: cookie, 'X-CSRF-Token': csrf, ...(body === undefined ? {} : { 'Content-Type': 'application/json' }), ...extra },
    body: body === undefined ? undefined : JSON.stringify(body) });
  const value = await response.json();
  assert(response.ok, `${method} ${path}: ${response.status} ${JSON.stringify(value)}`);
  return { value, response };
}
async function until(label, read, ready, attempts = 90) {
  for (let i = 0; i < attempts; i++) {
    const value = await read();
    if (ready(value)) return value;
    await new Promise(r => setTimeout(r, 500));
  }
  throw new Error(label + ' timed out');
}
const auth = await request('/auth/login', 'POST', { username: 'multilib-test', password: process.env.FOLIOPATH_TEST_PASSWORD });
cookie = auth.response.headers.get('set-cookie').split(';')[0]; csrf = auth.value.csrfToken;
await until('model', async () => (await request('/ai/models')).value, v => v.items.some(m => m.active && m.state === 'available'), 180);
const libraries = (await request('/libraries')).value.items;
const video = libraries.find(l => l.name === 'Synthetic alpha');
assert(video);
const image = libraries.find(l => l.name === 'Synthetic gamma') ?? (await request('/libraries', 'POST', { name: 'Synthetic gamma', rootPath: 'gamma' }, { 'Idempotency-Key': crypto.randomUUID() })).value.library;
const imageBase = `/libraries/${image.id}`;
const videoBase = `/libraries/${video.id}`;
await until('image scan', async () => (await request(imageBase + '/assets')).value, v => v.items.length === 1 && v.items[0].name === 'gamma.png');
const settings = await request(imageBase + '/ai/semantic');
if (!settings.value.enabled) await request(imageBase + '/ai/semantic', 'PATCH', { enabled: true }, { 'If-Match': settings.response.headers.get('etag') });
const original = await request('/settings');
const updateBudget = async n => { const current = await request('/settings'); return request('/settings', 'PATCH', { backgroundConcurrency: n }, { 'If-Match': current.response.headers.get('etag') }); };
const operation = async id => (await request('/ai/operations/' + id)).value;
const start = async (base, kind, mode) => (await request(base + `/ai/${kind}/jobs`, 'POST', { mode }, { 'Idempotency-Key': crypto.randomUUID() })).value;
const target = process.env.FOLIOPATH_TEST_CANCEL_TARGET ?? 'video';
assert(['image', 'video', 'clear', 'scan', 'tag-clear', 'tag'].includes(target));
const startTagClear = async () => {
  const snapshot = await request(imageBase + '/ai/tag-suggestions');
  return (await request(imageBase + '/ai/tag-suggestion-reviews/clear', 'POST',
    { confirmation: 'clear_tag_suggestion_reviews' },
    { 'If-Match': snapshot.response.headers.get('etag'), 'Idempotency-Key': crypto.randomUUID() })).value;
};
const waiting = target === 'tag' ? {base:imageBase,kind:'tag-suggestions'} : target !== 'video' ? { base: imageBase, kind: 'semantic' } : { base: videoBase, kind: 'video-semantic' };
const holding = target !== 'video' ? { base: videoBase, kind: 'video-semantic' } : { base: imageBase, kind: 'semantic' };
const waiterPath = id => (target === 'scan' ? '/scans/' : '/ai/operations/') + id;
const waiterState = value => target === 'scan' ? value.status : value.state;
const readWaiter = async id => (await request(waiterPath(id))).value;
let paused = false;
let originalVocabulary;
try {
  await updateBudget(1);
  const beforeAssets = (await request(imageBase + '/assets')).value.items;
  const curationPath = '/assets/' + beforeAssets[0].id + '/curation';
  const beforeCuration = target === 'tag' ? (await request(curationPath)).value : null;
  if (target === 'tag') {
    const vocabulary = await request('/ai/tag-vocabulary');
    originalVocabulary = vocabulary.value;
    const tag = (await request('/tags', 'POST', {name:'blue purple gradient'})).value;
    await request('/ai/tag-vocabulary','PUT',{tagIds:[tag.id]},{'If-Match':vocabulary.response.headers.get('etag')});
  }
  if (target === 'clear') {
    const seeded = await start(imageBase, 'semantic', 'missing');
    assert.equal((await until('seed image', () => operation(seeded.id), v => ['succeeded', 'failed'].includes(v.state))).state, 'succeeded');
  }
  execFileSync('docker', ['pause', ai]); paused = true;
  const holderJob = await start(holding.base, holding.kind, 'all');
  await until('holder running', () => operation(holderJob.id), v => v.state === 'running');
  // Give the first claimed processor time to reach its blocked inference call.
  await new Promise(r => setTimeout(r, 1000));
  let waitingJob;
  if (target === 'scan') {
    waitingJob = (await request(imageBase + '/scans', 'POST')).value;
  } else if (target === 'clear') {
    const current = await request(imageBase + '/ai/semantic');
    waitingJob = (await request(imageBase + '/ai/semantic/clear', 'POST', { confirmation: 'clear_semantic_data' },
      { 'If-Match': current.response.headers.get('etag'), 'Idempotency-Key': crypto.randomUUID() })).value;
  } else if (target === 'tag-clear') waitingJob = await startTagClear();
  else waitingJob = await start(waiting.base, waiting.kind, 'all');
  if (target === 'tag') assert.equal(waitingJob.totalItems, 1, 'require a non-empty generation job');
  await until('waiter claimed', () => readWaiter(waitingJob.id), v => waiterState(v) === 'running');
  const snapshot = await request(waiterPath(waitingJob.id));
  const began = Date.now();
  await request(waiterPath(waitingJob.id) + '/cancel', 'POST', undefined, { 'If-Match': snapshot.response.headers.get('etag') });
  const cancelled = await until('admission cancellation', () => readWaiter(waitingJob.id), v => waiterState(v) === 'cancelled', 50);
  assert.equal(target === 'scan' ? cancelled.discoveredAssets : cancelled.completedItems, 0);
  assert.deepEqual((await request(imageBase + '/assets')).value.items, beforeAssets);
  if (target === 'clear') assert.equal((await request(imageBase + '/ai/semantic')).value.coverage.completed, 1);
  assert.equal((await operation(holderJob.id)).state, 'running');
  assert.equal(execFileSync('docker', ['inspect', '--format', '{{.State.Paused}}', ai], { encoding: 'utf8' }).trim(), 'true');
  console.log('PASS waiter cancelled while holder remains running and AI paused', { target, elapsedMs: Date.now() - began, holder: holderJob.id, waiter: waitingJob.id });
  execFileSync('docker', ['unpause', ai]); paused = false;
  const holderDone = await until('holder completion', () => operation(holderJob.id), v => ['succeeded', 'failed', 'cancelled'].includes(v.state));
  assert.equal(holderDone.state, 'succeeded');
  if (target === 'scan') {
    const retry = (await request(imageBase + '/scans', 'POST')).value;
    assert.equal((await until('scan retry', () => readWaiter(retry.id), v => !['queued', 'running'].includes(v.status))).status, 'succeeded');
  } else if (target === 'tag-clear') {
    const retry = await startTagClear();
    assert.equal((await until('tag clear retry', () => operation(retry.id), v => ['succeeded', 'failed', 'cancelled'].includes(v.state))).state, 'succeeded');
  } else {
    if (target === 'clear') {
      const current = await request(imageBase + '/ai/semantic');
      await request(imageBase + '/ai/semantic', 'PATCH', { enabled: true }, { 'If-Match': current.response.headers.get('etag') });
    }
    const retry = await start(waiting.base, waiting.kind, 'missing');
    assert.equal((await until('waiter retry', () => operation(retry.id), v => ['succeeded', 'failed', 'cancelled'].includes(v.state))).state, 'succeeded');
  }
  assert.equal((await request(imageBase + '/ai/semantic')).value.coverage.completed, 1);
  assert.equal((await request(videoBase + '/ai/video-semantic')).value.coverage.completed, 1);
  assert.equal(waiterState(await readWaiter(waitingJob.id)), 'cancelled');
  if (target === 'tag') {
    const suggestions = (await request(imageBase + '/ai/tag-suggestions')).value;
    assert.equal(suggestions.coverage.completed, 1);
    assert.equal(suggestions.coverage.complete, true);
    assert.deepEqual((await request(curationPath)).value, beforeCuration);
  }
  console.log('PASS holder completed, recovery completed (clear: re-enable/backfill; others: retry), each coverage 1/1; cancelled history preserved', target);
} finally {
  if (paused && execFileSync('docker', ['inspect', '--format', '{{.State.Paused}}', ai], { encoding: 'utf8' }).trim() === 'true') execFileSync('docker', ['unpause', ai]);
  if (originalVocabulary) {
    const current = await request('/ai/tag-vocabulary');
    await request('/ai/tag-vocabulary','PUT',{tagIds:originalVocabulary.entries.map(t=>t.tagId)},{'If-Match':current.response.headers.get('etag')});
  }
  await updateBudget(original.value.backgroundConcurrency);
}
