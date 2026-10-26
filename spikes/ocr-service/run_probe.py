"""Offline, synthetic-only baseline -> OCR producer/Go consumer integration."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import tempfile
import time
import uuid

ROOT=Path(__file__).resolve().parents[2]
BASE="sha256:52f4a40c9f0169e87783b138200e24c8e96492fccf244b9d329ec1e05e45f626"


def command(args,**kwargs):return subprocess.run(args,check=True,capture_output=True,timeout=kwargs.pop("timeout",30),**kwargs)

def inspect_image(value):return command(["docker","image","inspect",value,"--format","{{.Id}}"],text=True).stdout.strip()


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--image",required=True);parser.add_argument("--probe",required=True)
    parser.add_argument("--output",type=Path,required=True)
    parser.add_argument("--cache",type=Path,default=Path("/Volumes/extended/foliopath/model-experiments/multilingual-integration-cache"))
    parser.add_argument("--ocr-cache",type=Path,default=Path("/Volumes/extended/foliopath/ocr-feasibility-cache"))
    args=parser.parse_args()
    for p in (args.output,args.cache,args.ocr_cache):assert p.is_absolute() and not p.is_symlink()
    args.output.mkdir(parents=True,exist_ok=True);output=args.output
    contracts=output/"contracts";contracts.mkdir(exist_ok=True)
    observation=output/"observations";observation.mkdir(exist_ok=True)
    image,probe=inspect_image(args.image),inspect_image(args.probe)
    prefix="fp-ocr-service-"+uuid.uuid4().hex[:8];network=prefix+"-net"
    service=prefix+"-ml";client=prefix+"-client"
    names=[];states={};metrics={}
    legacy_script='import server,json,photo_policy,multilingual_contract;from pathlib import Path;x=server.expected_identity(json.loads(Path("/opt/foliopath-inference/cache-manifest.json").read_bytes()));print(json.dumps({"semantic":multilingual_contract.expected_identity(),"face":{**x,"protocol":"foliopath-face-v1","model":"buffalo_l"},"tag":photo_policy.expected_identity()}))'
    baseline=command(["docker","run","--rm","--network","none","--read-only","--entrypoint","python",BASE,"-c",legacy_script]).stdout
    (contracts/"baseline-identities.json").write_bytes(baseline)
    expected=command(["docker","run","--rm","--network","none","--read-only","--entrypoint","cat",image,"/opt/foliopath-inference/ocr-identity.json"]).stdout
    (contracts/"ocr-identity.json").write_bytes(expected)
    def frozen(target):
        script='import hashlib,json;from pathlib import Path;print(json.dumps({p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in Path("/opt/foliopath-inference").iterdir() if p.is_file() and p.suffix in (".py",".json") and not p.name.startswith(("ocr_","ocr-"))},sort_keys=True))'
        return json.loads(command(["docker","run","--rm","--network","none","--read-only","--entrypoint","python",target,"-c",script]).stdout)
    original=frozen(BASE);assert frozen(image)==original,"legacy source bytes changed"
    def hash_file(path):
        with path.open("rb") as stream:return hashlib.file_digest(stream,"sha256").hexdigest()
    metadata_script='import json;from pathlib import Path;r=Path("/opt/foliopath-inference");print(json.dumps(sum([json.loads((r/p).read_text()) for p in ("cache-manifest.json","tag-manifest.json","multilingual-manifest.json")],[])))'
    entries=json.loads(command(["docker","run","--rm","--network","none","--read-only","--entrypoint","python",BASE,"-c",metadata_script]).stdout)
    model_pins={e["path"]:e for e in entries}
    for path,entry in model_pins.items():
        source=args.cache/path
        assert source.is_file() and not source.is_symlink() and source.stat().st_size==entry["size"] and hash_file(source)==entry["sha256"],"incomplete test cache: "+path
    fixture_pins=json.loads((ROOT/"docs/evidence/ocr/2026-10-25-ocr-feasibility-local.json").read_text())["fixtureHashes"]
    for name in ("bilingual.png","exif-oriented.jpg","wide.png","blank.png"):
        relative="tests/fixtures/ocr/"+name
        assert hash_file(ROOT/relative)==fixture_pins[relative],"synthetic fixture changed"
    command(["docker","network","create","--internal",network])
    try:
        for mode,target in (("baseline",BASE),("ocr",image),("missing",image)):
            scratch=Path(tempfile.mkdtemp(prefix=mode+"-scratch-",dir=output))
            run=["docker","run","-d","--name",service,"--network",network,"--network-alias","machine-learning","--read-only",
                 "--memory","4g","--memory-swap","4g","--cpus","2","--pids-limit","128","--cap-drop","ALL","--security-opt","no-new-privileges",
                 "--tmpfs","/tmp:size=256m","--mount",f"type=bind,src={args.cache},dst=/cache,readonly",
                 "--mount",f"type=bind,src={scratch},dst=/verified-cache"]
            if mode=="ocr":run += ["--mount",f"type=bind,src={args.ocr_cache},dst=/ocr-cache,readonly"]
            command([*run,target]);names.append(service)
            ready=False
            for _ in range(90):
                ping=subprocess.run(["docker","exec",service,"python","-c","import urllib.request;assert urllib.request.urlopen('http://127.0.0.1:3003/ping',timeout=1).read()==b'pong'"],capture_output=True,timeout=5)
                if ping.returncode==0:ready=True;break
                state=json.loads(command(["docker","inspect",service,"--format","{{json .State}}"],text=True).stdout)
                if not state["Running"]:break
                time.sleep(1)
            assert ready,"isolated service did not become ready"
            response=subprocess.run(["docker","run","--name",client,"--network",network,"--read-only","--memory","1g","--cpus","2","--cap-drop","ALL",
                "--security-opt","no-new-privileges","--tmpfs","/tmp:size=128m","--mount",f"type=bind,src={ROOT/'tests/fixtures/ocr'},dst=/fixtures,readonly",
                "--mount",f"type=bind,src={contracts},dst=/contracts,readonly","--mount",f"type=bind,src={observation},dst=/observations",
                "-e","FOLIOPATH_OCR_TEST_ORIGIN=http://machine-learning:3003","-e",f"FOLIOPATH_OCR_TEST_MODE={mode}",probe],capture_output=True,timeout=300)
            names.append(client);(output/(mode+"-probe.log")).write_bytes(response.stdout+response.stderr)
            state=json.loads(command(["docker","inspect",client,"--format","{{json .State}}"],text=True).stdout)
            states[mode+"Client"]=state;assert response.returncode==0 and not state["OOMKilled"],"actual service probe failed"
            metrics[mode]=command(["docker","exec",service,"python","-c","import json;from pathlib import Path;print(json.dumps({'peak':int(Path('/sys/fs/cgroup/memory.peak').read_text()),'events':dict(x.split() for x in Path('/sys/fs/cgroup/memory.events').read_text().splitlines())}))"],text=True).stdout
            metric=json.loads(metrics[mode]);assert int(metric["events"]["oom"])==0 and int(metric["events"]["oom_kill"])==0
            service_logs=command(["docker","logs",service]).stdout
            (output/(mode+"-service.log")).write_bytes(service_logs)
            command(["docker","stop","--time","15",service]);states[mode+"Service"]=json.loads(command(["docker","inspect",service,"--format","{{json .State}}"],text=True).stdout)
            assert not list(scratch.iterdir()),"private snapshot left behind"
            command(["docker","rm",client,service]);names.remove(client);names.remove(service);scratch.rmdir()
        for path,entry in model_pins.items():assert hash_file(args.cache/path)==entry["sha256"],"model cache modified"
        for name in ("bilingual.png","exif-oriented.jpg","wide.png","blank.png"):
            relative="tests/fixtures/ocr/"+name
            assert hash_file(ROOT/relative)==fixture_pins[relative],"synthetic fixture modified"
        result={"status":"passed","modelHashesPreserved":True,"syntheticSourcesPreserved":True,"image":image,"probeImage":probe,"baselineImage":BASE,"ocrExpectedIdentity":json.loads(expected),
                "baselineIdentities":json.loads(baseline),"legacySourceHashes":original,"legacySourcesPreserved":True,
                "states":states,"metrics":{k:json.loads(v) for k,v in metrics.items()},"baselineOutputSHA256":hashlib.sha256((observation/"baseline.json").read_bytes()).hexdigest(),
                "scope":"isolated actual producer/Go consumer and unchanged legacy output; no media/DB/main application composition",
                "privateSnapshotsRemoved":True,"containersRemoved":True,"networkInternal":True}
        (output/"result.json").write_text(json.dumps(result,ensure_ascii=False,indent=2)+"\n")
        print(json.dumps({"status":"passed","image":image,"legacyOutputSHA256":result["baselineOutputSHA256"]}))
    finally:
        for name in names:
            logs=subprocess.run(["docker","logs",name],capture_output=True,timeout=10)
            (output/(name+".failure.log")).write_bytes(logs.stdout+logs.stderr)
            subprocess.run(["docker","rm","-f",name],capture_output=True,timeout=20)
        subprocess.run(["docker","network","rm",network],capture_output=True,timeout=10)


if __name__=="__main__":main()
