// ADR-0025: local Docker, pinned real weights, synthetic media only.
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {copyFileSync,mkdirSync,mkdtempSync,readFileSync,realpathSync,writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createHash,randomUUID} from 'node:crypto';
import {crc32,deflateSync} from 'node:zlib';

const repo=fileURLToPath(new URL('../../',import.meta.url));
const pngUpgrade=process.env.FOLIOPATH_TEST_PNG_PROFILE==='1';
if(pngUpgrade)assert(process.env.FOLIOPATH_TEST_SEMANTIC_AUDIT,'Linux synthetic audit binary is required');
const run=(...args)=>execFileSync('docker',args,{encoding:'utf8',timeout:60000}).trim();
assert(JSON.parse(run('context','inspect'))[0].Endpoints.docker.Host.startsWith('unix://'));
for(const name of ['FOLIOPATH_TEST_APP_IMAGE','FOLIOPATH_TEST_AI_IMAGE','FOLIOPATH_TEST_MODEL_CACHE'])assert(process.env[name],name);
const appImage=run('image','inspect',process.env.FOLIOPATH_TEST_APP_IMAGE,'--format','{{.Id}}');
const aiImage=run('image','inspect',process.env.FOLIOPATH_TEST_AI_IMAGE,'--format','{{.Id}}');
const imageJSON=(...args)=>JSON.parse(run('run','--rm','--network','none','--entrypoint','python',aiImage,...args));
const identity=imageJSON('-m','multilingual_contract');
const legacy=imageJSON('-m','image_identity');
const tag=imageJSON('-m','image_identity','--tag');
assert.equal(identity.model,'XLM-Roberta-Large-Vit-B-16Plus');assert.equal(identity.dimension,640);
const manifest=imageJSON('-c','from pathlib import Path; print(Path("/opt/foliopath-inference/cache-manifest.json").read_text())');
const weights=['detection','recognition'].map(k=>manifest.find(e=>e.path===`facial-recognition/buffalo_l/${k}/model.onnx`));assert(weights.every(Boolean));
const deployment={origin:'http://machine-learning:3003',identity,model_bytes:2729566266,license_id:'unreviewed-local-test',
  tag:{identity:tag},face:{identity:{...legacy,protocol:'foliopath-face-v1',model:'buffalo_l'},
  detector_sha256:weights[0].sha256,embedder_sha256:weights[1].sha256,model_bytes:weights.reduce((s,e)=>s+e.size,0),license_id:'unreviewed-local-test',
  threshold:{schemaVersion:1,profileId:'multilingual-synthetic-only',coreSimilarity:.8,edgeSimilarity:.6,minCoreSize:2,qualitySummarySha256:createHash('sha256').update('NOT quality calibration').digest('hex')}}};
