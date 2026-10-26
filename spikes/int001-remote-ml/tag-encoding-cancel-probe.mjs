// Local synthetic instance only. Pausing the AI tests cancellation of a real
// uncached encoding request, not cancellation inside an ONNX kernel.
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
const app=JSON.parse(execFileSync('docker',['inspect','foliopath-multilib-app-local-1'],{encoding:'utf8'}))[0];
assert.equal(app.Config.Labels['com.docker.compose.project'],'foliopath-multilib');
assert(app.Mounts.some(m=>m.Destination==='/library'&&!m.RW&&/^(?:\/host_mnt)?\/(?:private\/)?tmp\/foliopath-multilib\.[A-Za-z0-9]+$/.test(m.Source)));
const ai='foliopath-multilib-machine-learning-1',origin='http://127.0.0.1:6091';
assert(process.env.FOLIOPATH_TEST_PASSWORD);
let cookie='',csrf='';
async function req(path,method='GET',body,extra={}){
 const r=await fetch(origin+'/api/v1'+path,{method,signal:AbortSignal.timeout(20000),headers:{Origin:origin,Cookie:cookie,'X-CSRF-Token':csrf,'Content-Type':'application/json',...extra},body:body===undefined?undefined:JSON.stringify(body)});
 const value=await r.json();assert(r.ok,`${path}: ${r.status} ${JSON.stringify(value)}`);return {value,r};
}
async function until(read,ready){for(let i=0;i<120;i++){const v=await read();if(ready(v))return v;await new Promise(r=>setTimeout(r,500));}throw Error('timed out');}
const login=await req('/auth/login','POST',{username:'multilib-test',password:process.env.FOLIOPATH_TEST_PASSWORD});cookie=login.r.headers.get('set-cookie').split(';')[0];csrf=login.value.csrfToken;
await until(async()=>(await req('/ai/models')).value,v=>v.items.some(m=>m.active&&m.state==='available'));
const library=(await req('/libraries')).value.items.find(l=>l.name==='Synthetic gamma');assert(library);
const base=`/libraries/${library.id}`;
const asset=(await req(base+'/assets')).value.items[0];assert.equal(asset.name,'gamma.png');
const original=await req('/ai/tag-vocabulary');
const tag=(await req('/tags','POST',{name:'gradient '+crypto.randomUUID().slice(0,8)})).value;
const curation=(await req('/assets/'+asset.id+'/curation')).value;
const operation=async id=>(await req('/ai/operations/'+id)).value;
const start=async()=>(await req(base+'/ai/tag-suggestions/jobs','POST',{mode:'all'},{'Idempotency-Key':crypto.randomUUID()})).value;
let paused=false;
try{
 await req('/ai/tag-vocabulary','PUT',{tagIds:[tag.id]},{'If-Match':original.r.headers.get('etag')});
 execFileSync('docker',['pause',ai]);paused=true;
 const job=await start();assert.equal(job.totalItems,1);
 await until(()=>operation(job.id),v=>v.state==='running');
 await new Promise(r=>setTimeout(r,1000));
 const current=await req('/ai/operations/'+job.id);assert.equal(current.value.state,'running');assert.equal(current.value.completedItems,0);
 const began=Date.now();
 await req('/ai/operations/'+job.id+'/cancel','POST',undefined,{'If-Match':current.r.headers.get('etag')});
 const done=await until(()=>operation(job.id),v=>['cancelled','failed','succeeded'].includes(v.state));
 assert.equal(done.state,'cancelled');assert.equal(done.completedItems,0);
 assert.equal(execFileSync('docker',['inspect','--format','{{.State.Paused}}',ai],{encoding:'utf8'}).trim(),'true');
 assert.equal((await req(base+'/ai/tag-suggestions')).value.items.length,0);
 console.log('PASS cancelled with AI paused',{job:job.id,elapsedMs:Date.now()-began});
 execFileSync('docker',['unpause',ai]);paused=false;
 const retry=await start();const completed=await until(()=>operation(retry.id),v=>['cancelled','failed','succeeded'].includes(v.state));
 assert.equal(completed.state,'succeeded');assert.equal(completed.completedItems,1);
 assert.deepEqual((await req('/assets/'+asset.id+'/curation')).value,curation);
 assert.equal((await operation(job.id)).state,'cancelled');
 console.log('PASS fresh text encoding retry completed; curation and cancelled history preserved',retry.id);
}finally{
 if(paused)execFileSync('docker',['unpause',ai]);
 const current=await req('/ai/tag-vocabulary');
 await req('/ai/tag-vocabulary','PUT',{tagIds:original.value.entries.map(t=>t.tagId)},{'If-Match':current.r.headers.get('etag')});
}
