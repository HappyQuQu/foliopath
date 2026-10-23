// Only for a fresh isolated compose project/data volume on localhost:6091.
import assert from 'node:assert/strict';
const origin = 'http://127.0.0.1:6091';
const password = process.env.FOLIOPATH_TEST_PASSWORD;
assert(password);
const phase = process.env.FOLIOPATH_TEST_PHASE ?? 'prepare';
assert(['prepare', 'backfill'].includes(phase));
let cookie = '', csrf = '';
async function request(path, method = 'GET', body, extra = {}) {
  const response = await fetch(origin + '/api/v1' + path, {
    method, signal: AbortSignal.timeout(40000),
    headers: { Origin: origin, Cookie: cookie, 'X-CSRF-Token': csrf,
      ...(body === undefined ? {} : { 'Content-Type': 'application/json' }), ...extra },
    body: body === undefined ? undefined : JSON.stringify(body),
  });
  const value = await response.json();
  assert(response.ok, `${method} ${path}: ${response.status} ${JSON.stringify(value)}`);
  return { value, response };
}
const credentials = { username: 'unrecorded-test', password };
const auth = phase === 'prepare'
  ? await request('/auth/setup', 'POST', { ...credentials, displayName: 'Unrecorded Docker Test' })
  : await request('/auth/login', 'POST', credentials);
cookie = auth.response.headers.get('set-cookie').split(';')[0];
csrf = auth.value.csrfToken;
async function until(label, read, predicate) {
  for (let i = 0; i < 60; i++) {
    const value = await read();
    if (predicate(value)) return value;
    await new Promise(resolve => setTimeout(resolve, 2000));
  }
  throw new Error(label + ' timed out');
}
let library;
if (phase === 'prepare') {
  library = (await request('/libraries', 'POST', { name: 'Unrecorded synthetic library', rootPath: '' }, { 'Idempotency-Key': crypto.randomUUID() })).value.library;
} else {
  library = (await request('/libraries')).value.items.find(item => item.name === 'Unrecorded synthetic library');
}
assert(library);
const base = `/libraries/${library.id}`;
await until('scan', async () => (await request(base + '/assets')).value, page => page.items.length === 3);
await until('model', async () => (await request('/ai/models')).value, page => page.items.some(model => model.active && model.state === 'available'));
const settings = await request(base + '/ai/semantic');
assert.equal(settings.value.coverageRecorded, false);
assert.equal(settings.value.coverage.complete, false);
assert(settings.value.activeGenerationId, 'test requires an active model generation');
console.log('PASS active model with media but no image progress', JSON.stringify(settings.value));
if (phase === 'backfill') {
  await request(base + '/ai/semantic', 'PATCH', { enabled: true }, { 'If-Match': settings.response.headers.get('etag') });
  const job = (await request(base + '/ai/semantic/jobs', 'POST', { mode: 'missing' }, { 'Idempotency-Key': crypto.randomUUID() })).value;
  const completed = await until('backfill', async () => (await request('/ai/operations/' + job.id)).value, value => ['succeeded', 'failed', 'cancelled'].includes(value.state));
  assert.equal(completed.state, 'succeeded');
  const result = (await request(base + '/ai/semantic')).value;
  assert.equal(result.coverageRecorded, true);
  assert.equal(result.coverage.completed, 1);
  assert.equal(result.coverage.eligible, 1);
  assert.equal(result.coverage.complete, true);
  console.log('PASS unknown to recorded after real inference', JSON.stringify(result));
}