const project='fp-multilingual-'+randomUUID().slice(0,8),root=realpathSync(mkdtempSync(join(tmpdir(),project+'-')));
for(const path of ['data','media/alpha','media/beta'])mkdirSync(join(root,path),{recursive:true});
copyFileSync(join(repo,'tests/fixtures/ai-faces/synthetic-pair.png'),join(root,'media/alpha/people.png'));
copyFileSync(join(repo,'tests/fixtures/media/viewer-blue-violet.png'),join(root,'media/alpha/blue.png'));
copyFileSync(join(repo,'tests/fixtures/ai-faces/synthetic-pair.png'),join(root,'media/beta/other.png'));
writeFileSync(join(root,'media/alpha/broken.jpg'),'Synthetic invalid media');
const sources=['alpha/people.png','alpha/blue.png','alpha/broken.jpg','beta/other.png'];
if(pngUpgrade){
  // Fully synthetic RGB PNG, valid pixels and real IDAT >32 MiB, not padding.
  const width=3344,height=5016,stride=width*3+1,raw=Buffer.alloc(stride*height);
  for(let y=0;y<height;y++)for(let x=0;x<width;x++){
    const offset=y*stride+1+x*3;raw[offset]=x*37+y*11;raw[offset+1]=x*19+y*29;raw[offset+2]=x^y;
  }
  const chunk=(type,data)=>{const out=Buffer.alloc(data.length+12);out.writeUInt32BE(data.length);out.write(type,4);data.copy(out,8);out.writeUInt32BE(crc32(out.subarray(4,data.length+8)),data.length+8);return out;};
  const ihdr=Buffer.alloc(13);ihdr.writeUInt32BE(width);ihdr.writeUInt32BE(height,4);ihdr[8]=8;ihdr[9]=2;
  const png=Buffer.concat([Buffer.from([137,80,78,71,13,10,26,10]),chunk('IHDR',ihdr),chunk('IDAT',deflateSync(raw,{level:0})),chunk('IEND',Buffer.alloc(0))]);
  assert(png.length>32*1024**2&&png.length<64*1024**2);
  writeFileSync(join(root,'media/alpha/large.png'),png);
  writeFileSync(join(root,'media/alpha/truncated.png'),png.subarray(0,Math.floor(png.length/2)));
  sources.push('alpha/large.png','alpha/truncated.png');
}
const successfulImages=pngUpgrade?3:2,failedImages=pngUpgrade?2:1;
const hashes=()=>sources.map(p=>createHash('sha256').update(readFileSync(join(root,'media',p))).digest('hex'));
const originalHashes=hashes();
const env={...process.env,FOLIOPATH_IMAGE:appImage,FOLIOPATH_AI_IMAGE:aiImage,FOLIOPATH_LIBRARY_PATH:join(root,'media'),FOLIOPATH_DATA_PATH:join(root,'data'),FOLIOPATH_MODEL_CACHE_PATH:realpathSync(process.env.FOLIOPATH_TEST_MODEL_CACHE),FOLIOPATH_BIND_ADDRESS:'127.0.0.1',FOLIOPATH_PORT:'0',FOLIOPATH_AI_DEPLOYMENT:JSON.stringify(deployment)};
const compose=(...args)=>execFileSync('docker',['compose','--env-file','/dev/null','-p',project,'-f',join(repo,'compose.ai.yaml'),'-f',join(repo,'compose.ai.multilingual.yaml'),'-f',join(repo,'tests/release/compose.ai.multilingual-test.yaml'),...args],{env,encoding:'utf8',timeout:60000}).trim();
const inspect=id=>JSON.parse(run('inspect',id))[0];
let app,ai,origin,cookie='',csrf='';
const credentials={username:'multilingual-test',password:'Synthetic-Multilingual-2026!'};
async function until(label,read,ready,timeout=240000){let last;const end=Date.now()+timeout;while(Date.now()<end){last=await read();if(ready(last))return last;await new Promise(r=>setTimeout(r,500));}throw Error(label+': '+JSON.stringify(last));}
const refresh=()=>{const port=inspect(app).NetworkSettings.Ports['8080/tcp'][0];assert.equal(port.HostIp,'127.0.0.1');origin='http://127.0.0.1:'+port.HostPort;};
async function request(path,method='GET',body,headers={}){return fetch(origin+'/api/v1'+path,{method,signal:AbortSignal.timeout(35000),headers:{Origin:origin,Cookie:cookie,'X-CSRF-Token':csrf,'Content-Type':'application/json',...headers},body:body===undefined?undefined:JSON.stringify(body)});}
async function json(path,method,body,headers){const r=await request(path,method,body,headers),v=await r.json();assert(r.ok,`${path}: ${r.status} ${JSON.stringify(v)}`);return v;}
async function login(){const r=await request('/auth/login','POST',credentials);assert(r.ok);cookie=r.headers.get('set-cookie').split(';')[0];csrf=(await r.json()).csrfToken;}
async function enable(library,capability,method='PATCH'){const r=await request(`/libraries/${library}/ai/${capability}`);assert(r.ok);return json(`/libraries/${library}/ai/${capability}`,method,{enabled:true},{'If-Match':r.headers.get('etag')});}
async function finish(op){const done=await until('operation',()=>json(`/ai/operations/${op.id}`),s=>!['queued','running','cancelling'].includes(s.state));assert.equal(done.state,'succeeded',JSON.stringify(done));return done;}
try{
  const config=JSON.parse(compose('config','--format','json'));assert.equal(Number(config.services['machine-learning'].mem_limit),4096*1024**2);
  assert.deepEqual(Object.keys(config.services).sort(),['foliopath','machine-learning']);assert(!config.services['machine-learning'].ports?.length);
  compose('up','-d','--pull','never');app=compose('ps','-q','foliopath');ai=compose('ps','-q','machine-learning');refresh();
  await until('app health',()=>inspect(app),s=>s.State.Health?.Status==='healthy');await until('AI health',()=>inspect(ai),s=>s.State.Health?.Status==='healthy');
  const mounts=inspect(ai).Mounts;assert.equal(mounts.find(m=>m.Destination==='/cache').RW,false);assert(mounts.some(m=>m.Destination==='/verified-cache'&&m.Type==='volume'));
  assert(!mounts.some(m=>['/library','/app/data','/var/run/docker.sock'].includes(m.Destination)));
  assert.equal((await request('/semantic/assets?q=test')).status,401);
  await json('/auth/setup','POST',{...credentials,displayName:'Synthetic multilingual test'});await login();
  await until('all three capability activation',()=>json('/ai/models'),v=>v.items.filter(m=>m.active&&m.state==='available').length===2);
  const create=async name=>{const value=await json('/libraries','POST',{name,rootPath:name},{'Idempotency-Key':randomUUID()});await until('scan',()=>json(`/scans/${value.scan.id}`),s=>s.status==='succeeded');return value.library.id;};
  const alpha=await create('alpha'),beta=await create('beta');
  const assets=(await json(`/libraries/${alpha}/assets`)).items;
  const photo=assets.find(a=>a.name==='people.png');assert(photo);
  const manual=await json('/tags','POST',{name:'人工中文保留'});
  const curation=await json(`/assets/${photo.id}/curation`);
  await json(`/assets/${photo.id}/tags/${manual.id}`,'PUT',undefined,{'If-Match':`"curation-r${curation.revision}"`});
  await enable(alpha,'semantic');
  let oldPNGGeneration;
  const audit=()=>{
    run('cp',realpathSync(process.env.FOLIOPATH_TEST_SEMANTIC_AUDIT),app+':/app/data/semantic-state-audit');
    return JSON.parse(run('exec',app,'/app/data/semantic-state-audit'));
  };
  if(pngUpgrade){
    await finish(await json(`/libraries/${alpha}/ai/semantic/jobs`,'POST',{mode:'missing'},{'Idempotency-Key':randomUUID()}));
    const old=await json(`/libraries/${alpha}/ai/semantic`);
    assert.equal(old.coverage.completed,2);assert.equal(old.coverage.failed,3);oldPNGGeneration=old.activeGenerationId;
    assert.equal(audit().find(g=>g.id===oldPNGGeneration).count,2);
    const beforeAI=inspect(ai).State.StartedAt;
    deployment.image_preprocessing='png-v4';env.FOLIOPATH_AI_DEPLOYMENT=JSON.stringify(deployment);
    compose('up','-d','--no-deps','foliopath');app=compose('ps','-q','foliopath');refresh();
    await until('PNG profile app health',()=>inspect(app),s=>s.State.Health?.Status==='healthy');await login();
    const next=await until('distinct PNG generation',()=>json(`/libraries/${alpha}/ai/semantic`),s=>s.activeGenerationId&&s.activeGenerationId!==oldPNGGeneration);
    assert.equal(next.coverage.completed,0);assert.equal(inspect(ai).State.StartedAt,beforeAI);
    const preserved=audit().find(g=>g.id===oldPNGGeneration);assert.equal(preserved.count,2);assert.equal(preserved.state,'retired');
    assert((await json(`/assets/${photo.id}/curation`)).tags.some(t=>t.id===manual.id&&t.source==='manual'));
    console.log('PASS explicit PNG v4 activation: new empty generation, old vectors/manual tags retained, AI not restarted');
  }
  const key=randomUUID(),op=await json(`/libraries/${alpha}/ai/semantic/jobs`,'POST',{mode:'missing'},{'Idempotency-Key':key});
  await finish(op);
  const settings=await json(`/libraries/${alpha}/ai/semantic`);assert.equal(settings.coverage.completed,successfulImages);assert.equal(settings.coverage.failed,failedImages);
  assert.equal((await json(`/libraries/${alpha}/ai/semantic/jobs`,'POST',{mode:'missing'},{'Idempotency-Key':key})).id,op.id);
  console.log('PASS real 640-dimensional synthetic indexing, corrupt input, idempotency and isolated mounts');
  for(const query of ['人物','蓝紫色背景','car','车']){
    const result=await json(`/semantic/assets?libraryId=${alpha}&q=${encodeURIComponent(query)}&limit=10`);
    assert.equal(result.items.length,successfulImages,JSON.stringify(result));assert(result.items.every(a=>a.libraryId===alpha));
    if(pngUpgrade)assert(result.items.some(a=>a.name==='large.png'));
    console.log('PASS query '+query+' order='+result.items.map(a=>a.name).join(','));
  }
  const before=(await json(`/libraries/${alpha}/ai/faces`));await enable(alpha,'faces');
  await finish(await json(`/libraries/${alpha}/ai/faces/jobs`,'POST',{mode:'missing'},{'Idempotency-Key':randomUUID()}));
  const faces=await json(`/libraries/${alpha}/ai/faces`);assert.equal(faces.coverage.completed,successfulImages);assert.equal(faces.coverage.failed,failedImages);
  await enable(alpha,'automatic-tags','PUT');
  await until('automatic tags',()=>json(`/libraries/${alpha}/ai/automatic-tags`),s=>s.coverage.completed===2&&!s.activeOperationId);
  if(pngUpgrade)assert.equal((await json(`/libraries/${alpha}/ai/automatic-tags`)).coverage.failed,3,'automatic tags must still use v3 input limits');
  const tags=await json(`/assets/${photo.id}/curation`);assert(tags.tags.some(t=>t.id===manual.id&&t.source==='manual'));assert(tags.tags.some(t=>t.source==='ai'));
  assert.equal((await json(`/libraries/${beta}/ai/faces`)).coverage.completed,0);
  assert.equal(inspect(app).RestartCount,0);assert.equal(inspect(ai).RestartCount,0);
  console.log('PASS shared scheduler legacy face/tag inference and manual Chinese tag protection');
  // Reproduce the real deployment's two background families. A foreground
  // query must wait inside its existing deadline, not fail with internal_error.
  const tagRebuild=await json(`/libraries/${alpha}/ai/automatic-tags/jobs`,'POST',{mode:'all',confirmation:'reprocess_automatic_tags'},{'Idempotency-Key':randomUUID()});
  const semanticRebuild=await json(`/libraries/${alpha}/ai/semantic/jobs`,'POST',{mode:'all'},{'Idempotency-Key':randomUUID()});
  await until('background running',()=>json(`/ai/operations/${tagRebuild.id}`),s=>s.state==='running');
  const concurrent=await json(`/semantic/assets?libraryId=${alpha}&q=${encodeURIComponent('人物')}`);
  assert.equal(concurrent.items.length,successfulImages);await finish(tagRebuild);await finish(semanticRebuild);
  console.log('PASS foreground Chinese search with both background families in flight');
  const peak=run('exec',ai,'sh','-c','cat /sys/fs/cgroup/memory.peak');
  // The main image is distroless (no shell). The PNG audit binary is copied
  // only into this isolated synthetic owner, and reads cgroup without SQLite.
  const appPeak=pngUpgrade?run('exec',app,'/app/data/semantic-state-audit','-memory'):null;
  compose('restart','foliopath');await until('restart',()=>inspect(app),s=>s.State.Health?.Status==='healthy');refresh();await login();
  assert.equal((await json(`/libraries/${alpha}/ai/semantic`)).coverage.completed,successfulImages);
  assert.equal((await json(`/semantic/assets?libraryId=${alpha}&q=${encodeURIComponent('人物')}`)).items.length,successfulImages);
  if(pngUpgrade)assert.equal(audit().find(g=>g.id===oldPNGGeneration).count,2);
  assert((await json(`/assets/${photo.id}/curation`)).tags.some(t=>t.id===manual.id&&t.source==='manual'));
  assert.deepEqual(hashes(),originalHashes);assert.equal(inspect(ai).State.OOMKilled,false);
  assert.equal(run('exec',app,'/app/foliopath','database-check'),'database check passed');
  assert.equal(inspect(app).RestartCount,0);assert.equal(inspect(ai).RestartCount,0);assert.equal(inspect(app).State.OOMKilled,false);
  console.log(JSON.stringify({passed:true,project,root,appImage,aiImage,identity,pngUpgrade,appPeakBytes:appPeak,aiPeakBytes:peak}));
}finally{
  if(process.env.FOLIOPATH_TEST_KEEP==='1')console.log(`Kept ${project}: ${root}`);
  else{compose('down','--remove-orphans');console.log(`Removed ${project}; synthetic data and disposable scratch retained`);}
}
