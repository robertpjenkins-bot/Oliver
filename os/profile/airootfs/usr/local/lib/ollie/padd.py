#!/usr/bin/env python3
"""Ollie OS's controller helper (ollie-padd.service, for each person):

- The first time a game controller is ever plugged in, it opens a window
  asking which button should open the Game Hub ("press it now").
- That button opens the Game Hub, or the Quick Menu while a game is running.
- While the Game Hub or the Quick Menu is open, the buttons drive it: it
  writes them to $XDG_RUNTIME_DIR/ollie-pad-events, which they read.
- Controller as a mouse (Ollie OS Settings): on the desktop, when no game
  and no Steam is running, the left stick moves the mouse, the right stick
  scrolls, A clicks, B right-clicks, the d-pad is the arrow keys, Back is
  Escape and Start opens the apps menu (through /dev/uinput).

  padd.py              run
  padd.py --selftest   check the button logic with made-up presses

Settings: ~/.config/ollie/controller.conf (HUB_BUTTON=316, MOUSE=on|off).
"""
import fcntl
import json
import os
import re
import select
import struct
import subprocess
import sys
import time

EVENT = struct.Struct("qqHHi")  # struct input_event on 64-bit Linux
EV_SYN, EV_KEY, EV_REL, EV_ABS = 0, 1, 2, 3
REL_X, REL_Y, REL_WHEEL, REL_HWHEEL = 0, 1, 8, 6
ABS_X, ABS_Y, ABS_RX, ABS_RY, ABS_HAT0X, ABS_HAT0Y = 0, 1, 3, 4, 16, 17
BTN_LEFT, BTN_RIGHT, BTN_MIDDLE = 0x110, 0x111, 0x112
KEY_ESC, KEY_ENTER, KEY_UP, KEY_DOWN, KEY_LEFT, KEY_RIGHT, KEY_LEFTMETA = 1, 28, 103, 108, 105, 106, 125
A, B, X, Y = 0x130, 0x131, 0x133, 0x134
LB, RB, BACK, START, HOME, LS, RS = 0x136, 0x137, 0x13a, 0x13b, 0x13c, 0x13d, 0x13e
DPAD = {0x220: "up", 0x221: "down", 0x222: "left", 0x223: "right"}
NAMES = {A: "A", B: "B", X: "X", Y: "Y", LB: "LB / L1", RB: "RB / R1", 0x138: "LT / L2", 0x139: "RT / R2",
         BACK: "Back / Select / Share", START: "Start / Menu / Options", HOME: "Home / Guide / PS",
         LS: "left stick press", RS: "right stick press", 0x13f: "Capture"}
# The Game Hub's and Quick Menu's words for the buttons.
WORDS = {A: "ok", B: "back", X: "x", Y: "y", LB: "lb", RB: "rb", START: "start", BACK: "select"}

HOME_DIR = os.path.expanduser("~")
CONF = os.path.join(HOME_DIR, ".config", "ollie", "controller.conf")
RUN = os.environ.get("XDG_RUNTIME_DIR", "/tmp")
EVENTS = os.path.join(RUN, "ollie-pad-events")
LEARN = os.path.join(RUN, "ollie-pad-learn")
HUB_OPEN = os.path.join(RUN, "ollie-hub-open")       # the Game Hub is open
MENU_OPEN = os.path.join(RUN, "ollie-quickmenu-open")  # the Quick Menu is open
LEARN_ASK = os.path.join(RUN, "ollie-pad-learn-request")  # Settings: choose the button again
CURRENT_GAME = os.path.join(RUN, "ollie-current-game")


def read_conf():
    conf = {"HUB_BUTTON": "", "MOUSE": "on"}
    try:
        for line in open(CONF):
            k, _, v = line.strip().partition("=")
            if k in conf:
                conf[k] = v
    except OSError:
        pass
    return conf


def game_running():
    try:
        return open(CURRENT_GAME).read().strip() != ""
    except OSError:
        return False


def steam_running():
    return subprocess.run(["pgrep", "-u", str(os.getuid()), "-x", "steam"], stdout=subprocess.DEVNULL).returncode == 0


