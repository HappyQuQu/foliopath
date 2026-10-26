#!/usr/bin/env python3
"""Verify an existing immutable AI image without rebuilding or model downloads."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import uuid

from inference_materialization_smoke import manifest

CONTRACT_MODULES = ('test_face', 'test_runtime_manifest', 'test_scheduler', 'test_service',
                    'test_tag', 'test_tcp_disconnect', 'ocr_tests')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--image', required=True)
    parser.add_argument('--repo', type=Path, default=Path(__file__).resolve().parents[2])
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    result = verify(args.repo.resolve(), args.image, args.output.resolve())
    print(json.dumps(result, sort_keys=True))


def verify(repo, image, output):
    output.mkdir(parents=True)
    expected = output / 'canonical-runtime'
    expected.mkdir()
    recipe = repo / 'services/inference/Dockerfile.release'
    commands = []
    for line in recipe.read_text().splitlines():
        if line.startswith('COPY '):
            source, destination = json.loads(line[5:])
            shutil.copyfile(repo / source, expected / Path(destination).name)
        elif line.startswith('RUN '):
            commands.append(line[4:])
    prefix = 'fp-ci-ai-' + uuid.uuid4().hex[:12]
    materializer = prefix + '-canonical'
    extractor = prefix + '-extract'
    contract = prefix + '-contracts'
    try:
        subprocess.run(['docker', 'run', '--rm', '--name', materializer, '--network', 'none',
                        '--read-only', '--memory', '256m', '--memory-swap', '256m', '--cpus', '1',
                        '--env', 'PYTHONDONTWRITEBYTECODE=1', '--entrypoint', 'sh',
                        '--mount', f'type=bind,src={expected},dst=/opt/foliopath-inference',
                        image, '-ec', '\n'.join(commands)], check=True)
        subprocess.run(['docker', 'create', '--name', extractor, '--network', 'none', '--read-only',
                        '--entrypoint', '/bin/true', image], check=True, stdout=subprocess.DEVNULL)
        actual = output / 'image-runtime'
        actual.mkdir()
        subprocess.run(['docker', 'cp', extractor + ':/opt/foliopath-inference/.', str(actual)], check=True)
        expected_files, actual_files = manifest(expected), manifest(actual)
        if len(expected_files) != 41 or expected_files != actual_files:
            raise ValueError('Exact-image runtime differs from canonical recipe materialization')
        command = ['docker', 'run', '--rm', '--name', contract, '--network', 'none', '--read-only',
                   '--memory', '1g', '--memory-swap', '1g', '--cpus', '2', '--pids-limit', '128',
                   '--tmpfs', '/tmp:size=256m', '--env', 'PYTHONDONTWRITEBYTECODE=1',
                   '--entrypoint', 'python', image, '-m', 'unittest', *CONTRACT_MODULES]
        with (output / 'contracts.log').open('w') as log:
            subprocess.run(command, check=True, stdout=log, stderr=subprocess.STDOUT)
        log = (output / 'contracts.log').read_text()
        if 'Ran 44 tests' not in log or '\nOK\n' not in log:
            raise ValueError('Expected all44 contract tests without skips')
        result = {'image': image, 'canonicalFiles': actual_files, 'runtimeFiles': 41,
                  'canonicalRecipeSHA256': hashlib.sha256(recipe.read_bytes()).hexdigest(),
                  'allRuntimeBytesMatch': True, 'contractTestsPassed': 44,
                  'ocrIdentity': json.loads((actual / 'ocr-identity.json').read_text()),
                  'modelInference': 'not_run_cache_not_provisioned',
                  'modelCacheInterface': 'independent runner-owned fixed cache required for separate real-inference verification;never upload privatecache',
                  'modelsOrMediaMounted': False, 'imageRebuilt': False}
        (output / 'verification.json').write_text(json.dumps(result, indent=2, sort_keys=True) + '\n')
        return {k: result[k] for k in ('image', 'runtimeFiles', 'allRuntimeBytesMatch', 'contractTestsPassed', 'modelInference')}
    finally:
        for name in (materializer, extractor, contract):
            subprocess.run(['docker', 'rm', '--force', name], capture_output=True)


if __name__ == '__main__':
    main()
