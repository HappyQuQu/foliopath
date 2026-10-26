"""Dispatch within the existing one-child deadline; never a second scheduler."""
import os
from pathlib import Path
import sys


if __name__ == "__main__":
    if sys.argv[1] == "tag":
        import photo_policy
        import inference_worker
        # Reuse the exact verified snapshot, image transform and ORT inference
        # owner. Only its explicit fixed output policy changes in this child.
        inference_worker.tag_contract = photo_policy
        inference_worker.main()
    else:
        target = "multilingual_worker.py" if Path(__file__).with_name("multilingual-manifest.json").is_file() else "inference_worker.py"
        os.execv(sys.executable, [sys.executable, str(Path(__file__).with_name(target)), *sys.argv[1:]])
