#!/usr/bin/env python3
"""Verify a built managed AI image without media, model mounts, or registry writes."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--image', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    repo = Path(__file__).resolve().parents[2]
    expected = json.loads((repo / 'internal/app/managed_ai.generated.json').read_text())
    command = '''
import json
import os
from pathlib import Path
import server, multilingual_contract, photo_policy, ocr_contract
from model_install import read_catalog
p=Path('/opt/foliopath-inference')
d=json.loads((p/'managed-deployment.generated.json').read_text())
assert d['identity']==multilingual_contract.expected_identity()
assert d['tag']['identity']==photo_policy.expected_identity()
assert d['ocr']['identity']==ocr_contract.expected_identity()
entries=json.loads((p/'cache-manifest.json').read_text())
face=dict(server.expected_identity(entries),protocol='foliopath-face-v1',model='buffalo_l')
assert d['face']['identity']==face
assert len(read_catalog())==186
print(json.dumps(d,sort_keys=True))
'''
    result = subprocess.run(['docker', 'run', '--rm', '--network', 'none', '--read-only',
                             '--entrypoint', 'python', args.image, '-c', command],
                            check=True, capture_output=True, text=True)
    if json.loads(result.stdout) != expected:
        raise ValueError('AI profile differs from the main-image embedded profile')
    args.output.mkdir(parents=True, exist_ok=True)
    with (args.output / 'installer-tests.log').open('w') as log:
        subprocess.run(['docker', 'run', '--rm', '--network', 'none', '--read-only',
                        '--tmpfs', '/tmp:size=64m', '--env', 'PYTHONDONTWRITEBYTECODE=1',
                        '--env', 'PYTHONPATH=/opt/foliopath-inference:/checks',
                        '--mount', f'type=bind,src={repo / "services/inference/test_model_install.py"},dst=/checks/test_model_install.py,readonly',
                        '--entrypoint', 'python', args.image, '-m', 'unittest', '-v', 'test_model_install'],
                       check=True, stdout=log, stderr=subprocess.STDOUT)
    subprocess.run(['python3', '-m', 'unittest', 'discover', '-s', 'tests/release',
                    '-p', 'test_managed_ai_real.py', '-v'], check=True,
                   env=dict(os.environ, FOLIOPATH_TEST_AI_IMAGE=args.image), cwd=repo)
    (args.output / 'verification.json').write_text(json.dumps(dict(image=args.image,
        profile='standard-v1', profile_sha256=hashlib.sha256(json.dumps(expected, sort_keys=True).encode()).hexdigest(),
        identity_parity=True, installer_tests=True, cleanup_regression=True, real_model_inference='not_run'), indent=2)+'\n')


if __name__ == '__main__': main()
