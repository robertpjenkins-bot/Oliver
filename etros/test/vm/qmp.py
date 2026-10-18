#!/usr/bin/env python3
"""Drive the VM: qmp.py SOCK shot FILE | dclick X Y (screen pixels, 1280x800)."""
import json, socket, sys, time
s = socket.socket(socket.AF_UNIX); s.connect(sys.argv[1]); f = s.makefile("rw")
f.readline()
def cmd(c):
    f.write(json.dumps(c) + "\n"); f.flush()
    while True:
        r = json.loads(f.readline())
        if "event" not in r:
            return r
cmd({"execute": "qmp_capabilities"})
if sys.argv[2] == "shot":
    print(cmd({"execute": "screendump", "arguments": {"filename": sys.argv[3], "format": "png"}}))
elif sys.argv[2] == "dclick":
    x, y = int(sys.argv[3]) * 32767 // 1280, int(sys.argv[4]) * 32767 // 800
    move = [{"type": "abs", "data": {"axis": "x", "value": x}}, {"type": "abs", "data": {"axis": "y", "value": y}}]
    cmd({"execute": "input-send-event", "arguments": {"events": move}})
    time.sleep(0.3)
    for _ in range(2):
        for down in (True, False):
            cmd({"execute": "input-send-event", "arguments": {"events": [{"type": "btn", "data": {"down": down, "button": "left"}}]}})
            time.sleep(0.05)
        time.sleep(0.08)
