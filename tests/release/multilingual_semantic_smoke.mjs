// ADR-0025: local Docker, pinned real weights, synthetic media only.
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {copyFileSync,cpSync,mkdirSync,mkdtempSync,readFileSync,realpathSync,writeFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createHash,randomUUID} from 'node:crypto';
import {crc32,deflateSync} from 'node:zlib';

const repo=fileURLToPath(new URL('../../',import.meta.url));
const fusedSearch=process.env.FOLIOPATH_TEST_FUSED_SEARCH==='1';
const fusedOnly=process.env.FOLIOPATH_TEST_FUSED_ONLY==='1';
if(fusedOnly)assert(fusedSearch);
const pngUpgrade=process.env.FOLIOPATH_TEST_PNG_PROFILE==='1';
const tagPNGUpgrade=process.env.FOLIOPATH_TEST_TAG_PNG_PROFILE==='1';
const tagRecovery=process.env.FOLIOPATH_TEST_TAG_RECOVERY==='1';
const photoPolicyUpgrade=process.env.FOLIOPATH_TEST_PHOTO_POLICY_UPGRADE==='1';
if(photoPolicyUpgrade){assert(tagPNGUpgrade&&tagRecovery,'photo upgrade requires PNG/manual/recovery fixtures');assert(process.env.FOLIOPATH_TEST_LEGACY_AI_IMAGE);}
if(tagRecovery)assert(tagPNGUpgrade,'tag recovery uses the nonempty manual/suppression fixtures');
if(tagPNGUpgrade)assert(pngUpgrade,'tag PNG upgrade requires the synthetic PNG profile fixtures');
if(pngUpgrade)assert(process.env.FOLIOPATH_TEST_SEMANTIC_AUDIT,'Linux synthetic audit binary is required');
const run=(...args)=>execFileSync('docker',args,{encoding:'utf8',timeout:60000}).trim();
assert(JSON.parse(run('context','inspect'))[0].Endpoints.docker.Host.startsWith('unix://'));
for(const name of ['FOLIOPATH_TEST_APP_IMAGE','FOLIOPATH_TEST_AI_IMAGE','FOLIOPATH_TEST_MODEL_CACHE'])assert(process.env[name],name);
const appImage=run('image','inspect',process.env.FOLIOPATH_TEST_APP_IMAGE,'--format','{{.Id}}');
const aiImage=run('image','inspect',process.env.FOLIOPATH_TEST_AI_IMAGE,'--format','{{.Id}}');
const legacyAIImage=photoPolicyUpgrade?run('image','inspect',process.env.FOLIOPATH_TEST_LEGACY_AI_IMAGE,'--format','{{.Id}}'):aiImage;
const imageJSON=(...args)=>JSON.parse(run('run','--rm','--network','none','--entrypoint','python',aiImage,...args));
const identity=imageJSON('-m','multilingual_contract');
const legacy=imageJSON('-m','image_identity');
const tag=imageJSON('-m','image_identity','--tag');
const photoTag=photoPolicyUpgrade?imageJSON('-m','photo_policy'):null;
const oldClasses=photoPolicyUpgrade?new Set(imageJSON('-c','import json,tag_contract; print(json.dumps(sorted(tag_contract.SAFE_LABELS)))')):null;
if(photoPolicyUpgrade){
  assert.equal(imageJSON('-c','import json,photo_policy; print(json.dumps(len(photo_policy.SAFE_LABELS)))'),153);
  assert.equal(photoTag.model_sha256,tag.model_sha256);assert.equal(photoTag.cache_manifest_sha256,tag.cache_manifest_sha256);
  assert.notEqual(photoTag.policy_sha256,tag.policy_sha256);assert.notEqual(photoTag.runtime_contract_sha256,tag.runtime_contract_sha256);
}
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
const bluePath=fusedSearch?'alpha/blue-background/blue.png':'alpha/blue.png';
if(fusedSearch)mkdirSync(join(root,'media/alpha/blue-background'));
copyFileSync(join(repo,'tests/fixtures/media/viewer-blue-violet.png'),join(root,'media',bluePath));
copyFileSync(join(repo,'tests/fixtures/ai-faces/synthetic-pair.png'),join(root,'media/beta/other.png'));
writeFileSync(join(root,'media/alpha/broken.jpg'),'Synthetic invalid media');
const sources=['alpha/people.png',bluePath,'alpha/broken.jpg','beta/other.png'];
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
const env={...process.env,FOLIOPATH_IMAGE:appImage,FOLIOPATH_AI_IMAGE:legacyAIImage,FOLIOPATH_LIBRARY_PATH:join(root,'media'),FOLIOPATH_DATA_PATH:join(root,'data'),FOLIOPATH_MODEL_CACHE_PATH:realpathSync(process.env.FOLIOPATH_TEST_MODEL_CACHE),FOLIOPATH_BIND_ADDRESS:'127.0.0.1',FOLIOPATH_PORT:'0',FOLIOPATH_AI_DEPLOYMENT:JSON.stringify(deployment)};
const compose=(...args)=>execFileSync('docker',['compose','--env-file','/dev/null','-p',project,'-f',join(repo,'compose.ai.yaml'),'-f',join(repo,'compose.ai.multilingual.yaml'),'-f',join(repo,'tests/release/compose.ai.multilingual-test.yaml'),...args],{env,encoding:'utf8',timeout:60000}).trim();
const inspect=id=>JSON.parse(run('inspect',id))[0];
let app,ai,origin,cookie='',csrf='';
const credentials={username:'multilingual-test',password:'Synthetic-Multilingual-2026!'};
async function until(label,read,ready,timeout=240000,interval=500){let last;const end=Date.now()+timeout;while(Date.now()<end){last=await read();if(ready(last))return last;await new Promise(r=>setTimeout(r,interval));}throw Error(label+': '+JSON.stringify(last));}
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
  for(const query of (fusedOnly?[]:['人物','蓝紫色背景','car','车'])){
    const result=await json(`/semantic/assets?libraryId=${alpha}&q=${encodeURIComponent(query)}&limit=10`);
    assert.equal(result.items.length,successfulImages,JSON.stringify(result));assert(result.items.every(a=>a.libraryId===alpha));
    if(pngUpgrade)assert(result.items.some(a=>a.name==='large.png'));
    console.log('PASS query '+query+' order='+result.items.map(a=>a.name).join(','));
  }
  if(fusedSearch){
    assert.equal((await request('/search/assets?q=blue', 'GET', undefined, {Cookie:''})).status,401);
    const fused=await json(`/search/assets?libraryId=${alpha}&q=blue&limit=10`);
    assert.equal(fused.semanticState,'ready');
    const blue=fused.items.find(h=>h.asset.name==='blue.png');assert(blue,JSON.stringify(fused));
    assert(blue.sources.includes('filename')&&blue.sources.includes('path'));
    assert(blue.sources.includes('semantic'),JSON.stringify(fused));assert(blue.semanticScore>=.2);
    assert.equal(new Set(fused.items.map(h=>h.asset.id)).size,fused.items.length);
    const chinese=await json(`/search/assets?libraryId=${alpha}&q=${encodeURIComponent('蓝紫色背景')}`);
    assert(chinese.items.some(h=>h.sources.includes('semantic')),JSON.stringify(chinese));
    const parent=await json(`/search/assets?libraryId=${alpha}&mode=filename&q=background`);
    assert.deepEqual(parent.items.map(h=>h.sources),[['path']]);
    const copied=await json(`/search/assets?libraryId=${alpha}&mode=filename&q=${encodeURIComponent('blue-background/blue.png')}`);assert.equal(copied.items.length,1);assert.deepEqual(copied.items[0].sources,['path']);
    const basename=await json(`/libraries/${alpha}/assets?q=people`);
    assert.deepEqual(basename.items.find(a=>a.name==='people.png').searchSources,['filename']);
    const all=await json('/search/assets?q=other');
    const betaHit=all.items.find(h=>h.asset.libraryId===beta);assert(betaHit);assert.deepEqual(betaHit.sources,['filename']);
    assert(all.coverage.excludedLibraries.some(l=>l.libraryId===beta));
    const first=await json(`/search/assets?libraryId=${alpha}&q=.png&limit=1`);assert(first.nextCursor);
    const second=await json(`/search/assets?libraryId=${alpha}&q=.png&limit=1&cursor=${encodeURIComponent(first.nextCursor)}`);
    assert.equal(second.items.length,1);assert.notEqual(second.items[0].asset.id,first.items[0].asset.id);
    const wrong=await request(`/search/assets?libraryId=${alpha}&q=blue&cursor=${encodeURIComponent(first.nextCursor)}`);assert.equal(wrong.status,400);
    const range=await json(`/search/assets?libraryId=${alpha}&q=.png&kind=video`);assert.equal(range.items.length,0);
    const future=await json(`/search/assets?libraryId=${alpha}&q=.png&modifiedFrom=2099-01-01T00%3A00%3A00Z`);assert.equal(future.items.length,0);
    const directory=blue.asset.directoryId;
    const scoped=await json(`/search/assets?libraryId=${alpha}&directoryId=${directory}&q=.png`);assert.equal(scoped.items.length,1);
    const configResponse=await request(`/libraries/${alpha}/ai/semantic`);
    await json(`/libraries/${alpha}/ai/semantic`,'PATCH',{enabled:false},{'If-Match':configResponse.headers.get('etag')});
    const stale=await request(`/search/assets?libraryId=${alpha}&q=.png&limit=1&cursor=${encodeURIComponent(first.nextCursor)}`);
    assert.equal(stale.status,409);assert.equal((await stale.json()).error.code,'search_cursor_stale');
    const disabled=await json(`/search/assets?libraryId=${alpha}&q=blue`);assert.equal(disabled.semanticState,'disabled');assert(disabled.items.every(h=>!h.sources.includes('semantic')));
    await enable(alpha,'semantic');
    // Only this isolated synthetic AI is stopped. No production task is touched.
    compose('stop','machine-learning');
    const down=await json(`/search/assets?libraryId=${alpha}&q=blue`);
    assert.equal(down.semanticState,'unavailable');assert.equal(down.items.length,1);assert.deepEqual(down.items[0].sources,['filename','path']);
    assert.equal((await request(`/search/assets?libraryId=${alpha}&mode=semantic&q=blue`)).status,503);
    compose('start','machine-learning');await until('synthetic AI recovered',()=>inspect(ai),v=>v.State.Health?.Status==='healthy');
    // A ping becomes healthy before the runtime/cache is ready for inference.
    const recovered=await until('functional fused recovery',()=>json(`/search/assets?libraryId=${alpha}&q=blue`),v=>v.semanticState==='ready',240000,5000);assert(recovered.items.some(h=>h.sources.includes('semantic')));
    assert.equal((await json(`/libraries/${alpha}/ai/semantic`)).coverage.completed,successfulImages);
    assert((await json(`/assets/${photo.id}/curation`)).tags.some(t=>t.id===manual.id&&t.source==='manual'));
    console.log('PASS CR-2026-035 real Chinese/English fused evidence, dedup, parent path, legacy attribution, keyset/query binding/stale, scope/kind/date, disabled/global exclusion, AI outage/recovery without losing vectors or manual tags');
    console.log(JSON.stringify({fusedSample:fused.items.map(h=>({name:h.asset.name,sources:h.sources,score:h.semanticScore})),chineseSources:chinese.items.map(h=>({name:h.asset.name,sources:h.sources,score:h.semanticScore}))}));
  }
  if(!fusedOnly){
  const before=(await json(`/libraries/${alpha}/ai/faces`));await enable(alpha,'faces');
  await finish(await json(`/libraries/${alpha}/ai/faces/jobs`,'POST',{mode:'missing'},{'Idempotency-Key':randomUUID()}));
  const faces=await json(`/libraries/${alpha}/ai/faces`);assert.equal(faces.coverage.completed,successfulImages);assert.equal(faces.coverage.failed,failedImages);
  await enable(alpha,'automatic-tags','PUT');
  await until('automatic tags',()=>json(`/libraries/${alpha}/ai/automatic-tags`),s=>s.coverage.completed===2&&!s.activeOperationId);
  if(pngUpgrade)assert.equal((await json(`/libraries/${alpha}/ai/automatic-tags`)).coverage.failed,3,'automatic tags must still use v3 input limits');
  if(tagPNGUpgrade){
    const beforeTags=await json(`/libraries/${alpha}/ai/automatic-tags`);
    const beforeSemantic=await json(`/libraries/${alpha}/ai/semantic`);
    const beforeFaces=await json(`/libraries/${alpha}/ai/faces`);
    const beforeAI=inspect(ai).State.StartedAt;
    const initial=(await json(`/assets/${photo.id}/curation`)).tags;
    const aiTags=initial.filter(t=>t.source==='ai'&&t.localization);assert(aiTags.length>=2,'fixture must exercise manual alias and removal separately');
    const removed=aiTags[0],promoted=aiTags[1];
    const chinese=await json('/tags','POST',{name:promoted.localization.zhCN});
    let current=await json(`/assets/${photo.id}/curation`);
    await json(`/assets/${photo.id}/tags/${chinese.id}`,'PUT',undefined,{'If-Match':`"curation-r${current.revision}"`});
    current=await json(`/assets/${photo.id}/curation`);
    await json(`/assets/${photo.id}/tags`,'PUT',{tagIds:current.tags.filter(t=>t.id!==removed.id).map(t=>t.id)},{'If-Match':`"curation-r${current.revision}"`});
    // Keep semantic/face/AI identities fixed; only tag input changes. Existing
    // enabled-library reconciliation must create exactly the new-identity work.
    deployment.tag.image_preprocessing='png-v4';env.FOLIOPATH_AI_DEPLOYMENT=JSON.stringify(deployment);
    compose('up','-d','--no-deps','foliopath');app=compose('ps','-q','foliopath');refresh();
    await until('tag PNG app health',()=>inspect(app),s=>s.State.Health?.Status==='healthy');await login();
    const upgraded=await until('tag PNG coverage',()=>json(`/libraries/${alpha}/ai/automatic-tags`),s=>s.coverage.completed===3&&s.coverage.failed===2&&!s.activeOperationId);
    assert.notDeepEqual(upgraded.model,beforeTags.model,'new input must bind a new tag identity');
    assert.equal(inspect(ai).State.StartedAt,beforeAI,'AI may not restart for tag input');
    const semantic=await json(`/libraries/${alpha}/ai/semantic`),face=await json(`/libraries/${alpha}/ai/faces`);
    assert.equal(semantic.activeGenerationId,beforeSemantic.activeGenerationId);assert.deepEqual(semantic.coverage,beforeSemantic.coverage);
    assert.equal(face.activeGenerationId,beforeFaces.activeGenerationId);assert.deepEqual(face.coverage,beforeFaces.coverage);
    const verifyManual=async()=>{
      const state=await json(`/assets/${photo.id}/curation`);
      assert(state.tags.some(t=>t.id===manual.id&&t.source==='manual'));
      assert(state.tags.some(t=>t.id===chinese.id&&t.source==='manual'&&!t.localization));
      assert(!state.tags.some(t=>t.id===removed.id),'removed category must remain suppressed across input generations');
      assert(!state.tags.some(t=>t.source==='ai'&&t.name===promoted.name),'Chinese manual synonym may not be duplicated by AI');
    };
    await verifyManual();
    const large=assets.find(a=>a.name==='large.png');assert(large);
    assert((await json(`/assets/${large.id}/curation`)).tags.some(t=>t.source==='ai'),'real large PNG needs actual AI tags, not just a ready checkpoint');
    const redo=await json(`/libraries/${alpha}/ai/automatic-tags/jobs`,'POST',{mode:'all',confirmation:'reprocess_automatic_tags'},{'Idempotency-Key':randomUUID()});await finish(redo);await verifyManual();
    if(photoPolicyUpgrade){
      const previous=await json(`/libraries/${alpha}/ai/automatic-tags`);
      const semanticBefore=await json(`/libraries/${alpha}/ai/semantic`),facesBefore=await json(`/libraries/${alpha}/ai/faces`);
      const vectorsBefore=audit();
      // Dedicated synthetic data only: complete stopped copy before changing
      // the tag policy, exactly as required for the real deployment.
      compose('stop','foliopath');assert.equal(inspect(app).State.ExitCode,0);
      assert.equal(inspect(app).State.Running,false);
      cpSync(join(root,'data'),join(root,'before-photo-policy'),{recursive:true,errorOnExist:true,force:false});
      writeFileSync(join(root,'before-photo-policy-deployment.json'),JSON.stringify(deployment));
      compose('stop','machine-learning');
      deployment.tag.identity=photoTag;env.FOLIOPATH_AI_IMAGE=aiImage;env.FOLIOPATH_AI_DEPLOYMENT=JSON.stringify(deployment);
      compose('up','-d','--no-deps','machine-learning');ai=compose('ps','-q','machine-learning');
      await until('153 category AI ready',()=>inspect(ai),s=>s.State.Health?.Status==='healthy');
      compose('up','-d','--no-deps','foliopath');app=compose('ps','-q','foliopath');refresh();
      await until('153 category app ready',()=>inspect(app),s=>s.State.Health?.Status==='healthy');await login();
      const current=await until('153 tag policy coverage',()=>json(`/libraries/${alpha}/ai/automatic-tags`),s=>s.model.identity!==previous.model.identity&&s.coverage.completed===3&&s.coverage.failed===2&&!s.activeOperationId);
      assert.notEqual(current.model.identity,previous.model.identity);await verifyManual();
      assert((await json(`/assets/${photo.id}/curation`)).tags.some(t=>t.source==='ai'&&!oldClasses.has(t.name)),'must produce a real newly allowed class');
      assert((await json(`/assets/${large.id}/curation`)).tags.some(t=>t.source==='ai'),'153 policy must retain actual large PNG labels');
      const semanticAfter=await json(`/libraries/${alpha}/ai/semantic`),facesAfter=await json(`/libraries/${alpha}/ai/faces`);
      assert.equal(semanticAfter.activeGenerationId,semanticBefore.activeGenerationId);assert.deepEqual(semanticAfter.coverage,semanticBefore.coverage);
      assert.equal(facesAfter.activeGenerationId,facesBefore.activeGenerationId);assert.deepEqual(facesAfter.coverage,facesBefore.coverage);
      assert.deepEqual(audit(),vectorsBefore,'all semantic generations/vectors must survive the tag-only policy change');
      assert.equal((await json(`/libraries/${beta}/ai/automatic-tags`)).coverage.completed,0);
      const policyRedo=await json(`/libraries/${alpha}/ai/automatic-tags/jobs`,'POST',{mode:'all',confirmation:'reprocess_automatic_tags'},{'Idempotency-Key':randomUUID()});await finish(policyRedo);await verifyManual();
      assert.deepEqual(hashes(),originalHashes);
      console.log('PASS stopped backup and 92-to-153 policy switch: new identity/new category, large PNG, manual/suppression and semantic/face retained');
    }
    if(tagRecovery){
      const recovery=await json(`/libraries/${alpha}/ai/automatic-tags/jobs`,'POST',{mode:'all',confirmation:'reprocess_automatic_tags'},{'Idempotency-Key':randomUUID()});
      // Stop only this isolated synthetic AI. Core browsing stays available and
      // the live operation must wait, never fabricate a failed/empty image.
      compose('stop','machine-learning');
      await new Promise(r=>setTimeout(r,3000));
      const waiting=await json(`/ai/operations/${recovery.id}`);
      assert.equal(waiting.state,'running');assert.equal(waiting.errorCode,null);
      await verifyManual();
      compose('start','machine-learning');
      await finish(recovery);await verifyManual();
      const recovered=await json(`/libraries/${alpha}/ai/automatic-tags`);
      assert.equal(recovered.coverage.completed,3);assert.equal(recovered.coverage.failed,2);
      assert.deepEqual(hashes(),originalHashes);
      console.log('PASS transient AI outage: same operation resumes, real labels/manual/suppression retained, bad sources remain failed');
    }
    compose('restart','foliopath');await until('tag PNG restart',()=>inspect(app),s=>s.State.Health?.Status==='healthy');refresh();await login();await verifyManual();
    assert.equal((await json(`/libraries/${beta}/ai/automatic-tags`)).coverage.completed,0);
    console.log('PASS independent PNG tag identity: real large PNG labels, bad sources retained, Chinese manual priority/removal suppression, rerun/restart, semantic/face unchanged');
  }
  const tags=await json(`/assets/${photo.id}/curation`);assert(tags.tags.some(t=>t.id===manual.id&&t.source==='manual'));
  // The tag-upgrade fixture deliberately removes/promotes every AI category on
  // photo. Require real AI output on the uncurated large PNG instead, while
  // retaining the original assertion for the unchanged semantic-only case.
  const aiOutput=tagPNGUpgrade?await json(`/assets/${assets.find(a=>a.name==='large.png').id}/curation`):tags;
  assert(aiOutput.tags.some(t=>t.source==='ai'));
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
  }
  const peak=run('exec',ai,'sh','-c','cat /sys/fs/cgroup/memory.peak');
  const aiMemoryEvents=Object.fromEntries(run('exec',ai,'sh','-c','cat /sys/fs/cgroup/memory.events').split('\n').map(line=>{const [key,value]=line.split(/\s+/);return [key,Number(value)];}));
  assert.equal(aiMemoryEvents.oom,0,'AI child/container OOM must not be hidden by service recovery');
  assert.equal(aiMemoryEvents.oom_kill,0,'recovery may not conceal a killed inference child');
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
  console.log(JSON.stringify({passed:true,project,root,appImage,aiImage,identity,fusedSearch,pngUpgrade,tagPNGUpgrade,tagRecovery,photoPolicyUpgrade,photoTag,appPeakBytes:appPeak,aiPeakBytes:peak,aiMemoryEvents}));
}finally{
  if(process.env.FOLIOPATH_TEST_KEEP==='1')console.log(`Kept ${project}: ${root}`);
  else{compose('down','--remove-orphans');console.log(`Removed ${project}; synthetic data and disposable scratch retained`);}
}
