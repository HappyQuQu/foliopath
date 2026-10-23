// CR-2026-031 T1/T2: isolated local Docker, synthetic media, real HTTP only.
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {copyFileSync, mkdtempSync, mkdirSync, readFileSync, realpathSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createHash, randomUUID} from 'node:crypto';

const repo = fileURLToPath(new URL('../../', import.meta.url));
const run = (...args) => execFileSync('docker', args, {encoding: 'utf8', timeout: 60000}).trim();
assert(process.env.FOLIOPATH_TEST_APP_IMAGE, 'Set FOLIOPATH_TEST_APP_IMAGE to an explicit locally built image');
const image = run('image', 'inspect', process.env.FOLIOPATH_TEST_APP_IMAGE, '--format', '{{.Id}}');
const runID = randomUUID();
const ownerLabel = 'org.foliopath.test.tag-cloud-run';
const name = 'fp-tags-' + runID.slice(0, 8);
const root = realpathSync(mkdtempSync(join(tmpdir(), name + '-')));
for (const directory of ['data', 'media/alpha', 'media/beta']) mkdirSync(join(root, directory), {recursive: true});
const fixtures = [
  ['viewer-blue-violet.png', 'alpha/first.png'],
  ['viewer-blue-violet.png', 'alpha/second.png'],
  ['viewer-blue-violet.png', 'beta/third.png'],
  ['range-video.mp4', 'alpha/movie.mp4'],
];
for (const [source, target] of fixtures) copyFileSync(join(repo, 'tests/fixtures/media', source), join(root, 'media', target));
const hashMedia = () => fixtures.map(([, target]) => createHash('sha256').update(readFileSync(join(root, 'media', target))).digest('hex'));
const before = hashMedia();
let cookie = '', csrf = '', origin, containerID;
async function request(path, method = 'GET', body, headers = {}) {
  return fetch(origin + '/api/v1' + path, {
    method, signal: AbortSignal.timeout(10000),
    headers: {'Content-Type': 'application/json', Origin: origin, Cookie: cookie, 'X-CSRF-Token': csrf, ...headers},
    body: body === undefined ? undefined : JSON.stringify(body),
  });
}
async function json(path, method, body, headers) {
  const response = await request(path, method, body, headers);
  const value = await response.json();
  assert(response.ok, `${method ?? 'GET'} ${path}: ${response.status} ${value.error?.code ?? 'unexpected response'}`);
  return value;
}
async function expectError(path, status, code, method = 'GET', body, headers) {
  const response = await request(path, method, body, headers);
  assert.equal(response.status, status, `${method} ${path}`);
  assert.equal((await response.json()).error?.code, code, `${method} ${path}`);
}
async function until(label, read, predicate, timeoutMs = 90000) {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const value = await read();
    if (predicate(value)) return value;
    await new Promise(resolve => setTimeout(resolve, 500));
  }
  throw Error(label + ' timed out');
}
try {
  // Capture the created ID before starting. Cleanup must never target an unrelated
  // pre-existing container if name allocation or startup fails.
  containerID = run('create', '--pull', 'never', '--name', name, '--label', `${ownerLabel}=${runID}`, '--read-only', '--cap-drop', 'ALL', '--security-opt', 'no-new-privileges:true',
    '--tmpfs', '/tmp:rw,noexec,nosuid,size=16m,mode=0700', '--memory', '1g', '--cpus', '1',
    '-p', '127.0.0.1::8080', '--mount', `type=bind,src=${join(root, 'media')},dst=/library,readonly`,
    '--mount', `type=bind,src=${join(root, 'data')},dst=/app/data`, image);
  assert.match(containerID, /^[a-f0-9]{64}$/);
  const inspect = () => JSON.parse(run('container', 'inspect', containerID))[0];
  run('start', containerID);
  const refreshOrigin = () => {
    const ports = inspect().NetworkSettings.Ports['8080/tcp'];
    assert.equal(ports.length, 1);
    assert.equal(ports[0].HostIp, '127.0.0.1');
    assert.match(ports[0].HostPort, /^\d+$/);
    origin = 'http://127.0.0.1:' + ports[0].HostPort;
  };
  refreshOrigin();
  await until('health', inspect, state => state.State.Running && state.State.Health?.Status === 'healthy');
  const initialContainer = inspect();
  assert.equal(initialContainer.Image, image);
  assert.equal(initialContainer.HostConfig.ReadonlyRootfs, true);
  assert.deepEqual(initialContainer.HostConfig.CapDrop, ['ALL']);
  assert.deepEqual(initialContainer.Mounts.map(({Source, Destination, RW}) => ({Source, Destination, RW})).sort((a, b) => a.Destination.localeCompare(b.Destination)), [
    {Source: join(root, 'data'), Destination: '/app/data', RW: true},
    {Source: join(root, 'media'), Destination: '/library', RW: false},
  ]);
  await expectError('/tags', 401, 'authentication_required');
  const credentials = {username: 'tag-test', password: 'Synthetic-Tags-2026!'};
  await json('/auth/setup', 'POST', {...credentials, displayName: 'Synthetic tag test'});
  async function login() {
    const response = await request('/auth/login', 'POST', credentials);
    assert(response.ok, `login failed: ${response.status}`);
    const setCookie = response.headers.get('set-cookie');
    assert(setCookie && /HttpOnly/i.test(setCookie) && /SameSite=Strict/i.test(setCookie), 'login must issue a protected session cookie');
    cookie = setCookie.split(';')[0];
    csrf = (await response.json()).csrfToken;
    assert.equal(typeof csrf, 'string');
    assert(csrf.length > 0);
  }
  await login();
  const initialLibraries = await json('/libraries');
  assert.deepEqual(initialLibraries.items, [], 'the smoke must start with an isolated empty catalog');
  assert.equal(initialLibraries.nextCursor, null);
  // Automatic discovery reconciles media inside configured libraries. It does
  // not turn /library children into libraries: create them through the real API.
  async function createLibrary(directory, expectedAssets) {
    const body = {name: directory, rootPath: directory};
    const headers = {'Idempotency-Key': `${runID}-${directory}`};
    const created = await json('/libraries', 'POST', body, headers);
    assert.equal(created.library.name, directory);
    assert.equal(created.library.rootPath, directory);
    assert.equal(created.scan.libraryId, created.library.id);
    const scan = await until(`${directory} scan`, () => json(`/scans/${created.scan.id}`), value => !['queued', 'running'].includes(value.status));
    assert.equal(scan.status, 'succeeded', `${directory} scan failed: ${scan.errorCode ?? scan.status}`);
    assert.equal(scan.errorCount, 0);
    const page = await json(`/libraries/${created.library.id}/assets`);
    assert.equal(page.items.length, expectedAssets);
    assert.equal(page.nextCursor, null);
    assert(page.items.every(asset => asset.libraryId === created.library.id));
    return {library: created.library, assets: page.items};
  }
  const {library: alpha, assets: a} = await createLibrary('alpha', 3);
  const {library: beta, assets: b} = await createLibrary('beta', 1);
  assert.equal(a.filter(asset => asset.kind === 'video').length, 1);
  assert.equal(a.filter(asset => asset.kind === 'image').length, 2);
  assert.equal(b[0].kind, 'image');
  const tag = await json('/tags', 'POST', {name: '蓝色'});
  const second = await json('/tags', 'POST', {name: '样片'});
  const unused = await json('/tags', 'POST', {name: '未使用'});
  async function replace(asset, tagIds) {
    assert(asset, 'expected synthetic asset is missing');
    const current = await request(`/assets/${asset.id}/curation`);
    assert.equal(current.status, 200);
    const etag = current.headers.get('etag');
    assert.match(etag ?? '', /^"curation-r[1-9][0-9]*"$/);
    assert.equal((await current.json()).assetId, asset.id);
    return json(`/assets/${asset.id}/tags`, 'PUT', {tagIds}, {'If-Match': etag});
  }
  await expectError(`/assets/${a[0].id}/tags`, 428, 'precondition_required', 'PUT', {tagIds: []});
  for (const asset of [...a, ...b]) await replace(asset, [tag.id]);
  await replace(a.find(asset => asset.name === 'first.png'), [tag.id, second.id]);
  const scope = `/tags?libraryId=${alpha.id}&kind=image,animated`;
  let cloud = await json(scope);
  assert.equal(cloud.items.find(item => item.id === tag.id).assetCount, 2);
  assert.equal(cloud.items.find(item => item.id === second.id).assetCount, 1);
  assert(!cloud.items.some(item => item.id === unused.id));
  const betaCloud = await json(`/tags?libraryId=${beta.id}&kind=image,animated`);
  assert.equal(betaCloud.items.length, 1);
  assert.equal(betaCloud.items[0].id, tag.id);
  assert.equal(betaCloud.items[0].assetCount, 1);
  const global = await json('/tags');
  assert.equal(global.items.find(item => item.id === tag.id).assetCount, 4);
  assert(global.items.some(item => item.id === unused.id && item.assetCount === 0));
  console.log('PASS library-scoped distinct image counts, video exclusion, global vocabulary compatibility');
  const firstPage = await json(scope + '&limit=1');
  assert(firstPage.nextCursor);
  const cursor = encodeURIComponent(firstPage.nextCursor);
  const nextPage = await json(`/tags?libraryId=${alpha.id}&kind=animated,image&limit=1&cursor=${cursor}`);
  assert.equal(nextPage.items.length, 1);
  assert.equal(nextPage.nextCursor, null);
  assert.notEqual(nextPage.items[0].id, firstPage.items[0].id);
  await expectError(`/tags?libraryId=${beta.id}&kind=image,animated&cursor=${cursor}`, 400, 'invalid_cursor');
  await expectError(`/tags?libraryId=${alpha.id}&kind=video&cursor=${cursor}`, 400, 'invalid_cursor');
  await expectError(scope + `&q=other&cursor=${cursor}`, 400, 'invalid_cursor');
  await expectError('/tags?libraryId=lib_999999', 404, 'library_not_found');
  await expectError('/tags?kind=unknown', 400, 'invalid_request');
  await expectError('/tags?libraryId=lib_0', 400, 'invalid_request');
  await expectError('/tags?kind=image&kind=video', 400, 'invalid_request');
  const assets = await json(`/tags/${tag.id}/assets?libraryId=${alpha.id}&kind=image,animated`);
  assert.equal(assets.items.length, 2);
  assert(assets.items.every(item => item.asset.libraryId === alpha.id && item.asset.kind !== 'video'));
  console.log('PASS scoped filtering, pagination, cross-scope cursor rejection and invalid scope errors');
  await replace(a.find(asset => asset.name === 'second.png'), []);
  cloud = await json(scope);
  assert.equal(cloud.items.find(item => item.id === tag.id).assetCount, 1);
  await expectError(scope + '&cursor=' + cursor, 400, 'invalid_cursor');
  const beforeRestart = inspect();
  run('restart', containerID);
  const restarted = await until('restarted health', inspect, state => state.State.Running && state.State.StartedAt !== beforeRestart.State.StartedAt && state.State.Health?.Status === 'healthy');
  assert.equal(restarted.Id, containerID);
  assert.equal(restarted.Image, image);
  assert.deepEqual(restarted.Mounts, beforeRestart.Mounts);
  refreshOrigin(); cookie = ''; csrf = ''; await login();
  assert.equal((await json(scope)).items.find(item => item.id === tag.id).assetCount, 1);
  assert.deepEqual(hashMedia(), before);
  console.log('PASS manual removal updates count, stale cursor rejected, restart persistence, originals unchanged');
  console.log(JSON.stringify({passed: true, name, image, root, origin, alpha: alpha.id, beta: beta.id}));
} finally {
  if (containerID && process.env.FOLIOPATH_TEST_KEEP !== '1') {
    try {
      const owned = JSON.parse(run('container', 'inspect', containerID))[0];
      assert.equal(owned.Config.Labels?.[ownerLabel], runID, 'refusing cleanup of a container not created by this run');
      run('rm', '-f', containerID);
      console.log(`Removed only test container ${name}; synthetic fixtures/database retained at ${root}`);
    } catch {
      // Preserve the original test failure; never broaden cleanup by name or path.
      console.error(`Could not safely clean up test container ${containerID}; inspect it manually. Synthetic data retained at ${root}`);
      process.exitCode = 1;
    }
  } else if (containerID) {
    console.log(`Kept isolated test container ${name} for browser QA; synthetic data: ${root}`);
  } else {
    console.log(`No test container was created; synthetic fixtures retained at ${root}`);
  }
}
