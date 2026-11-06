#!/usr/bin/env python3
"""Mouse Maker's helper (ollie-mouse): a drawn mouse pointer, made into a real
one. The drawing is a grid of N x N squares (N = 24), one "RRGGBB" each, or
"......" for an empty (see-through) one, row by row.

  mouse.py use X Y GRID [links]   the pointer, with its tip (what clicks) at
                                  square X, Y: a cursor theme of its own in
                                  ~/.local/share/icons, switched on; "links":
                                  over links and buttons too
  mouse.py normal                 back to the pointer from before

The theme takes the rest (the text bar, the arrows for sizes) from the pointer
that was on before. Each time it gets a new name (ollie-my-mouse-TIME), so
Plasma loads the new drawing; the old ones are taken away. The drawing is kept
in ~/.local/share/ollie/my-mouse (X Y GRID links), for the next time.
"""
import glob
import os
import shutil
import struct
import subprocess
import sys
import time

N = 24
SIZES = (24, 32, 36, 48, 64, 72, 96)  # each pointer size Plasma can ask for
ARROW = ("left_ptr", "default", "arrow", "top_left_arrow", "left_arrow")
LINKS = ("pointer", "hand1", "hand2", "pointing_hand", "e29285e634086352946a0e7090d73106",
         "9d800788f1b08800ae810202380a0822")
DEFAULT = "Bibata-Modern-Ice"  # Ollie OS's pointer (/etc/xdg/kcminputrc)

home = os.path.expanduser("~")
data = os.environ.get("XDG_DATA_HOME") or os.path.join(home, ".local/share")
icons = os.path.join(data, "icons")
keep = os.path.join(data, "ollie")


def parse(grid):
    if len(grid) != N * N * 6:
        raise ValueError("the drawing is the wrong size")
    out = []
    for i in range(N * N):
        cell = grid[i * 6:i * 6 + 6]
        if cell == "......":
            out.append(None)
        else:
            out.append(tuple(int(cell[k:k + 2], 16) for k in (0, 2, 4)))
    return out


def xcursor(cells, hx, hy):
    """An Xcursor file: the drawing at each size (square by square, bigger)."""
    images = []
    for size in SIZES:
        px = []
        for y in range(size):
            for x in range(size):
                c = cells[(y * N // size) * N + (x * N // size)]
                px.append(0 if c is None else 0xFF000000 | c[0] << 16 | c[1] << 8 | c[2])
        # The tip: the middle of its square, at this size.
        tx = min(size - 1, (hx * size + size // 2) // N) if size > N else hx * size // N
        ty = min(size - 1, (hy * size + size // 2) // N) if size > N else hy * size // N
        images.append((size, tx, ty, px))
    header = 16
    toc = 12 * len(images)
    chunks, pos, entries = [], header + toc, []
    for size, tx, ty, px in images:
        chunk = struct.pack("<9I", 36, 0xFFFD0002, size, 1, size, size, tx, ty, 0) + struct.pack(f"<{len(px)}I", *px)
        entries.append(struct.pack("<3I", 0xFFFD0002, size, pos))
        chunks.append(chunk)
        pos += len(chunk)
    return b"Xcur" + struct.pack("<3I", header, 0x10000, len(images)) + b"".join(entries) + b"".join(chunks)


def config(key, value=None):
    """Plasma's pointer setting (kcminputrc, Mouse): read it, or write it."""
    if value is None:
        try:
            return subprocess.run(["kreadconfig6", "--file", "kcminputrc", "--group", "Mouse", "--key", key],
                                  capture_output=True, text=True, timeout=10).stdout.strip()
        except (OSError, subprocess.SubprocessError):
            return ""
    try:
        subprocess.run(["kwriteconfig6", "--file", "kcminputrc", "--group", "Mouse", "--key", key, value], timeout=10)
    except (OSError, subprocess.SubprocessError):
        pass
    return value


def switch(theme):
    """Switch the pointer now (and for the next start)."""
    try:
        if subprocess.run(["plasma-apply-cursortheme", theme], capture_output=True, timeout=20).returncode == 0:
            return
    except (OSError, subprocess.SubprocessError):
        pass
    config("cursorTheme", theme)


def before():
    """The pointer from before Mouse Maker's (kept the first time)."""
    path = os.path.join(keep, "my-mouse-before")
    now = config("cursorTheme")
    if now and not now.startswith("ollie-my-mouse"):
        os.makedirs(keep, exist_ok=True)
        with open(path, "w") as f:
            f.write(now + "\n")
        return now
    try:
        with open(path) as f:
            return f.read().strip() or DEFAULT
    except OSError:
        return DEFAULT


def use(hx, hy, grid, links):
    cells = parse(grid)
    if not 0 <= hx < N or not 0 <= hy < N:
        raise ValueError("the tip is not on the drawing")
    if all(c is None for c in cells):
        raise ValueError("the drawing is empty")
    parent = before()
    name = "ollie-my-mouse-%d" % time.time_ns()
    theme = os.path.join(icons, name)
    os.makedirs(os.path.join(theme, "cursors"), exist_ok=True)
    with open(os.path.join(theme, "index.theme"), "w") as f:
        f.write(f"[Icon Theme]\nName=My own mouse (Mouse Maker)\nComment=Drawn in Mouse Maker\nInherits={parent}\n")
    with open(os.path.join(theme, "cursors", "left_ptr"), "wb") as f:
        f.write(xcursor(cells, hx, hy))
    for other in ARROW[1:] + (LINKS if links else ()):
        os.symlink("left_ptr", os.path.join(theme, "cursors", other))
    os.makedirs(keep, exist_ok=True)
    with open(os.path.join(keep, "my-mouse"), "w") as f:
        f.write(f"{hx} {hy} {grid} {'links' if links else 'arrow'}\n")
    switch(name)
    for old in glob.glob(os.path.join(icons, "ollie-my-mouse-*")):
        if os.path.basename(old) != name and os.path.isdir(old):
            shutil.rmtree(old, ignore_errors=True)


def normal():
    parent = before()
    switch(parent)
    for old in glob.glob(os.path.join(icons, "ollie-my-mouse-*")):
        if os.path.isdir(old):
            shutil.rmtree(old, ignore_errors=True)


def main(a):
    try:
        if len(a) >= 4 and a[0] == "use":
            use(int(a[1]), int(a[2]), a[3], len(a) > 4 and a[4] == "links")
        elif a[:1] == ["normal"]:
            normal()
        else:
            print(__doc__, file=sys.stderr)
            return 2
    except (ValueError, OSError) as e:
        print(f"Error: {e}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
