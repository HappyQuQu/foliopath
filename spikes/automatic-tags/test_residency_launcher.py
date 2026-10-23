import contextlib
import importlib.util
import io
import json
import subprocess
import unittest
from pathlib import Path
from unittest.mock import patch


spec = importlib.util.spec_from_file_location("residency_launcher", Path(__file__).with_name("run-co-residency.py"))
launcher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(launcher)


def completed(code=0, output=""):
    return subprocess.CompletedProcess([], code, stdout=output, stderr="")


class ResidencyLauncherTests(unittest.TestCase):
    def run_mocked(self, responses):
        with patch.object(launcher.subprocess, "run", side_effect=responses) as run, \
                patch.object(Path, "is_dir", return_value=True), \
                patch.object(Path, "is_symlink", return_value=False), \
                contextlib.redirect_stdout(io.StringIO()) as output:
            result = launcher.main()
        return result, run.call_args_list, output.getvalue()

    def test_oom_is_reported_as_failure_and_container_is_removed(self):
        result, calls, output = self.run_mocked([
            completed(), completed(137),
            completed(output=json.dumps({"OOMKilled": True, "ExitCode": 137})), completed(),
        ])
        self.assertEqual(result, 137)
        create = calls[0].args[0]
        self.assertIn("2304m", create)
        self.assertIn("/verified-cache:rw,noexec,nosuid,size=1200m,mode=0700", create)
        self.assertEqual(create[create.index("--network") + 1], "none")
        self.assertEqual(calls[-1].args[0][:3], ["docker", "rm", "--force"])
        self.assertIn('"OOMKilled": true', output)

    def test_timeout_stops_then_removes_only_own_container(self):
        result, calls, _output = self.run_mocked([
            completed(), subprocess.TimeoutExpired("docker start", 150), completed(),
            completed(output=json.dumps({"OOMKilled": False, "ExitCode": 143})), completed(),
        ])
        self.assertEqual(result, 124)
        self.assertEqual(calls[1].kwargs["timeout"], 150)
        self.assertEqual(calls[2].args[0][:4], ["docker", "stop", "--time", "3"])
        self.assertEqual(calls[2].args[0][-1], calls[-1].args[0][-1])
        self.assertTrue(calls[-1].args[0][-1].startswith("fp-tag-residency-spike-"))

    def test_create_timeout_still_attempts_cleanup(self):
        with patch.object(launcher.subprocess, "run", side_effect=[
            subprocess.TimeoutExpired("docker create", 10), completed(),
        ]) as run, patch.object(Path, "is_dir", return_value=True), \
                patch.object(Path, "is_symlink", return_value=False), \
                contextlib.redirect_stdout(io.StringIO()):
            with self.assertRaises(subprocess.TimeoutExpired):
                launcher.main()
        self.assertEqual(run.call_args_list[-1].args[0][:3], ["docker", "rm", "--force"])


if __name__ == "__main__":
    unittest.main()
