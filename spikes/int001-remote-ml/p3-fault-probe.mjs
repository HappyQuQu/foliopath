import assert from 'node:assert/strict';
import {readFileSync,writeFileSync,truncateSync} from 'node:fs';
import {join} from 'node:path';
import {randomUUID} from 'node:crypto';
import {environment,until,docker} from './p3-environment.mjs';
const e=await environment({port:6097});
try{
  const base='/libraries/'+e.libraries[0].id;
  const assets=(await e.request(base+'/assets')).value.items;
  const faceAsset=assets.find(a=>a.name==='faces.png');
  const faces=(await e.request('/assets/'+faceAsset.id+'/faces')).value;
  const generation=(await e.request(base+'/ai/faces')).value.activeGenerationId;
  const manifest=JSON.parse(readFileSync(new URL('./cache-baseline.json',import.meta.url)));
  const weight=manifest.find(m=>m.path==='facial-recognition/buffalo_l/detection/model.onnx');
  const corrupt=join(e.root,'corrupt-model.onnx');writeFileSync(corrupt,'');truncateSync(corrupt,weight.size);
  const cache=['--mount','type=bind,src=/Volumes/extended/foliopath/model-cache,dst=/cache,readonly'];
  for(const [name,args] of [
    ['missing-model',['--tmpfs','/cache','--tmpfs','/verified-cache:size=1200m']],
    ['corrupt-model',[...cache,'--mount',`type=bind,src=${corrupt},dst=/cache/${weight.path},readonly`,'--tmpfs','/verified-cache:size=1200m']],
    ['snapshot-disk-full',[...cache,'--tmpfs','/verified-cache:size=64k']],
    ['snapshot-readonly',[...cache]],
    ['memory-pressure',[...cache,'--memory','128m','--memory-swap','128m','--tmpfs','/verified-cache:size=1200m']],
  ]){
    docker('stop',e.ai);docker('rename',e.ai,e.ai+'-good');let made=false;
    try{
      docker('run','-d','--name',e.ai,'--network',e.key,'--read-only','--tmpfs','/tmp',...args,e.aiImage);made=true;
      await until(name+' rejects availability',e.models,items=>items.filter(m=>m.active).every(m=>m.state==='unavailable'));
      if(name==='memory-pressure')await until('bounded AI OOM',async()=>JSON.parse(docker('inspect',e.ai,'--format','{{json .State}}')),s=>!s.Running&&s.OOMKilled,60);
      assert.equal((await e.request(base+'/assets')).value.items.length,4);
      assert.deepEqual((await e.request('/assets/'+faceAsset.id+'/faces')).value,faces);
      assert.equal((await e.request(base+'/ai/faces')).value.activeGenerationId,generation);
      await e.request(base+'/ai/faces/jobs','POST',{mode:'all'},{'Idempotency-Key':randomUUID()},409);
      console.log('PASS C04 '+name+': unavailable, no false activation, old catalog/faces preserved');
    }finally{
      if(made)docker('rm','-f',e.ai);docker('rename',e.ai+'-good',e.ai);docker('start',e.ai);await e.readyModels();
    }
  }
  await e.done(await e.job(base,'faces','missing'));
  console.log('PASS C04 recovery after all isolated fault containers');
}finally{await e.cleanup();}
