// FR-SCN-004/006: local Docker only, synthetic media and a private temporary DB.
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {copyFileSync, existsSync, mkdirSync, mkdtempSync, readFileSync, realpathSync, writeFileSync} from 'node:fs';
import {join} from 'node:path';
import {tmpdir} from 'node:os';
import {fileURLToPath} from 'node:url';
import {createHash, randomUUID} from 'node:crypto';

const repo=fileURLToPath(new URL('../../',import.meta.url));
const docker=(...args)=>execFileSync('docker',args,{encoding:'utf8',timeout:60000,maxBuffer:32*1024*1024}).trim();
assert(JSON.parse(docker('context','inspect'))[0].Endpoints.docker.Host.startsWith('unix://'),'local Docker only');
assert(process.env.FOLIOPATH_TEST_APP_IMAGE,'explicit local app image required');
const image=docker('image','inspect',process.env.FOLIOPATH_TEST_APP_IMAGE,'--format','{{.Id}}');
const runID=randomUUID(),name='fp-scan-worker-'+runID.slice(0,8),label='org.foliopath.test.scan-worker';
const root=realpathSync(mkdtempSync(join(tmpdir(),name+'-')));
mkdirSync(join(root,'data'));
const files=[];
for(let library=0;library<4;library++) {
  mkdirSync(join(root,'media',`lib-${library}`),{recursive:true});
  for(let i=0;i<25;i++) {
    const file=join(root,'media',`lib-${library}`,`photo-${i}.png`);
    copyFileSync(join(repo,'tests/fixtures/media/viewer-blue-violet.png'),file);files.push(file);
  }
}
const hashes=()=>files.map(file=>createHash('sha256').update(readFileSync(file)).digest('hex'));
const before=hashes(),credentials={username:'scan-worker-test',password:'Synthetic-Scan-Worker-2026!'};
let id,origin,cookie='',csrf='',requests=0,cancelled=0,databaseChecks=0;
const inspect=()=>JSON.parse(docker('inspect',id))[0];
async function until(label,read,predicate,timeout=90000) {
  const end=Date.now()+timeout;let last;
  while(Date.now()<end) { last=await read(); if(predicate(last))return last; await new Promise(r=>setTimeout(r,200)); }
  throw Error(label+' timed out');
}
const refreshOrigin=()=>{const port=inspect().NetworkSettings.Ports['8080/tcp'][0];assert.equal(port.HostIp,'127.0.0.1');origin='http://127.0.0.1:'+port.HostPort;};
async function request(path,method='GET',body,headers={},signal=AbortSignal.timeout(20000)) {
  return fetch(origin+'/api/v1'+path,{method,signal,headers:{'Content-Type':'application/json',Origin:origin,Cookie:cookie,'X-CSRF-Token':csrf,...headers},body:body===undefined?undefined:JSON.stringify(body)});
}
async function json(path,method,body,headers) { const r=await request(path,method,body,headers);const v=await r.json();assert(r.ok,`${path}: ${r.status} ${v.error?.code}`);return v; }
async function login(){const r=await request('/auth/login','POST',credentials);assert(r.ok);cookie=r.headers.get('set-cookie').split(';')[0];csrf=(await r.json()).csrfToken;}
async function scanned(job){return until('scan',()=>json('/scans/'+job.id),s=>s.status==='succeeded');}
try {
  id=docker('create','--pull','never','--name',name,'--label',`${label}=${runID}`,'--restart','unless-stopped','--read-only','--cap-drop','ALL','--security-opt','no-new-privileges:true',
    '--tmpfs','/tmp:rw,noexec,nosuid,size=128m,mode=0700','--memory','1g','--memory-swap','1g','--cpus','1','-p','127.0.0.1::8080',
    ...(process.env.FOLIOPATH_TEST_AI_DEPLOYMENT?['--env','FOLIOPATH_AI_DEPLOYMENT='+process.env.FOLIOPATH_TEST_AI_DEPLOYMENT]:[]),
    '--mount',`type=bind,src=${join(root,'media')},dst=/library,readonly`,'--mount',`type=bind,src=${join(root,'data')},dst=/app/data`,image);
  docker('start',id);refreshOrigin();await until('health',async()=>inspect(),s=>s.State.Health?.Status==='healthy');
  assert.equal((await request('/libraries')).status,401);
  await json('/auth/setup','POST',{...credentials,displayName:'Synthetic scan worker'});await login();
  const libraries=[];
  for(let i=0;i<4;i++) { const v=await json('/libraries','POST',{name:`Synthetic ${i}`,rootPath:`lib-${i}`},{'Idempotency-Key':randomUUID()});libraries.push(v.library.id);await scanned(v.scan); }
  const startTime=inspect().State.StartedAt;
  const duration=Number(process.env.FOLIOPATH_TEST_SCAN_SECONDS??120);
  assert(Number.isInteger(duration)&&duration>=10&&duration<=600);
  const end=Date.now()+duration*1000;
  const paths=libraries.flatMap(library=>[
    `/libraries/${library}/assets`,`/libraries/${library}/scans`,`/tags?libraryId=${library}&kind=image`,
    ...(process.env.FOLIOPATH_TEST_AI_DEPLOYMENT?['faces','semantic','automatic-tags','video-semantic'].map(capability=>`/libraries/${library}/ai/${capability}`):[]),
  ]);
  const load=Array.from({length:8},(_,worker)=>(async()=>{
    let step=worker;
    while(Date.now()<end) {
      const abort=step%5===0;
      try {
        const r=await request(paths[step%paths.length],'GET',undefined,{},abort?AbortSignal.timeout(1):AbortSignal.timeout(20000));
        const result=await r.json();requests++;
        assert(r.ok||r.status===429||abort,`unexpected live read ${paths[step%paths.length]} status ${r.status} ${result.error?.code}`);
      } catch(error) {
        if(abort&&['TimeoutError','AbortError'].includes(error.name)) cancelled++;
        else throw error;
      }
      step++;await new Promise(r=>setTimeout(r,30));
    }
  })());
  const scans=(async()=>{
    let rounds=0;
    while(Date.now()<end) {
      for(const library of libraries) {
        if(Date.now()>=end)break;
        const job=await json(`/libraries/${library}/scans`,'POST');await scanned(job);rounds++;
      }
      await new Promise(r=>setTimeout(r,100));
    }
    return rounds;
  })();
  const checks=(async()=>{
    while(Date.now()<end) {
      assert.equal(docker('exec',id,'/app/foliopath','database-check'),'database check passed');databaseChecks++;
      await new Promise(r=>setTimeout(r,500));
    }
  })();
  const results=await Promise.all([...load,checks,scans]);
  const current=inspect();assert.equal(current.RestartCount,0);assert.equal(current.State.StartedAt,startTime);assert.equal(current.State.OOMKilled,false);
  assert(!docker('logs',id).includes('application.component_stopped'),'component exited during concurrent scans and reads');
  assert.deepEqual(hashes(),before);
  assert(databaseChecks>0);
  docker('restart',id);await until('restart health',async()=>inspect(),s=>s.State.Health?.Status==='healthy');refreshOrigin();await login();
  for(const library of libraries) await until('restart scan',()=>json(`/libraries/${library}`),s=>s.status==='ready');
  assert.equal(docker('exec',id,'/app/foliopath','database-check'),'database check passed');
  assert.deepEqual(hashes(),before);
  // Negative CLI checks have no server, model, media mount, or writable database.
  for(const kind of ['missing','corrupt']) {
    const data=join(root,kind+'-check');mkdirSync(data);
    if(kind==='corrupt')writeFileSync(join(data,'foliopath.db'),'synthetic private secret not a database');
    let failed=false;
    try {docker('run','--rm','--network','none','--read-only','--cap-drop','ALL','--mount',`type=bind,src=${data},dst=/app/data,readonly`,image,'database-check');}
    catch(error) {failed=true;assert.equal(error.status,1);assert.equal(error.stdout.trim(),'');assert.equal(error.stderr.trim(),'foliopath: database check failed');}
    assert(failed,kind+' must fail');
    if(kind==='missing')assert(!existsSync(join(data,'foliopath.db')));
  }
  assert.equal(inspect().RestartCount,0);assert.equal(inspect().State.OOMKilled,false);
  console.log(JSON.stringify({passed:true,name,root,origin,image,duration,requests,cancelled,databaseChecks,scans:results.at(-1),restartCount:current.RestartCount,explicitRestart:true}));
} finally {
  if(id) {
    assert.equal(inspect().Config.Labels[label],runID);
    if(process.env.FOLIOPATH_TEST_KEEP==='1') console.log(`Kept isolated ${name}`);
    else {docker('rm','-f',id);console.log(`Removed owned test container ${name}; synthetic data retained at ${root}`);}
  }
}
