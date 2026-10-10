#!/usr/bin/env python3
"""Draws the Annie Rainbow mouse cursor theme.

Usage: make-cursors.py THEME_DIR

Writes THEME_DIR/index.theme and THEME_DIR/cursors/* in the Xcursor format
X and Wayland desktops read. Cursors not drawn here come from Adwaita.
"""
import math
import os
import struct
import sys

from PIL import Image, ImageDraw, ImageFilter

SIZES = (24, 32, 48, 64)
MASTER = 256
RAINBOW = [(255, 92, 120), (255, 160, 80), (255, 222, 90),
           (110, 220, 140), (90, 170, 255), (170, 120, 255)]
OUTLINE = (60, 30, 90, 255)


def rainbow_fill(size, box):
    """A diagonal rainbow gradient running corner to corner across box."""
    x0, y0, x1, y1 = box
    img = Image.new("RGBA", (size, size))
    px = img.load()
    span = (x1 - x0) * 0.4 + (y1 - y0) * 0.6 or 1
    for y in range(size):
        for x in range(size):
            t = ((x - x0) * 0.4 + (y - y0) * 0.6) / span
            t = min(max(t, 0), 0.999) * (len(RAINBOW) - 1)
            i, f = int(t), t - int(t)
            c0, c1 = RAINBOW[i], RAINBOW[min(i + 1, len(RAINBOW) - 1)]
            px[x, y] = tuple(int(c0[k] + (c1[k] - c0[k]) * f) for k in range(3)) + (255,)
    return img


def shape(points, size=MASTER, sparkle=None, rim=1.0):
    """Rainbow-filled polygon with a dark outline, white rim and soft shadow."""
    s = size / 32
    pts = [(x * s, y * s) for x, y in points]
    mask = Image.new("L", (size, size), 0)
    ImageDraw.Draw(mask).polygon(pts, fill=255)
    out = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    shadow = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    ImageDraw.Draw(shadow).polygon([(x + 1.2 * s, y + 1.6 * s) for x, y in pts], fill=(30, 0, 50, 110))
    out.alpha_composite(shadow.filter(ImageFilter.GaussianBlur(1.4 * s)))
    d = ImageDraw.Draw(out)
    d.polygon(pts, fill=OUTLINE)
    inner = mask.filter(ImageFilter.MinFilter(int(2 * s * rim) | 1))
    white = Image.new("RGBA", (size, size), (255, 255, 255, 255))
    out.paste(white, (0, 0), inner)
    core = inner.filter(ImageFilter.MinFilter(int(1.6 * s * rim) | 1))
    out.paste(rainbow_fill(size, mask.getbbox()), (0, 0), core)
    if sparkle:
        sx, sy, r = sparkle[0] * s, sparkle[1] * s, sparkle[2] * s
        d.ellipse([sx - r * 0.45, sy - r * 0.45, sx + r * 0.45, sy + r * 0.45], fill=(255, 240, 150, 255))
        star = [(sx, sy - r), (sx + r / 3.5, sy), (sx, sy + r), (sx - r / 3.5, sy)]
        star2 = [(sx - r, sy), (sx, sy - r / 3.5), (sx + r, sy), (sx, sy + r / 3.5)]
        for p in (star, star2):
            d.polygon(p, fill=(255, 250, 220, 255), outline=OUTLINE, width=max(1, int(s / 2)))
    return out


ARROW = [(2, 1), (2, 24), (8, 18.5), (12, 28), (16, 26.2), (12, 17), (20, 17)]
HAND = [(9, 2), (12, 2), (12, 12), (14, 11), (16, 11.5), (17, 12.5), (19, 12), (21, 13), (22, 14),
        (24, 14), (26, 16), (26, 24), (23, 30), (12, 30), (6, 21), (5, 18), (6, 16.5), (9, 19)]
IBEAM = [(10, 2), (22, 2), (22, 6.5), (18.5, 6.5), (18.5, 25.5), (22, 25.5), (22, 30), (10, 30), (10, 25.5),
         (13.5, 25.5), (13.5, 6.5), (10, 6.5)]


