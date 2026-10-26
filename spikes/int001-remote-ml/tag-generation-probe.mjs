import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
const app=JSON.parse(execFileSync('docker',['inspect','foliopath-multilib-app-local-1'],{encoding:'utf8'}))[0];
assert.equal(app.Config.Labels['com.docker.compose.project'],'foliopath-multilib');
assert(app.Mounts.some(m=>m.Destination==='/library'&&!m.RW&&/^(?:\/host_mnt)?\/(?:private\/)?tmp\/foliopath-multilib\.[A-Za-z0-9]+$/.test(m.Source)));
const origin='http://127.0.0.1:6091';
assert(process.env.FOLIOPATH_TEST_PASSWORD);
let cookie='',csrf='';
async function req(path,method='GET',body,extra={}) {
  const r=await fetch(origin+'/api/v1'+path,{method,signal:AbortSignal.timeout(20000),headers:{Origin:origin,Cookie:cookie,'X-CSRF-Token':csrf,'Content-Type':'application/json',...extra},body:body===undefined?undefined:JSON.stringify(body)});
  const value=await r.json(); assert(r.ok,`${path}: ${r.status} ${JSON.stringify(value)}`); return {value,r};
}
const auth=await req('/auth/login','POST',{username:'multilib-test',password:process.env.FOLIOPATH_TEST_PASSWORD});
cookie=auth.r.headers.get('set-cookie').split(';')[0];csrf=auth.value.csrfToken;
const library=(await req('/libraries')).value.items.find(l=>l.name==='Synthetic gamma');assert(library);
const vocabulary=await req('/ai/tag-vocabulary');
const tag=(await req('/tags','POST',{name:'blue purple gradient'})).value;
console.log('tag',tag);
const tagId=tag.id ?? tag.tag?.id;assert(tagId);
await req('/ai/tag-vocabulary','PUT',{tagIds:[tagId]},{'If-Match':vocabulary.r.headers.get('etag')});
try {
  const job=(await req(`/libraries/${library.id}/ai/tag-suggestions/jobs`,'POST',{mode:'all'},{'Idempotency-Key':crypto.randomUUID()})).value;
  let done;
  for(let i=0;i<120;i++) {
    done=(await req('/ai/operations/'+job.id)).value;
    if(['succeeded','failed','cancelled'].includes(done.state)) break;
    await new Promise(r=>setTimeout(r,500));
  }
  console.log('operation',done);assert.equal(done.state,'succeeded');
  const result=(await req(`/libraries/${library.id}/ai/tag-suggestions`)).value;
  assert.equal(result.coverage.completed,1);assert.equal(result.coverage.complete,true);
  console.log('PASS generation completed',JSON.stringify(result));
  assert(result.items.length>0,'non-empty suggestions required for review probe');
  const suggestion=result.items.find(s=>s.tag.tagId===tagId);assert(suggestion);
  const assetPath='/assets/'+suggestion.asset.id+'/curation';
  const before=(await req(assetPath)).value;
  assert(!before.tags.some(t=>t.id===tagId),'test tag already assigned; use a fresh synthetic library');
  const reviewBody={items:[{suggestionId:suggestion.id,action:'accept',expectedSuggestionRevision:suggestion.revision,expectedCurationRevision:before.revision}]};
  const key=crypto.randomUUID();
  const accepted=(await req('/ai/tag-suggestion-reviews','POST',reviewBody,{'Idempotency-Key':key})).value;
  assert.equal(accepted.items[0].outcome,'accepted');
  const replay=(await req('/ai/tag-suggestion-reviews','POST',reviewBody,{'Idempotency-Key':key})).value;
  assert.deepEqual(replay,accepted);
  const tagged=(await req(assetPath)).value;assert(tagged.tags.some(t=>t.id===tagId));
  const reviews=await req(`/libraries/${library.id}/ai/tag-suggestions?status=accepted`);
  assert(reviews.value.items.some(s=>s.tag.tagId===tagId));
  const clear=(await req(`/libraries/${library.id}/ai/tag-suggestion-reviews/clear`,'POST',{confirmation:'clear_tag_suggestion_reviews'},{'If-Match':reviews.r.headers.get('etag'),'Idempotency-Key':crypto.randomUUID()})).value;
  let cleared;
  for(let i=0;i<120;i++) {
    cleared=(await req('/ai/operations/'+clear.id)).value;
    if(['succeeded','failed','cancelled'].includes(cleared.state)) break;
    await new Promise(r=>setTimeout(r,500));
  }
  assert.equal(cleared.state,'succeeded');assert(cleared.completedItems>=1);
  assert.equal((await req(`/libraries/${library.id}/ai/tag-suggestions?status=accepted`)).value.items.length,0);
  assert.deepEqual((await req(assetPath)).value,tagged);
  console.log('PASS non-empty review, idempotent replay, clear, formal tag preserved', {suggestion:suggestion.id,clear:clear.id,deleted:cleared.completedItems});
} finally {
  const current=await req('/ai/tag-vocabulary');
  await req('/ai/tag-vocabulary','PUT',{tagIds:vocabulary.value.entries.map(t=>t.tagId)},{'If-Match':current.r.headers.get('etag')});
}
