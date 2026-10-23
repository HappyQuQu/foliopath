FROM ghcr.io/immich-app/immich-machine-learning:v3.1.0@sha256:5a0839dc5303cd7215bcd2180a26aed3af41675aefb3e75e5157e9f10ad16e6e
COPY services/inference/*.py /opt/foliopath-inference/
# Candidate baseline only: provenance/release approval remains a separate gate.
COPY spikes/int001-remote-ml/cache-baseline.json /opt/foliopath-inference/local-baseline.json
ARG FOLIOPATH_INCLUDE_FACE=0
ARG FOLIOPATH_INCLUDE_TAG=0
RUN python -c 'import json,pathlib,os; p=pathlib.Path("/opt/foliopath-inference"); entries=json.loads((p/"local-baseline.json").read_text()); prefixes=("clip/ViT-B-32__openai/", "facial-recognition/buffalo_l/") if os.environ["FOLIOPATH_INCLUDE_FACE"] == "1" else ("clip/ViT-B-32__openai/",); (p/"cache-manifest.json").write_text(json.dumps([e for e in entries if e["path"].startswith(prefixes)],sort_keys=True,separators=(",",":")))'
RUN PYTHONPATH=/opt/foliopath-inference python -c 'import json,pathlib,os,tag_contract; pathlib.Path("/opt/foliopath-inference/tag-manifest.json").write_text(json.dumps(tag_contract.MANIFEST if os.environ["FOLIOPATH_INCLUDE_TAG"] == "1" else [],sort_keys=True,separators=(",",":")))'
ENV PYTHONPATH=/opt/foliopath-inference
CMD ["python", "-m", "uvicorn", "server:application", "--host", "0.0.0.0", "--port", "3003", "--workers", "1", "--limit-concurrency", "4", "--no-access-log"]
