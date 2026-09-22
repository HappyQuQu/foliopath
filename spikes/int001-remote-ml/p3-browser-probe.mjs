import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {mkdirSync,copyFileSync} from 'node:fs';
import {join} from 'node:path';
import {randomUUID} from 'node:crypto';
import {environment,until,docker,repository} from './p3-environment.mjs';
const e=await environment();
try{
  const base='/libraries/'+e.libraries[0].id;
  const tag=(await e.request('/tags','POST',{name:'blue purple gradient'})).value;
  const vocabulary=await e.request('/ai/tag-vocabulary');
  await e.request('/ai/tag-vocabulary','PUT',{tagIds:[tag.id]},{'If-Match':vocabulary.r.headers.get('etag')});
  await e.done(await e.job(base,'tag-suggestions'));
  assert((await e.request(base+'/ai/tag-suggestions')).value.items.length>0);
  const reviewLibraries={};
  for(const browser of ['chromium','firefox','webkit']){
    const rootPath='review-'+browser;mkdirSync(join(e.media,rootPath));
    for(let i=0;i<52;i++)copyFileSync(join(repository,'tests/fixtures/media/viewer-blue-violet.png'),join(e.media,rootPath,String(i).padStart(3,'0')+'.png'));
    const lib=(await e.request('/libraries','POST',{name:rootPath,rootPath},{'Idempotency-Key':randomUUID()})).value.library;
    const path='/libraries/'+lib.id;
    await until('review catalog',async()=>(await e.request(path+'/assets?limit=100')).value.items,v=>v.length===52);
    const settings=await e.request(path+'/ai/semantic');await e.request(path+'/ai/semantic','PATCH',{enabled:true},{'If-Match':settings.r.headers.get('etag')});
    await e.done(await e.job(path,'semantic'));await e.done(await e.job(path,'tag-suggestions'));
    assert((await e.request(path+'/ai/tag-suggestions')).value.nextCursor);
    reviewLibraries[browser]=lib.id;
  }
  mkdirSync(join(e.media,'large'));
  for(let i=0;i<400;i++)copyFileSync(join(repository,'tests/fixtures/media/viewer-blue-violet.png'),join(e.media,'large',String(i).padStart(4,'0')+'.png'));
  const large=(await e.request('/libraries','POST',{name:'P3 large',rootPath:'large'},{'Idempotency-Key':randomUUID()})).value.library;
  await until('large catalog',async()=>(await e.request('/libraries/'+large.id+'/assets?limit=100')).value,v=>v.items.length===100);
  const zero=(await e.request(base+'/assets')).value.items.find(a=>a.name==='gradient.png');
  const faceAsset=(await e.request(base+'/assets')).value.items.find(a=>a.name==='faces.png');
  docker('pause',e.ai);const task=await e.job(base,'faces');
  await until('face claimed',()=>e.operation(task.id),v=>v.state==='running');
  const state=await e.request('/ai/operations/'+task.id);
  await e.request('/ai/operations/'+task.id+'/cancel','POST',{}, {'If-Match':state.r.headers.get('etag')});
  docker('unpause',e.ai);assert.equal((await e.terminal(task)).state,'cancelled');
  function runBrowser(offline=false,d04=false){try{execFileSync('docker',['run','--rm','--network',e.key,
    '-v',join(repository,'web/playwright.config.ts')+':/src/web/playwright.config.ts:ro',
    '-v',join(repository,'web/tests/e2e')+':/src/web/tests/e2e'+(process.env.FOLIOPATH_D04_UPDATE==='1'?'':':ro'),
    '-v',join(repository,'tests/fixtures')+':/src/tests/fixtures:ro',
    '-v',join(repository,'test-results/p3-real'+(offline?'-offline':d04?'-d04':''))+':/src/test-results/playwright',
    '-e','FOLIOPATH_WEB_E2E_URL=http://'+e.app+':8080','-e','FOLIOPATH_P3_E2E=1',
    '-e','FOLIOPATH_P3_PASSWORD='+e.password,'-e','FOLIOPATH_P3_LIBRARIES='+e.libraries.map(l=>l.id).join(','),
    '-e','FOLIOPATH_P3_CANCELLED='+task.id,
    '-e','FOLIOPATH_P3_REVIEW_LIBRARIES='+JSON.stringify(reviewLibraries),
    '-e','FOLIOPATH_P3_LARGE='+large.id,'-e','FOLIOPATH_P3_ZERO_FACE='+zero.id,
    '-e','FOLIOPATH_P3_FACE_ASSET='+faceAsset.id,
    '-e','FOLIOPATH_P3_OFFLINE='+(offline?'1':'0'),
    '-e','FOLIOPATH_D04_VISUAL='+(process.env.FOLIOPATH_D04_VISUAL??'0'),
    'foliopath-p3-browser:local','npx','playwright','test',...(d04?['d04-real.spec.ts','--project=mobile-chromium',...(process.env.FOLIOPATH_D04_UPDATE==='1'?['--update-snapshots']:[])]:['p3-real.spec.ts',...(offline?['--grep','real offline']:process.env.FOLIOPATH_P3_TEST_GREP?['--grep',process.env.FOLIOPATH_P3_TEST_GREP]:['media-matrix.spec.ts','--project=mobile-chromium'])]),'--project=chromium','--project=firefox','--project=webkit'],{stdio:'inherit',timeout:1800000});}
    catch{throw Error('P3 browser assertion failed; see test-results/p3-real'+(offline?'-offline':''));}}
  runBrowser();
  if(process.env.FOLIOPATH_P3_D04==='1')runBrowser(false,true);
  docker('stop',e.ai);await until('offline models',e.models,items=>items.filter(m=>m.active).every(m=>m.state==='unavailable'));
  runBrowser(true);docker('start',e.ai);await e.readyModels();
  console.log('PASS selected real browser cases; task completion is tracked separately in the acceptance checklist');
}finally{await e.cleanup();}
