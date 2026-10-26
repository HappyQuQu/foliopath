"""Request-scoped canonical PNG -> OCR child. Scheduler owns process lifetime."""
from contextlib import redirect_stdout
import io
import json
import os
import sys

from cache_snapshot import snapshot, InvalidCache
from ocr_contract import (MANIFEST, MAX_BYTES, INPUT, OUTPUT, InvalidInput, InvalidMedia, InputLimit,
                          InvalidOutput, fingerprint, validate_png)


def main():
    kind,source,work=sys.argv[1:]
    if kind!="ocr": raise ValueError("invalid worker kind")
    try:
        payload=sys.stdin.buffer.read(MAX_BYTES+1)
        dimensions=validate_png(payload)
        cache=str(snapshot(source,work,MANIFEST))
        for key in list(os.environ):
            if key.startswith("MACHINE_LEARNING_"):del os.environ[key]
        os.environ.update({"MACHINE_LEARNING_CACHE_FOLDER":cache,"HF_HUB_OFFLINE":"1","HF_HUB_DISABLE_TELEMETRY":"1",
            "MACHINE_LEARNING_REQUEST_THREADS":"1","MACHINE_LEARNING_MODEL_INTRA_OP_THREADS":"2","MACHINE_LEARNING_MODEL_INTER_OP_THREADS":"1",
            "MACHINE_LEARNING_MAX_BATCH_SIZE__OCR":str(OUTPUT["recognition_batch"]),"MPLCONFIGDIR":"/tmp/matplotlib"})
        sys.path.insert(0,"/usr/src")
        from PIL import Image
        Image.MAX_IMAGE_PIXELS=INPUT["max_output_pixels"]
        try:
            image=Image.open(io.BytesIO(payload))
        except (Image.DecompressionBombError,Image.DecompressionBombWarning):raise InputLimit() from None
        except (OSError,ValueError):raise InvalidMedia() from None
        try:
            try:
                if image.format!="PNG" or image.mode!="RGB" or image.size!=dimensions or getattr(image,"n_frames",1)!=1:raise InvalidMedia()
                image.load()
            except (Image.DecompressionBombError,Image.DecompressionBombWarning):raise InputLimit() from None
            except (OSError,ValueError):raise InvalidMedia() from None
            from ocr_engine import infer
            with redirect_stdout(sys.stderr):result=infer(cache,image)
        finally:image.close()

    except InvalidInput:result={"identity":fingerprint(),"error":"invalid_input"}
    except InvalidMedia:result={"identity":fingerprint(),"error":"invalid_media"}
    except InputLimit:result={"identity":fingerprint(),"error":"input_limit"}
    except InvalidOutput:result={"identity":fingerprint(),"error":"output_invalid"}
    except InvalidCache:result={"identity":fingerprint(),"error":"model_unavailable"}
    except Exception:result={"identity":fingerprint(),"error":"inference_failed"}
    sys.stdout.write(json.dumps(result,separators=(",",":"),allow_nan=False));sys.stdout.flush()


if __name__=="__main__":main()
