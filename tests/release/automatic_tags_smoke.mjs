// CR-2026-031 T3/T4: actual local two-service Compose and synthetic media only.
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {copyFileSync, mkdirSync, mkdtempSync, readFileSync, realpathSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createHash, randomUUID} from 'node:crypto';

const repo = fileURLToPath(new URL('../../', import.meta.url));
const run = (...args) => execFileSync('docker', args, {encoding:'utf8',timeout:60000}).trim();
for (const name of ['FOLIOPATH_TEST_APP_IMAGE','FOLIOPATH_TEST_AI_IMAGE','FOLIOPATH_TEST_MODEL_CACHE']) assert(process.env[name], `${name} required`);
const context = JSON.parse(run('context','inspect'))[0];
assert(context.Endpoints.docker.Host.startsWith('unix://'), 'local Docker only');
const appImage = run('image','inspect',process.env.FOLIOPATH_TEST_APP_IMAGE,'--format','{{.Id}}');
const aiImage = run('image','inspect',process.env.FOLIOPATH_TEST_AI_IMAGE,'--format','{{.Id}}');
// Expected identities come from trusted image files, never from the remote HTTP response.
const identity = JSON.parse(run('run','--rm','--network','none','--entrypoint','python',aiImage,'-m','image_identity'));
const tagIdentity = JSON.parse(run('run','--rm','--network','none','--entrypoint','python',aiImage,'-m','image_identity','--tag'));
const deployment = {origin:'http://machine-learning:3003',identity,model_bytes:610841665,license_id:'unreviewed-local-test',tag:{identity:tagIdentity}};
const project = 'fp-auto-tags-' + randomUUID().slice(0,8);
const root = realpathSync(mkdtempSync(join(tmpdir(),project+'-')));
for (const directory of ['data','media/alpha','media/beta']) mkdirSync(join(root,directory),{recursive:true});
const files = [];
for (let index=0; index<4; index++) {
  const target = `alpha/person-${index}.png`;
  copyFileSync(join(repo,'tests/fixtures/ai-faces/synthetic-pair.png'),join(root,'media',target)); files.push(target);
}
copyFileSync(join(repo,'tests/fixtures/media/viewer-blue-violet.png'),join(root,'media/alpha/control.png')); files.push('alpha/control.png');
copyFileSync(join(repo,'tests/fixtures/media/range-video.mp4'),join(root,'media/alpha/movie.mp4')); files.push('alpha/movie.mp4');
copyFileSync(join(repo,'tests/fixtures/ai-faces/synthetic-pair.png'),join(root,'media/beta/person.png')); files.push('beta/person.png');
const hashes = () => files.map(name => createHash('sha256').update(readFileSync(join(root,'media',name))).digest('hex'));
const beforeHashes = hashes();
const env = {...process.env,FOLIOPATH_IMAGE:appImage,FOLIOPATH_AI_IMAGE:aiImage,FOLIOPATH_LIBRARY_PATH:join(root,'media'),
  FOLIOPATH_DATA_PATH:join(root,'data'),FOLIOPATH_MODEL_CACHE_PATH:realpathSync(process.env.FOLIOPATH_TEST_MODEL_CACHE),
  FOLIOPATH_BIND_ADDRESS:'127.0.0.1',FOLIOPATH_PORT:'0',FOLIOPATH_AI_DEPLOYMENT:JSON.stringify(deployment)};
