// Local isolated synthetic media only. Never mounts or mutates developer app data.
import { execFileSync } from 'node:child_process';
import { readFileSync, writeFileSync, mkdtempSync, mkdirSync, copyFileSync, rmSync } from 'node:fs';
import { createHash, randomUUID } from 'node:crypto';
import { tmpdir } from 'node:os';
import { join, resolve } from 'node:path';
import assert from 'node:assert/strict';

const docker=(...args)=>execFileSync('docker',args,{encoding:'utf8',timeout:240000}).trim();
const root=resolve(import.meta.dirname,'../..');
const groupReview=process.env.FOLIOPATH_GROUP_REVIEW_REAL==='1';
const directoryReview=process.env.FOLIOPATH_PEOPLE_DIRECTORY_REAL==='1';
const photoReview=process.env.FOLIOPATH_PHOTO_FACE_REAL==='1';
const undoReview=process.env.FOLIOPATH_FACE_UNDO_REAL==='1';
const cannotLinkReview=process.env.FOLIOPATH_CANNOT_LINK_REAL==='1';
const recoveryReview=process.env.FOLIOPATH_PERSON_RECOVERY_REAL==='1';
const key=`fp-${groupReview?'cluster':'person'}-${randomUUID().slice(0,8)}`,app=key+'-app',ai=key+'-ai',volume=key+'-data';
const appImage=docker('image','inspect',process.env.FOLIOPATH_PERSON_APP_IMAGE??'foliopath:local-person-review-20261025','--format','{{.Id}}');
const aiImage=docker('image','inspect','foliopath-inference:face-candidate','--format','{{.Id}}');
const browserImage=docker('image','inspect','foliopath-p3-browser:local','--format','{{.Id}}');
const fixture=mkdtempSync(join(tmpdir(),key+'-'));mkdirSync(join(fixture,'synthetic'));
for(const name of ['a.png','b.png'])copyFileSync(join(root,'tests/fixtures/ai-faces/synthetic-pair.png'),join(fixture,'synthetic',name));
const hash=()=>['a.png','b.png'].map(n=>createHash('sha256').update(readFileSync(join(fixture,'synthetic',n))).digest('hex'));
const before=hash();
const manifest=JSON.parse(readFileSync(join(root,'spikes/int001-remote-ml/cache-baseline.json')));
const identity=JSON.parse(docker('run','--rm','--network','none','--entrypoint','python',aiImage,'-m','image_identity'));
const weights=['detection','recognition'].map(k=>manifest.find(e=>e.path===`facial-recognition/buffalo_l/${k}/model.onnx`));assert(weights.every(Boolean));
const config={origin:`http://${ai}:3003`,identity,model_bytes:610841665,license_id:'unreviewed-local-test',face:{identity:{...identity,protocol:'foliopath-face-v1',model:'buffalo_l'},detector_sha256:weights[0].sha256,embedder_sha256:weights[1].sha256,model_bytes:weights.reduce((n,e)=>n+e.size,0),license_id:'unreviewed-local-test',threshold:{schemaVersion:1,profileId:'synthetic-person-review-only',coreSimilarity:.8,edgeSimilarity:.6,minCoreSize:2,qualitySummarySha256:createHash('sha256').update('synthetic review test, not quality calibration').digest('hex')}}};
let origin='',cookie='',csrf=''; const created=[];
async function request(path,method='GET',body,headers={}){const r=await fetch(origin+'/api/v1'+path,{method,signal:AbortSignal.timeout(10000),headers:{Origin:origin,Cookie:cookie,'X-CSRF-Token':csrf,'Content-Type':'application/json',...headers},body:body===undefined?undefined:JSON.stringify(body)});assert(r.ok,`${method} ${path}: ${r.status} ${await r.clone().text()}`);return {r,value:r.status===204?null:await r.json()};}
async function until(label,fn,ready){const end=Date.now()+180000;while(Date.now()<end){let v;try{v=await fn();}catch{}if(v&&ready(v))return v;await new Promise(r=>setTimeout(r,1000));}throw Error(label+' timed out');}
try{
 docker('network','create','--internal',key);created.push(['network',key]);docker('volume','create',volume);created.push(['volume',volume]);
 docker('run','-d','--name',ai,'--network',key,'--read-only','--memory','4g','--cpus','2','--tmpfs','/tmp','--tmpfs','/verified-cache:size=1200m','--mount','type=bind,src=/Volumes/extended/foliopath/model-cache,dst=/cache,readonly',aiImage);created.push(['container',ai]);
 docker('run','-d','--name',app,'--network',key,'--read-only','--tmpfs','/tmp','--mount',`type=volume,src=${volume},dst=/app/data`,'--mount',`type=bind,src=${fixture},dst=/library,readonly`,'-p','127.0.0.1::8080','-e','FOLIOPATH_LISTEN=0.0.0.0:8080','-e','FOLIOPATH_AI_DEPLOYMENT='+JSON.stringify(config),appImage);created.push(['container',app]);docker('network','connect','bridge',app);
 const inspection=JSON.parse(docker('inspect',app))[0];origin='http://127.0.0.1:'+inspection.NetworkSettings.Ports['8080/tcp'][0].HostPort;
 await until('app ready',()=>fetch(origin+'/health/ready',{signal:AbortSignal.timeout(5000)}),r=>r.ok);
 const password='Synthetic-'+randomUUID()+'!';await request('/auth/setup','POST',{username:'person-review-test',displayName:'Synthetic person review',password});
 const login=await request('/auth/login','POST',{username:'person-review-test',password});cookie=login.r.headers.get('set-cookie').split(';')[0];csrf=login.value.csrfToken;
 await until('face available',async()=>(await request('/ai/models')).value.items,a=>a.some(m=>m.active&&m.kind==='face'&&m.state==='available')||a.length===2&&a.every(m=>m.active&&m.state==='available'));
 const library=(await request('/libraries','POST',{name:'Synthetic people review',rootPath:'synthetic'},{'Idempotency-Key':randomUUID()})).value.library;
 const assets=await until('scan',async()=>(await request(`/libraries/${library.id}/assets`)).value.items,a=>a.length===2);
 const setting=await request(`/libraries/${library.id}/ai/faces`);await request(`/libraries/${library.id}/ai/faces`,'PATCH',{enabled:true},{'If-Match':setting.r.headers.get('etag')});
 const job=(await request(`/libraries/${library.id}/ai/faces/jobs`,'POST',{mode:'missing'},{'Idempotency-Key':randomUUID()})).value;
 const done=await until('face analysis',async()=>(await request('/ai/operations/'+job.id)).value,o=>['succeeded','failed','cancelled'].includes(o.state));
 assert.equal(done.state,'succeeded',JSON.stringify(done));console.log('PASS isolated face model and two synthetic photos');
 if(groupReview){
 const groups=await until('groups',async()=>(await request(`/libraries/${library.id}/ai/face-clusters`)).value,v=>v.items.length===2);assert.equal(groups.groupAssignmentAllowed,false);
 assert.deepEqual(groups.counts,{totalGroups:2,pendingGroups:2,totalFaces:4,pendingFaces:4});
 const initialPending=(await request(`/libraries/${library.id}/ai/face-clusters?review=pending&limit=1`)).value;assert(initialPending.nextCursor);
 const group=(await request(`/libraries/${library.id}/ai/face-clusters/${groups.items[0].id}`)).value;assert.equal(group.items.length,2);
 const asset=assets[0];const tag=(await request('/tags','POST',{name:'Protected group manual tag'})).value;const c=await request(`/assets/${asset.id}/curation`);await request(`/assets/${asset.id}/tags`,'PUT',{tagIds:[tag.id]},{'If-Match':c.r.headers.get('etag')});
 const output=join(root,'test-results/cluster-review-real');mkdirSync(output,{recursive:true});
 console.log(docker('run','--rm','--network',key,'--mount',`type=bind,src=${join(root,'web/tests/e2e')},dst=/src/web/tests/e2e,readonly`,'--mount',`type=bind,src=${join(root,'web/playwright.config.ts')},dst=/src/web/playwright.config.ts,readonly`,'--mount',`type=bind,src=${output},dst=/src/test-results/playwright`,'-e',`FOLIOPATH_WEB_E2E_URL=http://${app}:8080`,'-e','FOLIOPATH_GROUP_REVIEW_REAL=1','-e','FOLIOPATH_PEOPLE_DIRECTORY_REAL='+(directoryReview?'1':'0'),'-e','FOLIOPATH_DIRECTORY_PREVIEW_ASSET='+group.items[0].assetId,'-e','FOLIOPATH_PERSON_REVIEW_PASSWORD='+password,'-e','FOLIOPATH_PERSON_REVIEW_LIBRARY='+library.id,'-e','FOLIOPATH_CLUSTER_REVIEW_ID='+groups.items[0].id,'-e','FOLIOPATH_CLUSTER_REVIEW_FIRST_FACE='+group.items[0].faceId,'-e','FOLIOPATH_CLUSTER_REVIEW_SECOND_FACE='+group.items[1].faceId,browserImage,'npx','playwright','test','cluster-review-real.spec.ts','--project=chromium'));
 if(photoReview){
  const bound=(await request(`/assets/${group.items[0].assetId}/faces`)).value.items.find(f=>f.faceId===group.items[0].faceId);
  const owner=(await request('/people/'+bound.personId)).value;const guardKey=randomUUID();
  const stale=await fetch(origin+'/api/v1/ai/face-reviews',{method:'POST',headers:{Origin:origin,Cookie:cookie,'X-CSRF-Token':csrf,'Content-Type':'application/json','Idempotency-Key':guardKey},body:JSON.stringify({action:'exclude_face',faceId:bound.faceId,expectedFaceRevision:bound.revision}),signal:AbortSignal.timeout(5000)});
  assert.equal(stale.status,409);assert.equal((await stale.json()).error.code,'face_constraint_conflict');assert.equal((await request('/people/'+owner.id)).value.revision,owner.revision);
  const destination=(await request('/people','POST',{name:'Photo review target'},{'Idempotency-Key':randomUUID()})).value;
  console.log(docker('run','--rm','--network',key,'--mount',`type=bind,src=${join(root,'web/tests/e2e')},dst=/src/web/tests/e2e,readonly`,'--mount',`type=bind,src=${join(root,'web/playwright.config.ts')},dst=/src/web/playwright.config.ts,readonly`,'--mount',`type=bind,src=${output},dst=/src/test-results/playwright`,'-e',`FOLIOPATH_WEB_E2E_URL=http://${app}:8080`,'-e','FOLIOPATH_PHOTO_FACE_REAL=1','-e','FOLIOPATH_PERSON_REVIEW_PASSWORD='+password,'-e','FOLIOPATH_PERSON_REVIEW_LIBRARY='+library.id,'-e','FOLIOPATH_PHOTO_ASSET='+bound.assetId,'-e','FOLIOPATH_PHOTO_FACE='+bound.faceId,'-e','FOLIOPATH_PHOTO_ORDINAL='+bound.ordinal,'-e','FOLIOPATH_PHOTO_SOURCE='+owner.id,'-e','FOLIOPATH_PHOTO_SOURCE_REV='+owner.revision,'-e','FOLIOPATH_PHOTO_TARGET='+destination.id,browserImage,'npx','playwright','test','photo-face-review-real.spec.ts','--project=chromium'));
  assert.equal((await request('/people/'+owner.id)).value.revision,owner.revision+2);
  assert.equal((await request('/people/'+destination.id)).value.confirmedFaceCount,0);
  if(undoReview){
   console.log(docker('run','--rm','--network',key,'--mount',`type=bind,src=${join(root,'web/tests/e2e')},dst=/src/web/tests/e2e,readonly`,'--mount',`type=bind,src=${join(root,'web/playwright.config.ts')},dst=/src/web/playwright.config.ts,readonly`,'--mount',`type=bind,src=${output},dst=/src/test-results/playwright`,'-e',`FOLIOPATH_WEB_E2E_URL=http://${app}:8080`,'-e','FOLIOPATH_FACE_UNDO_REAL=1','-e','FOLIOPATH_PERSON_REVIEW_PASSWORD='+password,'-e','FOLIOPATH_PERSON_REVIEW_LIBRARY='+library.id,'-e','FOLIOPATH_PHOTO_ASSET='+bound.assetId,'-e','FOLIOPATH_PHOTO_FACE='+bound.faceId,'-e','FOLIOPATH_PHOTO_ORDINAL='+bound.ordinal,'-e','FOLIOPATH_PHOTO_SOURCE='+owner.id,'-e','FOLIOPATH_PHOTO_TARGET='+destination.id,browserImage,'npx','playwright','test','face-review-undo-real.spec.ts','--project=chromium'));
  }

  if(recoveryReview){
   console.log(docker('run','--rm','--network',key,'--mount',`type=bind,src=${join(root,'web/tests/e2e')},dst=/src/web/tests/e2e,readonly`,'--mount',`type=bind,src=${join(root,'web/playwright.config.ts')},dst=/src/web/playwright.config.ts,readonly`,'--mount',`type=bind,src=${output},dst=/src/test-results/playwright`,'-e',`FOLIOPATH_WEB_E2E_URL=http://${app}:8080`,'-e','FOLIOPATH_PERSON_RECOVERY_REAL=1','-e','FOLIOPATH_PERSON_REVIEW_PASSWORD='+password,'-e','FOLIOPATH_PERSON_REVIEW_LIBRARY='+library.id,'-e','FOLIOPATH_PHOTO_SOURCE='+owner.id,'-e','FOLIOPATH_PHOTO_TARGET='+destination.id,browserImage,'npx','playwright','test','person-recovery-real.spec.ts','--project=chromium'));
  }

  if(cannotLinkReview){
   console.log(docker('run','--rm','--network',key,'--mount',`type=bind,src=${join(root,'web/tests/e2e')},dst=/src/web/tests/e2e,readonly`,'--mount',`type=bind,src=${join(root,'web/playwright.config.ts')},dst=/src/web/playwright.config.ts,readonly`,'--mount',`type=bind,src=${output},dst=/src/test-results/playwright`,'-e',`FOLIOPATH_WEB_E2E_URL=http://${app}:8080`,'-e','FOLIOPATH_CANNOT_LINK_REAL=1','-e','FOLIOPATH_MERGE_CHECK_REAL='+(process.env.FOLIOPATH_MERGE_CHECK_REAL==='1'?'1':'0'),'-e','FOLIOPATH_PERSON_REVIEW_PASSWORD='+password,'-e','FOLIOPATH_PERSON_REVIEW_LIBRARY='+library.id,'-e','FOLIOPATH_PHOTO_ASSET='+bound.assetId,'-e','FOLIOPATH_PHOTO_SOURCE='+owner.id,'-e','FOLIOPATH_PHOTO_TARGET='+destination.id,browserImage,'npx','playwright','test','face-cannot-link-real.spec.ts','--project=chromium'));
  }

 }
 const checkDirectory=async(restarted=false)=>{
  const pending=(await request(`/libraries/${library.id}/ai/face-clusters?review=pending`)).value;
  assert.deepEqual(pending.counts,{totalGroups:2,pendingGroups:1,totalFaces:4,pendingFaces:2});assert.equal(pending.items.length,1);assert.notEqual(pending.items[0].id,groups.items[0].id);
  const all=(await request(`/libraries/${library.id}/ai/face-clusters?review=all`)).value;assert.equal(all.items.length,2);assert.equal(all.items.find(g=>g.id===groups.items[0].id).pendingMemberCount,0);
  const stale=await fetch(origin+`/api/v1/libraries/${library.id}/ai/face-clusters?review=pending&limit=1&cursor=${encodeURIComponent(initialPending.nextCursor)}`,{headers:{Cookie:cookie},signal:AbortSignal.timeout(5000)});assert.equal(stale.status,restarted?400:409);assert.equal((await stale.json()).error.code,restarted?'invalid_cursor':'face_cursor_stale');
 };await checkDirectory();
 const people=(await request('/people')).value.items;assert.equal(people.length,directoryReview||photoReview?2:1);const person=people.find(p=>p.name==='Reviewed synthetic');if(directoryReview){assert.equal(person.previewAssetId,group.items[0].assetId);assert.equal(people.find(p=>p.name==='Empty directory fixture').previewAssetId,null);}assert.equal(person.confirmedFaceCount,1);assert.equal(person.assetCount,1);
 const statuses=async()=>{const all=[];for(const a of assets)all.push(...(await request(`/assets/${a.id}/faces`)).value.items);return all};const final=await statuses();assert.equal(final.length,4);assert.equal(final.filter(f=>f.state==='assigned').length,1);assert.equal(final.filter(f=>f.state==='excluded').length,1);assert.equal(final.filter(f=>f.state==='anonymous_core').length,2);
 assert.equal((await request(`/assets/${asset.id}/curation`)).value.tags[0].id,tag.id);assert.deepEqual(hash(),before);assert.equal((await request(`/libraries/${library.id}/assets`)).value.items.length,2);
 docker('restart',app);origin='http://127.0.0.1:'+JSON.parse(docker('inspect',app))[0].NetworkSettings.Ports['8080/tcp'][0].HostPort;await until('restart ready',()=>fetch(origin+'/health/ready',{signal:AbortSignal.timeout(5000)}),r=>r.ok);assert.equal((await request('/people/'+person.id)).value.confirmedFaceCount,1);
 const restored=await until('face projection ready after model activation',statuses,faces=>faces.length===4);if(directoryReview){assert.equal((await request('/people/'+person.id)).value.previewAssetId,group.items[0].assetId);}assert.equal((await request(`/assets/${asset.id}/curation`)).value.tags[0].id,tag.id);
 assert.deepEqual(restored.map(f=>[f.faceId,f.state,f.personId]),final.map(f=>[f.faceId,f.state,f.personId]));assert.deepEqual(hash(),before);await until('cluster directory ready after model activation',async()=>(await request(`/libraries/${library.id}/ai/face-clusters?review=pending`)).value,v=>Array.isArray(v.items));await checkDirectory(true);assert.match(docker('exec',app,'/app/foliopath','database-check'),/passed/);
 writeFileSync(join(output,'evidence.json'),JSON.stringify({timestamp:new Date().toISOString(),appImage,aiImage,browserImage,synthetic:true,realModel:true,photoFaceReview:photoReview,faceReviewUndo:undoReview,personReviewRecovery:recoveryReview,cannotLinkReview,personMergeCheck:process.env.FOLIOPATH_MERGE_CHECK_REAL==='1',mergeBlockedThenAllowedAfterUndo:cannotLinkReview,cannotLinkCommittedReplay:cannotLinkReview,committedSplitMoveNonemptyMergeReplay:recoveryReview,mergedSource404Recovery:recoveryReview,undoAcrossPhotoPersonMergeGroup:undoReview,undoConflictPreservesLaterCuration:undoReview,committedUndoSameKeyReplay:undoReview,exclusionAnonymousRaceRejected:photoReview,realUnknownSuccessReplay:photoReview,peopleDirectoryPreview:directoryReview,emptyDirectoryPerson:directoryReview,pendingDirectoryCounts:true,allGroupsHistory:true,reviewCursorStale:true,groupAssignmentAllowed:false,emptyPersonCreatedSeparately:true,exactAssignment:true,exactExclusion:true,unchangedAnonymousFaces:2,assigned:1,excluded:1,manualTagPreserved:true,sourceHashesPreserved:true,restart:'passed',database:'passed'},null,2)+'\n');console.log('PASS real model single-face group confirmation, other faces, manual tags, source hash, restart and database');
 }else{
 const source=(await request('/people','POST',{name:'Synthetic source'},{'Idempotency-Key':randomUUID()})).value;
 const target=(await request('/people','POST',{name:'Synthetic target'},{'Idempotency-Key':randomUUID()})).value;
 const a=assets.find(a=>a.name==='a.png'),b=assets.find(a=>a.name==='b.png');
 const faces=await until('a faces',async()=>(await request(`/assets/${a.id}/faces`)).value.items,a=>a.length===2);
 const other=await until('b faces',async()=>(await request(`/assets/${b.id}/faces`)).value.items,a=>a.length===2);
 for(const [owner,items]of [[source,faces],[target,[other[0]]]])for(const face of items){const p=(await request('/people/'+owner.id)).value;await request('/ai/face-reviews','POST',{action:'assign_face',faceId:face.faceId,expectedFaceRevision:face.revision,personId:p.id,expectedPersonRevision:p.revision},{'Idempotency-Key':randomUUID()});}
 const tag=(await request('/tags','POST',{name:'Protected synthetic manual tag'})).value;const c=await request(`/assets/${a.id}/curation`);await request(`/assets/${a.id}/tags`,'PUT',{tagIds:[tag.id]},{'If-Match':c.r.headers.get('etag')});
 const output=join(root,'test-results/person-review-real');mkdirSync(output,{recursive:true});
 const browserOutput=docker('run','--rm','--network',key,'--mount',`type=bind,src=${join(root,'web/tests/e2e')},dst=/src/web/tests/e2e,readonly`,'--mount',`type=bind,src=${join(root,'web/playwright.config.ts')},dst=/src/web/playwright.config.ts,readonly`,'--mount',`type=bind,src=${output},dst=/src/test-results/playwright`,'-e',`FOLIOPATH_WEB_E2E_URL=http://${app}:8080`,'-e','FOLIOPATH_PERSON_REVIEW_REAL=1','-e','FOLIOPATH_PERSON_REVIEW_PASSWORD='+password,'-e','FOLIOPATH_PERSON_REVIEW_SOURCE='+source.id,'-e','FOLIOPATH_PERSON_REVIEW_TARGET='+target.id,'-e','FOLIOPATH_PERSON_REVIEW_FACE='+faces[1].faceId,'-e','FOLIOPATH_PERSON_REVIEW_LIBRARY='+library.id,browserImage,'npx','playwright','test','person-review-real.spec.ts','--project=chromium');console.log(browserOutput);
 const targetAfter=(await request('/people/'+target.id)).value;
 assert.equal(targetAfter.confirmedFaceCount,2);assert.equal(targetAfter.assetCount,2);
 assert(!(await request('/people')).value.items.some(p=>p.id===source.id));
 assert.equal((await request(`/assets/${a.id}/curation`)).value.tags[0].id,tag.id);assert.deepEqual(hash(),before);assert.equal((await request(`/libraries/${library.id}/assets`)).value.items.length,2);
 docker('restart',app);
 // Docker may reassign an automatically published host port on restart.
 origin='http://127.0.0.1:'+JSON.parse(docker('inspect',app))[0].NetworkSettings.Ports['8080/tcp'][0].HostPort;
 await until('restart ready',()=>fetch(origin+'/health/ready',{signal:AbortSignal.timeout(5000)}),r=>r.ok);assert.equal((await request('/people/'+target.id)).value.confirmedFaceCount,2);assert.deepEqual(hash(),before);
 const database=docker('exec',app,'/app/foliopath','database-check');assert.match(database,/passed/);
 const evidence={timestamp:new Date().toISOString(),appImage,aiImage,browserImage,synthetic:true,realModel:true,multiFaceRemoval:'other face and photo preserved',exactMove:true,confirmedMerge:true,sourceTombstoned:true,manualTagsPreserved:true,sourceHashesPreserved:true,restart:'passed',database:'passed',targetFaceCount:targetAfter.confirmedFaceCount,targetAssetCount:targetAfter.assetCount};
 writeFileSync(join(output,'evidence.json'),JSON.stringify(evidence,null,2)+'\n');console.log('PASS isolated real-model Docker person review, manual tag, source hashes, restart and database');
 }
}finally{for(const [kind,name]of created.reverse()){try{if(kind==='container')docker('rm','-f',name);else docker(kind,'rm',name);}catch{}}rmSync(fixture,{recursive:true,force:true});}
