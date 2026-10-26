// CR-2026-033 / ADR-0026. Real pinned models, isolated local Docker, synthetic media.
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {copyFileSync,mkdirSync,mkdtempSync,readFileSync,realpathSync,writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createHash,randomUUID} from 'node:crypto';

const repo=fileURLToPath(new URL('../../',import.meta.url));
const docker=(...args)=>execFileSync('docker',args,{encoding:'utf8',timeout:60000}).trim();
assert(JSON.parse(docker('context','inspect'))[0].Endpoints.docker.Host.startsWith('unix://'));
for(const key of ['FOLIOPATH_TEST_APP_IMAGE','FOLIOPATH_TEST_AI_IMAGE','FOLIOPATH_TEST_MODEL_CACHE'])assert(process.env[key],key);
const appImage=docker('image','inspect',process.env.FOLIOPATH_TEST_APP_IMAGE,'--format','{{.Id}}');
const aiImage=docker('image','inspect',process.env.FOLIOPATH_TEST_AI_IMAGE,'--format','{{.Id}}');
const imageJSON=(...args)=>JSON.parse(docker('run','--rm','--network','none','--entrypoint','python',aiImage,...args));
const identity=imageJSON('-m','multilingual_contract'), tag=imageJSON('-m','photo_policy');
const legacyClasses=new Set(imageJSON('-c','import json,tag_contract; print(json.dumps(sorted(tag_contract.SAFE_LABELS)))'));
const legacy=imageJSON('-m','image_identity'), oldTag=imageJSON('-m','image_identity','--tag');
assert.equal(identity.runtime_contract_sha256,'81ec3de561ee05318103ba0f354b6472cb2d1dc0d2ef0f33bf2becee2c710659');
assert.equal(legacy.runtime_contract_sha256,'be1d1baaccb2f8532d9df872483b0453e66e1221d5d1768432357dc76c1cfc8a');
assert.notEqual(tag.policy_sha256,oldTag.policy_sha256);assert.notEqual(tag.runtime_contract_sha256,oldTag.runtime_contract_sha256);
assert.equal(tag.model_sha256,oldTag.model_sha256);assert.equal(tag.cache_manifest_sha256,oldTag.cache_manifest_sha256);
const manifest=imageJSON('-c','from pathlib import Path; print(Path("/opt/foliopath-inference/cache-manifest.json").read_text())');
const weights=['detection','recognition'].map(k=>manifest.find(e=>e.path===`facial-recognition/buffalo_l/${k}/model.onnx`));assert(weights.every(Boolean));
const deployment={origin:'http://machine-learning:3003',identity,model_bytes:2729566266,license_id:'unreviewed-local-test',tag:{identity:tag},
  face:{identity:{...legacy,protocol:'foliopath-face-v1',model:'buffalo_l'},detector_sha256:weights[0].sha256,embedder_sha256:weights[1].sha256,
    model_bytes:weights.reduce((s,e)=>s+e.size,0),license_id:'unreviewed-local-test',threshold:{schemaVersion:1,profileId:'localized-tags-synthetic-only',coreSimilarity:.8,edgeSimilarity:.6,minCoreSize:2,qualitySummarySha256:createHash('sha256').update('NOT quality calibration').digest('hex')}}};
