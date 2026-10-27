#!/usr/bin/env python3
"""Empty-cache preparation, interrupted resume, offline restart and real synthetic inference."""
import argparse
from contextlib import contextmanager, nullcontext
import json
from pathlib import Path
import subprocess
import time
import uuid


def docker(*args, **kwargs):
    return subprocess.check_output(['docker', *args], text=True, **kwargs)


def status(container):
    code = "import json,urllib.request; print(urllib.request.urlopen('http://127.0.0.1:3003/api/v1/preparation',timeout=3).read().decode())"
    return json.loads(docker('exec', container, 'python', '-c', code, stderr=subprocess.DEVNULL))


def wait_ready(container, deadline):
    while time.monotonic() < deadline:
        try:
            value = status(container)
            if value['state'] == 'ready': return value
            print(json.dumps(value), flush=True)
        except (subprocess.CalledProcessError, json.JSONDecodeError): pass
        time.sleep(10)
    raise TimeoutError('model preparation did not become ready')


def infer(container, repo):
    fixtures = repo / 'tests/fixtures'
    for source, name in [('ai-faces/synthetic-pair.png', 'faces.png'), ('ocr/bilingual.png', 'text.png'), ('ocr/blank.png', 'blank.png')]:
        docker('cp', str(fixtures/source), container+':/tmp/'+name)
    code = '''
import io,json,math,urllib.request
import ocr_contract
from PIL import Image
from pathlib import Path

def get(path):
    return json.load(urllib.request.urlopen('http://127.0.0.1:3003'+path,timeout=30))
def post(path,fields):
    boundary='foliopath-native-probe'
    body=bytearray()
    for name,value,filename in fields:
        body.extend(('--'+boundary+'\\r\\nContent-Disposition: form-data; name="'+name+'"'+('; filename="'+filename+'"' if filename else '')+'\\r\\n\\r\\n').encode())
        body.extend(value);body.extend(b'\\r\\n')
    body.extend(('--'+boundary+'--\\r\\n').encode())
    req=urllib.request.Request('http://127.0.0.1:3003'+path,data=body,headers={'Content-Type':'multipart/form-data; boundary='+boundary})
    return json.load(urllib.request.urlopen(req,timeout=30))
root=Path('/opt/foliopath-inference')
expected=json.loads((root/'managed-deployment.generated.json').read_text())
assert get('/api/v1/identity')==expected['identity']
assert get('/api/v1/face/identity')==expected['face']['identity']
assert get('/api/v1/tag/identity')==expected['tag']['identity']
assert get('/api/v1/ocr/identity')==expected['ocr']['identity']
# Re-encode synthetic fixtures to the canonical RGB PNG transport shape.
def png(name):
    b=io.BytesIO();Image.open('/tmp/'+name).convert('RGB').save(b,format='PNG');return b.getvalue()
image=png('faces.png')
for kind in ['textual','visual']:
    fields=[('entries',json.dumps({'clip':{kind:{'modelName':expected['identity']['model']}}}).encode(),None)]
    fields += [('text','two people'.encode(),None)] if kind=='textual' else [('image',image,'input')]
    value=post('/predict',fields)
    vector=json.loads(value['clip'])
    assert len(vector)==640 and all(math.isfinite(v) for v in vector)
    print(json.dumps({'kind':kind,'dimension':len(vector)}),flush=True)
face=post('/api/v1/face/predict',[('image',image,'input')])
assert len(face['faces'])>=1
assert all(len(f['embedding'])==512 for f in face['faces'])
tags=post('/api/v1/tag/predict',[('image',image,'input')])
assert set(tags)=={'labels'}
ocr=post('/api/v1/ocr/predict',[('image',png('text.png'),'input')])
assert ocr['lines'] and any(line['text'].strip() for line in ocr['lines'])
blank=post('/api/v1/ocr/predict',[('image',png('blank.png'),'input')])
assert blank==ocr_contract.normalize([])
print(json.dumps({'faces':len(face['faces']),'tags':len(tags['labels']),'ocrLines':len(ocr['lines']),'blankOCR':True}),flush=True)
'''
    try:
        return docker('exec', '-i', container, 'python', '-', input=code)
    except subprocess.CalledProcessError as error:
        print(error.output, flush=True)
        raise


@contextmanager
def validation_container(image):
    container = 'fp-managed-native-' + uuid.uuid4().hex[:12]
    volumes = []
    try:
        for suffix in ('cache', 'ocr', 'scratch'):
            volume = container + '-' + suffix
            docker('volume', 'create', volume)
            volumes.append(volume)
        docker('run', '-d', '--name', container, '--memory', '6g', '--memory-swap', '6g', '--cpus', '3',
               '-v', volumes[0] + ':/cache', '-v', volumes[1] + ':/ocr-cache',
               '-v', volumes[2] + ':/verified-cache', image)
        yield container
    finally:
        # Docker owns root-created cache files; the unprivileged runner never unlinks them.
        subprocess.run(['docker', 'rm', '-f', container], check=False, capture_output=True)
        if volumes:
            docker('volume', 'rm', *volumes)


def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--image',required=True)
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--container',help='Reuse an isolated validation container already downloading; never a production instance')
    args=parser.parse_args()
    repo=Path(__file__).resolve().parents[2]
    owned=not args.container
    args.output.mkdir(parents=True,exist_ok=True)
    deadline=time.monotonic()+2700
    owner = validation_container(args.image) if owned else nullcontext(args.container)
    with owner as container:
        if owned:
            while time.monotonic()<deadline:
                try:
                    if status(container)['completedFiles']>0: break
                except (subprocess.CalledProcessError,json.JSONDecodeError): pass
                time.sleep(1)
            docker('stop','--time','30',container)
            inspected=json.loads(docker('inspect',container))[0]
            if inspected['State']['ExitCode'] not in (0,143) or inspected['State']['OOMKilled']:
                raise ValueError('bootstrap did not stop gracefully')
            if 'Application shutdown complete.' not in docker('logs',container,stderr=subprocess.STDOUT):
                raise ValueError('bootstrap lifespan did not finish before exit')
            docker('start',container)
        value=wait_ready(container,deadline)
        # No routes during inference/restart: all bytes must come from verified cache.
        networks=json.loads(docker('inspect',container))[0]['NetworkSettings']['Networks']
        if 'bridge' in networks: docker('network','disconnect','bridge',container)
        docker('restart','--time','30',container)
        warm=wait_ready(container,time.monotonic()+120)
        result=infer(container,repo)
        (args.output/'inference.log').write_text(result)
        (args.output/'verification.json').write_text(json.dumps(dict(image=args.image,
            freshInterruptedDownload=owned,preparation=value,offlineRestart=warm,realInference=True,
            syntheticFixturesOnly=True),indent=2)+'\n')
        print(result,flush=True)


if __name__=='__main__': main()
