// P5/F01: actual standalone Compose, isolated generated fixtures and data only.
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {mkdtempSync, mkdirSync, copyFileSync, rmSync, readFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {randomUUID, createHash} from 'node:crypto';
import {exerciseModelFaults} from './ai_compose_faults.mjs';
import {exerciseDelivery} from './ai_compose_delivery.mjs';

const repo = fileURLToPath(new URL('../../', import.meta.url));
const run = (...args) => execFileSync('docker', args, {encoding: 'utf8', timeout: 60000}).trim();
const appImage = run('image', 'inspect', process.env.FOLIOPATH_TEST_APP_IMAGE ?? 'foliopath-app:p5-delivery', '--format', '{{.Id}}');
const aiImage = run('image', 'inspect', process.env.FOLIOPATH_TEST_AI_IMAGE ?? 'foliopath-inference:p5-delivery', '--format', '{{.Id}}');
const cache = process.env.FOLIOPATH_TEST_MODEL_CACHE;
assert(cache, 'FOLIOPATH_TEST_MODEL_CACHE must name the copied model cache (not media)');
const identity = JSON.parse(run('run', '--rm', '--network', 'none', '--entrypoint', 'python', aiImage, '-m', 'image_identity'));
const faults = process.argv.includes('--faults');
const delivery = process.argv.includes('--delivery');
assert(!(faults && delivery), 'run fault and delivery matrices separately');
const withFace = faults || delivery;
const deployment = {origin: 'http://machine-learning:3003', identity,
  model_bytes: 610841665, license_id: 'unreviewed-local-test'};
if (withFace) {
  const manifest = JSON.parse(run('run', '--rm', '--network', 'none', '--entrypoint', 'python', aiImage,
    '-c', 'from pathlib import Path; print(Path("/opt/foliopath-inference/cache-manifest.json").read_text())'));
  const weights = ['detection', 'recognition'].map(kind => manifest.find(e => e.path === `facial-recognition/buffalo_l/${kind}/model.onnx`));
  assert(weights.every(Boolean), 'fault matrix requires a face-capable candidate image');
  deployment.face = {identity: {...identity, protocol: 'foliopath-face-v1', model: 'buffalo_l'},
    detector_sha256: weights[0].sha256, embedder_sha256: weights[1].sha256,
    model_bytes: weights.reduce((sum, e) => sum + e.size, 0), license_id: 'unreviewed-local-test',
    threshold: {schemaVersion: 1, profileId: 'p5-synthetic-only', coreSimilarity: .8, edgeSimilarity: .6,
      minCoreSize: 2, qualitySummarySha256: createHash('sha256').update('NOT quality calibration').digest('hex')}};
}
const project = 'fp-p5-' + randomUUID().slice(0, 8);
const root = mkdtempSync(join(tmpdir(), project + '-'));
for (const dir of ['media', 'data']) mkdirSync(join(root, dir));
copyFileSync(join(repo, 'tests/fixtures/media/viewer-blue-violet.png'), join(root, 'media', 'synthetic.png'));
if (withFace) copyFileSync(join(repo, 'tests/fixtures/ai-faces/synthetic-pair.png'), join(root, 'media', 'faces.png'));
const env = {...process.env, FOLIOPATH_IMAGE: appImage, FOLIOPATH_AI_IMAGE: aiImage,
  FOLIOPATH_LIBRARY_PATH: join(root, 'media'), FOLIOPATH_DATA_PATH: join(root, 'data'),
  FOLIOPATH_MODEL_CACHE_PATH: cache, FOLIOPATH_BIND_ADDRESS: '127.0.0.1', FOLIOPATH_PORT: '0',
  FOLIOPATH_AI_DEPLOYMENT: JSON.stringify(deployment)};
const offlineArgs = process.argv.includes('--offline') ? ['-f', join(repo, 'tests/release/compose.ai.offline.yaml')] : [];
const compose = (...args) => execFileSync('docker', ['compose', '-p', project, '-f', join(repo, 'compose.ai.yaml'), ...offlineArgs, ...args],
  {env, encoding: 'utf8', timeout: 60000}).trim();
async function until(label, read, ready, attempts = 120) {
  for (let i = 0; i < attempts; i++) {
    const value = await read();
    if (ready(value)) return value;
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
  throw Error(label + ' timed out');
}
const inspect = id => JSON.parse(run('inspect', id))[0];
let origin, cookie = '', csrf = '';
async function request(path, method = 'GET', body, authenticated = true, headers = {}) {
  if (offlineArgs.length) {
    const ai = compose('ps', '-q', 'machine-learning');
    const base = ai ? 'http://foliopath:8080' : 'http://127.0.0.1:8080';
    const probe = join(repo, 'tests/release/ai_http_probe.py');
    const args = ai ? ['exec', '-i', ai, 'python', '-c', readFileSync(probe, 'utf8')] : ['run', '--rm', '-i', '--read-only', '--network', 'container:' + compose('ps', '-q', 'foliopath'),
      '--mount', `type=bind,src=${join(repo, 'tests/release/ai_http_probe.py')},dst=/probe.py,readonly`,
      '--entrypoint', 'python', aiImage, '/probe.py'];
    const output = execFileSync('docker', args, {encoding: 'utf8', timeout: 30000,
      input: JSON.stringify({base, path, method, body: body ?? null, headers: {'Content-Type': 'application/json', Origin: base,
        ...(authenticated ? {Cookie: cookie, 'X-CSRF-Token': csrf} : {}), ...headers}})});
    const result = JSON.parse(output);
    return new Response(result.status === 204 ? null : Buffer.from(result.body, 'base64'), {status: result.status, headers: result.headers});
  }
  return fetch(origin + '/api/v1' + path, {method, signal: AbortSignal.timeout(10000),
    headers: {'Content-Type': 'application/json', Origin: origin,
      ...(authenticated ? {Cookie: cookie, 'X-CSRF-Token': csrf} : {}), ...headers},
    body: body === undefined ? undefined : JSON.stringify(body)});
}
try {
  const config = JSON.parse(compose('config', '--format', 'json'));
  assert.deepEqual(Object.keys(config.services).sort(), ['foliopath', 'machine-learning']);
  const core = config.services.foliopath, ml = config.services['machine-learning'];
  assert(!core.depends_on);
  assert(!ml.ports?.length);
  assert.deepEqual(Object.keys(ml.networks), ['inference']);
  assert.equal(config.networks.inference.internal, true);
  if (offlineArgs.length) assert.equal(config.networks.web.internal, true);
  assert.deepEqual(ml.volumes.map(v => v.target), ['/cache']);
  for (const service of [core, ml]) {
    assert.equal(service.read_only, true);
    assert.equal(service.pull_policy, 'never');
    assert.deepEqual(service.cap_drop, ['ALL']);
    assert.equal(service.mem_limit, service.memswap_limit);
    for (const volume of service.volumes) assert.equal(volume.bind.create_host_path, false);
  }
  console.log('PASS resolved two-service configuration and isolation');
  compose('up', '-d', 'foliopath');
  const app = compose('ps', '-q', 'foliopath');
  await until('core healthy without AI', () => inspect(app).State.Health.Status, s => s === 'healthy');
  function refreshOrigin() {
    if (offlineArgs.length) { origin = 'http://127.0.0.1:8080'; return; }
    const port = inspect(compose('ps', '-q', 'foliopath')).NetworkSettings.Ports['8080/tcp'][0];
    assert.equal(port.HostIp, '127.0.0.1');
    origin = 'http://127.0.0.1:' + port.HostPort;
  }
  refreshOrigin();
  assert.equal((await request('/libraries', 'GET', undefined, false)).status, 401);
  const password = 'Synthetic-' + randomUUID() + '!';
  assert((await request('/auth/setup', 'POST', {username: 'p5-test', displayName: 'P5 test', password})).ok);
  const login = await request('/auth/login', 'POST', {username: 'p5-test', password});
  assert(login.ok);
  assert(login.headers.get('set-cookie').includes('HttpOnly'));
  assert(/SameSite=Strict/i.test(login.headers.get('set-cookie')));
  cookie = login.headers.get('set-cookie').split(';')[0];
  csrf = (await login.json()).csrfToken;
  assert((await request('/libraries')).ok);
  console.log('PASS fresh setup and authenticated core while AI absent');
  compose('up', '-d', 'machine-learning');
  const ai = compose('ps', '-q', 'machine-learning');
  await until('AI health', () => inspect(ai).State.Health.Status, s => s === 'healthy');
  const models = await until('semantic activation', async () => {
    const response = await request('/ai/models'); assert(response.ok); return (await response.json()).items;
  }, items => items.filter(m => m.active && m.state === 'available').length === (withFace ? 2 : 1));
  assert.equal(models.filter(m => m.active).length, withFace ? 2 : 1, 'face is opt-in, no test threshold enabled by Compose');
  for (const [id, memory, cpu] of [[app, 1024 * 1024 ** 2, 1e9], [ai, 2304 * 1024 ** 2, 3e9]]) {
    const actual = inspect(id);
    assert.equal(actual.HostConfig.Memory, memory);
    assert.equal(actual.HostConfig.MemorySwap, memory);
    assert.equal(actual.HostConfig.NanoCpus, cpu);
    assert.equal(actual.HostConfig.ReadonlyRootfs, true);
    assert.deepEqual(actual.HostConfig.CapDrop, ['ALL']);
  }
  const aiState = inspect(ai);
  assert.deepEqual(aiState.HostConfig.PortBindings, {});
  assert.equal(Object.keys(aiState.NetworkSettings.Networks).length, 1);
  assert.equal(aiState.Mounts.find(m => m.Destination === '/cache').RW, false);
  assert(!aiState.Mounts.some(m => ['/library', '/app/data', '/var/run/docker.sock'].includes(m.Destination)));
  assert.equal(inspect(app).Mounts.find(m => m.Destination === '/library').RW, false);
  console.log('PASS verified offline model activation and runtime mount/resource boundaries');
  if (faults) await exerciseModelFaults({root, cache, env, deployment, compose, run, inspect, until, request, refreshOrigin});
  if (delivery) await exerciseDelivery({root, cache, env, compose, run, inspect, until, request, refreshOrigin});
  const currentApp = compose('ps', '-q', 'foliopath');
  compose('restart', 'foliopath');
  await until('core restart', () => inspect(currentApp).State.Health.Status, s => s === 'healthy');
  // Docker may allocate a different host port when restarting a port-0 container.
  refreshOrigin();
  const relogin = await request('/auth/login', 'POST', {username: 'p5-test', password});
  assert(relogin.ok, 'administrator persists across restart');
  console.log('PASS persisted administrator after restart');
  console.log(JSON.stringify({project, appImage, aiImage, identity, passed: true}));
} finally {
  // Only this unique test project's containers/networks and generated temporary data.
  compose('down', '--timeout', '15');
  rmSync(root, {recursive: true, force: true});
}
