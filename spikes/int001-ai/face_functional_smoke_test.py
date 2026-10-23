from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys
import tempfile
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import face_functional_smoke as smoke


def arguments(**overrides: object) -> argparse.Namespace:
    values: dict[str, object] = {
        "catalog": Path("catalog.json"),
        "model_root": Path("models"),
        "media_root": Path("media"),
        "dataset_id": "local-functional-v1",
        "authorization_ref": "operator-local-2026-08-31",
        "max_per_group": 15,
        "group_depth": 1,
        "max_dimension": 1600,
        "max_file_bytes": 250 * 1024 * 1024,
        "detector_score_threshold": 0.9,
        "pair_limit": 100_000,
    }
    values.update(overrides)
    return argparse.Namespace(**values)


class FunctionalSmokeContractTest(unittest.TestCase):
    def test_rejects_placeholder_authorization_and_unbounded_inputs(self) -> None:
        for authorization_ref in ("", "AUTHORIZATION-REQUIRED", "replace-me"):
            with self.subTest(authorization_ref=authorization_ref):
                with self.assertRaises(SystemExit):
                    smoke.validate_args(arguments(authorization_ref=authorization_ref))
        for name, value in (
            ("max_per_group", 101),
            ("group_depth", 4),
            ("max_dimension", 4097),
            ("max_file_bytes", 1024 * 1024 * 1024 + 1),
            ("detector_score_threshold", 0.49),
            ("pair_limit", 99),
        ):
            with self.subTest(name=name):
                with self.assertRaises(SystemExit):
                    smoke.validate_args(arguments(**{name: value}))

    def test_selection_is_balanced_and_skips_symlinks_and_oversized_files(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "media"
            root.mkdir()
            for group_name in ("a", "b"):
                group = root / group_name
                group.mkdir()
                for index in range(3):
                    (group / f"{index}.jpg").write_bytes(b"image")
            (root / "a" / "oversized.png").write_bytes(b"123456")
            os.symlink(root / "a" / "0.jpg", root / "a" / "linked.jpg")
            os.symlink(root / "a", root / "linked-group")

            selected, groups, skipped_oversized = smoke.select_images(
                arguments(media_root=root, max_per_group=2, max_file_bytes=5)
            )

            self.assertEqual(groups, 2)
            self.assertEqual(len(selected), 4)
            self.assertEqual(skipped_oversized, 1)
            self.assertTrue(all(not path.is_symlink() for path in selected))

    def test_selection_supports_bounded_nested_groups(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary) / "media"
            for top in ("a", "b"):
                for nested in ("one", "two"):
                    group = root / top / nested
                    group.mkdir(parents=True)
                    (group / "image.jpg").write_bytes(b"image")

            selected, groups, skipped_oversized = smoke.select_images(
                arguments(media_root=root, group_depth=2)
            )

            self.assertEqual(groups, 4)
            self.assertEqual(len(selected), 4)
            self.assertEqual(skipped_oversized, 0)

    def test_rejects_symlink_media_root(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            target = Path(temporary) / "target"
            target.mkdir()
            link = Path(temporary) / "link"
            os.symlink(target, link)
            with self.assertRaises(SystemExit):
                smoke.select_images(arguments(media_root=link))

    def test_model_files_are_bound_to_catalog_size_and_digest(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            model_root = root / "models"
            model_root.mkdir()
            detector = model_root / "detector.onnx"
            embedder = model_root / "embedder.onnx"
            detector.write_bytes(b"detector")
            embedder.write_bytes(b"embedder")

            def entry(identifier: str, purpose: str, path: Path) -> dict[str, object]:
                content = path.read_bytes()
                return {
                    "id": identifier,
                    "purpose": purpose,
                    "version": "test",
                    "filename": path.name,
                    "size_bytes": len(content),
                    "sha256": hashlib.sha256(content).hexdigest(),
                }

            catalog = root / "catalog.json"
            catalog.write_text(
                json.dumps(
                    {
                        "models": [
                            entry("detector", "face_detection", detector),
                            entry("embedder", "face_embedding", embedder),
                        ]
                    }
                ),
                encoding="utf-8",
            )
            args = arguments(catalog=catalog, model_root=model_root)
            detector_entry, embedder_entry = smoke.load_models(args)
            self.assertEqual(detector_entry["id"], "detector")
            self.assertEqual(embedder_entry["id"], "embedder")

            embedder.write_bytes(b"tampered")
            with self.assertRaises(SystemExit):
                smoke.load_models(args)

    def test_directory_group_pair_score_is_bounded_and_does_not_claim_identity_metrics(self) -> None:
        result = smoke.score_directory_group_pairs(
            [
                ("a", [1.0, 0.0]),
                ("a", [0.99, 0.01]),
                ("b", [0.0, 1.0]),
                ("b", [0.01, 0.99]),
            ],
            100,
        )
        self.assertEqual(result["functional_groups"], 2)
        self.assertEqual(result["within_group_pairs"], 2)
        self.assertEqual(result["cross_group_pairs"], 4)
        self.assertEqual(
            result["group_semantics"],
            "directory-group-only-not-identity-ground-truth",
        )
        self.assertNotIn("same_pairs", result)
        self.assertNotIn("different_pairs", result)
        for metric in result["threshold_metrics"]:
            self.assertNotIn("same_pair_recall", metric)
            self.assertNotIn("different_pair_false_positive_rate", metric)
        self.assertFalse(result["identity_labels_persisted"])
        self.assertFalse(result["embeddings_persisted"])
        self.assertNotIn("observations", result)

    def test_pair_limit_is_balanced_across_group_pairs(self) -> None:
        observations = [
            (group, [1.0, float(index + 1)])
            for index, group in enumerate(("a", "a", "b", "b", "c", "c", "d", "d"))
        ]
        result = smoke.score_directory_group_pairs(observations, 100)
        self.assertEqual(result["cross_group_pairs"], 24)
        self.assertEqual(
            result["sampling"], "deterministic-balanced-across-group-pairs"
        )

        quotas = smoke.balanced_quotas([100, 100, 100, 100], 10)
        self.assertEqual(sum(quotas), 10)
        self.assertLessEqual(max(quotas) - min(quotas), 1)


if __name__ == "__main__":
    unittest.main()
