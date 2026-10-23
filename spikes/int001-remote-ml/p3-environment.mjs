// Isolated local-Docker acceptance harness. No real media or production state.
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {readFileSync,mkdtempSync,mkdirSync,copyFileSync,rmSync} from 'node:fs';
import {randomUUID,createHash} from 'node:crypto';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {fileURLToPath} from 'node:url';
export const repository=fileURLToPath(new URL('../../',import.meta.url));
export const docker=(...args)=>{
  try{return execFileSync('docker',args,{encoding:'utf8',timeout:60000}).trim();}
  catch(error){throw Error('Docker acceptance command failed: '+(error.stdout??'')+(error.stderr??''));}
};
export async function until(label,read,ready,attempts=150){
  let value;
  for(let i=0;i<attempts;i++){value=await read();if(ready(value))return value;await new Promise(r=>setTimeout(r,1000));}
  throw Error(label+' timed out: '+JSON.stringify(value));
}
export async function environment({port=6095}={}){
  const resourceProfile=process.env.FOLIOPATH_TEST_RESOURCE_PROFILE??'';
  assert(['','p4-native','p4-large'].includes(resourceProfile),'unknown test resource profile');
  const appMemory=resourceProfile==='p4-large'?'1024m':'768m';
  const aiMemory=resourceProfile==='p4-large'?'2304m':'2560m';
  const appLimits=resourceProfile?['--cpus','1','--memory',appMemory,'--memory-swap',appMemory]:[];
  const aiLimits=resourceProfile?['--cpus','3','--memory',aiMemory,'--memory-swap',aiMemory]:[];
  const modelCache=process.env.FOLIOPATH_TEST_MODEL_CACHE??'/Volumes/extended/foliopath/model-cache';
  const runLabel=['--label',`foliopath.test.run=${process.env.FOLIOPATH_TEST_RUN_ID??'unassigned'}`];
  const key='fp-p3-'+randomUUID().slice(0,8),app=key+'-app',ai=key+'-ai',volume=key+'-data';
  const root=mkdtempSync(join(tmpdir(),'fp-p3-')),media=join(root,'media');mkdirSync(media);
  for(const library of ['one','two']){
    const dir=join(media,library);mkdirSync(dir);
    copyFileSync(join(repository,'tests/fixtures/ai-faces/synthetic-pair.png'),join(dir,'faces.png'));
    copyFileSync(join(repository,'tests/fixtures/media/viewer-blue-violet.png'),join(dir,'gradient.png'));
    copyFileSync(join(repository,'tests/fixtures/media/range-video.mp4'),join(dir,'short.mp4'));
  }
  // Fixture preparation only; every application/inference assertion runs in Docker.
  const appImage=docker('image','inspect',process.env.FOLIOPATH_P3_APP_IMAGE ?? 'foliopath-app:p3','--format','{{.Id}}');
  // Hold a unique tag across stop/recreate cycles, including concurrent rebuilds
  // of the developer's candidate tag under Docker's containerd image store.
  docker('tag',appImage,key+'-app-image');
  try {execFileSync('ffmpeg',['-hide_banner','-loglevel','error','-f','lavfi','-i','testsrc2=size=160x96:rate=12','-t','3','-an','-c:v','libx264','-preset','ultrafast','-pix_fmt','yuv420p',join(root,'clip.mp4')],{timeout:30000});}
  catch(error){rmSync(root,{recursive:true,force:true});throw error;}
  for(const library of ['one','two'])copyFileSync(join(root,'clip.mp4'),join(media,library,'clip.mp4'));
  const aiImage=docker('image','inspect',process.env.FOLIOPATH_P3_AI_IMAGE??'foliopath-inference:face-candidate','--format','{{.Id}}');
  const identity=JSON.parse(docker('run','--rm','--network','none','--entrypoint','python',aiImage,'-m','image_identity'));
  const manifest=JSON.parse(readFileSync(new URL('./cache-baseline.json',import.meta.url)));
  const weights=['detection','recognition'].map(k=>manifest.find(e=>e.path===`facial-recognition/buffalo_l/${k}/model.onnx`));
  const config={origin:`http://${ai}:3003`,identity,model_bytes:610841665,license_id:'unreviewed-local-test',face:{
    identity:{...identity,protocol:'foliopath-face-v1',model:'buffalo_l'},detector_sha256:weights[0].sha256,embedder_sha256:weights[1].sha256,
    model_bytes:weights.reduce((n,w)=>n+w.size,0),license_id:'unreviewed-local-test',threshold:{schemaVersion:1,profileId:'p3-synthetic-only',coreSimilarity:.8,edgeSimilarity:.6,minCoreSize:2,qualitySummarySha256:createHash('sha256').update('NOT quality calibration').digest('hex')}}};
  const origin='http://127.0.0.1:'+port,password='Synthetic-'+randomUUID()+'!';let cookie='',csrf='',running=false,aiRunning=false;
  const volumes=[volume],containers=[];
  docker('network','create','--internal',key);docker('network','create',key+'-edge');docker('volume','create',volume);
  async function request(path,method='GET',body,headers={},expected){
    // Synchronous browser/Docker runs pause Node's socket lifecycle handling.
    // Do not reuse an idle socket afterwards, and never blindly replay mutations.
    const r=await fetch(origin+'/api/v1'+path,{method,signal:AbortSignal.timeout(20000),headers:{Connection:'close',Origin:origin,Cookie:cookie,'X-CSRF-Token':csrf,'Content-Type':'application/json',...headers},body:body===undefined?undefined:JSON.stringify(body)});
    const value=r.status===204?null:await r.json();assert(expected?r.status===expected:r.ok,`${method} ${path}: ${r.status} ${JSON.stringify(value)}`);return{r,value};
  }
  async function startApp(image=appImage,data=volume){
    docker('run','-d','--name',app,...runLabel,...appLimits,'--network',key+'-edge','--network',key,'--read-only','--tmpfs','/tmp','-p',`127.0.0.1:${port}:8080`,
      '--mount',`type=volume,src=${data},dst=/app/data`,'--mount',`type=bind,src=${media},dst=/library,readonly`,
      '-e','FOLIOPATH_LISTEN=0.0.0.0:8080','-e','FOLIOPATH_AI_DEPLOYMENT='+JSON.stringify(config),image);running=true;
    await until('core ready',async()=>{
      if(docker('inspect',app,'--format','{{.State.Running}}')!=='true')throw Error('Core exited: '+docker('logs',app));
      try{return(await fetch(origin+'/health/ready',{signal:AbortSignal.timeout(3000)})).status;}catch{return 0;}
    },v=>v===200);
  }
  function stopApp(kill=false){if(running){docker(kill?'kill':'stop',app);docker('rm',app);running=false;}}
  function startAI(){docker('run','-d','--name',ai,...runLabel,...aiLimits,'--network',key,'--read-only','--tmpfs','/tmp','--tmpfs','/verified-cache:size=1200m','--mount',`type=bind,src=${modelCache},dst=/cache,readonly`,aiImage);aiRunning=true;}
  const models=async()=>(await request('/ai/models')).value.items;
  const readyModels=()=>until('models available',models,v=>v.filter(m=>m.active&&m.state==='available').length===2);
  const job=async(base,kind,mode='all')=>(await request(base+`/ai/${kind}/jobs`,'POST',{mode},{'Idempotency-Key':randomUUID()})).value;
  const operation=async id=>(await request('/ai/operations/'+id)).value;
  const terminal=op=>until('operation '+op.id,()=>operation(op.id),v=>['succeeded','failed','cancelled'].includes(v.state));
  const done=async op=>{const value=await terminal(op);assert.equal(value.state,'succeeded',JSON.stringify(value));return value;};
  const rescan=async base=>{const scan=(await request(base+'/scans','POST')).value;const result=await until('scan',async()=>(await request('/scans/'+scan.id)).value,v=>!['queued','running'].includes(v.status));assert.equal(result.status,'succeeded');};
  async function cleanup(){
    if(aiRunning&&docker('inspect',ai,'--format','{{.State.Paused}}')==='true')docker('unpause',ai);
    stopApp();if(aiRunning){docker('stop',ai);docker('rm',ai);}
    for(const name of containers)docker('rm','-f',name);
    for(const name of volumes)docker('volume','rm',name);
    docker('network','rm',key);docker('network','rm',key+'-edge');docker('image','rm',key+'-app-image');rmSync(root,{recursive:true,force:true});
  }
  try{
    await startApp();await request('/auth/setup','POST',{username:'p3-test',displayName:'P3 synthetic',password});
    const auth=await request('/auth/login','POST',{username:'p3-test',password});cookie=auth.r.headers.get('set-cookie').split(';')[0];csrf=auth.value.csrfToken;
    startAI();await readyModels();const libraries=[];
    for(const rootPath of ['one','two']){
      const lib=(await request('/libraries','POST',{name:'P3 '+rootPath,rootPath},{'Idempotency-Key':randomUUID()})).value.library;libraries.push(lib);
      const base='/libraries/'+lib.id;await until('catalog',async()=>(await request(base+'/assets')).value.items,v=>v.length===4);
      for(const kind of ['semantic','faces']){const s=await request(base+'/ai/'+kind);await request(base+'/ai/'+kind,'PATCH',{enabled:true},{'If-Match':s.r.headers.get('etag')});}
      await done(await job(base,'semantic'));await done(await job(base,'faces'));
      await request(base+'/media-processing/repair?mode=missing','POST');
      await until('storyboard',async()=>(await request(base+'/assets')).value.items,v=>v.some(a=>a.name==='clip.mp4'&&a.storyboard.status==='ready'));
      await done(await job(base,'video-semantic'));
    }
    return{key,app,ai,volume,volumes,containers,root,media,appImage,aiImage,config,password,origin,libraries,request,models,readyModels,job,operation,terminal,done,rescan,startApp,stopApp,cleanup};
  }catch(error){await cleanup();throw error;}
}
