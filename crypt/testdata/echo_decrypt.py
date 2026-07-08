#!/usr/bin/env python3
import sys, json, base64

for line in sys.stdin:
    req = json.loads(line)
    rid = req["id"]
    hook = req["hook"]
    if hook == "key":
        resp = {"id": rid, "ok": True, "key": req["raw_key"], "iv": req.get("iv", "")}
    elif hook == "segment":
        data = base64.b64decode(req["ciphertext"])
        resp = {"id": rid, "ok": True, "data": base64.b64encode(data).decode()}
    else:
        resp = {"id": rid, "ok": False, "error": "unknown hook"}
    print(json.dumps(resp), flush=True)
