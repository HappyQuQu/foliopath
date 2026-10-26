import tempfile
import unittest
from pathlib import Path

import numpy as np
from PIL import Image

from probe import SAFE_LABELS, prepare_image, select_safe_tags, sha256_file, verify_model


class ProbeTests(unittest.TestCase):
    def test_emits_only_allowed_trained_classes(self):
        labels = ["car", "sky", "breasts", "touhou", "ethnicity", "unknown"]
        results = select_safe_tags(np.array([[3, 0, 20, 20, 20, 20]]), labels)
        self.assertEqual([row["tag"] for row in results], ["car", "sky"])
        self.assertNotIn("breasts", SAFE_LABELS)
        self.assertNotIn("touhou", SAFE_LABELS)

    def test_no_forced_top_tag_below_threshold(self):
        self.assertEqual(select_safe_tags(np.array([[-8, -8]]), ["car", "sky"]), [])

    def test_rejects_unexpected_or_nonfinite_outputs(self):
        for values in (np.array([[float("nan")]]), np.array([[float("inf")]]), np.array([1])):
            with self.assertRaises(ValueError):
                select_safe_tags(values, ["car"])

    def test_rejects_corrupt_or_symlinked_model_file(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            source = directory / "weights"
            source.write_bytes(b"fixture")
            manifest = {"files": [{"name": "weights", "bytes": 7, "sha256": sha256_file(source)}]}
            verify_model(directory, manifest)
            source.write_bytes(b"changed")
            with self.assertRaises(ValueError):
                verify_model(directory, manifest)
            link = directory / "linked"
            link.symlink_to(source)
            manifest["files"][0]["name"] = "linked"
            with self.assertRaises(ValueError):
                verify_model(directory, manifest)

    def test_preprocessing_float32_nchw_and_white_square_padding(self):
        with tempfile.TemporaryDirectory() as temporary:
            path = Path(temporary) / "synthetic.png"
            Image.new("RGB", (8, 4), (0, 0, 0)).save(path)
            tensor = prepare_image(path, image_size=8)
            self.assertEqual(tensor.shape, (1, 3, 8, 8))
            self.assertEqual(tensor.dtype, np.float32)
            self.assertTrue(tensor.flags.c_contiguous)
            self.assertTrue((tensor[0, :, 0, 0] > 1).all())
            self.assertTrue((tensor[0, :, 4, 4] < -1).all())


if __name__ == "__main__":
    unittest.main()
