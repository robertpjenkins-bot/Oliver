#!/usr/bin/env python3
"""Draws the Annie OS unicorn wallpaper.

Usage: make-wallpaper.py OUTPUT.png [WIDTH HEIGHT]
"""
import math
import random
import sys

from PIL import Image, ImageDraw, ImageFilter

RAINBOW = [(255, 92, 120), (255, 160, 80), (255, 222, 90),
           (110, 220, 140), (90, 170, 255), (170, 120, 255)]


def lerp(a, b, t):
    return tuple(int(a[i] + (b[i] - a[i]) * t) for i in range(3))


def sky(w, h):
    img = Image.new("RGB", (w, h))
    d = ImageDraw.Draw(img)
    top, mid, bottom = (120, 90, 200), (240, 150, 210), (255, 214, 230)
    for y in range(h):
        t = y / h
        c = lerp(top, mid, t / 0.6) if t < 0.6 else lerp(mid, bottom, (t - 0.6) / 0.4)
        d.line([(0, y), (w, y)], fill=c)
    return img


def stars(img, rng):
    d = ImageDraw.Draw(img)
    w, h = img.size
    for _ in range(140):
        x, y = rng.randrange(w), rng.randrange(int(h * 0.55))
        r = rng.choice([1, 1, 2, 2, 3])
        d.ellipse([x - r, y - r, x + r, y + r], fill=(255, 250, 235))
    for _ in range(12):  # four-point sparkles
        x, y = rng.randrange(w), rng.randrange(int(h * 0.5))
        s = rng.randrange(8, 18)
        d.polygon([(x, y - s), (x + s / 4, y), (x, y + s), (x - s / 4, y)], fill="white")
        d.polygon([(x - s, y), (x, y - s / 4), (x + s, y), (x, y + s / 4)], fill="white")


def rainbow(img):
    w, h = img.size
    layer = Image.new("RGBA", img.size, (0, 0, 0, 0))
    d = ImageDraw.Draw(layer)
    cx, cy = int(w * 0.30), int(h * 0.95)
    band = int(h * 0.035)
    outer = int(h * 0.80)
    for i, c in enumerate(RAINBOW):
        r = outer - i * band
        d.ellipse([cx - r, cy - r, cx + r, cy + r], fill=c + (170,))
    r = outer - len(RAINBOW) * band
    d.ellipse([cx - r, cy - r, cx + r, cy + r], fill=(0, 0, 0, 0))
    layer = layer.filter(ImageFilter.GaussianBlur(3))
    img.paste(layer, (0, 0), layer)


def clouds(img, rng):
    w, h = img.size
    layer = Image.new("RGBA", img.size, (0, 0, 0, 0))
    d = ImageDraw.Draw(layer)
    for _ in range(7):
        x, y = rng.randrange(w), rng.randrange(int(h * 0.6), h)
        for _ in range(6):
            rx, ry = rng.randrange(60, 160), rng.randrange(40, 90)
            ox, oy = rng.randrange(-120, 120), rng.randrange(-30, 30)
            d.ellipse([x + ox - rx, y + oy - ry, x + ox + rx, y + oy + ry], fill=(255, 255, 255, 200))
    layer = layer.filter(ImageFilter.GaussianBlur(6))
    img.paste(layer, (0, 0), layer)