def ring_frame(i, n, size=MASTER):
    """A spinning rainbow ring for the busy cursor."""
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    s = size / 32
    box = [5 * s, 5 * s, 27 * s, 27 * s]
    d.ellipse([box[0] - s, box[1] - s, box[2] + s, box[3] + s], fill=OUTLINE)
    d.ellipse([box[0] + 5.5 * s, box[1] + 5.5 * s, box[2] - 5.5 * s, box[3] - 5.5 * s], fill=(0, 0, 0, 0))
    seg = 360 / len(RAINBOW)
    rot = i * 360 / n
    for k, c in enumerate(RAINBOW):
        d.pieslice(box, rot + k * seg, rot + (k + 1) * seg, fill=c + (255,))
    hole = Image.new("L", (size, size), 0)
    ImageDraw.Draw(hole).ellipse([box[0] + 4.5 * s, box[1] + 4.5 * s, box[2] - 4.5 * s, box[3] - 4.5 * s], fill=255)
    img.paste((0, 0, 0, 0), (0, 0), hole)
    ImageDraw.Draw(img).ellipse([box[0] + 4.5 * s, box[1] + 4.5 * s, box[2] - 4.5 * s, box[3] - 4.5 * s],
                                outline=OUTLINE, width=int(s))
    return img


def xcursor(frames, hotspot, delay=0):
    """frames: list of master images (one per animation step)."""
    chunks = []
    for size in SIZES:
        for f in frames:
            im = f.resize((size, size), Image.LANCZOS)
            hx, hy = (round(hotspot[0] * size / 32), round(hotspot[1] * size / 32))
            data = bytearray()
            for r, g, b, a in (im.get_flattened_data() if hasattr(im, 'get_flattened_data') else im.getdata()):  # premultiplied ARGB, little endian
                data += struct.pack("<I", (a << 24) | ((r * a // 255) << 16) | ((g * a // 255) << 8) | (b * a // 255))
            hdr = struct.pack("<9I", 36, 0xFFFD0002, size, 1, size, size, min(hx, size - 1), min(hy, size - 1), delay)
            chunks.append((size, hdr + bytes(data)))
    pos = 16 + 12 * len(chunks)
    toc, body = b"", b""
    for size, c in chunks:
        toc += struct.pack("<3I", 0xFFFD0002, size, pos + len(body))
        body += c
    return b"Xcur" + struct.pack("<3I", 16, 0x10000, len(chunks)) + toc + body


NAMES = {
    "left_ptr": ["default", "arrow", "top_left_arrow", "left_arrow"],
    "hand2": ["hand1", "hand", "pointer", "pointing_hand", "e29285e634086352946a0e7090d73106",
              "9d800788f1b08800ae810202380a0822"],
    "xterm": ["text", "ibeam"],
    "watch": ["wait", "progress", "left_ptr_watch", "half-busy",
              "3ecb610c1bf2410f44200f48c40d3599", "08e8e1c95fe2fc01f976f1e063a24ccd"],
}


def main():
    theme = sys.argv[1]
    cur = os.path.join(theme, "cursors")
    os.makedirs(cur, exist_ok=True)
    files = {
        "left_ptr": xcursor([shape(ARROW, sparkle=(21, 7, 5))], (2, 1)),
        "hand2": xcursor([shape(HAND, sparkle=(5, 6, 3.5))], (10.5, 2)),
        "xterm": xcursor([shape(IBEAM, rim=0.45)], (16, 16)),
        "watch": xcursor([ring_frame(i, 12) for i in range(12)], (16, 16), delay=60),
    }
    for name, data in files.items():
        with open(os.path.join(cur, name), "wb") as f:
            f.write(data)
        for alias in NAMES[name]:
            link = os.path.join(cur, alias)
            if os.path.lexists(link):
                os.remove(link)
            os.symlink(name, link)
    with open(os.path.join(theme, "index.theme"), "w") as f:
        f.write("[Icon Theme]\nName=Annie Rainbow\nComment=Rainbow mouse cursors for Annie OS\nInherits=Adwaita\n")
    # a preview for the README
    prev = Image.new("RGBA", (4 * 80, 80), (245, 238, 255, 255))
    for i, im in enumerate([shape(ARROW, sparkle=(21, 7, 5)), shape(HAND, sparkle=(5, 6, 3.5)), shape(IBEAM, rim=0.45), ring_frame(0, 12)]):
        prev.alpha_composite(im.resize((64, 64), Image.LANCZOS), (i * 80 + 8, 8))
    prev.save(os.path.join(theme, "preview.png"))


if __name__ == "__main__":
    main()