const compose = (...args) => execFileSync('docker',['compose','--env-file','/dev/null','-p',project,'-f',join(repo,'compose.ai.yaml'),...args],{env,encoding:'utf8',timeout:60000}).trim();
const inspect = id => JSON.parse(run('inspect',id))[0];
const credentials = {username:'autotag-test',password:'Synthetic-Automatic-Tags-2026!'};
let origin, cookie='', csrf='', appID, aiID;
async function until(label,read,predicate,timeout=180000) {
  const deadline=Date.now()+timeout;
  let last;
  while(Date.now()<deadline) { last=await read(); if(predicate(last)) return last; await new Promise(r=>setTimeout(r,500)); }
  throw Error(label+' timed out: '+JSON.stringify(last));
}
async function request(path,method='GET',body,headers={}) {
  return fetch(origin+'/api/v1'+path,{method,signal:AbortSignal.timeout(35000),headers:{'Content-Type':'application/json',Origin:origin,Cookie:cookie,'X-CSRF-Token':csrf,...headers},body:body===undefined?undefined:JSON.stringify(body)});
}
async function json(path,method,body,headers) {
  const response=await request(path,method,body,headers), value=await response.json();
  assert(response.ok,`${method??'GET'} ${path}: ${response.status} ${JSON.stringify(value)}`); return value;
}
function refreshOrigin() { const port=inspect(appID).NetworkSettings.Ports['8080/tcp'][0]; assert.equal(port.HostIp,'127.0.0.1'); origin='http://127.0.0.1:'+port.HostPort; }
async function login() {
  const response=await request('/auth/login','POST',credentials); assert(response.ok);
  cookie=response.headers.get('set-cookie').split(';')[0]; csrf=(await response.json()).csrfToken;
}
async function settings(library) { const response=await request(`/libraries/${library}/ai/automatic-tags`); assert(response.ok); return {body:await response.json(),etag:response.headers.get('etag')}; }
async function enable(library,enabled) { const {etag}=await settings(library); return json(`/libraries/${library}/ai/automatic-tags`,'PUT',{enabled},{'If-Match':etag}); }
async function state(asset) { return json(`/assets/${asset}/curation`); }
async function manual(asset,tag) { const current=await state(asset); return json(`/assets/${asset}/tags/${tag}`,'PUT',undefined,{'If-Match':`"curation-r${current.revision}"`}); }
async function replace(asset,tagIds) { const current=await state(asset); return json(`/assets/${asset}/tags`,'PUT',{tagIds},{'If-Match':`"curation-r${current.revision}"`}); }
async function start(library,mode='all',key=randomUUID()) { return json(`/libraries/${library}/ai/automatic-tags/jobs`,'POST',{mode,...(mode==='all'?{confirmation:'reprocess_automatic_tags'}:{})},{'Idempotency-Key':key}); }
async function finish(operation,expected='succeeded') {
  const result=await until('operation '+operation.id,()=>json(`/ai/operations/${operation.id}`),o=>!['queued','running','cancelling'].includes(o.state));
  assert.equal(result.state,expected,JSON.stringify(result));
  if (expected==='succeeded') assert.equal(result.totalItems,result.completedItems);
  return result;
}
async function inferenceRunning(operation) {
  await until('real inference for '+operation.id,async()=>({
    operation:await json(`/ai/operations/${operation.id}`),
    processes:run('top',aiID,'-eo','pid,args'),
  }),s=>s.operation.state==='running'&&s.processes.includes('inference_worker'));
}
async function remainsStopped(library,seconds=35) {
  const before=(await settings(library)).body;
  const deadline=Date.now()+seconds*1000;
  while(Date.now()<deadline) {
    const current=(await settings(library)).body;
    assert.equal(current.activeOperationId,null,'terminal failure/cancel must not automatically retry');
    assert.deepEqual(current.coverage,before.coverage,'no late inference commits after terminal operation');
    await new Promise(resolve=>setTimeout(resolve,1000));
  }
}
async function tagSnapshot(assets) {
  const snapshot=[];
  for(const asset of assets) snapshot.push({assetId:asset.id,tags:(await state(asset.id)).tags});
  return snapshot;
}
async function covered(library,count) { return until('coverage',async()=>(await settings(library)).body,s=>s.coverage.completed===count&&!s.activeOperationId); }
try {
  const config=JSON.parse(compose('config','--format','json'));
  assert.deepEqual(Object.keys(config.services).sort(),['foliopath','machine-learning']);
  assert.equal(Number(config.services['machine-learning'].mem_limit),2304*1024**2);
  assert(!config.services['machine-learning'].ports?.length);
  compose('up','-d','--no-build','--pull','never'); appID=compose('ps','-q','foliopath'); aiID=compose('ps','-q','machine-learning'); refreshOrigin();
  await until('app health',()=>inspect(appID),s=>s.State.Health?.Status==='healthy');
  await until('AI health',()=>inspect(aiID),s=>s.State.Health?.Status==='healthy');
  assert.equal(inspect(aiID).HostConfig.MemorySwap,2304*1024**2);
  assert.equal(inspect(appID).Mounts.find(m=>m.Destination==='/library').RW,false);
  assert.deepEqual(inspect(aiID).Mounts.map(m=>m.Destination),['/cache']);
  assert.equal((await request('/libraries/lib_1/ai/automatic-tags')).status,401);
  await json('/auth/setup','POST',{...credentials,displayName:'Synthetic automatic tags'}); await login();
  const createLibrary=async name=>{
    const result=await json('/libraries','POST',{name,rootPath:name},{'Idempotency-Key':randomUUID()});
    const scan=await until('scan',()=>json(`/scans/${result.scan.id}`),s=>!['queued','running'].includes(s.status)); assert.equal(scan.status,'succeeded');
    return {id:result.library.id,assets:(await json(`/libraries/${result.library.id}/assets`)).items};
  };
  const alpha=await createLibrary('alpha'),beta=await createLibrary('beta');
  const photo=alpha.assets.find(a=>a.name==='person-0.png'),video=alpha.assets.find(a=>a.kind==='video'); assert(photo&&video);
  const keep=await json('/tags','POST',{name:'人工保留'}); await manual(photo.id,keep.id);
  assert.equal((await settings(alpha.id)).body.enabled,false);
  const noCSRF=await request(`/libraries/${alpha.id}/ai/automatic-tags`,'PUT',{enabled:true},{'If-Match':(await settings(alpha.id)).etag,'X-CSRF-Token':''}); assert.equal(noCSRF.status,403);
  const stale=(await settings(alpha.id)).etag; await enable(alpha.id,true);
  assert.equal((await request(`/libraries/${alpha.id}/ai/automatic-tags`,'PUT',{enabled:false},{'If-Match':stale})).status,412);
  const coverage=await covered(alpha.id,5); assert.equal(coverage.coverage.eligible,5); assert.equal(coverage.coverage.failed,0);
  const initial=await state(photo.id); assert(initial.tags.some(t=>t.id===keep.id&&t.source==='manual'));
  const automatic=initial.tags.find(t=>t.source==='ai'); assert(automatic,'real model must produce a nonempty tag on the synthetic photo');
  assert.equal((await state(video.id)).tags.length,0); assert.equal((await state(beta.assets[0].id)).tags.length,0);
  const cloud=await json(`/tags?libraryId=${alpha.id}&kind=image,animated`); assert(cloud.items.some(t=>t.id===automatic.id&&t.assetCount>0));
  console.log('PASS real automatic inference, manual priority, image-only counts and library isolation');

  await replace(photo.id,initial.tags.filter(t=>t.id!==automatic.id).map(t=>t.id));
  const key=randomUUID(), job=await start(alpha.id,'all',key), replay=await start(alpha.id,'all',key); assert.equal(job.id,replay.id); await finish(job);
  let current=await state(photo.id); assert(!current.tags.some(t=>t.name===automatic.name),'manual removal must survive a rerun'); assert(current.tags.some(t=>t.id===keep.id&&t.source==='manual'));
  await manual(photo.id,automatic.id); assert((await state(photo.id)).tags.some(t=>t.id===automatic.id&&t.source==='manual'));
  await finish(await start(alpha.id)); current=await state(photo.id); assert(current.tags.some(t=>t.id===automatic.id&&t.source==='manual'));
  console.log('PASS removal suppression, explicit manual restore, rerun protection and idempotency');

  // Restart with a durable active job; no direct database writes or fake predictions.
  const resumable=await start(alpha.id); await inferenceRunning(resumable); compose('restart','foliopath');
  await until('restart health',()=>inspect(appID),s=>s.State.Health?.Status==='healthy'); refreshOrigin(); cookie='';csrf='';await login();
  const restartedAt=Date.parse(inspect(appID).State.StartedAt);
  const resumed=await finish(resumable);
  assert(Number.isFinite(restartedAt)&&Date.parse(resumed.updatedAt)>=restartedAt,'operation completed before the new app process started; restart recovery was not exercised');
  assert.equal(resumed.completedItems,5,'restart must finish all five eligible images');
  assert.equal(resumed.totalItems,5);
  assert((await state(photo.id)).tags.some(t=>t.id===automatic.id&&t.source==='manual'));
  const cancellable=await start(alpha.id);
  await inferenceRunning(cancellable);
  const beforeCancellation=await tagSnapshot([...alpha.assets,...beta.assets]);
  const opResponse=await request(`/ai/operations/${cancellable.id}`); assert(opResponse.ok); const opETag=opResponse.headers.get('etag');
  await json(`/ai/operations/${cancellable.id}/cancel`,'POST',undefined,{'If-Match':opETag}); await finish(cancellable,'cancelled');
  await remainsStopped(alpha.id);
  assert.deepEqual(await tagSnapshot([...alpha.assets,...beta.assets]),beforeCancellation,'cancellation must preserve every alpha/beta asset tag set, including video and the disabled library');
  console.log('PASS running inference restart recovery, cancellation fence and no automatic retry');

  const unavailable=await start(alpha.id); await inferenceRunning(unavailable); compose('stop','machine-learning');
  await finish(unavailable,'failed');
  const failureTags=(await state(photo.id)).tags;
  compose('start','machine-learning'); await until('restored AI health',()=>inspect(aiID),s=>s.State.Health?.Status==='healthy');
  await remainsStopped(alpha.id);
  assert.deepEqual((await state(photo.id)).tags,failureTags,'failed inference must preserve existing tags');
  await finish(await start(alpha.id));
  assert((await state(photo.id)).tags.some(t=>t.id===automatic.id&&t.source==='manual'));
  console.log('PASS real service interruption preserves tags, does not retry forever, and explicit reprocessing recovers');
  await enable(alpha.id,false); const saved=(await state(photo.id)).tags;
  compose('restart','foliopath'); await until('disabled restart health',()=>inspect(appID),s=>s.State.Health?.Status==='healthy'); refreshOrigin();cookie='';csrf='';await login();
  assert.equal((await settings(alpha.id)).body.enabled,false); assert.deepEqual((await state(photo.id)).tags,saved);
  assert.deepEqual(hashes(),beforeHashes); assert.equal(inspect(aiID).State.OOMKilled,false); assert.equal(inspect(appID).State.OOMKilled,false);
  console.log('PASS restart recovery, cancellation, disable persistence, manual state and original hashes unchanged');
  console.log(JSON.stringify({passed:true,project,root,origin,appImage,aiImage,library:alpha.id,photo:photo.id,aiMemoryLimit:inspect(aiID).HostConfig.Memory}));
} finally {
  if(process.env.FOLIOPATH_TEST_KEEP==='1') console.log(`Kept isolated test project ${project} at ${root}`);
  else {
    for(const id of [appID,aiID].filter(Boolean)) assert.equal(inspect(id).Config.Labels['com.docker.compose.project'],project,'refusing unrelated cleanup');
    compose('down','--remove-orphans'); console.log(`Removed isolated project ${project}; synthetic data retained at ${root}`);
  }
}
