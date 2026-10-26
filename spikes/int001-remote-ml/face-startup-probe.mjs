// Local synthetic deployment/worker test. Only repository media fixtures, no production data or host AI port.
import { execFileSync } from 'node:child_process';
import { readFileSync } from 'node:fs';
import { createHash, randomUUID } from 'node:crypto';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
const docker = (...args) => execFileSync('docker', args, { encoding: 'utf8', timeout: 60000 }).trim();
const key = `fp-face-${randomUUID().slice(0, 8)}`;
const network = key, volume = key + '-data', app = key + '-app', ai = key + '-ai';
const origin = 'http://127.0.0.1:6094';
const manifest = JSON.parse(readFileSync(new URL('./cache-baseline.json', import.meta.url)));
const identity = JSON.parse(docker('run', '--rm', '--network', 'none', '--entrypoint', 'python', 'foliopath-inference:face-candidate', '-m', 'image_identity'));
const weights = ['detection', 'recognition'].map(k => manifest.find(e => e.path === `facial-recognition/buffalo_l/${k}/model.onnx`));
assert(weights.every(Boolean));
const config = { origin: `http://${ai}:3003`, identity, model_bytes: 610841665, license_id: 'unreviewed-local-test', face: {
  identity: { ...identity, protocol: 'foliopath-face-v1', model: 'buffalo_l' },
  detector_sha256: weights[0].sha256, embedder_sha256: weights[1].sha256,
  model_bytes: weights.reduce((n,e)=>n+e.size,0), license_id: 'unreviewed-local-test',
  threshold: { schemaVersion: 1, profileId: 'synthetic-startup-only', coreSimilarity: .8, edgeSimilarity: .6, minCoreSize: 2,
    qualitySummarySha256: createHash('sha256').update('synthetic startup fixture; NOT quality calibration').digest('hex') },
} };
let cookie = '', csrf = '';
async function request(path, method = 'GET', body, headers = {}) {
  const r = await fetch(origin + '/api/v1' + path, { method, signal: AbortSignal.timeout(5000),
    headers: { Origin: origin, Cookie: cookie, 'X-CSRF-Token': csrf, 'Content-Type': 'application/json', ...headers },
    body: body === undefined ? undefined : JSON.stringify(body) });
  assert(r.ok, `${path}: ${r.status} ${await r.clone().text()}`);
  return { r, value: r.status === 204 ? null : await r.json() };
}
async function until(label, action, ready) {
  for (let n = 0; n < 70; n++) {
    try { const value = await action(); if (ready(value)) return value; } catch (e) { if(n===69) throw e; }
    await new Promise(r => setTimeout(r, 2000));
  }
  throw Error(label + ' timed out');
}
let madeNetwork = false, madeVolume = false, madeApp = false, madeAI = false, paused = false;
try {
  docker('network', 'create', '--internal', network); madeNetwork = true;
  docker('volume', 'create', volume); madeVolume = true;
  docker('run', '-d', '--name', app, '--network', network, '--read-only', '--tmpfs', '/tmp',
    '--mount', `type=volume,src=${volume},dst=/app/data`, '-p', '127.0.0.1:6094:8080',
    '--mount', `type=bind,src=${fileURLToPath(new URL('../../tests/fixtures/media', import.meta.url))},dst=/library,readonly`,
    '-e', 'FOLIOPATH_LISTEN=0.0.0.0:8080', '-e', 'FOLIOPATH_AI_DEPLOYMENT=' + JSON.stringify(config), 'foliopath-app:face-worker'); madeApp = true;
  docker('network', 'connect', 'bridge', app);
  await until('startup', () => fetch(origin + '/health/ready'), r => r.ok);
  const password = 'Synthetic-' + randomUUID() + '!';
  await request('/auth/setup', 'POST', { username: 'face-startup-test', displayName: 'Synthetic startup test', password });
  const login = await request('/auth/login', 'POST', { username: 'face-startup-test', password });
  cookie = login.r.headers.get('set-cookie').split(';')[0]; csrf = login.value.csrfToken;
  const models = async () => (await request('/ai/models')).value.items;
  assert.equal((await models()).length, 0);
  console.log('PASS first start offline: core ready, no unverified models');
  docker('run', '-d', '--name', ai, '--network', network, '--read-only', '--tmpfs', '/tmp', '--tmpfs', '/verified-cache:size=1200m',
    '--mount', 'type=bind,src=/Volumes/extended/foliopath/model-cache,dst=/cache,readonly', 'foliopath-inference:face-candidate'); madeAI = true;
  const active = await until('both activations', models, a => a.length===2 && a.every(m=>m.active && m.state==='available'));
  const ids = active.map(m=>m.id).sort();
  console.log('PASS CLIP and face active', ids);
  docker('stop', ai);
  await until('offline marking', models, a => a.length===2 && a.every(m=>m.state==='unavailable'));
  console.log('PASS unavailable models retained');
  docker('start', ai);
  await until('recovery', models, a => a.length===2 && a.every(m=>m.active && m.state==='available'));
  docker('restart', app);
  const restarted = await until('restart', models, a => a.length===2 && a.every(m=>m.active && m.state==='available'));
  assert.deepEqual(restarted.map(m=>m.id).sort(), ids);
  console.log('PASS recovery/restart preserve two model identities');
  const library = (await request('/libraries','POST',{name:'Synthetic face worker',rootPath:''},{'Idempotency-Key':randomUUID()})).value.library;
  const base = `/libraries/${library.id}`;
  await until('scan',async()=>(await request(base+'/assets')).value,v=>v.items.length===2);
  const settings = await request(base+'/ai/faces');
  await request(base+'/ai/faces','PATCH',{enabled:true},{'If-Match':settings.r.headers.get('etag')});
  const key = randomUUID();
  const job = (await request(base+'/ai/faces/jobs','POST',{mode:'all'},{'Idempotency-Key':key})).value;
  const completed = await until('face analysis',async()=>(await request('/ai/operations/'+job.id)).value,v=>['succeeded','failed'].includes(v.state));
  assert.equal(completed.state,'succeeded',JSON.stringify(completed));
  const coverage = (await request(base+'/ai/faces')).value;
  assert.equal(coverage.coverage.eligible,1);
  assert.equal(coverage.coverage.completed,1);
  assert.equal(coverage.coverage.failed,0);
  assert.equal(coverage.state,'ready');
  const replay=(await request(base+'/ai/faces/jobs','POST',{mode:'all'},{'Idempotency-Key':key})).value;
  assert.equal(replay.id,job.id);
  console.log('PASS formal HTTP admission, safe fixture inference, persisted coverage, idempotency');
  const personKey = randomUUID();
  const created = await request('/people', 'POST', {name: 'Synthetic person'}, {'Idempotency-Key': personKey});
  const personPath = '/people/' + created.value.id;
  const duplicate = await request('/people', 'POST', {name: 'Synthetic person'}, {'Idempotency-Key': personKey});
  assert.equal(duplicate.value.id, created.value.id);
  assert.equal(duplicate.r.headers.get('idempotency-replayed'), 'true');
  assert.equal((await request('/people')).value.items.length, 1);
  assert.equal((await request(personPath + '/assets')).value.items.length, 0);
  assert.equal((await request(base + '/ai/face-clusters')).value.items.length, 0);
  const renamed = await request(personPath, 'PATCH', {name: 'Synthetic renamed'}, {'If-Match': created.r.headers.get('etag')});
  assert.equal(renamed.value.name, 'Synthetic renamed');
  console.log('PASS production people routes: create, replay, list, rename, empty projections');
  // Freeze only this ephemeral AI container, then kill the app after it claims
  // a job. A real lease expiry must recover the interrupted claim on restart.
  docker('pause',ai); paused=true;
  const interrupted=(await request(base+'/ai/faces/jobs','POST',{mode:'all'},{'Idempotency-Key':randomUUID()})).value;
  await until('claimed',async()=>(await request('/ai/operations/'+interrupted.id)).value,v=>v.state==='running');
  docker('kill',app);
  docker('unpause',ai); paused=false;
  docker('start',app);
  const recovered=await until('lease recovery',async()=>(await request('/ai/operations/'+interrupted.id)).value,v=>['succeeded','failed'].includes(v.state));
  assert.equal(recovered.state,'succeeded',JSON.stringify(recovered));
  const after=(await request(base+'/ai/faces')).value;
  assert.equal(after.activeGenerationId,coverage.activeGenerationId);
  assert.equal(after.coverage.completed,1);
  assert.equal(after.coverage.failed,0);
  const persistedPerson = await request(personPath);
  assert.equal(persistedPerson.value.name, 'Synthetic renamed');
  await request(personPath, 'DELETE', undefined, {'If-Match': persistedPerson.r.headers.get('etag')});
  assert.equal((await request('/people')).value.items.length, 0);
  console.log('PASS person persists across process kill/restart and deletes through production API');
  console.log('PASS killed in-flight app: durable claim recovered after lease expiry, same generation');
  console.log('P1 PASS; synthetic zero-face result only, not P2 nonempty-face or quality acceptance');
} finally {
  if(paused) docker('unpause',ai);
  if(madeApp) { docker('stop',app); docker('rm',app); }
  if(madeAI) { docker('stop',ai); docker('rm',ai); }
  if(madeVolume) docker('volume','rm',volume);
  if(madeNetwork) docker('network','rm',network);
}
