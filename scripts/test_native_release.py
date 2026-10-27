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


class ManagedPlanTests(unittest.TestCase):
    def test_actual_cli_derives_paired_tags_and_rejects_component_mix(self):
        with tempfile.TemporaryDirectory() as directory:
            main, ai = Path(directory) / "main.json", Path(directory) / "ai.json"
            release.write_json(main, plan())
            subprocess.run(["python3", str(Path(release.__file__)), "managed-plan", "--plan", str(main), "--output", str(ai)], check=True)
            value = release.read_plan(ai)
            self.assertEqual(value["tags"], [release.IMAGE+":1.4.0-ai", release.IMAGE+":latest-ai"])
            self.assertEqual(value["component"], "ai")
            for arch in release.ARCHITECTURES:
                release.write_json(Path(directory) / "records" / (arch+".json"), dict(image=release.IMAGE,
                    source=value["source"], version=value["version"], arch=arch, digest="sha256:"+"c"*64))
            with self.assertRaisesRegex(ValueError, "component"):
                release.publish(value, Path(directory) / "records")
            value["tags"] = [release.IMAGE+":latest"]
            with self.assertRaises(ValueError): release.validate_plan(value)


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

    def test_verification_plans_share_controlled_repository_without_tag_permission(self):
        import os
        script = Path(__file__).with_name("native_release.py")
        for component in ("main", "ai"):
            output = self.root / (component + "-plan.json")
            env = dict(os.environ, SOURCE="a" * 40, VERSION="ci-20261026")
            subprocess.run(["python3", str(script), "verification-plan", "--kind", component,
                            "--output", str(output)], check=True, env=env)
            value = release.read_plan(output)
            self.assertEqual(value["component"], component)
            self.assertEqual(value["image"], release.IMAGE)
            self.assertEqual(value["labels"]["org.foliopath.component"], component)
            self.assertEqual(value["tags"], [])
            forbidden = Mock(side_effect=AssertionError("verification must never access registry for promotion"))
            with self.assertRaisesRegex(ValueError, "cannot publish"):
                release.publish(value, self.root, forbidden)
            forbidden.assert_not_called()
            for mode in ("true", 1):
                with self.assertRaises(ValueError):
                    release.validate_plan(dict(value, verification_only=mode))
            with self.assertRaises(ValueError):
                release.validate_plan(dict(value, tags=[release.IMAGE + ":latest"]))

    def test_same_repository_wrong_component_fails_before_runtime(self):
        value = dict(self.plan, component="ai", verification_only=True, tags=[])
        value["labels"] = dict(value["labels"], **{"org.foliopath.component": "ai"})
        image = dict(Architecture="amd64", Os="linux", Config=dict(Labels=dict(
            value["labels"], **{"org.foliopath.component": "main"})))
        run = Mock(side_effect=[SimpleNamespace(), SimpleNamespace(stdout=json.dumps([image]))])
        with self.assertRaisesRegex(ValueError, "identity"):
            release.verify(value, "amd64", "sha256:" + "b" * 64, self.root / "not-written.json", run,
                           dict(SOURCE=value["source"], VERSION=value["version"]))
        self.assertFalse((self.root / "not-written.json").exists())
        self.assertEqual(run.call_count, 2)

    def test_workflow_verification_path_has_four_native_builds_and_no_publish_dependency(self):
        root = Path(__file__).resolve().parent.parent
        workflow = (root / ".github/workflows/dockerhub.yml").read_text()
        native = (root / ".github/workflows/native-images.yml").read_text()
        verification = workflow.split("  verification-images:", 1)[1]
        self.assertIn("verification_only: true", verification)
        self.assertNotIn("release-please", verification)
        self.assertNotIn("publish", verification.split("name:", 1)[0])
        job = native.split("  verification-image:", 1)[1]
        self.assertIn("kind: [main, ai]", job)
        self.assertIn("arch: [amd64, arm64]", job)
        self.assertEqual(job.count("uses: docker/build-push-action@v6"), 1)
        self.assertNotIn("tags:", job)
        self.assertIn("push-by-digest=true", job)
        self.assertIn("sbom: true", job)
        self.assertIn("provenance: true", job)
        self.assertNotIn("setup-qemu", job)

    def test_verification_records_cannot_be_relabelled_for_promotion(self):
        for arch in release.ARCHITECTURES:
            path = self.root / (arch + ".json")
            value = json.loads(path.read_text())
            value["verification_only"] = True
            release.write_json(path, value)
        run = Mock(side_effect=AssertionError("verification result must never write a tag"))
        with self.assertRaisesRegex(ValueError, "records cannot publish"):
            release.publish(self.plan, self.root, run)
        run.assert_not_called()

    def test_same_build_oci_parser_binds_actual_blob_bytes_and_registry_runtime(self):
        import hashlib
        blobs = {}
        def add(value):
            data = json.dumps(value, sort_keys=True).encode()
            digest = "sha256:" + hashlib.sha256(data).hexdigest()
            blobs[digest] = data
            return dict(digest=digest, size=len(data))
        config = add(dict(architecture="amd64", os="linux", config=dict(Labels=self.plan["labels"])))
        layer = add({"synthetic": "actual fixture bytes"})
        runtime_doc = dict(config=config, layers=[layer])
        runtime = add(runtime_doc)
        runtime["platform"] = dict(architecture="amd64", os="linux")
        payloads = [dict(add({"predicateType": kind}), annotations={"in-toto.io/predicate-type": kind})
                    for kind in ("https://spdx.dev/Document", "https://slsa.dev/provenance/v1")]
        attestation = add(dict(config=add({}), layers=payloads))
        attestation.update(platform=dict(architecture="unknown", os="unknown"), annotations={
            "vnd.docker.reference.type": "attestation-manifest", "vnd.docker.reference.digest": runtime["digest"]})
        index = dict(manifests=[runtime, attestation])
        archive = self.root / "native.oci.tar"
        def package(corrupt=False):
            with tarfile.open(archive, "w") as target:
                entries = {"index.json": json.dumps(index).encode()}
                entries.update({"blobs/sha256/" + key.split(":")[1]: value for key,value in blobs.items()})
                if corrupt:
                    key = "blobs/sha256/" + layer["digest"].split(":")[1]
                    entries[key] = bytes([entries[key][0] ^ 1]) + entries[key][1:]
                for name,data in entries.items():
                    entry = tarfile.TarInfo(name); entry.size = len(data)
                    target.addfile(entry, io.BytesIO(data))
        package()
        run = Mock(side_effect=[SimpleNamespace(stdout=json.dumps(index)), SimpleNamespace(stdout=json.dumps(runtime_doc))])
        observed = release.verify_oci_archive(archive, release.IMAGE + "@sha256:" + "b"*64, "amd64", self.plan, run)
        self.assertTrue(observed["runtime_and_layers_equal"])
        self.assertEqual(observed["config_digest"], config["digest"])
        package(corrupt=True)
        run = Mock(side_effect=[SimpleNamespace(stdout=json.dumps(index)), SimpleNamespace(stdout=json.dumps(runtime_doc))])
        with self.assertRaisesRegex(ValueError, "hash differs"):
            release.verify_oci_archive(archive, release.IMAGE + "@sha256:" + "b"*64, "amd64", self.plan, run)


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
