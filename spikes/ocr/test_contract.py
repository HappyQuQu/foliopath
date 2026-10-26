import copy
import unittest

import ocr_contract as c


def raw():
    return {"text": ["中文 FolioPath 2026"], "boxes": [[.1, .1, .9, .1, .9, .3, .1, .3]], "detection": [.9], "recognition": [.95]}

class ContractTests(unittest.TestCase):
    def test_bilingual_and_blank_are_complete_results(self):
        self.assertEqual(c.normalize(raw())["lines"][0]["text"], "中文 FolioPath 2026")
        self.assertEqual(c.normalize({key: [] for key in raw()}), {"lines": []})

    def test_preserves_plain_text_instead_of_interpreting_html(self):
        value = raw(); value["text"] = ["<b>synthetic</b>"]
        self.assertEqual(c.normalize(value)["lines"][0]["text"], value["text"][0])

    def test_unicode_nfc(self):
        value = raw(); value["text"] = ["Cafe\u0301"]
        self.assertEqual(c.normalize(value)["lines"][0]["text"], "Café")

    def test_rejects_entire_malformed_result(self):
        changes = [lambda v: v.update(extra="not allowed"), lambda v: v["recognition"].clear(),
                   lambda v: v.update(text=[None]), lambda v: v.update(text=[" "]),
                   lambda v: v.update(text=["a\0b"]), lambda v: v.update(text=["\ud800"]),
                   lambda v: v.update(text=["x" * (c.MAX_LINE + 1)]),
                   lambda v: v.update(text=["\u0344" * c.MAX_LINE]),
                   lambda v: v.update(boxes=[[0] * 8]), lambda v: v.update(boxes=[[0] * 7]),
                   lambda v: v["boxes"][0].__setitem__(0, -0.1),
                   lambda v: v["boxes"][0].__setitem__(0, float("nan")),
                   lambda v: v.update(detection=[True]), lambda v: v.update(recognition=[float("inf")]),
                   lambda v: v.update(detection=[.49]), lambda v: v.update(recognition=[.79]),
                   lambda v: v.update(boxes=[[.1, .1, .9, .3, .9, .1, .1, .3]])]
        for change in changes:
            with self.subTest(change=change):
                value = raw(); change(value)
                with self.assertRaises(c.InvalidOutput): c.normalize(value)

    def test_rejects_region_and_total_text_overflow(self):
        base = raw()
        for size, text in [(c.MAX_REGIONS + 1, "ok"), (20, "x" * c.MAX_LINE)]:
            value = {key: copy.deepcopy(items * size) for key, items in base.items()}
            value["text"] = [text] * size
            with self.assertRaises(c.InvalidOutput): c.normalize(value)

    def test_detection_scores_follow_boxes_when_sorting(self):
        a, b = [[0, 0], [8, 0], [8, 2], [0, 2]], [[0, 4], [8, 4], [8, 6], [0, 6]]
        self.assertEqual(c.align_detections([b, a], [.6, .9], [a, b]), [.9, .6])
        self.assertEqual(c.align_detections([a, a], [.6, .9], [a, a]), [.6, .9])
        with self.assertRaises(c.InvalidOutput): c.align_detections([a, b], [.6, .9], [a, a])
        with self.assertRaises(c.InvalidOutput): c.align_detections([a], [.4], [a])

    def test_row_order_and_scores_are_not_model_quality_claims(self):
        top_left = [[0, 0], [8, 0], [8, 4], [0, 4]]
        top_right = [[12, 1], [20, 1], [20, 5], [12, 5]]
        bottom = [[0, 12], [8, 12], [8, 16], [0, 16]]
        self.assertEqual(c.reading_order([bottom, top_right, top_left]), [2, 1, 0])
        self.assertEqual(c.reading_order([]), [])
        with self.assertRaises(c.InvalidOutput): c.reading_order([[[0, 0]] * 4])

if __name__ == "__main__": unittest.main()
