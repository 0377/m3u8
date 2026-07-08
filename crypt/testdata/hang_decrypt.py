#!/usr/bin/env python3
import sys, time

# Hangs until stdin receives a line, then sleeps (for timeout tests).
for line in sys.stdin:
    time.sleep(5)
    print('{"id": 0, "ok": false, "error": "should not reach"}', flush=True)
