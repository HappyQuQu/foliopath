// Read-only local candidate probe. No media, inference, settings writes or model downloads.
import assert from 'node:assert/strict';
const origin = 'http://127.0.0.1:6091';
assert(process.env.FOLIOPATH_TEST_PASSWORD, 'Set isolated candidate password');
const login = await fetch(origin + '/api/v1/auth/login', {
  method: 'POST', headers: { Origin: origin, 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'candidate-test', password: process.env.FOLIOPATH_TEST_PASSWORD }),
  signal: AbortSignal.timeout(10000),
});
assert(login.ok);
const cookie = login.headers.get('set-cookie').split(';')[0];
async function read(path) {
  const response = await fetch(origin + '/api/v1' + path, { headers: { Cookie: cookie }, signal: AbortSignal.timeout(10000) });
  assert.equal(response.status, 200, path);
  return response.json();
}
const expected = process.env.FOLIOPATH_TEST_MODEL_STATE ?? 'available';
assert(['available', 'unavailable'].includes(expected));
let matched = false;
for (let attempt = 0; attempt < 30; attempt++) {
  const models = await read('/ai/models');
  if (models.items.some(model => model.active && model.state === expected)) { matched = true; break; }
  await new Promise(resolve => setTimeout(resolve, 2000));
}
assert(matched, `active model did not become ${expected}`);
const libraries = await read('/libraries');
const library = libraries.items.find(item => item.name === 'Synthetic candidate chain');
assert(library);
const status = await read(`/libraries/${library.id}/ai/video-semantic`);
assert.deepEqual(Object.fromEntries(['eligible', 'completed', 'degraded', 'failed', 'stale', 'complete'].map(key => [key, status.coverage[key]])),
  { eligible: 2, completed: 1, degraded: 1, failed: 0, stale: 0, complete: false });
console.log(JSON.stringify({ modelState: expected, videoStatus: status }));
const imageStatus = await read(`/libraries/${library.id}/ai/semantic`);
assert.equal(imageStatus.activeGenerationId, expected === 'available' ? status.activeGenerationId : null);
assert.deepEqual(Object.fromEntries(['eligible', 'completed', 'failed', 'stale', 'complete'].map(key => [key, imageStatus.coverage[key]])),
  { eligible: 1, completed: 1, failed: 0, stale: 0, complete: true });
console.log(JSON.stringify({ modelState: expected, imageStatus }));
console.log('PASS recorded video progress remains readable without inference');
console.log('PASS recorded image progress remains readable without inference');
