"""Real Docker cleanup regression: privileged model files, unprivileged test runner."""
import json
import os
import subprocess
import unittest

from managed_ai_real import docker, validation_container


class ValidationCacheTests(unittest.TestCase):
    def test_root_owned_model_cache_is_removed_on_success_and_failure(self):
        image = os.environ['FOLIOPATH_TEST_AI_IMAGE']
        for fail in (False, True):
            with self.subTest(failure=fail):
                container = None
                volumes = []
                try:
                    with validation_container(image) as container:
                        # Stop bootstrap so this test needs neither models nor downloads.
                        docker('stop', '--timeout', '30', container)
                        mounts = json.loads(docker('inspect', container))[0]['Mounts']
                        self.assertEqual({m['Destination'] for m in mounts}, {'/cache', '/ocr-cache', '/verified-cache'})
                        self.assertTrue(all(m['Type'] == 'volume' for m in mounts))
                        volumes = [m['Name'] for m in mounts]
                        # Use the same UID as production to create a non-writable subtree.
                        docker('run', '--rm', '--network', 'none', '--user', '0',
                               '-v', volumes[0] + ':/cache', '--entrypoint', 'python', image, '-c',
                               "from pathlib import Path; p=Path('/cache/root-owned'); p.mkdir(mode=0o700); (p/'model.onnx').write_bytes(b'synthetic'); p.chmod(0o500)")
                        if fail:
                            raise RuntimeError('synthetic inference failure')
                except RuntimeError as error:
                    self.assertTrue(fail)
                    self.assertEqual(str(error), 'synthetic inference failure')
                for resource, name in [('container', container), *(('volume', v) for v in volumes)]:
                    result = subprocess.run(['docker', resource, 'inspect', name], capture_output=True)
                    self.assertNotEqual(result.returncode, 0, name)


if __name__ == '__main__':
    unittest.main()
