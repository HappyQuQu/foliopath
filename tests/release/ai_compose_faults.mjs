// F02 failure matrix for the real standalone Compose. Mutations target temp fixtures only.
import assert from 'node:assert/strict';
import {mkdirSync, copyFileSync, openSync, readSync, writeSync, closeSync, readFileSync} from 'node:fs';
import {dirname, join} from 'node:path';
import {createHash, randomUUID} from 'node:crypto';

export async function exerciseModelFaults(e) {
  const {root, cache, env, deployment, compose, run, inspect, until, request, refreshOrigin} = e;
  const json = async (path, method = 'GET', body, headers = {}) => {
    const response = await request(path, method, body, true, headers);
    const value = await response.json();
    assert(response.ok, `${method} ${path}: ${response.status} ${JSON.stringify(value)}`);
    return value;
  };
  const models = () => json('/ai/models');
  const ready = () => until('both models recovered', models,
    page => page.items.filter(m => m.active && m.state === 'available').length === 2);
  const library = (await json('/libraries', 'POST', {name: 'P5 synthetic', rootPath: ''}, {'Idempotency-Key': randomUUID()})).library;
  const base = '/libraries/' + library.id;
  await until('synthetic scan', () => json(base + '/assets'), value => value.items.length === 2);
  async function analyze(kind) {
    const operation = await json(base + `/ai/${kind}/jobs`, 'POST', {mode: 'all'}, {'Idempotency-Key': randomUUID()});
    const final = await until(kind + ' completed', () => json('/ai/operations/' + operation.id),
      value => ['succeeded', 'failed', 'cancelled'].includes(value.state));
    assert.equal(final.state, 'succeeded', JSON.stringify(final));
    assert.equal(final.completedItems, 2);
  }
  for (const kind of ['semantic', 'faces']) {
    const settings = await request(base + '/ai/' + kind);
    assert(settings.ok);
    await json(base + '/ai/' + kind, 'PATCH', {enabled: true}, {'If-Match': settings.headers.get('etag')});
    await analyze(kind);
  }
  const originalModels = (await models()).items.filter(m => m.active).map(m => m.id).sort();
  const generations = Object.fromEntries(await Promise.all(['semantic', 'faces'].map(async kind =>
    [kind, (await json(base + '/ai/' + kind)).activeGenerationId])));
  let semanticCoverage = (await json(base + '/ai/semantic')).coverage;
  const assets = (await json(base + '/assets')).items;
  const faceAsset = assets.find(a => a.name === 'faces.png');
  let faceSnapshot = await json('/assets/' + faceAsset.id + '/faces');
  assert(faceSnapshot.items.length > 0, 'nonempty synthetic faces required');
  const mediaHashes = assets.map(a => createHash('sha256').update(readFileSync(join(root, 'media', a.name))).digest('hex'));
  async function preserved(offline = false) {
    assert.deepEqual((await json(base + '/assets')).items.map(a => a.id).sort(), assets.map(a => a.id).sort());
    assert.deepEqual(await json('/assets/' + faceAsset.id + '/faces'), faceSnapshot);
    for (const kind of ['semantic', 'faces']) {
      const settings = await json(base + '/ai/' + kind);
      // Documented API contract hides the semantic admission generation while
      // offline, but retains coverage; recovery must expose the original ID.
      assert.equal(settings.activeGenerationId, offline && kind === 'semantic' ? null : generations[kind]);
      if (kind === 'semantic') assert.deepEqual(settings.coverage, semanticCoverage);
    }
  }
  async function unavailable(label) {
    await until(label + ' unavailable', models, page => {
      const active = page.items.filter(m => m.active);
      return active.length === 2 && active.every(m => m.state === 'unavailable');
    });
    await preserved(true);
    for (const kind of ['semantic', 'faces']) {
      const response = await request(base + `/ai/${kind}/jobs`, 'POST', {mode: 'all'}, true, {'Idempotency-Key': randomUUID()});
      assert.equal(response.status, 409);
      const body = await response.json();
      assert.equal(body.error.code, kind === 'faces' ? 'face_model_unavailable' : 'model_unavailable');
      assert(!JSON.stringify(body).includes(root), 'no host path in public errors');
    }
    console.log('PASS F02 ' + label + ': unavailable, jobs rejected, catalog/faces/generations retained');
  }
  async function recovered(label) {
    const page = await ready();
    assert.deepEqual(page.items.filter(m => m.active).map(m => m.id).sort(), originalModels);
    await preserved();
    for (const kind of ['semantic', 'faces']) await analyze(kind);
    // A deliberate reanalysis may replace derived observation rows; compare each
    // subsequent outage against the state immediately before that outage.
    faceSnapshot = await json('/assets/' + faceAsset.id + '/faces');
    assert(faceSnapshot.items.length > 0);
    semanticCoverage = (await json(base + '/ai/semantic')).coverage;
    console.log('PASS F02 ' + label + ': same models/generations restored, both analyses succeed');
  }
  // Empty source and same-size wrong bytes fail cache verification before HTTP readiness.
  const missing = join(root, 'missing-cache'); mkdirSync(missing);
  const corrupt = join(root, 'corrupt-cache'); mkdirSync(corrupt);
  const aiImage = env.FOLIOPATH_AI_IMAGE;
  const entries = JSON.parse(run('run', '--rm', '--network', 'none', '--entrypoint', 'python', aiImage,
    '-c', 'from pathlib import Path; print(Path("/opt/foliopath-inference/cache-manifest.json").read_text())'));
  const first = entries[0]; assert(first.size > 0);
  const badFile = join(corrupt, first.path); mkdirSync(dirname(badFile), {recursive: true});
  copyFileSync(join(cache, first.path), badFile);
  const fd = openSync(badFile, 'r+');
  try { const byte = Buffer.alloc(1); readSync(fd, byte, 0, 1, 0); byte[0] ^= 255; writeSync(fd, byte, 0, 1, 0); }
  finally { closeSync(fd); }
  for (const [label, path] of [['missing cache', missing], ['same-size corrupt cache', corrupt]]) {
    env.FOLIOPATH_MODEL_CACHE_PATH = path;
    compose('up', '-d', '--force-recreate', 'machine-learning');
    await unavailable(label);
    const ai = compose('ps', '-a', '-q', 'machine-learning');
    assert.notEqual(inspect(ai).State.Health?.Status, 'healthy');
    await until('cache verification rejects startup', () => inspect(ai), value => value.RestartCount > 0 || value.State.ExitCode !== 0);
    env.FOLIOPATH_MODEL_CACHE_PATH = cache;
    compose('up', '-d', '--force-recreate', 'machine-learning');
    await recovered(label);
  }
  // Change the trusted expectation, not the service. The healthy service must be rejected.
  const wrong = structuredClone(deployment);
  wrong.identity.runtime_contract_sha256 = '0'.repeat(64);
  wrong.face.identity.runtime_contract_sha256 = '0'.repeat(64);
  env.FOLIOPATH_AI_DEPLOYMENT = JSON.stringify(wrong);
  compose('up', '-d', '--force-recreate', 'foliopath');
  await until('core with mismatched identity healthy', () => inspect(compose('ps', '-q', 'foliopath')).State.Health.Status, s => s === 'healthy');
  refreshOrigin();
  await unavailable('identity mismatch');
  env.FOLIOPATH_AI_DEPLOYMENT = JSON.stringify(deployment);
  compose('up', '-d', '--force-recreate', 'foliopath');
  await until('restored core healthy', () => inspect(compose('ps', '-q', 'foliopath')).State.Health.Status, s => s === 'healthy');
  refreshOrigin();
  await recovered('identity mismatch');
  assert.deepEqual(assets.map(a => createHash('sha256').update(readFileSync(join(root, 'media', a.name))).digest('hex')), mediaHashes);
  console.log('PASS F02 synthetic original hashes unchanged; no downloaded models or new generations');
}
