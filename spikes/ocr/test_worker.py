"""Runs inside the existing inference runtime; no model or real media access."""
import io
import unittest
from PIL import Image
import numpy as np
from ocr_contract import InputLimit, InvalidOutput, MAX_EDGE
from ocr_worker import prepare_image, check_contour_budget, check_crop_budget

def picture(size, format="PNG"):
    stream=io.BytesIO();Image.new("RGB",size,"white").save(stream,format=format);return stream.getvalue()

class WorkerTests(unittest.TestCase):
    def test_header_pixels_and_bytes_rejected_before_inference(self):
        with self.assertRaises(InputLimit):prepare_image(picture((3000,2000)))
        with self.assertRaises(InputLimit):prepare_image(b"x"*(8*1024*1024+1))
        with self.assertRaises(ValueError):prepare_image(b"")
        with self.assertRaises(OSError):prepare_image(b"synthetic broken png")

    def test_long_edge_and_exif_orientation_are_explicit(self):
        image,source=prepare_image(picture((3000,1000)))
        self.assertEqual(source,[3000,1000]);self.assertEqual(max(image.size),MAX_EDGE)
        exif=Image.Exif();exif[274]=6;stream=io.BytesIO();Image.new("RGB",(100,200),"white").save(stream,format="JPEG",exif=exif)
        image,source=prepare_image(stream.getvalue());self.assertEqual(source,[100,200]);self.assertEqual(image.size,(200,100))

    def test_contour_overflow_is_rejected_instead_of_truncated(self):
        bitmap=np.zeros((100,100),dtype=np.uint8);bitmap[::4,::4]=1
        with self.assertRaises(InputLimit):check_contour_budget(bitmap)
        self.assertEqual(check_contour_budget(np.zeros((10,10),dtype=np.uint8)),0)

    def test_aggregate_crop_budget_precedes_native_crops(self):
        box=[[0,0],[2048,0],[2048,2048],[0,2048]]
        with self.assertRaises(InputLimit):check_crop_budget([box]*5)
        with self.assertRaises(InvalidOutput):check_crop_budget([[[0,0]]*4])

if __name__=="__main__":unittest.main()
