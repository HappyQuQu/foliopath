import argparse
import tempfile
import unittest
from pathlib import Path

import numpy as np

from face_ground_truth_prepare import connected_components, validate_paths, write_review


class ConnectedComponentsTests(unittest.TestCase):
    def test_review_csv_uses_unix_lines_and_stays_pending(self):
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory)
            (output / "thumbnails").mkdir()
            write_review(
                output,
                [{
                    "item_id": "face-001",
                    "source_group": "group-001",
                    "relative_path": "group-001/image.jpg",
                    "thumbnail": "thumbnails/face-001.jpg",
                }],
                [np.asarray([1, 0], dtype=np.float32)],
                [1],
            )

            content = (output / "review.csv").read_bytes()

            self.assertNotIn(b"\r\n", content)
            self.assertIn(b"face-001,candidate-001,group-001,pending,", content)

    def test_large_finite_embeddings_are_normalized_without_overflow(self):
        scale = np.float32(3e30)
        embeddings = [
            np.array([scale, scale, 0], dtype=np.float32),
            np.array([scale, scale, 1], dtype=np.float32),
            np.array([scale, -scale, 0], dtype=np.float32),
        ]

        clusters = connected_components(np, embeddings, 0.9)

        self.assertEqual(clusters[0], clusters[1])
        self.assertNotEqual(clusters[0], clusters[2])

    def test_private_review_allows_bounded_expanded_group_sample(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            media = root / "media"
            media.mkdir()
            args = argparse.Namespace(
                authorization_ref="operator-review-001",
                max_per_group=500,
                max_images=5000,
                max_dimension=1600,
                detector_score_threshold=0.9,
                cluster_threshold=0.6,
                media_root=media,
                output=root / "private-output",
            )

            resolved_media, resolved_output = validate_paths(args)

            self.assertEqual(resolved_media, media.resolve())
            self.assertTrue(resolved_output.is_dir())

    def test_chunked_similarity_stays_finite_for_review_sized_input(self):
        generator = np.random.default_rng(42)
        embeddings = generator.normal(size=(300, 512)).astype(np.float32)
        embeddings[1] = embeddings[0]

        with np.errstate(all="raise"):
            clusters = connected_components(np, embeddings, 0.99)

        self.assertEqual(len(clusters), 300)
        self.assertEqual(clusters[0], clusters[1])

    def test_transitive_bridge_does_not_join_dissimilar_candidates(self):
        embeddings = np.asarray(
            [[1, 0], [.8, .6], [.28, .96]], dtype=np.float32
        )

        clusters = connected_components(
            np, embeddings, .75, ["face-001", "face-002", "face-003"]
        )

        self.assertEqual(clusters[0], clusters[1])
        self.assertNotEqual(clusters[0], clusters[2])


if __name__ == "__main__":
    unittest.main()
