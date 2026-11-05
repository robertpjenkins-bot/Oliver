#!/usr/bin/env python3
"""Couch Co-op's controller watcher (ollie-coop): writes which game
controllers are plugged in, and which of their buttons are held right now,
to a small file that coop.qml reads many times a second.

  pads.py OUT.json

Controllers are the input devices the kernel gives a joystick handler (js);
plugging one in or out is noticed within 2 seconds. Reading them needs no
admin rights: Ollie OS lets the person at the computer read game
controllers (udev's uaccess).
"""
import json
import os
import re
import select
import struct
import sys
import time

EVENT = struct.Struct("qqHHi")  # struct input_event on 64-bit Linux
EV_KEY, EV_ABS = 1, 3
NAMES = {0x130: "A", 0x131: "B", 0x133: "Y", 0x134: "X", 0x136: "LB", 0x137: "RB", 0x138: "LT", 0x139: "RT",
         0x13a: "Back", 0x13b: "Start", 0x13c: "Home", 0x13d: "L-stick", 0x13e: "R-stick",
         0x220: "Up", 0x221: "Down", 0x222: "Left", 0x223: "Right",
         0x120: "1", 0x121: "2", 0x122: "3", 0x123: "4", 0x124: "5", 0x125: "6", 0x126: "7", 0x127: "8"}


def controllers():
    """{event device: name} of the joysticks and game pads."""
    found, name = {}, ""
    try:
        text = open("/proc/bus/input/devices").read()
    except OSError:
        return found
    for block in text.split("\n\n"):
        m = re.search(r'^N: Name="(.*)"', block, re.M)
        h = re.search(r"^H: Handlers=(.*)$", block, re.M)
        if not m or not h:
            continue
        handlers = h.group(1).split()
        ev = [x for x in handlers if x.startswith("event")]
        if ev and any(x.startswith("js") for x in handlers):
            found["/dev/input/" + ev[0]] = m.group(1)
    return found


def main(out):
    pads = {}  # device: {"fd", "name", "held": set, "since"}
    order = []  # devices in the order they came: Player 1, 2, 3...
    last_scan, last_write, dirty = 0.0, 0.0, True
    while True:
        now = time.time()
        if now - last_scan > 2:
            last_scan = now
            seen = controllers()
            for dev, name in seen.items():
                if dev not in pads:
                    try:
                        fd = os.open(dev, os.O_RDONLY | os.O_NONBLOCK)
                    except OSError:
                        continue
                    pads[dev] = {"fd": fd, "name": name, "held": set(), "hat": [0, 0], "presses": 0}
                    order.append(dev)
                    dirty = True
            for dev in [d for d in pads if d not in seen]:
                os.close(pads[dev]["fd"])
                del pads[dev]
                order.remove(dev)
                dirty = True
        fds = {p["fd"]: d for d, p in pads.items()}
        ready = select.select(list(fds), [], [], 0.25)[0] if fds else (time.sleep(0.25) or [])
        for fd in ready:
            dev = fds[fd]
            p = pads.get(dev)
            try:
                data = os.read(fd, EVENT.size * 64)
            except BlockingIOError:
                continue
            except OSError:  # unplugged
                os.close(fd)
                del pads[dev]
                order.remove(dev)
                dirty = True
                continue
            for i in range(0, len(data) - EVENT.size + 1, EVENT.size):
                _, _, etype, code, value = EVENT.unpack_from(data, i)
                if etype == EV_KEY and code >= 0x100:
                    if value:
                        p["held"].add(code)
                        p["presses"] += 1
                    else:
                        p["held"].discard(code)
                    dirty = True
                elif etype == EV_ABS and code in (0x10, 0x11):  # the D-pad, as a hat
                    p["hat"][code - 0x10] = value
                    dirty = True
        if dirty and time.time() - last_write > 0.04:
            players = []
            for dev in order:
                p = pads[dev]
                held = [NAMES.get(c, "Button") for c in sorted(p["held"])]
                hx, hy = p["hat"]
                held += (["Left"] if hx < 0 else ["Right"] if hx > 0 else []) + (["Up"] if hy < 0 else ["Down"] if hy > 0 else [])
                players.append({"name": p["name"], "held": held, "presses": p["presses"]})
            tmp = out + ".new"
            with open(tmp, "w") as f:
                json.dump({"players": players}, f)
            os.replace(tmp, out)
            last_write, dirty = time.time(), False


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(__doc__)
        sys.exit(2)
    try:
        main(sys.argv[1])
    except KeyboardInterrupt:
        pass
