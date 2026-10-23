"""Test controller only: requests inside an offline application's network namespace."""
import base64
import json
import sys
import urllib.error
import urllib.request

payload = json.load(sys.stdin)
request = urllib.request.Request(
    payload.get("base", "http://127.0.0.1:8080") + "/api/v1" + payload["path"],
    data=None if payload["body"] is None else json.dumps(payload["body"]).encode(),
    headers=payload["headers"], method=payload["method"],
)
opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
try:
    response = opener.open(request, timeout=15)
except urllib.error.HTTPError as error:
    response = error
with response:
    print(json.dumps({"status": response.status, "headers": list(response.headers.items()),
                      "body": base64.b64encode(response.read()).decode()}))