const project='fp-localized-tags-'+randomUUID().slice(0,8),root=realpathSync(mkdtempSync(join(tmpdir(),project+'-')));
for(const path of ['data','media/alpha','media/beta'])mkdirSync(join(root,path),{recursive:true});
for(const [source,target]of [['ai-faces/synthetic-pair.png','alpha/people.png'],['media/viewer-blue-violet.png','alpha/blue.png'],['ai-faces/synthetic-pair.png','beta/people.png']])copyFileSync(join(repo,'tests/fixtures',source),join(root,'media',target));
writeFileSync(join(root,'media/alpha/broken.jpg'),'Synthetic invalid image');
const paths=['alpha/people.png','alpha/blue.png','alpha/broken.jpg','beta/people.png'];
const hashes=()=>paths.map(p=>createHash('sha256').update(readFileSync(join(root,'media',p))).digest('hex'));
const before=hashes();
const env={...process.env,FOLIOPATH_IMAGE:appImage,FOLIOPATH_AI_IMAGE:aiImage,FOLIOPATH_LIBRARY_PATH:join(root,'media'),FOLIOPATH_DATA_PATH:join(root,'data'),FOLIOPATH_MODEL_CACHE_PATH:realpathSync(process.env.FOLIOPATH_TEST_MODEL_CACHE),FOLIOPATH_BIND_ADDRESS:'127.0.0.1',FOLIOPATH_PORT:process.env.FOLIOPATH_TEST_PORT??'0',FOLIOPATH_AI_DEPLOYMENT:JSON.stringify(deployment)};
const compose=(...args)=>execFileSync('docker',['compose','--env-file','/dev/null','-p',project,'-f',join(repo,'compose.ai.yaml'),'-f',join(repo,'compose.ai.multilingual.yaml'),'-f',join(repo,'tests/release/compose.ai.multilingual-test.yaml'),...args],{env,encoding:'utf8',timeout:60000}).trim();
const inspect=id=>JSON.parse(docker('inspect',id))[0];
let app,ai,origin,cookie='',csrf='';
const credentials={username:'localized-tag-test',password:'Synthetic-Localized-Tags-2026!'};
async function until(label,read,ready,timeout=240000){let last;const end=Date.now()+timeout;while(Date.now()<end){last=await read();if(ready(last))return last;await new Promise(r=>setTimeout(r,500));}throw Error(label+': '+JSON.stringify(last));}
function refresh(){const port=inspect(app).NetworkSettings.Ports['8080/tcp'][0];assert.equal(port.HostIp,'127.0.0.1');origin='http://127.0.0.1:'+port.HostPort;}
async function request(path,method='GET',body,headers={}){return fetch(origin+'/api/v1'+path,{method,signal:AbortSignal.timeout(35000),headers:{Origin:origin,Cookie:cookie,'X-CSRF-Token':csrf,'Content-Type':'application/json',...headers},body:body===undefined?undefined:JSON.stringify(body)});}
async function json(path,method,body,headers){const r=await request(path,method,body,headers),value=await r.json();assert(r.ok,`${path}: ${r.status} ${JSON.stringify(value)}`);return value;}
async function login(){const r=await request('/auth/login','POST',credentials);assert(r.ok);cookie=r.headers.get('set-cookie').split(';')[0];csrf=(await r.json()).csrfToken;}
async function enable(lib){const r=await request(`/libraries/${lib}/ai/automatic-tags`);assert(r.ok);return json(`/libraries/${lib}/ai/automatic-tags`,'PUT',{enabled:true},{'If-Match':r.headers.get('etag')});}
async function settings(lib){return json(`/libraries/${lib}/ai/automatic-tags`);}
async function covered(lib){return until('automatic coverage',()=>settings(lib),s=>s.coverage.completed===2&&!s.activeOperationId);}
async function state(asset){return json(`/assets/${asset}/curation`);}
async function manual(asset,tagId){const current=await state(asset);return json(`/assets/${asset}/tags/${tagId}`,'PUT',undefined,{'If-Match':`"curation-r${current.revision}"`});}
async function remove(asset,tagId){const current=await state(asset);return json(`/assets/${asset}/tags`,'PUT',{tagIds:current.tags.filter(t=>t.id!==tagId).map(t=>t.id)},{'If-Match':`"curation-r${current.revision}"`});}
async function rerun(lib){const op=await json(`/libraries/${lib}/ai/automatic-tags/jobs`,'POST',{mode:'all',confirmation:'reprocess_automatic_tags'},{'Idempotency-Key':randomUUID()});const result=await until('rerun',()=>json(`/ai/operations/${op.id}`),s=>!['queued','running','cancelling'].includes(s.state));assert.equal(result.state,'succeeded',JSON.stringify(result));}
async function runCapability(lib,capability){const r=await request(`/libraries/${lib}/ai/${capability}`);assert(r.ok);await json(`/libraries/${lib}/ai/${capability}`,'PATCH',{enabled:true},{'If-Match':r.headers.get('etag')});const op=await json(`/libraries/${lib}/ai/${capability}/jobs`,'POST',{mode:'missing'},{'Idempotency-Key':randomUUID()});const done=await until(capability,()=>json(`/ai/operations/${op.id}`),s=>!['queued','running','cancelling'].includes(s.state));assert.equal(done.state,'succeeded',JSON.stringify(done));const value=await json(`/libraries/${lib}/ai/${capability}`);assert.equal(value.coverage.completed,2);assert.equal(value.coverage.failed,1);}
try{
  compose('up','-d','--pull','never');app=compose('ps','-q','foliopath');ai=compose('ps','-q','machine-learning');refresh();
  await until('AI health',()=>inspect(ai),s=>s.State.Health?.Status==='healthy');await until('app health',()=>inspect(app),s=>s.State.Health?.Status==='healthy');
  assert.equal(inspect(ai).HostConfig.Memory,4096*1024**2);assert.equal(inspect(ai).NetworkSettings.Ports['3003/tcp'],undefined);
  const mounts=inspect(ai).Mounts;assert.equal(mounts.find(m=>m.Destination==='/cache').RW,false);assert(!mounts.some(m=>['/library','/app/data','/var/run/docker.sock'].includes(m.Destination)));
  assert.equal((await request('/tags?q='+encodeURIComponent('汽车'))).status,401);
  await json('/auth/setup','POST',{...credentials,displayName:'Synthetic localized tags'});await login();
  await until('models',()=>json('/ai/models'),v=>v.items.some(m=>m.active&&m.state==='available'));
  async function create(name){const value=await json('/libraries','POST',{name,rootPath:name},{'Idempotency-Key':randomUUID()});await until('scan',()=>json(`/scans/${value.scan.id}`),s=>s.status==='succeeded');return value.library.id;}
  const alpha=await create('alpha'),beta=await create('beta');
  const photo=(await json(`/libraries/${alpha}/assets`)).items.find(a=>a.name==='people.png');assert(photo);
  const keep=await json('/tags','POST',{name:'人工中文保留'});await manual(photo.id,keep.id);
  await runCapability(alpha,'semantic');await runCapability(alpha,'faces');
  assert.equal((await json(`/semantic/assets?libraryId=${alpha}&q=${encodeURIComponent('人物')}`)).items.length,2);
  console.log('PASS unchanged 640-dimensional Chinese retrieval and legacy face inference through the shared scheduler');
  await enable(alpha);const coverage=await covered(alpha);assert.equal(coverage.coverage.failed,1);
  let current=await state(photo.id);const automatic=current.tags.find(t=>t.source==='ai'&&t.localization);assert(automatic,'real tag must expose category metadata');
  assert(current.tags.some(t=>t.source==='ai'&&!legacyClasses.has(t.name)),'synthetic fixture must exercise an expanded class, not only legacy output');
  assert.equal(automatic.name,automatic.localization.key);assert(automatic.localization.zhCN);assert(automatic.localization.en);
  const search=await json(`/tags?libraryId=${alpha}&kind=image,animated&q=${encodeURIComponent(automatic.localization.zhCN)}`);
  assert(search.items.some(t=>t.id===automatic.id&&t.name===automatic.name));
  assert.equal((await json(`/tags?libraryId=${beta}&q=${encodeURIComponent(automatic.localization.zhCN)}`)).items.length,0);
  const literal=current.tags.find(t=>t.id===keep.id);assert.equal(literal.name,'人工中文保留');assert(!literal.localization);
  console.log('PASS real new-policy inference, scoped Chinese aliases, stable original names/IDs and manual text');
  const chinese=await json('/tags','POST',{name:automatic.localization.zhCN});await manual(photo.id,chinese.id);
  await rerun(alpha);current=await state(photo.id);
  assert(current.tags.some(t=>t.id===chinese.id&&t.source==='manual'&&!t.localization));
  assert(!current.tags.some(t=>t.source==='ai'&&t.name===automatic.name),'must not duplicate a Chinese manual category');
  await remove(photo.id,chinese.id);await rerun(alpha);current=await state(photo.id);
  assert(!current.tags.some(t=>t.name===automatic.name||t.id===chinese.id),'Chinese removal suppression survives rerun');
  await manual(photo.id,automatic.id);await rerun(alpha);current=await state(photo.id);
  assert(current.tags.some(t=>t.id===automatic.id&&t.name===automatic.name&&t.source==='manual'&&!t.localization));
  assert(current.tags.some(t=>t.id===keep.id&&t.name==='人工中文保留'));
  console.log('PASS Chinese manual alias priority/removal suppression, explicit manual restore and repeated real inference');
  const revision=current.revision;
  for(const query of [automatic.localization.zhCN,automatic.localization.en,automatic.name])await json(`/tags?libraryId=${alpha}&q=${encodeURIComponent(query)}`);
  assert.equal((await state(photo.id)).revision,revision,'display/search reads may not write curation');
  compose('restart','foliopath');await until('restart',()=>inspect(app),s=>s.State.Health?.Status==='healthy');refresh();await login();
  current=await state(photo.id);assert(current.tags.some(t=>t.id===automatic.id&&t.source==='manual'&&!t.localization));
  assert.equal((await settings(alpha)).coverage.completed,2);assert.equal((await settings(beta)).enabled,false);
  assert.deepEqual(hashes(),before);assert.equal(docker('exec',app,'/app/foliopath','database-check'),'database check passed');
  const events=docker('exec',ai,'sh','-c','cat /sys/fs/cgroup/memory.events');assert.match(events,/oom_kill 0/);
  assert.equal(inspect(app).RestartCount,0);assert.equal(inspect(ai).RestartCount,0);assert.equal(inspect(ai).State.OOMKilled,false);
  console.log(JSON.stringify({passed:true,project,root,origin,appImage,aiImage,tag,aiPeakBytes:docker('exec',ai,'sh','-c','cat /sys/fs/cgroup/memory.peak'),events,example:automatic}));
}finally{
  if(process.env.FOLIOPATH_TEST_KEEP==='1')console.log(`Kept ${project}: ${root}`);
  else {compose('down','--remove-orphans');console.log(`Removed ${project}; synthetic data/scratch retained`);}
}
