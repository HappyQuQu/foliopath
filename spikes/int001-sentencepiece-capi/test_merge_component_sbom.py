import json
import pathlib
import tempfile
import unittest

from merge_component_sbom import merge


DIGEST_A = "sha256:" + "a" * 64
DIGEST_B = "sha256:" + "b" * 64
DIGEST_C = "sha256:" + "c" * 64


class MergeComponentSBOMTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.root = pathlib.Path(self.temp.name)
        self.base = self.root / "base.json"
        self.component = self.root / "component.json"
        self.base.write_text(
            json.dumps(
                {
                    "bomFormat": "CycloneDX",
                    "specVersion": "1.5",
                    "components": [{"type": "library", "bom-ref": "base", "name": "base"}],
                    "metadata": {
                        "component": {
                            "type": "container",
                            "bom-ref": "root-" + "c" * 64,
                            "purl": "pkg:oci/test@" + DIGEST_C,
                        }
                    },
                    "dependencies": [{"ref": "root-" + "c" * 64, "dependsOn": ["base"]}],
                }
            ),
            encoding="utf-8",
        )
        self.write_component("native")

    def tearDown(self):
        self.temp.cleanup()

    def write_component(self, ref):
        self.component.write_text(
            json.dumps(
                {
                    "bomFormat": "CycloneDX",
                    "specVersion": "1.6",
                    "components": [
                        {
                            "type": "library",
                            "bom-ref": ref,
                            "name": ref,
                            "purl": f"pkg:generic/{ref}@1?arch=arm64&os=linux",
                        }
                    ],
                }
            ),
            encoding="utf-8",
        )

    def run_merge(self):
        return merge(self.base, [self.component], "arm64", DIGEST_A, DIGEST_B, DIGEST_C)

    def test_merges_explicit_component_and_binds_root(self):
        result = self.run_merge()
        self.assertEqual([item["bom-ref"] for item in result["components"]], ["base", "native"])
        root = next(item for item in result["dependencies"] if item["ref"].startswith("root-"))
        self.assertEqual(root["dependsOn"], ["base", "native"])
        self.assertEqual(result["specVersion"], "1.6")
        self.assertEqual(result["compositions"][0]["aggregate"], "incomplete")

    def test_rejects_duplicate_component_ref(self):
        self.write_component("base")
        with self.assertRaisesRegex(ValueError, "duplicate component bom-ref base"):
            self.run_merge()

    def test_rejects_wrong_architecture(self):
        raw = json.loads(self.component.read_text(encoding="utf-8"))
        raw["components"][0]["purl"] = "pkg:generic/native@1?arch=amd64&os=linux"
        self.component.write_text(json.dumps(raw), encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "does not target arch=arm64"):
            self.run_merge()

    def test_normalizes_nested_scanner_component_order(self):
        raw = json.loads(self.base.read_text(encoding="utf-8"))
        raw["components"][0]["components"] = [
            {"type": "file", "bom-ref": "z", "name": "z"},
            {"type": "file", "bom-ref": "a", "name": "a"},
        ]
        self.base.write_text(json.dumps(raw), encoding="utf-8")
        first = self.run_merge()
        raw["components"][0]["components"].reverse()
        self.base.write_text(json.dumps(raw), encoding="utf-8")
        second = self.run_merge()
        self.assertEqual(first, second)
        self.assertEqual(
            [item["bom-ref"] for item in first["components"]],
            ["a", "base", "native", "z"],
        )

    def test_consolidates_dependencies_and_rejects_unknown_refs(self):
        raw = json.loads(self.base.read_text(encoding="utf-8"))
        root_ref = raw["metadata"]["component"]["bom-ref"]
        raw["dependencies"].append({"ref": root_ref, "dependsOn": ["base"]})
        self.base.write_text(json.dumps(raw), encoding="utf-8")
        result = self.run_merge()
        root_rows = [item for item in result["dependencies"] if item["ref"] == root_ref]
        self.assertEqual(len(root_rows), 1)

        raw["dependencies"].append({"ref": root_ref, "dependsOn": ["missing"]})
        self.base.write_text(json.dumps(raw), encoding="utf-8")
        with self.assertRaisesRegex(ValueError, "unknown ref missing"):
            self.run_merge()


if __name__ == "__main__":
    unittest.main()