class Brain:
    """What each press does: no devices here, so it can be tested."""

    def __init__(self, out):
        self.out = out            # out(kind, *args): "launch" NAME, "event" WORD, "key" CODE 0/1, "rel" AXIS N, "learned" CODE
        self.conf = read_conf()
        self.learning = False
        self.seq = 0
        self.held_dirs = set()
        self.axes = {}

    def hub_button(self):
        try:
            return int(self.conf.get("HUB_BUTTON") or 0)
        except ValueError:
            return 0

    def first_controller(self):
        """A controller came: the first time ever, ask about the button."""
        if not os.path.exists(CONF) and not self.learning:
            self.learning = True
            self.out("launch", "setup")

    def mode(self, hub_open, game, steam):
        if self.learning:
            return "learn"
        if hub_open:
            return "hub"
        if game or steam or self.conf.get("MOUSE") != "on":
            return "off"
        return "mouse"

    def key(self, code, value, hub_open=False, game=False, steam=False):
        mode = self.mode(hub_open, game, steam)
        if mode == "learn":
            if value == 1 and code not in DPAD:
                self.out("learned", code)
            return
        if value == 1 and code and code == self.hub_button():
            self.out("launch", "quickmenu" if game else "hub")
            return
        if mode == "hub":
            if value == 1:
                word = DPAD.get(code) or WORDS.get(code)
                if word:
                    self.out("event", word)
        elif mode == "mouse":
            target = {A: BTN_LEFT, B: BTN_RIGHT, Y: BTN_MIDDLE, BACK: KEY_ESC, START: KEY_LEFTMETA,
                      X: KEY_ENTER, 0x220: KEY_UP, 0x221: KEY_DOWN, 0x222: KEY_LEFT, 0x223: KEY_RIGHT}.get(code)
            if target and value in (0, 1):
                self.out("key", target, value)

    def axis(self, code, value, lo, hi, hub_open=False, game=False, steam=False):
        """A stick or a d-pad that is an axis (most pads report the d-pad so)."""
        mode = self.mode(hub_open, game, steam)
        if code in (ABS_HAT0X, ABS_HAT0Y):
            names = ("left", "right") if code == ABS_HAT0X else ("up", "down")
            if mode == "hub" and value != 0:
                self.out("event", names[0] if value < 0 else names[1])
            elif mode == "mouse":
                keys = (KEY_LEFT, KEY_RIGHT) if code == ABS_HAT0X else (KEY_UP, KEY_DOWN)
                for i, k in enumerate(keys):
                    want = (value < 0) if i == 0 else (value > 0)
                    if want != (k in self.held_dirs):
                        self.held_dirs.symmetric_difference_update({k})
                        self.out("key", k, 1 if want else 0)
            return
        mid, half = (lo + hi) / 2, max(1, (hi - lo) / 2)
        self.axes[code] = max(-1.0, min(1.0, (value - mid) / half))
        if mode == "hub" and code in (ABS_X, ABS_Y):
            v = self.axes[code]
            name = ("left", "right") if code == ABS_X else ("up", "down")
            was = self.axes.get(("pushed", code), 0)
            now = -1 if v < -0.6 else 1 if v > 0.6 else 0
            if now != was:
                self.axes[("pushed", code)] = now
                if now:
                    self.out("event", name[0] if now < 0 else name[1])

    def tick(self, dt, hub_open=False, game=False, steam=False):
        """Called many times a second: the sticks move the mouse."""
        if self.mode(hub_open, game, steam) != "mouse":
            return
        def curve(v):
            v = 0.0 if abs(v) < 0.15 else (abs(v) - 0.15) / 0.85 * (1 if v > 0 else -1)
            return v * abs(v)          # gentle near the middle, fast at the edge
        dx, dy = curve(self.axes.get(ABS_X, 0)) * 1400 * dt, curve(self.axes.get(ABS_Y, 0)) * 1400 * dt
        if dx or dy:
            self.out("rel", REL_X, dx)
            self.out("rel", REL_Y, dy)
        wheel = curve(-self.axes.get(ABS_RY, 0)) * 12 * dt
        if wheel:
            self.out("rel", REL_WHEEL, wheel)


