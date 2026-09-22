// Explicit local-Docker upgrade/paired rollback diagnostic; synthetic media only.
import assert from 'node:assert/strict';
import {randomUUID,createHash} from 'node:crypto';
import {readFileSync,writeFileSync} from 'node:fs';
import {join} from 'node:path';
import {environment,until,docker} from './p3-environment.mjs';
assert(process.env.FOLIOPATH_P4_UPGRADE_IMAGE && process.env.FOLIOPATH_P4_UPGRADE_REPORT);
const next=docker('image','inspect',process.env.FOLIOPATH_P4_UPGRADE_IMAGE,'--format','{{.Id}}');
const e=await environment({port:6097});
const report={oldImage:e.appImage,newImage:next,steps:[],pass:false};
const bases=e.libraries.map(l=>'/libraries/'+l.id);
const status=async(base,kind='semantic')=>(await e.request(base+'/ai/'+kind)).value;
try{
  assert.notEqual(next,e.appImage);
  const before=await Promise.all(bases.map(b=>status(b)));
  report.videoBefore=await Promise.all(bases.map(b=>status(b,'video-semantic')));
  // The shared fixture intentionally includes a short/degraded video, as in
  // p3-business-probe. Require the exact baseline, not false full coverage.
  const videoCounts=s=>['eligible','completed','degraded','failed','stale','complete'].map(k=>s.coverage[k]);
  for(const s of report.videoBefore)assert.deepEqual(videoCounts(s),[2,1,1,0,0,false]);
  assert(before.every(s=>s.coverage.complete && s.coverage.completed>0));
  const oldGeneration=before[0].activeGenerationId;
  const faceGeneration=(await status(bases[0],'faces')).activeGenerationId;
  const asset=(await e.request(bases[0]+'/assets')).value.items.find(a=>a.name==='faces.png');
  const originalHash=createHash('sha256').update(readFileSync(join(e.media,'one','faces.png'))).digest('hex');
  const person=(await e.request('/people','POST',{name:'Upgrade synthetic'},{'Idempotency-Key':randomUUID()})).value;
  const face=(await e.request('/assets/'+asset.id+'/faces')).value.items[0];
  await e.request('/ai/face-reviews','POST',{action:'assign_face',faceId:face.faceId,expectedFaceRevision:face.revision,personId:person.id,expectedPersonRevision:person.revision},{'Idempotency-Key':randomUUID()});
  const tag=(await e.request('/tags','POST',{name:'Upgrade manual'})).value;
  const c=await e.request('/assets/'+asset.id+'/curation');
  await e.request('/assets/'+asset.id+'/tags','PUT',{tagIds:[tag.id]},{'If-Match':c.r.headers.get('etag')});
  async function preserved(){
    assert.equal((await e.request('/people/'+person.id)).value.confirmedFaceCount,1);
    assert((await e.request('/assets/'+asset.id+'/curation')).value.tags.some(t=>t.id===tag.id));
    assert.equal((await status(bases[0],'faces')).activeGenerationId,faceGeneration);
    assert.equal(createHash('sha256').update(readFileSync(join(e.media,'one','faces.png'))).digest('hex'),originalHash);
  }
  e.stopApp();
  const backup=e.key+'-preupgrade';docker('volume','create',backup);e.volumes.push(backup);
  docker('run','--rm','--network','none','--user','0','--entrypoint','sh',
    '--mount',`type=volume,src=${e.volume},dst=/source,readonly`,'--mount',`type=volume,src=${backup},dst=/target`,
    'foliopath-release-http-client:local','-c','cd /source && tar --exclude=./cache --exclude=./tmp -cf - . | tar -xf - -C /target');
  await e.startApp(next);await until('new semantic identity',()=>status(bases[0]),s=>s.activeGenerationId && s.activeGenerationId!==oldGeneration);
  await e.readyModels();
  const upgraded=await Promise.all(bases.map(b=>status(b)));
  assert(upgraded.every(s=>s.coverage.completed===0),'old vectors counted as new generation');
  await preserved();report.oldGeneration=oldGeneration;report.newGeneration=upgraded[0].activeGenerationId;
  report.steps.push('new generation starts without counting old vectors; curation and face generation retained');
  report.videoAfter=[];
  for(const base of bases){
    await e.done(await e.job(base,'semantic'));await e.done(await e.job(base,'video-semantic'));
    assert((await status(base)).coverage.complete);
    const video=await status(base,'video-semantic');report.videoAfter.push(video);
    assert.equal(video.activeGenerationId,report.newGeneration);
    assert.deepEqual(videoCounts(video),[2,1,1,0,0,false]);
  }
  const results=(await e.request('/semantic/assets?libraryId='+e.libraries[0].id+'&q=person&limit=20')).value;
  assert(results.items.length>0);await preserved();
  report.steps.push('both libraries rebuilt; known short-video degradation unchanged; semantic search returns results');
  e.stopApp();await e.startApp(next);await e.readyModels();
  assert.equal((await status(bases[0])).activeGenerationId,report.newGeneration);
  assert((await status(bases[0])).coverage.complete);await preserved();
  report.steps.push('new generation and manual state persist after restart');
  e.stopApp();await e.startApp(e.appImage,backup);await e.readyModels();
  assert.equal((await status(bases[0])).activeGenerationId,oldGeneration);
  assert((await status(bases[0])).coverage.complete);await preserved();
  report.steps.push('old image paired with preupgrade backup restores old generation and manual state');
  report.pass=true;
}finally{
  writeFileSync(process.env.FOLIOPATH_P4_UPGRADE_REPORT,JSON.stringify(report,null,2)+'\n');
  await e.cleanup();
}
console.log(JSON.stringify(report));
