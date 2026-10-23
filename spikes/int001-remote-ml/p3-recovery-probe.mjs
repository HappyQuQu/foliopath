// C03/C05: real requests, real inference, stopped-volume backup and paired restore.
import assert from 'node:assert/strict';
import {randomUUID,createHash} from 'node:crypto';
import {readFileSync} from 'node:fs';
import {join} from 'node:path';
import {environment,until,docker} from './p3-environment.mjs';
const e=await environment({port:6096});const {request,job,done,operation}=e;
const base='/libraries/'+e.libraries[0].id;
try{
  const assets=(await request(base+'/assets')).value.items;
  const asset=assets.find(a=>a.name==='faces.png');
  const mediaHash=()=>createHash('sha256').update(readFileSync(join(e.media,'one','faces.png'))).digest('hex');
  const originalHash=mediaHash();
  const person=(await request('/people','POST',{name:'Recovery synthetic'},{'Idempotency-Key':randomUUID()})).value;
  const face=(await request('/assets/'+asset.id+'/faces')).value.items[0];
  await request('/ai/face-reviews','POST',{action:'assign_face',faceId:face.faceId,expectedFaceRevision:face.revision,personId:person.id,expectedPersonRevision:person.revision},{'Idempotency-Key':randomUUID()});
  const tag=(await request('/tags','POST',{name:'Recovery manual'})).value;
  const curation=await request('/assets/'+asset.id+'/curation');
  await request('/assets/'+asset.id+'/tags','PUT',{tagIds:[tag.id]},{'If-Match':curation.r.headers.get('etag')});
  async function preserved(){
    assert.equal((await request(base+'/assets')).value.items.length,4);
    assert.equal((await request('/people/'+person.id)).value.confirmedFaceCount,1);
    assert((await request('/assets/'+asset.id+'/curation')).value.tags.some(t=>t.id===tag.id));
    assert.equal(mediaHash(),originalHash);
  }
  // Killing and graceful shutdown are not user cancellation. The persisted job
  // resumes after its actual lease expires; no direct DB edits advance the clock.
  for(const kill of [false,true]){
    docker('pause',e.ai);const pending=await job(base,'faces');
    await until('face running',()=>operation(pending.id),v=>v.state==='running');
    await new Promise(r=>setTimeout(r,500));e.stopApp(kill);docker('unpause',e.ai);
    await e.startApp();await e.readyModels();
    const recovering=await operation(pending.id);assert.notEqual(recovering.state,'cancelled');
    await done(pending);await preserved();
    console.log('PASS C03 '+(kill?'SIGKILL':'graceful shutdown')+' persisted face lease resumes, curation retained');
  }
  // Break an in-flight connection, not only a health check while idle.
  docker('pause',e.ai);const interrupted=await job(base,'faces');
  await until('face running',()=>operation(interrupted.id),v=>v.state==='running');
  await new Promise(r=>setTimeout(r,500));docker('unpause',e.ai);docker('kill',e.ai);
  const failed=await e.terminal(interrupted);assert.equal(failed.state,'failed',JSON.stringify(failed));
  await preserved();await until('stopped AI unavailable',e.models,items=>items.filter(m=>m.active).every(m=>m.state==='unavailable'));
  docker('start',e.ai);await e.readyModels();await done(await job(base,'faces','missing'));
  console.log('PASS C03 inference interruption fails without false success; restart recovers');
  docker('pause',e.ai);const timedOut=await job(base,'faces');
  await until('face timeout terminal',()=>operation(timedOut.id),v=>['failed','cancelled','succeeded'].includes(v.state),90);
  assert.equal((await operation(timedOut.id)).state,'failed');await preserved();
  await until('paused AI unavailable',e.models,items=>items.filter(m=>m.active).every(m=>m.state==='unavailable'));
  docker('unpause',e.ai);await e.readyModels();await done(await job(base,'faces','missing'));
  console.log('PASS C03 real HTTP timeout fails without user cancellation; service recovery permits retry');
  // Pin wrong expected identity. The existing active generation must not move.
  const generation=(await request(base+'/ai/faces')).value.activeGenerationId;
  e.stopApp();const expected=e.config.identity.runtime_contract_sha256;
  e.config.identity.runtime_contract_sha256='0'.repeat(64);e.config.face.identity.runtime_contract_sha256='0'.repeat(64);
  await e.startApp();await until('mismatched identity unavailable',e.models,items=>items.filter(m=>m.active).every(m=>m.state==='unavailable'));
  assert.equal((await request(base+'/ai/faces')).value.activeGenerationId,generation);
  await request(base+'/ai/faces/jobs','POST',{mode:'all'},{'Idempotency-Key':randomUUID()},409);await preserved();
  e.stopApp();e.config.identity.runtime_contract_sha256=expected;e.config.face.identity.runtime_contract_sha256=expected;await e.startApp();await e.readyModels();
  console.log('PASS C03 identity mismatch fails closed without cross-generation write');
  // Back up only after both queue completion and an orderly stop; never copy live WAL.
  e.stopApp();const backup=e.key+'-backup',restored=e.key+'-restore';
  for(const volume of [backup,restored]){docker('volume','create',volume);e.volumes.push(volume);}
  function copyVolume(source,target){docker('run','--rm','--network','none','--user','0','--entrypoint','sh',
    '--mount',`type=volume,src=${source},dst=/source,readonly`,'--mount',`type=volume,src=${target},dst=/target`,'foliopath-release-http-client:local',
    '-c','cd /source && tar --exclude=./cache --exclude=./tmp -cf - . | tar -xf - -C /target');}
  copyVolume(e.volume,backup);copyVolume(backup,restored);
  await e.startApp(e.appImage,restored);await e.readyModels();await preserved();
  const settings=await request(base+'/ai/faces');await done((await request(base+'/ai/faces/derived-clear','POST',{confirmation:'clear_derived_face_data'},{'If-Match':settings.r.headers.get('etag'),'Idempotency-Key':randomUUID()})).value);
  const disabled=await request(base+'/ai/faces');await request(base+'/ai/faces','PATCH',{enabled:true},{'If-Match':disabled.r.headers.get('etag')});
  await done(await job(base,'faces'));await preserved();
  console.log('PASS C05 stopped-volume backup restores people/manual tags and rebuildable face relationships; originals unchanged');
}finally{await e.cleanup();}
