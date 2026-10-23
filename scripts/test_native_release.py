import copy
import json
import io
from pathlib import Path
import subprocess
import tempfile
import tarfile
from types import SimpleNamespace
import unittest
from unittest.mock import Mock

import native_release as release
import buildkit_cache_mounts as cache


def plan():
    return dict(image=release.IMAGE, source="a" * 40, version="v1.4.0",
                labels={"org.opencontainers.image.revision": "a" * 40,
                        "org.opencontainers.image.version": "v1.4.0"},
                tags=[release.IMAGE + ":1.4.0", release.IMAGE + ":latest"])


class NativeReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.plan = plan()
        for arch, token in [("amd64", "b"), ("arm64", "c")]:
            release.write_json(self.root / (arch + ".json"), dict(
                image=release.IMAGE, source=self.plan["source"], version=self.plan["version"],
                arch=arch, digest="sha256:" + token * 64))

    def test_rejects_missing_or_cross_commit_records_before_registry_access(self):
        for field, bad in [("source", "d" * 40), ("version", "v1.3.0"),
                           ("image", "another/repository"), ("arch", "amd64"), ("digest", "latest")]:
            with self.subTest(field=field):
                path = self.root / "arm64.json"
                original = json.loads(path.read_text())
                changed = dict(original, **{field: bad})
                release.write_json(path, changed)
                run = Mock()
                with self.assertRaises(ValueError):
                    release.publish(self.plan, self.root, run)
                run.assert_not_called()
                release.write_json(path, original)
        (self.root / "arm64.json").unlink()
        with self.assertRaises(ValueError):
            release.publish(self.plan, self.root, Mock())

    def test_rejects_duplicate_digests_and_unplanned_tags(self):
        path = self.root / "arm64.json"
        record = json.loads(path.read_text())
        record["digest"] = "sha256:" + "b" * 64
        release.write_json(path, record)
        with self.assertRaises(ValueError):
            release.publish(self.plan, self.root, Mock())
        for tags in [[], ["other/repo:latest"], [release.IMAGE + ":bad;tag"]]:
            with self.subTest(tags=tags), self.assertRaises(ValueError):
                release.validate_plan(dict(self.plan, tags=tags))

    def test_failed_runtime_or_wrong_identity_never_produces_success_record(self):
        output = self.root / "verified.json"
        identity = dict(Architecture="arm64", Os="linux", Config=dict(Labels=self.plan["labels"]))
        run = Mock(side_effect=[SimpleNamespace(), SimpleNamespace(stdout=json.dumps([identity])),
                                subprocess.CalledProcessError(1, "smoke")])
        env = dict(SOURCE=self.plan["source"], VERSION=self.plan["version"])
        with self.assertRaises(subprocess.CalledProcessError):
            release.verify(self.plan, "arm64", "sha256:" + "c" * 64, output, run, env)
        self.assertFalse(output.exists())
        self.assertEqual(run.call_args.kwargs["env"]["FOLIOPATH_RELEASE_BUILD_IMAGE"], "0")
        run = Mock(side_effect=[SimpleNamespace(), SimpleNamespace(stdout=json.dumps([
            dict(identity, Architecture="amd64")]))])
        with self.assertRaises(ValueError):
            release.verify(self.plan, "arm64", "sha256:" + "c" * 64, output, run, env)
        self.assertFalse(output.exists())
        self.assertEqual(run.call_count, 2)

    def test_only_successfully_verified_image_is_recorded(self):
        output = self.root / "verified.json"
        run = Mock(side_effect=[SimpleNamespace(), SimpleNamespace(stdout=json.dumps([
            dict(Architecture="amd64", Os="linux", Config=dict(Labels=self.plan["labels"]))])), SimpleNamespace()])
        release.verify(self.plan, "amd64", "sha256:" + "b" * 64, output, run,
                       dict(SOURCE=self.plan["source"], VERSION=self.plan["version"]))
        self.assertEqual(json.loads(output.read_text())["source"], self.plan["source"])
        self.assertEqual(run.call_args.args[0], ["sh", "tests/release/image_smoke.sh"])

    @staticmethod
    def index(arch, token):
        return {"manifests": [
            {"digest": "sha256:" + token * 64, "platform": {"os": "linux", "architecture": arch}},
            {"digest": "sha256:" + ("d" if arch == "amd64" else "e") * 64,
             "platform": {"os": "unknown", "architecture": "unknown"},
             "annotations": {"vnd.docker.reference.type": "attestation-manifest",
                             "vnd.docker.reference.digest": "sha256:" + token * 64}},
        ]}

    def test_promotes_exact_digests_without_build_and_preserves_attestations(self):
        amd64, arm64 = self.index("amd64", "1"), self.index("arm64", "2")
        final = {"manifests": amd64["manifests"] + arm64["manifests"]}
        attestation = {"layers": [{"annotations": {"in-toto.io/predicate-type": value}} for value in
                                  ["https://spdx.dev/Document", "https://slsa.dev/provenance/v0.2"]]}
        run = Mock(side_effect=[SimpleNamespace(stdout=json.dumps(amd64)),
                                SimpleNamespace(stdout=json.dumps(attestation)),
                                SimpleNamespace(stdout=json.dumps(arm64)),
                                SimpleNamespace(stdout=json.dumps(attestation)), SimpleNamespace(),
                                SimpleNamespace(stdout=json.dumps(final))])
        release.publish(self.plan, self.root, run)
        command = run.call_args_list[4].args[0]
        self.assertEqual(command[:4], ["docker", "buildx", "imagetools", "create"])
        self.assertEqual(command[-2:], [release.IMAGE + "@sha256:" + c * 64 for c in ["b", "c"]])
        self.assertIn(release.IMAGE + ":1.4.0", command)
        self.assertNotIn("build", command)

    def test_missing_attestations_or_wrong_platform_fail_before_promotion(self):
        for index in [self.index("arm64", "1"), {"manifests": self.index("amd64", "1")["manifests"][:1]}]:
            run = Mock(return_value=SimpleNamespace(stdout=json.dumps(index)))
            with self.assertRaises(ValueError):
                release.publish(self.plan, self.root, run)
            self.assertEqual(run.call_count, 1)

    def test_missing_sbom_or_provenance_fails_before_tag_write(self):
        for predicate in ["https://spdx.dev/Document", "https://slsa.dev/provenance/v1"]:
            run = Mock(side_effect=[SimpleNamespace(stdout=json.dumps(self.index("amd64", "1"))),
                                    SimpleNamespace(stdout=json.dumps({"layers": [
                                        {"annotations": {"in-toto.io/predicate-type": predicate}}]}))])
            with self.subTest(predicate=predicate), self.assertRaises(ValueError):
                release.publish(self.plan, self.root, run)
            self.assertEqual(run.call_count, 2)

    def test_rejects_label_identity_changes_and_output_injection(self):
        for key, value in [("org.opencontainers.image.revision", "f" * 40),
                           ("org.opencontainers.image.version", "old"), ("description", "bad\nlabels=oops")]:
            candidate = copy.deepcopy(self.plan)
            candidate["labels"][key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                release.validate_plan(candidate)


class BuildCacheTests(unittest.TestCase):
    def test_cache_helpers_use_the_canonical_pinned_base_and_only_approved_mounts(self):
        dockerfile = Path(__file__).resolve().parent.parent / "Dockerfile"
        base = cache.cache_base(dockerfile.read_text())
        self.assertIn("@sha256:", base)
        for target in cache.MOUNTS.values():
            self.assertIn("FROM " + base, cache.dancefile(base, target, "inject"))
            self.assertIn("COPY --from=cache /export/ /", cache.dancefile(base, target, "extract"))
        with self.assertRaises(ValueError):
            cache.cache_base("FROM debian:latest AS expat")
        with self.assertRaises(ValueError):
            cache.dancefile(base, "/library", "extract")

    def test_inject_and_extract_restore_actual_mounts_and_export_to_separate_directories(self):
        with tempfile.TemporaryDirectory() as directory:
            dockerfile = Path(__file__).resolve().parent.parent / "Dockerfile"
            for mode in ["inject", "extract"]:
                def simulate_export(command, **_kwargs):
                    if "--output" in command:
                        output = command[command.index("--output") + 1].split("dest=", 1)[1]
                        with tarfile.open(output, "w"):
                            pass
                run = Mock(side_effect=simulate_export)
                cache.transfer(mode, directory, dockerfile, run)
                self.assertEqual(run.call_count, 3)
                for call in run.call_args_list:
                    self.assertEqual("--output" in call.args[0], mode == "extract")
                    self.assertIn("--mount=type=cache", call.kwargs["input"])
                    self.assertTrue(call.kwargs["check"])
                    self.assertEqual(call.kwargs["timeout"], 60)

    def test_unsafe_archive_cannot_escape_or_replace_the_previous_snapshot(self):
        with tempfile.TemporaryDirectory() as directory:
            snapshot = Path(directory) / "go-mod"
            snapshot.mkdir()
            (snapshot / "previous").write_text("usable")

            dockerfile = Path(__file__).resolve().parent.parent / "Dockerfile"
            for name, kind in [("../../escaped", tarfile.REGTYPE), ("/absolute", tarfile.REGTYPE),
                               ("link", tarfile.SYMTYPE), ("hard", tarfile.LNKTYPE),
                               ("fifo", tarfile.FIFOTYPE)]:
                def unsafe_export(command, **_kwargs):
                    output = command[command.index("--output") + 1].split("dest=", 1)[1]
                    with tarfile.open(output, "w") as archive:
                        member = tarfile.TarInfo(name)
                        member.type = kind
                        member.linkname = "../../escaped"
                        member.size = 3 if member.isfile() else 0
                        archive.addfile(member, io.BytesIO(b"bad"))
                with self.subTest(name=name), self.assertRaises(ValueError):
                    cache.transfer("extract", directory, dockerfile, unsafe_export)
                self.assertEqual((snapshot / "previous").read_text(), "usable")

    def test_timeout_preserves_the_previous_cache_snapshot(self):
        with tempfile.TemporaryDirectory() as directory:
            snapshot = Path(directory) / "go-mod"
            snapshot.mkdir()
            (snapshot / "previous").write_text("usable")
            run = Mock(side_effect=subprocess.TimeoutExpired("docker", 60))
            dockerfile = Path(__file__).resolve().parent.parent / "Dockerfile"
            with self.assertRaises(subprocess.TimeoutExpired):
                cache.transfer("extract", directory, dockerfile, run)
            self.assertEqual((snapshot / "previous").read_text(), "usable")


if __name__ == "__main__":
    unittest.main()
