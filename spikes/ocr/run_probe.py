"""Run the immutable OCR experiment against synthetic fixtures and a separate cache."""
import argparse
import json
from pathlib import Path
import subprocess
import uuid

parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument("--image",default="foliopath-inference:ocr-feasibility-20261025")
parser.add_argument("--cache",type=Path,default=Path("/Volumes/extended/foliopath/ocr-feasibility-cache"))
parser.add_argument("--fixtures",type=Path,required=True)
parser.add_argument("--output",type=Path,required=True)
args=parser.parse_args()
for path in (args.cache,args.fixtures):
    if not path.is_absolute() or path.is_symlink() or not path.is_dir():raise ValueError("Use explicit nonsymlink experiment directories")
image=subprocess.check_output(["docker","image","inspect",args.image,"--format","{{.Id}}"],text=True,timeout=10).strip()
name="fp-ocr-"+uuid.uuid4().hex[:10]
args.output.mkdir(parents=True,exist_ok=True)
try:
    result=subprocess.run(["docker","run","--name",name,"--network","none","--read-only","--memory","4g","--cpus","2",
        "--cap-drop","ALL","--security-opt","no-new-privileges","--tmpfs","/tmp:size=256m","--tmpfs","/work:size=400m",
        "--mount",f"type=bind,src={args.cache},dst=/ocr-cache,readonly",
        "--mount",f"type=bind,src={args.fixtures},dst=/fixtures,readonly",image],text=True,capture_output=True,timeout=360)
    inspected=json.loads(subprocess.check_output(["docker","inspect",name],text=True,timeout=10))[0]
    state=inspected["State"]
    assert inspected["HostConfig"]["NetworkMode"] == "none"
    assert not inspected["HostConfig"]["PortBindings"]
    assert {mount["Destination"] for mount in inspected["Mounts"]} == {"/fixtures", "/ocr-cache"}
    assert all(not mount["RW"] for mount in inspected["Mounts"])
    (args.output/"container.json").write_text(json.dumps({"image":image,"state":state,"memoryLimitBytes":4*1024**3,"cpus":2,"isolated":True},indent=2)+"\n")
    (args.output/"stderr.log").write_text(result.stderr)
    (args.output/"stdout.log").write_text(result.stdout)
    if result.returncode or state["OOMKilled"]:raise RuntimeError(f"Experiment failed ({result.returncode}); inspect the experiment logs")
    value=json.loads(result.stdout)
    assert value["status"]=="passed"
    value["image"]=image
    value["containerState"]={"exitCode":state["ExitCode"],"oomKilled":state["OOMKilled"]}
    (args.output/"result.json").write_text(json.dumps(value,ensure_ascii=False,indent=2)+"\n")
    print(json.dumps({"status":"passed","image":image,"caseCount":value["caseCount"],"memoryPeakBytes":value["memoryPeakBytes"]}))
finally:
    subprocess.run(["docker","rm","-f",name],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL,timeout=20)