def unicorn(img):
    """A side-on unicorn facing left, scaled to the image height."""
    w, h = img.size
    s = h / 1080
    ox, oy = w * 0.58, h * 0.30

    def P(x, y):
        return (ox + x * s, oy + y * s)

    layer = Image.new("RGBA", img.size, (0, 0, 0, 0))
    d = ImageDraw.Draw(layer)
    body = (255, 252, 255)
    shade = (232, 222, 245)
    outline = (150, 110, 190)

    # tail: rainbow strands sweeping back
    for i, c in enumerate(RAINBOW):
        pts = [P(560, 260 + i * 6)]
        for k in range(1, 13):
            t = k / 12
            pts.append(P(560 + 150 * t + 30 * math.sin(t * 5 + i), 260 + i * 14 + 260 * t))
        d.line(pts, fill=c, width=int(22 * s), joint="curve")

    # far legs (shaded)
    for x in (230, 470):
        d.rounded_rectangle([P(x, 380), P(x + 46, 640)], radius=int(20 * s), fill=shade, outline=outline, width=int(4 * s))
    # body
    d.ellipse([P(180, 180), P(600, 450)], fill=body, outline=outline, width=int(5 * s))
    # near legs + hooves
    for x in (180, 300, 420, 520):
        d.rounded_rectangle([P(x, 380), P(x + 50, 660)], radius=int(22 * s), fill=body, outline=outline, width=int(4 * s))
        d.rounded_rectangle([P(x - 4, 630), P(x + 54, 670)], radius=int(10 * s), fill=(255, 200, 90), outline=outline, width=int(3 * s))
    # neck
    d.polygon([P(200, 330), P(110, 90), P(230, 40), P(330, 260)], fill=body, outline=outline)
    d.ellipse([P(160, 200), P(340, 380)], fill=body)
    # head
    d.ellipse([P(30, 20), P(240, 170)], fill=body, outline=outline, width=int(5 * s))
    d.ellipse([P(-30, 80), P(110, 190)], fill=body, outline=outline, width=int(5 * s))
    d.ellipse([P(0, 125), P(18, 143)], fill=outline)  # nostril
    # ear
    d.polygon([P(170, 40), P(205, -40), P(225, 50)], fill=body, outline=outline)
    # horn: gold spiral
    d.polygon([P(125, 30), P(70, -150), P(165, 15)], fill=(255, 215, 110), outline=(210, 150, 40))
    for k in range(1, 5):
        t = k / 5
        a, b = P(125 - 55 * t, 30 - 180 * t), P(165 - 95 * t, 15 - 165 * t)
        d.line([a, b], fill=(210, 150, 40), width=int(4 * s))
    # eye + lashes + blush
    d.ellipse([P(100, 70), P(140, 112)], fill=(60, 40, 90))
    d.ellipse([P(110, 76), P(124, 90)], fill="white")
    for dx in (0, 14, 28):
        d.line([P(104 + dx, 72), P(96 + dx * 1.2, 56)], fill=(60, 40, 90), width=int(4 * s))
    d.ellipse([P(130, 120), P(175, 145)], fill=(255, 170, 200))
    # mane: rainbow locks down the neck
    for i, c in enumerate(RAINBOW):
        x0, y0 = 200 + i * 22, 10 + i * 34
        d.ellipse([P(x0, y0), P(x0 + 110, y0 + 90)], fill=c)
        d.ellipse([P(x0 + 30, y0 + 40), P(x0 + 120, y0 + 120)], fill=c)

    shadow = Image.new("RGBA", img.size, (0, 0, 0, 0))
    ImageDraw.Draw(shadow).ellipse([P(140, 645), P(640, 700)], fill=(120, 60, 140, 90))
    shadow = shadow.filter(ImageFilter.GaussianBlur(10))
    img.paste(shadow, (0, 0), shadow)
    img.paste(layer, (0, 0), layer)


def title(img):
    d = ImageDraw.Draw(img)
    w, h = img.size
    try:
        from PIL import ImageFont
        font = ImageFont.truetype("DejaVuSans-Bold.ttf", int(h * 0.06))
    except OSError:
        return
    text = "Annie OS"
    x, y = int(w * 0.06), int(h * 0.10)
    d.text((x + 4, y + 4), text, font=font, fill=(110, 60, 150))
    d.text((x, y), text, font=font, fill=(255, 255, 255))


def main():
    out = sys.argv[1]
    w = int(sys.argv[2]) if len(sys.argv) > 2 else 1920
    h = int(sys.argv[3]) if len(sys.argv) > 3 else 1080
    rng = random.Random(7)
    img = sky(w, h)
    stars(img, rng)
    rainbow(img)
    clouds(img, rng)
    unicorn(img)
    title(img)
    img.save(out, optimize=True)


if __name__ == "__main__":
    main()
