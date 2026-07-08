#!/usr/bin/env python3
import sys, json, base64

for line in sys.stdin:
    req = json.loads(line)
    rid = req["id"]
    hook = req["hook"]
    if hook == "segment":
        data = base64.b64decode(req["ciphertext"])
        if data == b"first":
            sys.exit(1)
        resp = {"id": rid, "ok": True, "data": base64.b64encode(data).decode()}
        print(json.dumps(resp), flush=True)
    else:
        resp = {"id": rid, "ok": False, "error": "not implemented"}
        print(json.dumps(resp), flush=True)
