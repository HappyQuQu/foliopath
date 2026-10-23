// Local synthetic deployment/worker test. Only repository media fixtures, no production data or host AI port.
import { execFileSync } from 'node:child_process';
import { readFileSync, mkdtempSync, mkdirSync, copyFileSync, rmSync, renameSync, utimesSync } from 'node:fs';
import { createHash, randomUUID } from 'node:crypto';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
const docker = (...args) => {
  try { return execFileSync('docker', args, { encoding: 'utf8', timeout: 60000 }).trim(); }
  catch (error) { throw new Error('Docker test command failed: '+(error.stdout ?? '')+(error.stderr ?? '')); }
};
const key = `fp-face-${randomUUID().slice(0, 8)}`;
const appImage = docker('image','inspect',process.env.FOLIOPATH_FACE_APP_IMAGE ?? 'foliopath-app:face-worker','--format','{{.Id}}');
const network = key, volume = key + '-data', app = key + '-app', ai = key + '-ai';
const fixtures = mkdtempSync(join(tmpdir(), 'fp-face-fixtures-'));
for (const library of ['one', 'two']) {
  mkdirSync(join(fixtures, library));
  for (const name of ['a.png','b.png']) copyFileSync(fileURLToPath(new URL('../../tests/fixtures/ai-faces/synthetic-pair.png', import.meta.url)), join(fixtures, library, name));
}
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
async function request(path, method = 'GET', body, headers = {}, expectedStatus) {
  const r = await fetch(origin + '/api/v1' + path, { method, signal: AbortSignal.timeout(5000),
    headers: { Origin: origin, Cookie: cookie, 'X-CSRF-Token': csrf, 'Content-Type': 'application/json', ...headers },
    body: body === undefined ? undefined : JSON.stringify(body) });
  assert(expectedStatus ? r.status === expectedStatus : r.ok, `${path}: ${r.status} ${await r.clone().text()}`);
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
function startApp() {
  docker('run', '-d', '--name', app, '--network', network, '--read-only', '--tmpfs', '/tmp',
    '--mount', `type=volume,src=${volume},dst=/app/data`, '-p', '127.0.0.1:6094:8080',
    '--mount', `type=bind,src=${fixtures},dst=/library,readonly`,
    '-e', 'FOLIOPATH_LISTEN=0.0.0.0:8080', '-e', 'FOLIOPATH_AI_DEPLOYMENT=' + JSON.stringify(config), appImage); madeApp = true;
  docker('network', 'connect', 'bridge', app);
}
try {
  docker('network', 'create', '--internal', network); madeNetwork = true;
  docker('volume', 'create', volume); madeVolume = true;
  startApp();
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

  const libs = [];
  for (const rootPath of ['one','two']) {
    const lib = (await request('/libraries','POST',{name:'Synthetic '+rootPath,rootPath},{'Idempotency-Key':randomUUID()})).value.library;
    libs.push(lib);
    const base='/libraries/'+lib.id;
    await until('scan',async()=>(await request(base+'/assets')).value,v=>v.items.length===2);
    const settings=await request(base+'/ai/faces');
    await request(base+'/ai/faces','PATCH',{enabled:true},{'If-Match':settings.r.headers.get('etag')});
    const job=(await request(base+'/ai/faces/jobs','POST',{mode:'all'},{'Idempotency-Key':randomUUID()})).value;
    const done=await until('analysis',async()=>(await request('/ai/operations/'+job.id)).value,v=>['succeeded','failed'].includes(v.state));
    assert.equal(done.state,'succeeded',JSON.stringify(done));
    const assets=(await request(base+'/assets')).value.items;
    for (const asset of assets) {
      const faces=(await request('/assets/'+asset.id+'/faces')).value.items;
      assert.equal(faces.length,2,JSON.stringify(faces));
    }
    const groups=(await request(base+'/ai/face-clusters')).value;
    assert.equal(groups.items.length,2);
    assert(groups.items.every(g=>g.memberCount===2 && g.kind==='core'));
    assert.equal(groups.groupAssignmentAllowed,false);
  }
  console.log('PASS real nonempty inference, observations, two clusters per library');
  const base = '/libraries/'+libs[0].id, other = '/libraries/'+libs[1].id;
  const assets = (await request(base+'/assets')).value.items;
  const tag=(await request('/tags','POST',{name:'Preserved manual tag'})).value;
  const curation=await request('/assets/'+assets[0].id+'/curation');
  await request('/assets/'+assets[0].id+'/tags','PUT',{tagIds:[tag.id]},{'If-Match':curation.r.headers.get('etag')});
  const mediaHash=createHash('sha256').update(readFileSync(join(fixtures,'one','a.png'))).digest('hex');
  const faces = async (asset=assets[0]) => (await request('/assets/'+asset.id+'/faces')).value.items;
  const person = async id => (await request('/people/'+id)).value;
  const review = body => request('/ai/face-reviews','POST',body,{'Idempotency-Key':randomUUID()});
  const undo = event => request('/ai/face-reviews/'+event.value.reviewId+'/undo','POST',{},{
    'Idempotency-Key':randomUUID(), 'If-Match':event.r.headers.get('etag'),
  });
  const done = async operation => {
    const result=await until('operation',async()=>(await request('/ai/operations/'+operation.id)).value,v=>['succeeded','failed','cancelled'].includes(v.state));
    assert.equal(result.state,'succeeded',JSON.stringify(result));
  };
  const analyze = async (path=base,mode='all') => done((await request(path+'/ai/faces/jobs','POST',{mode},{'Idempotency-Key':randomUUID()})).value);
  const alice=(await request('/people','POST',{name:'Synthetic A'},{'Idempotency-Key':randomUUID()})).value;
  const bob=(await request('/people','POST',{name:'Synthetic B'},{'Idempotency-Key':randomUUID()})).value;
  const groups=(await request(base+'/ai/face-clusters')).value.items;
  await request('/people','POST',{name:'Forbidden automatic group',sourceClusterId:groups[0].id,expectedClusterRevision:groups[0].revision},{'Idempotency-Key':randomUUID()},409);
  await request('/ai/face-reviews','POST',{action:'assign_cluster',clusterId:groups[0].id,personId:alice.id,expectedClusterRevision:groups[0].revision,expectedPersonRevision:alice.revision},{'Idempotency-Key':randomUUID()},409);
  const assign = async (f,id) => review({action:'assign_face',faceId:f.faceId,expectedFaceRevision:f.revision,personId:id,expectedPersonRevision:(await person(id)).revision});
  for (const asset of assets) {
    const observed=await faces(asset);
    await assign(observed[0],alice.id);
    await assign(observed[1],bob.id);
  }
  assert.equal((await person(alice.id)).confirmedFaceCount,2);
  assert.equal((await request('/people/'+alice.id+'/assets')).value.items.length,2);
  let observed=await faces();
  const excluded=await review({action:'exclude_face',faceId:observed[0].faceId,expectedFaceRevision:observed[0].revision});
  assert.equal((await person(alice.id)).confirmedFaceCount,1);
  await undo(excluded);
  assert.equal((await person(alice.id)).confirmedFaceCount,2);
  observed=await faces();
  const split=await review({action:'split_face',faceId:observed[0].faceId,expectedFaceRevision:observed[0].revision,sourcePersonId:alice.id,expectedSourceRevision:(await person(alice.id)).revision});
  await undo(split);
  const merged=await review({action:'merge_people',sourcePersonId:bob.id,targetPersonId:alice.id,expectedSourceRevision:(await person(bob.id)).revision,expectedTargetRevision:(await person(alice.id)).revision,conflictsAcknowledged:true});
  assert.equal((await person(alice.id)).confirmedFaceCount,4);
  await undo(merged);
  assert.equal((await person(alice.id)).confirmedFaceCount,2);
  observed=await faces();
  await review({action:'cannot_link',leftFaceId:observed[0].faceId,rightFaceId:observed[1].faceId,expectedLeftRevision:observed[0].revision,expectedRightRevision:observed[1].revision});
  await request('/ai/face-reviews','POST',{action:'merge_people',sourcePersonId:bob.id,targetPersonId:alice.id,expectedSourceRevision:(await person(bob.id)).revision,expectedTargetRevision:(await person(alice.id)).revision,conflictsAcknowledged:true},{'Idempotency-Key':randomUUID()},409);
  console.log('PASS individual assignment, projections, exclusion/split/merge undo, cannot-link, server group gate');
  await analyze(base,'missing');
  await analyze();
  assert.equal((await person(alice.id)).confirmedFaceCount,2);
  let settings=await request(base+'/ai/faces');
  await request(base+'/ai/faces','PATCH',{enabled:false},{'If-Match':settings.r.headers.get('etag')});
  settings=await request(base+'/ai/faces');
  await request(base+'/ai/faces','PATCH',{enabled:true},{'If-Match':settings.r.headers.get('etag')});
  await analyze(base,'missing');
  assert.equal((await person(alice.id)).confirmedFaceCount,2);
  settings=await request(base+'/ai/faces');
  await done((await request(base+'/ai/faces/derived-clear','POST',{confirmation:'clear_derived_face_data'},{'Idempotency-Key':randomUUID(),'If-Match':settings.r.headers.get('etag')})).value);
  await request('/assets/'+assets[0].id+'/faces','GET',undefined,{},409);
  assert.equal((await request(other+'/ai/face-clusters')).value.items.length,2);
  settings=await request(base+'/ai/faces');
  await request(base+'/ai/faces','PATCH',{enabled:true},{'If-Match':settings.r.headers.get('etag')});
  await analyze();
  assert.equal((await person(alice.id)).confirmedFaceCount,2);
  observed=await faces();
  await request('/ai/face-reviews','POST',{action:'merge_people',sourcePersonId:bob.id,targetPersonId:alice.id,expectedSourceRevision:(await person(bob.id)).revision,expectedTargetRevision:(await person(alice.id)).revision,conflictsAcknowledged:true},{'Idempotency-Key':randomUUID()},409);
  console.log('PASS missing/all rebuild, disable/enable, derived clear preserves manual relationships and cannot-link, library isolation');

  const previousGeneration=(await request(base+'/ai/faces')).value.activeGenerationId;
  const previousOther=(await request(other+'/ai/faces')).value.activeGenerationId;
  const previousOtherGroups=(await request(other+'/ai/face-clusters')).value.items.map(g=>g.id);
  docker('kill',app); docker('rm',app); madeApp=false;
  config.face.threshold.profileId='synthetic-business-v2';
  startApp();
  await until('new deployment active',models,items=>items.filter(m=>m.active && m.state==='available').length===2 && items.length===3);
  assert.deepEqual((await request(other+'/ai/face-clusters')).value.items.map(g=>g.id),previousOtherGroups);
  assert.equal((await person(alice.id)).confirmedFaceCount,2);
  docker('pause',ai); paused=true;
  const switching=(await request(base+'/ai/faces/jobs','POST',{mode:'all'},{'Idempotency-Key':randomUUID()})).value;
  await until('switch claimed',async()=>(await request('/ai/operations/'+switching.id)).value,v=>v.state==='running');
  assert.equal((await faces()).length,2);
  assert.deepEqual((await request(other+'/ai/face-clusters')).value.items.map(g=>g.id),previousOtherGroups);
  const running=await request('/ai/operations/'+switching.id);
  await request('/ai/operations/'+switching.id+'/cancel','POST',{}, {'If-Match':running.r.headers.get('etag')});
  docker('unpause',ai); paused=false;
  const cancelled=await until('cancel',async()=>(await request('/ai/operations/'+switching.id)).value,v=>['cancelled','failed','succeeded'].includes(v.state));
  assert.equal(cancelled.state,'cancelled',JSON.stringify(cancelled));
  await analyze();
  assert.notEqual((await request(base+'/ai/faces')).value.activeGenerationId,previousGeneration);
  assert.equal((await request(other+'/ai/faces')).value.activeGenerationId,previousOther);
  assert.equal((await person(alice.id)).confirmedFaceCount,2);
  assert((await faces()).every(f=>f.state==='assigned'));
  assert.deepEqual((await request(other+'/ai/face-clusters')).value.items.map(g=>g.id),previousOtherGroups);
  await analyze(other);
  assert.equal((await request(other+'/ai/face-clusters')).value.items.length,2);
  docker('stop',ai);
  await until('offline',models,items=>items.filter(m=>m.active).every(m=>m.state==='unavailable'));
  assert.equal((await faces()).length,2);
  assert.equal((await request(other+'/ai/face-clusters')).value.items.length,2);
  docker('start',ai);
  await until('online',models,items=>items.filter(m=>m.active).every(m=>m.state==='available'));
  console.log('PASS generation retarget, cancel, old per-library visible results, exact manual rebind, offline reads and recovery');
  const otherAssets=(await request(other+'/assets')).value.items;
  for(let n=0;n<51;n++) await request('/people','POST',{name:'Pagination '+String(n).padStart(3,'0')},{'Idempotency-Key':randomUUID()});
  const runBrowser = (offline=false) => docker('run','--rm','--network',network,
    '-v',fileURLToPath(new URL('../../web/tests/e2e',import.meta.url))+':/src/web/tests/e2e:ro',
    '-v',fileURLToPath(new URL('../../test-results/face-real/'+(offline?'offline':'workflow'),import.meta.url))+':/src/test-results/playwright',
    '-e','FOLIOPATH_WEB_E2E_URL=http://'+app+':8080',
    '-e','FOLIOPATH_FACE_REAL_E2E=1',
    '-e','FOLIOPATH_FACE_OFFLINE_E2E='+(offline?'1':'0'),
    '-e','FOLIOPATH_FACE_LIBRARY_ID='+libs[1].id,
    '-e','FOLIOPATH_FACE_ASSET_ID='+otherAssets[0].id,
    '-e','FOLIOPATH_FACE_PASSWORD='+password,
    'foliopath-face-browser:local','npx','playwright','test','face-real.spec.ts','--project=chromium');
  runBrowser();
  console.log('PASS browser nonempty face workflow');
  settings=await request(base+'/ai/faces');
  await done((await request(base+'/ai/faces/manual-clear','POST',{confirmation:'clear_manual_face_relationships',expectedPersonCount:2,expectedAssignmentCount:4,expectedConstraintCount:1},{'Idempotency-Key':randomUUID(),'If-Match':settings.r.headers.get('etag')})).value);
  assert.equal((await person(alice.id)).confirmedFaceCount,0);
  assert.equal((await person(bob.id)).confirmedFaceCount,0);
  assert.equal((await request(other+'/assets')).value.items.length,2);
  assert.equal((await request(base+'/assets')).value.items.length,2);
  assert.equal((await request('/assets/'+assets[0].id+'/curation')).value.tags[0].id,tag.id);
  assert.equal(createHash('sha256').update(readFileSync(join(fixtures,'one','a.png'))).digest('hex'),mediaHash);
  console.log('PASS manual clear limited to requested library; original synthetic files remain');
  renameSync(join(fixtures,'two'),join(fixtures,'temporarily-offline'));
  await request(other+'/scans','POST');
  await until('library offline',async()=>(await request('/libraries')).value.items.find(l=>l.id===libs[1].id),l=>l.status==='offline');
  await request(other+'/ai/faces/jobs','POST',{mode:'all'},{'Idempotency-Key':randomUUID()},409);
  runBrowser(true);
  assert.equal((await request(base+'/assets')).value.items.length,2);
  renameSync(join(fixtures,'temporarily-offline'),join(fixtures,'two'));
  await request(other+'/scans','POST');
  await until('library restored',async()=>(await request('/libraries')).value.items.find(l=>l.id===libs[1].id),l=>l.status==='ready');
  assert.equal((await request(other+'/assets')).value.items.length,2);
  assert.equal((await request(other+'/ai/face-clusters')).value.items.length,2);
  console.log('PASS offline media root preserves index and other library, restoration retains results');
  const rescan = async () => {
    const scan=(await request(other+'/scans','POST')).value;
    const result=await until('source rescan',async()=>(await request('/scans/'+scan.id)).value,v=>!['queued','running','cancelling'].includes(v.status));
    assert.equal(result.status,'succeeded',JSON.stringify(result));
  };
  utimesSync(join(fixtures,'two','a.png'),new Date(),new Date(Date.now()+10000));
  await rescan();
  await analyze(other,'missing');
  const changed=(await request(other+'/assets')).value.items.find(a=>a.name==='a.png');
  assert((await request('/assets/'+changed.id+'/faces')).value.items.every(f=>f.state!=='assigned'));
  renameSync(join(fixtures,'two','b.png'),join(fixtures,'removed-synthetic.png'));
  await rescan();
  assert.equal((await request(other+'/assets')).value.items.length,1);
  await analyze(other);
  assert.equal((await request(base+'/assets')).value.items.length,2);
  assert.equal((await request('/assets/'+assets[0].id+'/curation')).value.tags[0].id,tag.id);
  console.log('PASS changed source requires review, removed synthetic asset reconciles, other library/manual tags preserved');
} finally {
  if(paused) docker('unpause',ai);
  if(madeApp) { docker('stop',app); docker('rm',app); }
  if(madeAI) { docker('stop',ai); docker('rm',ai); }
  if(madeVolume) docker('volume','rm',volume);
  if(madeNetwork) docker('network','rm',network);
  rmSync(fixtures,{recursive:true,force:true});
}
