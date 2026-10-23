import uvicorn

if __name__ == "__main__":
    uvicorn.run("photo_server:application", host="0.0.0.0", port=3003, workers=1,
                limit_concurrency=4, access_log=False)
