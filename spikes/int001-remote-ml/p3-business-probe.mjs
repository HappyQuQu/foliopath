import assert from 'node:assert/strict';
import {copyFileSync,utimesSync,renameSync,readFileSync} from 'node:fs';
import {join} from 'node:path';
import {randomUUID,createHash} from 'node:crypto';
import {environment,until,docker,repository} from './p3-environment.mjs';
const e=await environment({port:6098});const {request,job,done,libraries,operation,terminal,rescan}=e;
const bases=libraries.map(l=>'/libraries/'+l.id);
try{
  console.log('PASS P3 bootstrap: real image/video/face inference in two synthetic libraries');
  // B05: existing aggregate-only video failures must conservatively become pending
  // after membership changes, not be attributed to arbitrary surviving assets.
  const video=async base=>(await request(base+'/ai/video-semantic')).value;
  const images=async base=>(await request(base+'/ai/semantic')).value;
  for(const base of bases){const v=await video(base);assert.equal(v.coverage.eligible,2);assert.equal(v.coverage.completed,1);assert.equal(v.coverage.degraded,1);assert.equal(v.coverage.complete,false);assert.equal((await images(base)).coverage.completed,2);}
  const otherBefore=(await video(bases[1])).coverage;
  copyFileSync(join(e.root,'clip.mp4'),join(e.media,'one','added.mp4'));
  copyFileSync(join(repository,'tests/fixtures/media/viewer-blue-violet.png'),join(e.media,'one','added.png'));
  await rescan(bases[0]);assert.equal((await video(bases[0])).coverage.eligible,3);assert.equal((await video(bases[0])).coverage.complete,false);
  await request(bases[0]+'/media-processing/repair?mode=missing','POST');
  await until('added storyboard',async()=>(await request(bases[0]+'/assets')).value.items,v=>v.find(a=>a.name==='added.mp4')?.storyboard.status==='ready');
  await done(await job(bases[0],'video-semantic','missing'));await done(await job(bases[0],'semantic','missing'));
  assert.equal((await video(bases[0])).coverage.completed,2);assert.equal((await images(bases[0])).coverage.completed,3);
  for(const name of ['added.mp4','added.png'])utimesSync(join(e.media,'one',name),new Date(),new Date(Date.now()+10000));
  await rescan(bases[0]);assert.equal((await video(bases[0])).coverage.stale,1);assert.equal((await images(bases[0])).coverage.stale,1);
  await request(bases[0]+'/media-processing/repair?mode=missing','POST');
  await until('changed storyboard',async()=>(await request(bases[0]+'/assets')).value.items,v=>v.find(a=>a.name==='added.mp4')?.storyboard.status==='ready');
  await done(await job(bases[0],'video-semantic','missing'));await done(await job(bases[0],'semantic','missing'));
  for(const name of ['added.mp4','added.png'])renameSync(join(e.media,'one',name),join(e.root,name));
  await rescan(bases[0]);const shrunk=await video(bases[0]);assert.equal(shrunk.coverage.eligible,2);assert.equal(shrunk.coverage.completed,1);assert.equal(shrunk.coverage.degraded,0);assert.equal(shrunk.coverage.complete,false);
  await done(await job(bases[0],'video-semantic','missing'));assert.equal((await video(bases[0])).coverage.degraded,1);
  assert.deepEqual((await video(bases[1])).coverage,otherBefore);
  console.log('PASS B05 image/video add-change-remove, stale exclusion, conservative pending, reclassification, other library unchanged');
  // C01: actual generated suggestions and mixed-library durable replay.
  const tag=(await request('/tags','POST',{name:'blue purple gradient'})).value;
  const vocab=await request('/ai/tag-vocabulary');await request('/ai/tag-vocabulary','PUT',{tagIds:[tag.id]},{'If-Match':vocab.r.headers.get('etag')});
  const suggestions=[];
  for(const base of bases){await done(await job(base,'tag-suggestions'));const s=(await request(base+'/ai/tag-suggestions')).value.items.find(s=>s.asset.name==='gradient.png');assert(s,'nonempty real gradient suggestion required');suggestions.push(s);}
  const rows=[];for(const [i,s] of suggestions.entries()){const c=(await request('/assets/'+s.asset.id+'/curation')).value;rows.push({suggestionId:s.id,action:i?'dismiss':'accept',expectedSuggestionRevision:s.revision,expectedCurationRevision:c.revision});}
  const reviewKey=randomUUID(),body={items:rows};
  const stale=(await request('/ai/tag-suggestion-reviews','POST',{items:rows.map(r=>({...r,expectedSuggestionRevision:r.expectedSuggestionRevision+1}))},{'Idempotency-Key':randomUUID()})).value;
  assert(stale.items.every(i=>i.outcome==='conflict'),JSON.stringify(stale));
  const reviewed=(await request('/ai/tag-suggestion-reviews','POST',body,{'Idempotency-Key':reviewKey})).value;
  assert.deepEqual(reviewed.items.map(i=>i.outcome),['accepted','dismissed']);
  assert.deepEqual((await request('/ai/tag-suggestion-reviews','POST',body,{'Idempotency-Key':reviewKey})).value,reviewed);
  await request('/ai/tag-suggestion-reviews','POST',{items:[{...rows[0],action:'dismiss'}]},{'Idempotency-Key':reviewKey},409);
  assert.deepEqual((await request('/ai/tag-suggestion-reviews','POST',body,{'Idempotency-Key':randomUUID()})).value,reviewed);
  const tagged=(await request('/assets/'+suggestions[0].asset.id+'/curation')).value;assert(tagged.tags.some(t=>t.id===tag.id));
  const otherReview=(await request(bases[1]+'/ai/tag-suggestions?status=dismissed')).value;
  const snapshot=await request(bases[0]+'/ai/tag-suggestions?status=accepted');
  await done((await request(bases[0]+'/ai/tag-suggestion-reviews/clear','POST',{confirmation:'clear_tag_suggestion_reviews'},{'If-Match':snapshot.r.headers.get('etag'),'Idempotency-Key':randomUUID()})).value);
  assert.deepEqual((await request('/assets/'+suggestions[0].asset.id+'/curation')).value,tagged);
  assert.deepEqual((await request(bases[1]+'/ai/tag-suggestions?status=dismissed')).value,otherReview);
  const afterClearReplay=(await request('/ai/tag-suggestion-reviews','POST',body,{'Idempotency-Key':reviewKey})).value;
  assert.equal(afterClearReplay.items[0].outcome,'conflict',JSON.stringify(afterClearReplay));
  assert.deepEqual(afterClearReplay.items[1],reviewed.items[1]);
  assert.deepEqual((await request('/assets/'+suggestions[0].asset.id+'/curation')).value,tagged);
  console.log('PASS C01 mixed-library accept/dismiss, replay/conflict, cleared mixed ledger not stale-replayed, clear isolation and manual tag preservation; no tag undo endpoint exists');
  // C02: face holds the shared slot; unrelated work cancels before processing.
  const settings=await request('/settings');await request('/settings','PATCH',{backgroundConcurrency:1},{'If-Match':settings.r.headers.get('etag')});
  const cancelled=[];
  for(const kind of ['semantic','video-semantic','tag-suggestions','scan','face-clear']){
    docker('pause',e.ai);
    const holder=await job(bases[0],'faces');await until('face holder',()=>operation(holder.id),v=>v.state==='running');
    await new Promise(r=>setTimeout(r,500));
    let waiting;
    if(kind==='scan')waiting=(await request(bases[1]+'/scans','POST')).value;
    else if(kind==='face-clear'){const s=await request(bases[1]+'/ai/faces');waiting=(await request(bases[1]+'/ai/faces/derived-clear','POST',{confirmation:'clear_derived_face_data'},{'If-Match':s.r.headers.get('etag'),'Idempotency-Key':randomUUID()})).value;}
    else waiting=await job(bases[1],kind);
    const path=(kind==='scan'?'/scans/':'/ai/operations/')+waiting.id;
    await until('waiter claimed',async()=>(await request(path)).value,v=>(v.state??v.status)==='running');
    const state=await request(path);await request(path+'/cancel','POST',undefined,{'If-Match':state.r.headers.get('etag')});
    const result=await until('waiter cancelled',async()=>(await request(path)).value,v=>(v.state??v.status)==='cancelled',30);
    assert.equal(result.completedItems??result.discoveredAssets,0);assert.equal((await operation(holder.id)).state,'running');
    const browse=(await request(bases[1]+'/assets')).value;assert.equal(browse.items.length,4);
    cancelled.push(waiting.id);docker('unpause',e.ai);await done(holder);
  }
  console.log('PASS C02 face shared-budget holder, image/video/tag/scan/clear wait cancellation, browse preserved',cancelled.length);
  console.log('P3 business probe completed');
}finally{await e.cleanup();}
