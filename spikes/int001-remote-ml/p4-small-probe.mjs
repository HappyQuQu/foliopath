// Explicit small-sample diagnostic; never imported by default tests or CI.
import assert from 'node:assert/strict';
import {readFileSync,cpSync,mkdirSync,writeFileSync,createReadStream} from 'node:fs';
import {createHash,randomUUID} from 'node:crypto';
import {join} from 'node:path';
import {environment,until,docker} from './p3-environment.mjs';
import {fullSmallCoverage} from './p4-small-result.mjs';
const source=process.env.FOLIOPATH_SMALL_SAMPLE_ROOT;
assert(/^\/tmp\/foliopath-p4-small\.[A-Za-z0-9]+$/.test(source??''));
const manifest=JSON.parse(readFileSync(process.env.FOLIOPATH_SMALL_MANIFEST));
assert.equal(manifest.entries.length,100);
async function verify(root){for(const row of manifest.entries){assert(/^sample-\d{3}\.(jpg|jpeg|png|webp|gif|mp4|mov|mkv)$/.test(row.id));const h=createHash('sha256');for await(const chunk of createReadStream(join(root,row.id)))h.update(chunk);assert.equal(h.digest('hex'),row.sha256);}}
await verify(join(source,'samples'));
const report={profile:'p4-small-functional-v1',quality:'not-scored-no-human-ground-truth',samples:100,images:manifest.images,videos:manifest.videos,steps:[],queries:[]};
report.manifestSHA256=createHash('sha256').update(readFileSync(process.env.FOLIOPATH_SMALL_MANIFEST)).digest('hex');
report.resourceProfile=process.env.FOLIOPATH_TEST_RESOURCE_PROFILE??'';
const e=await environment({port:6099});
try{
  report.appImage=e.appImage;report.aiImage=e.aiImage;
  const settings=await e.request('/settings');
  await e.request('/settings','PATCH',{backgroundConcurrency:1,contentReadConcurrency:1},{'If-Match':settings.r.headers.get('etag')});
  report.backgroundConcurrency=1;report.contentReadConcurrency=1;
  // Copy already isolated anonymous samples, never a developer media library.
  mkdirSync(join(e.media,'sample'));cpSync(join(source,'samples'),join(e.media,'sample'),{recursive:true,errorOnExist:true});
  const created=(await e.request('/libraries','POST',{name:'P4 100 sample',rootPath:'sample'},{'Idempotency-Key':randomUUID()})).value;
  const base='/libraries/'+created.library.id;
  await until('sample scan',async()=>(await e.request('/scans/'+created.scan.id)).value,v=>!['queued','running'].includes(v.status),600).then(v=>assert.equal(v.status,'succeeded'));
  const lib=(await e.request(base)).value;report.catalogAssets=lib.assetCount;assert.equal(lib.assetCount,100);
  for(const kind of ['semantic','faces']){const s=await e.request(base+'/ai/'+kind);await e.request(base+'/ai/'+kind,'PATCH',{enabled:true},{'If-Match':s.r.headers.get('etag')});}
  for(const kind of ['semantic','faces','video-semantic']){
    if(kind==='video-semantic'){
      await e.request(base+'/media-processing/repair?mode=missing','POST');
      await until('video storyboard settlement',async()=>(await e.request(base+'/assets?limit=100')).value.items,v=>v.filter(a=>a.kind==='video').every(a=>['ready','failed','unavailable'].includes(a.storyboard.status)),300);
    }
    const start=Date.now(),op=await e.job(base,kind);let browseRequests=0;
    const terminal=await until(kind,async()=>{await e.request(base+'/assets?limit=10');browseRequests++;return e.operation(op.id);},v=>['succeeded','failed','cancelled'].includes(v.state),1800);
    const status=(await e.request(base+'/ai/'+kind)).value;
    report.steps.push({kind,state:terminal.state,elapsedMs:Date.now()-start,browseRequests,completedItems:terminal.completedItems,coverage:status.coverage});
    console.log(JSON.stringify(report.steps.at(-1)));assert.equal(terminal.state,'succeeded');
  }
  const tags=[];for(const name of ['person','outdoor','indoors','landscape','animal'])tags.push((await e.request('/tags','POST',{name})).value.id);
  const vocab=await e.request('/ai/tag-vocabulary');await e.request('/ai/tag-vocabulary','PUT',{tagIds:tags},{'If-Match':vocab.r.headers.get('etag')});
  const op=await e.job(base,'tag-suggestions');const tagResult=await until('tags',()=>e.operation(op.id),v=>['succeeded','failed','cancelled'].includes(v.state),600);assert.equal(tagResult.state,'succeeded');report.tags=(await e.request(base+'/ai/tag-suggestions?limit=100')).value.coverage;
  for(const q of ['person','outdoor','人物','室外'])for(const type of ['assets','videos']){
    const start=Date.now();const value=(await e.request('/semantic/'+type+'?libraryId='+created.library.id+'&q='+encodeURIComponent(q)+'&limit=20')).value;
    const results=value.items.map(x=>{
      const name=(x.asset??x).name;
      assert(manifest.entries.some(row=>row.id===name),'result absent from anonymous manifest');
      if(type==='videos')assert(Number.isFinite(x.score));
      return type==='videos'?{sample:name,score:x.score,frame:x.matchedFrame}:{sample:name};
    });
    report.queries.push({q,type,elapsedMs:Date.now()-start,count:value.items.length,ids:value.items.map(x=>x.asset?.id??x.id),results});
  }
  report.stats=docker('stats','--no-stream','--format','{{.Name}} {{.MemUsage}}',e.app,e.ai);
  await verify(join(e.media,'sample'));await verify(join(source,'samples'));report.sampleHashesUnchanged=true;
  report.functionalPass=fullSmallCoverage(report);
  assert(report.functionalPass,'Settled jobs do not imply complete coverage; inspect report');
}catch(error){
  report.functionalPass=false;report.failure=error.name;
  report.containerStates=[e.app,e.ai].map(name=>JSON.parse(docker('inspect','--format','{{json .State}}',name)));
  throw error;
}finally{
  writeFileSync(process.env.FOLIOPATH_SMALL_REPORT,JSON.stringify(report,null,2)+'\n');await e.cleanup();
}
console.log('PASS small-sample functional run (not quality/capacity acceptance)');
