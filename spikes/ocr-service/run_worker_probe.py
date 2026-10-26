"""Synthetic-only OCR application integration. Never mounts application DB or real media."""
import argparse,hashlib,json,shutil,subprocess,tempfile,time,uuid,zlib,struct
from pathlib import Path
ROOT=Path(__file__).resolve().parents[2]
DEFAULT_AI="sha256:e995f5748c36ae8de9f3dd86633e4a2f60f3f6d6b7eaaba6616bbdf6ee73da21"
def cmd(args,timeout=60):return subprocess.run(args,check=True,capture_output=True,timeout=timeout)
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--probe',required=True);p.add_argument('--ai',default=DEFAULT_AI);p.add_argument('--output',type=Path,required=True);a=p.parse_args()
 AI=cmd(['docker','image','inspect',a.ai,'--format','{{.Id}}']).stdout.decode().strip()
 out=a.output.resolve();out.mkdir(parents=True,exist_ok=True);library=out/'library';library.mkdir();contracts=out/'contracts';contracts.mkdir();observations=out/'observations';observations.mkdir();scratch=out/'scratch';scratch.mkdir()
 for name in ('bilingual.png','exif-oriented.jpg','blank.png','corrupt.png'):shutil.copyfile(ROOT/'tests/fixtures/ocr'/name,library/name)
 (library/'empty.png').write_bytes(b'')
 digest=lambda f:hashlib.sha256(f.read_bytes()).hexdigest()
 source_hashes={p.name:digest(p) for p in library.iterdir()}
 trusted='import server,json,photo_policy,multilingual_contract;from pathlib import Path;x=server.expected_identity(json.loads(Path("/opt/foliopath-inference/cache-manifest.json").read_bytes()));print(json.dumps({"semantic":multilingual_contract.expected_identity(),"face":{**x,"protocol":"foliopath-face-v1","model":"buffalo_l"},"tag":photo_policy.expected_identity()}))'
 (contracts/'baseline-identities.json').write_bytes(cmd(['docker','run','--rm','--network','none','--read-only','--entrypoint','python',AI,'-c',trusted]).stdout)
 (contracts/'ocr-identity.json').write_bytes(cmd(['docker','run','--rm','--network','none','--read-only','--entrypoint','cat',AI,'/opt/foliopath-inference/ocr-identity.json']).stdout)
 probe=cmd(['docker','image','inspect',a.probe,'--format','{{.Id}}']).stdout.decode().strip()
 prefix='fp-ocr-worker-'+uuid.uuid4().hex[:8];network=prefix+'-net';service=prefix+'-ml';client=prefix+'-app';names=[]
 cmd(['docker','network','create','--internal',network])
 try:
  cmd(['docker','run','-d','--name',service,'--network',network,'--network-alias','machine-learning','--read-only','--memory','4g','--memory-swap','4g','--cpus','2','--pids-limit','128','--cap-drop','ALL','--security-opt','no-new-privileges','--tmpfs','/tmp:size=256m','--mount','type=bind,src=/Volumes/extended/foliopath/model-experiments/multilingual-integration-cache,dst=/cache,readonly','--mount','type=bind,src=/Volumes/extended/foliopath/ocr-feasibility-cache,dst=/ocr-cache,readonly','--mount',f'type=bind,src={scratch},dst=/verified-cache',AI]);names.append(service)
  ready=False
  for _ in range(80):
   try:cmd(['docker','exec',service,'python','-c',"import urllib.request;assert urllib.request.urlopen('http://127.0.0.1:3003/ping',timeout=1).read()==b'pong'"],5);ready=True;break
   except subprocess.CalledProcessError:time.sleep(1)
  assert ready,'AI service not ready'
  cmd(['docker','create','--name',client,'--network',network,'--read-only','--memory','2g','--memory-swap','2g','--cpus','2','--cap-drop','ALL','--security-opt','no-new-privileges','--tmpfs','/tmp:size=512m','--mount',f'type=bind,src={library},dst=/library,readonly','--mount',f'type=bind,src={contracts},dst=/contracts,readonly','--mount',f'type=bind,src={observations},dst=/observations','-e','FOLIOPATH_OCR_TEST_ORIGIN=http://machine-learning:3003',probe]);names.append(client)
  result=subprocess.run(['docker','start','-a',client],capture_output=True,timeout=480);(out/'probe.log').write_bytes(result.stdout+result.stderr)
  state=json.loads(cmd(['docker','inspect',client]).stdout)[0]['State'];assert result.returncode==0 and state['ExitCode']==0 and not state['OOMKilled'],'worker application probe failed'
  metrics=json.loads(cmd(['docker','exec',service,'python','-c',"import json;from pathlib import Path;print(json.dumps({'peak':int(Path('/sys/fs/cgroup/memory.peak').read_text()),'events':dict(x.split() for x in Path('/sys/fs/cgroup/memory.events').read_text().splitlines())}))"]).stdout)
  assert int(metrics['events']['oom_kill'])==0
  cmd(['docker','stop','--time','15',service]);assert not list(scratch.iterdir()),'snapshot leak'
  assert {p.name:digest(p) for p in library.iterdir()}==source_hashes,'synthetic source modified'
  report={'status':'passed','aiImage':AI,'probeImage':probe,'clientState':state,'serviceMetrics':metrics,'syntheticSourceSHA256':source_hashes,'observations':json.loads((observations/'worker.json').read_text()),'isolation':'internal network, no ports; single readonly synthetic media mount; DB only inside Linux tmpfs','scope':'actual application/management/safe source/native OCR/durable tasks, not quality or capacity/release'}
  (out/'result.json').write_text(json.dumps(report,ensure_ascii=False,indent=2)+'\n');print(json.dumps({'status':'passed','probeImage':probe}))
 finally:
  for name in names:
   logs=subprocess.run(['docker','logs',name],capture_output=True,timeout=15);(out/(name+'.log')).write_bytes(logs.stdout+logs.stderr)
   subprocess.run(['docker','rm','-f',name],capture_output=True,timeout=20)
  subprocess.run(['docker','network','rm',network],capture_output=True,timeout=15)
if __name__=='__main__':main()