# --- The devices ---------------------------------------------------------------------------
def controllers():
    """{event device: name} of the game pads and joysticks."""
    found = {}
    try:
        text = open("/proc/bus/input/devices").read()
    except OSError:
        return found
    for block in text.split("\n\n"):
        m = re.search(r'^N: Name="(.*)"', block, re.M)
        h = re.search(r"^H: Handlers=(.*)$", block, re.M)
        if not m or not h or "Ollie OS controller mouse" in m.group(1):
            continue
        handlers = h.group(1).split()
        ev = [x for x in handlers if x.startswith("event")]
        if ev and any(x.startswith("js") for x in handlers):
            found["/dev/input/" + ev[0]] = m.group(1)
    return found


def abs_range(fd, code):
    """The lowest and highest value of an axis (EVIOCGABS)."""
    buf = bytearray(24)
    try:
        fcntl.ioctl(fd, 0x80184540 + code, buf)
        _, lo, hi, _, _, _ = struct.unpack("6i", buf)
        return lo, hi
    except OSError:
        return -32768, 32767


class Mouse:
    """A made-up mouse and keyboard (/dev/uinput) the controller drives."""

    def __init__(self):
        self.fd = None
        self.rest = {}

    def open(self):
        if self.fd is not None:
            return True
        try:
            fd = os.open("/dev/uinput", os.O_WRONLY | os.O_NONBLOCK)
        except OSError:
            return False
        for ev in (EV_KEY, EV_REL, EV_SYN):
            fcntl.ioctl(fd, 0x40045564, ev)                      # UI_SET_EVBIT
        for k in (BTN_LEFT, BTN_RIGHT, BTN_MIDDLE, KEY_ESC, KEY_ENTER, KEY_UP, KEY_DOWN, KEY_LEFT, KEY_RIGHT, KEY_LEFTMETA):
            fcntl.ioctl(fd, 0x40045565, k)                       # UI_SET_KEYBIT
        for r in (REL_X, REL_Y, REL_WHEEL, REL_HWHEEL):
            fcntl.ioctl(fd, 0x40045566, r)                       # UI_SET_RELBIT
        dev = struct.pack("80sHHHHi" + "i" * 256, b"Ollie OS controller mouse", 0x06, 0x4f4c, 0x4c45, 1, 0, *([0] * 256))
        os.write(fd, dev)
        fcntl.ioctl(fd, 0x5501)                                  # UI_DEV_CREATE
        self.fd = fd
        return True

    def send(self, typ, code, value):
        if not self.open():
            return
        if typ == EV_REL:   # keep the bits under one pixel for next time
            total = value + self.rest.get(code, 0.0)
            value = int(total)
            self.rest[code] = total - value
            if value == 0:
                return
        try:
            os.write(self.fd, EVENT.pack(0, 0, typ, code, int(value)) + EVENT.pack(0, 0, EV_SYN, 0, 0))
        except OSError:
            pass


def launch(what):
    cmd = {"setup": ["/usr/local/bin/ollie-gamehub", "--setup"], "hub": ["/usr/local/bin/ollie-gamehub", "--toggle"],
           "quickmenu": ["/usr/local/bin/ollie-quickmenu", "--toggle"]}[what]
    subprocess.Popen(cmd, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL, start_new_session=True)


