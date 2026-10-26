"""ADR-0034 fitness: routing cannot become a second admission/lifecycle owner."""
import ast
from pathlib import Path

root=Path(__file__).resolve().parents[1]
source=ast.parse((root/"services/inference/ocr_scheduler.py").read_text())
classes=[node for node in source.body if isinstance(node,ast.ClassDef)]
assert len(classes)==1 and isinstance(classes[0].bases[0],ast.Name) and classes[0].bases[0].id=="InferenceScheduler"
methods={node.name:node for node in classes[0].body if isinstance(node,(ast.FunctionDef,ast.AsyncFunctionDef))}
assert set(methods)=={"__init__","infer","_run"},"OCR route adapter must inherit reaping/close from core"
for node in ast.walk(source):
    if isinstance(node,ast.Call) and isinstance(node.func,ast.Attribute):
        assert node.func.attr not in {"Lock","Semaphore","create_subprocess_exec","create_subprocess_shell","timeout"},"OCR adapter added a second gate/lifecycle"
    if isinstance(node,(ast.Assign,ast.AugAssign,ast.AnnAssign)):
        targets=node.targets if isinstance(node,ast.Assign) else [node.target]
        for target in targets:
            if isinstance(target,ast.Attribute):
                assert target.attr not in {"pending","lock","requests","processes","closed","timeout"},"OCR adapter duplicated core state policy"

def delegates(method,target):
    return sum(isinstance(n,ast.Call) and isinstance(n.func,ast.Attribute) and n.func.attr==target and
        isinstance(n.func.value,ast.Call) and isinstance(n.func.value.func,ast.Name) and n.func.value.func.id=="super"
        for n in ast.walk(method))
assert delegates(methods["infer"],"infer")==1,"all admissions must delegate to core"
assert delegates(methods["_run"],"_run")==2,"both OCR and legacy branches must use core child lifecycle"
recipe=(root/"services/inference/Dockerfile.ocr").read_text()
assert "@sha256:52f4a40c9f0169e87783b138200e24c8e96492fccf244b9d329ec1e05e45f626" in recipe
for line in recipe.splitlines():
    if line.startswith("COPY services/inference/"):
        assert line=="COPY services/inference/ocr_*.py services/inference/ocr-*.json /opt/foliopath-inference/", "OCR overlay may not overwrite legacy sources"
for name in ("ocr_server.py","ocr_worker.py","ocr_engine.py","ocr_contract.py"):
    assert "spikes/" not in (root/"services/inference"/name).read_text(),"experimental implementation entered production graph"
print("OCR scheduler ownership and immutable-base boundary verified")
