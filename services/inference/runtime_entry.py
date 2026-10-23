"""Image-owned profile selection, never a request or ambient model name."""
from pathlib import Path
import uvicorn

if __name__ == "__main__":
    multilingual = Path(__file__).with_name("multilingual-manifest.json").is_file()
    uvicorn.run("multilingual_server:application" if multilingual else "server:application",
                host="0.0.0.0", port=3003, workers=1, limit_concurrency=4, access_log=False)