def main():
    mouse = Mouse()
    events = []        # the newest button words for the Game Hub
    seq = [0]

    def write_events():
        tmp = EVENTS + ".tmp"
        with open(tmp, "w") as f:
            json.dump({"events": events[-20:]}, f)
        os.replace(tmp, EVENTS)

    def out(kind, *a):
        if kind == "launch":
            launch(a[0])
        elif kind == "event":
            seq[0] += 1
            events.append([seq[0], a[0]])
            del events[:-20]
            write_events()
        elif kind == "learned":
            seq[0] += 1
            with open(LEARN + ".tmp", "w") as f:
                json.dump({"code": a[0], "name": NAMES.get(a[0], "button " + str(a[0])), "seq": seq[0]}, f)
            os.replace(LEARN + ".tmp", LEARN)
        elif kind == "key":
            mouse.send(EV_KEY, a[0], a[1])
        elif kind == "rel":
            mouse.send(EV_REL, a[0], a[1])

    brain = Brain(out)
    write_events()
    pads = {}          # device: {"fd", "ranges"}
    conf_time, last_scan, last_state, last = 0, 0.0, 0.0, time.time()
    state = (False, False, False)
    while True:
        now = time.time()
        if now - last_scan > 2:
            last_scan = now
            seen = controllers()
            for dev in seen:
                if dev not in pads:
                    try:
                        fd = os.open(dev, os.O_RDONLY | os.O_NONBLOCK)
                    except OSError:
                        continue
                    pads[dev] = {"fd": fd, "ranges": {c: abs_range(fd, c) for c in (ABS_X, ABS_Y, ABS_RX, ABS_RY, ABS_HAT0X, ABS_HAT0Y)}}
                    brain.first_controller()
            for dev in [d for d in pads if d not in seen]:
                os.close(pads[dev]["fd"])
                del pads[dev]
            try:
                t = os.path.getmtime(CONF)
            except OSError:
                t = 0
            if t != conf_time:
                conf_time = t
                brain.conf = read_conf()
                if t:
                    brain.learning = False
        if now - last_state > 0.5:
            last_state = now
            state = (os.path.exists(HUB_OPEN) or os.path.exists(MENU_OPEN), game_running(), steam_running() if pads else False)
            if os.path.exists(LEARN_ASK):
                os.remove(LEARN_ASK)
                brain.learning = True
        fds = {p["fd"]: d for d, p in pads.items()}
        ready = select.select(list(fds), [], [], 0.016)[0] if fds else (time.sleep(0.5) or [])
        for fd in ready:
            dev = fds[fd]
            try:
                data = os.read(fd, EVENT.size * 64)
            except BlockingIOError:
                continue
            except OSError:
                continue
            for i in range(0, len(data) - EVENT.size + 1, EVENT.size):
                _, _, typ, code, value = EVENT.unpack_from(data, i)
                if typ == EV_KEY:
                    brain.key(code, value, *state)
                elif typ == EV_ABS:
                    lo, hi = pads[dev]["ranges"].get(code, (-32768, 32767))
                    brain.axis(code, value, lo, hi, *state)
        now = time.time()
        brain.tick(min(0.05, now - last), *state)
        last = now


def selftest():
    got = []
    b = Brain(lambda *a: got.append(a))
    b.conf = {"HUB_BUTTON": str(HOME), "MOUSE": "on"}
    ok = True

    def expect(what, want):
        nonlocal ok
        if got != want:
            ok = False
            print("FAIL", what, got, "wanted", want)
        got.clear()

    b.key(HOME, 1)
    expect("hub button on the desktop opens the Game Hub", [("launch", "hub")])
    b.key(HOME, 1, game=True)
    expect("hub button in a game opens the Quick Menu", [("launch", "quickmenu")])
    b.key(A, 1, hub_open=True)
    b.axis(ABS_HAT0Y, 1, -1, 1, hub_open=True)
    b.axis(ABS_X, -30000, -32768, 32767, hub_open=True)
    b.axis(ABS_X, -31000, -32768, 32767, hub_open=True)
    expect("in the Game Hub the buttons and sticks are words", [("event", "ok"), ("event", "down"), ("event", "left")])
    b.key(A, 1)
    b.key(A, 0)
    expect("on the desktop A clicks", [("key", BTN_LEFT, 1), ("key", BTN_LEFT, 0)])
    b.key(A, 1, game=True)
    b.key(A, 1, steam=True)
    expect("no mouse in games or with Steam open", [])
    b.axis(ABS_X, 32767, -32768, 32767)
    b.tick(0.1)
    moved = [g for g in got if g[0] == "rel" and g[1] == REL_X]
    got.clear()
    if not moved or moved[0][2] <= 50:
        ok = False
        print("FAIL the stick moves the mouse", moved)
    b.conf["MOUSE"] = "off"
    b.key(A, 1)
    expect("controller mouse switched off", [])
    b.conf = {"HUB_BUTTON": "", "MOUSE": "on"}
    b.learning = True
    b.key(0x220, 1)
    b.key(START, 1)
    expect("learning the Game Hub button (not the d-pad)", [("learned", START)])
    print("selftest", "ok" if ok else "FAILED")
    return ok


if __name__ == "__main__":
    if sys.argv[1:] == ["--selftest"]:
        sys.exit(0 if selftest() else 1)
    main()
