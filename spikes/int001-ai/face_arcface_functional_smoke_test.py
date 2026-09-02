from __future__ import annotations

import sys
import unittest
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))
import face_arcface_functional_smoke as smoke


class Value:
    def __init__(self, name, shape, value_type):
        self.name = name
        self.shape = shape
        self.type = value_type


class Session:
    def __init__(self, input_value, output_value):
        self.input_value = input_value
        self.output_value = output_value

    def get_inputs(self):
        return [self.input_value]

    def get_outputs(self):
        return [self.output_value]


class ArcFaceFunctionalContractTest(unittest.TestCase):
    def test_accepts_pinned_auraface_dynamic_batch_contract(self):
        contract = smoke.validate_embedder_contract(
            {"id": "fal-auraface-v1-glintr100"},
            Session(
                Value("data", ["None", 3, 112, 112], "tensor(float)"),
                Value("1333", [1, 512], "tensor(float)"),
            ),
        )
        self.assertEqual(
            contract["preprocess"],
            "insightface-rgb-minus-127.5-div-127.5-v1",
        )

    def test_rejects_unknown_model_and_tensor_drift(self):
        session = Session(
            Value("data", ["batch", 3, 112, 112], "tensor(float)"),
            Value("1333", [1, 512], "tensor(float)"),
        )
        with self.assertRaises(SystemExit):
            smoke.validate_embedder_contract({"id": "unknown"}, session)
        session.output_value = Value("embedding", [1, 512], "tensor(float)")
        with self.assertRaises(SystemExit):
            smoke.validate_embedder_contract(
                {"id": "fal-auraface-v1-glintr100"}, session
            )


if __name__ == "__main__":
    unittest.main()
