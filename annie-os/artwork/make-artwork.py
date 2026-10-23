#!/usr/bin/env python3
"""Draw the Annie OS artwork as SVG: the magic wand wallpaper, the boot
splash and the logo. Run render.sh afterwards to turn them into PNGs."""

import math
import os
import random

HERE = os.path.dirname(os.path.abspath(__file__))


def sparkle(cx, cy, r, fill, opacity=1.0, rot=0.0):
    """A four-pointed star: long thin points, the classic 'magic' glint."""
    pts = []
    for i in range(8):
        a = rot + i * math.pi / 4
        rr = r if i % 2 == 0 else r * 0.18
        pts.append(f"{cx + rr * math.sin(a):.1f},{cy - rr * math.cos(a):.1f}")
    return f'<polygon points="{" ".join(pts)}" fill="{fill}" opacity="{opacity:.2f}"/>'


def starfield(rng, w, h, n, avoid=None):
    out = []
    for _ in range(n):
        x, y = rng.uniform(0, w), rng.uniform(0, h)
        r = rng.choice([0.6, 0.8, 1.0, 1.2, 1.6, 2.2])
        o = rng.uniform(0.25, 0.9)
        out.append(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="{r}" fill="#fff" opacity="{o:.2f}"/>')
    return "\n".join(out)


def wand(x0, y0, x1, y1, width):
    """A wand from handle (x0,y0) to tip (x1,y1), drawn along the x axis then rotated."""
    length = math.hypot(x1 - x0, y1 - y0)
    angle = math.degrees(math.atan2(y1 - y0, x1 - x0))
    w = width
    handle = length * 0.30
    parts = [
        f'<g transform="translate({x0:.1f},{y0:.1f}) rotate({angle:.2f})">',
        # shadowy body tapering towards the tip
        f'<path d="M0,{-w/2} L{length},{-w*0.16} L{length},{w*0.16} L0,{w/2} Z" fill="url(#shaft)"/>',
        # handle wrap
        f'<path d="M0,{-w/2*1.12} L{handle},{-w/2*0.98} L{handle},{w/2*0.98} L0,{w/2*1.12} Z" fill="url(#handle)"/>',
    ]
    for t in (0.04, 0.12, 0.20, 0.30):
        x = handle * t / 0.30
        hw = w / 2 * (1.18 - 0.2 * t)
        parts.append(f'<rect x="{x-5:.1f}" y="{-hw:.1f}" width="10" height="{2*hw:.1f}" rx="4" fill="url(#gold)"/>')
    # rounded pommel
    parts.append(f'<circle cx="-4" cy="0" r="{w*0.62:.1f}" fill="url(#gold)"/>')
    # highlight along the shaft
    parts.append(f'<path d="M{handle},{-w*0.30} L{length*0.97},{-w*0.10}" stroke="#fff" stroke-opacity="0.25" stroke-width="2" stroke-linecap="round"/>')
    parts.append("</g>")
    return "\n".join(parts)


DEFS = """
<radialGradient id="sky" cx="62%" cy="38%" r="85%">
  <stop offset="0" stop-color="#4a1f7a"/>
  <stop offset="0.45" stop-color="#26104a"/>
  <stop offset="1" stop-color="#0a0418"/>
</radialGradient>
<radialGradient id="nebula" cx="50%" cy="50%" r="50%">
  <stop offset="0" stop-color="#d16ba5" stop-opacity="0.35"/>
  <stop offset="1" stop-color="#d16ba5" stop-opacity="0"/>
</radialGradient>
<radialGradient id="nebula2" cx="50%" cy="50%" r="50%">
  <stop offset="0" stop-color="#5ab0ff" stop-opacity="0.22"/>
  <stop offset="1" stop-color="#5ab0ff" stop-opacity="0"/>
</radialGradient>
<radialGradient id="glow" cx="50%" cy="50%" r="50%">
  <stop offset="0" stop-color="#fffbe8" stop-opacity="1"/>
  <stop offset="0.12" stop-color="#ffe9a8" stop-opacity="0.9"/>
  <stop offset="0.4" stop-color="#ffb3e6" stop-opacity="0.35"/>
  <stop offset="1" stop-color="#b57bff" stop-opacity="0"/>
</radialGradient>
<linearGradient id="shaft" x1="0" y1="0" x2="0" y2="1">
  <stop offset="0" stop-color="#5a3a2a"/>
  <stop offset="0.5" stop-color="#2a160e"/>
  <stop offset="1" stop-color="#120804"/>
</linearGradient>
<linearGradient id="handle" x1="0" y1="0" x2="0" y2="1">
  <stop offset="0" stop-color="#3b2416"/>
  <stop offset="1" stop-color="#0d0603"/>
</linearGradient>
<linearGradient id="gold" x1="0" y1="0" x2="0" y2="1">
  <stop offset="0" stop-color="#fff1b0"/>
  <stop offset="0.5" stop-color="#e2a93b"/>
  <stop offset="1" stop-color="#8a5a12"/>
</linearGradient>
<linearGradient id="trail" x1="0" y1="1" x2="1" y2="0">
  <stop offset="0" stop-color="#ffe9a8" stop-opacity="0.85"/>
  <stop offset="0.5" stop-color="#ff9ad5" stop-opacity="0.45"/>
  <stop offset="1" stop-color="#8fd3ff" stop-opacity="0"/>
</linearGradient>
<filter id="soft" x="-50%" y="-50%" width="200%" height="200%"><feGaussianBlur stdDeviation="6"/></filter>
<filter id="blur" x="-50%" y="-50%" width="200%" height="200%"><feGaussianBlur stdDeviation="2"/></filter>
"""


def trail(rng, tip, scale):
    """A swirl of sparkles flowing up and to the right from the wand tip."""
    tx, ty = tip
    out = []
    # the ribbon of light
    c1 = (tx + 260 * scale, ty - 330 * scale)
    c2 = (tx + 560 * scale, ty - 40 * scale)
    end = (tx + 720 * scale, ty - 300 * scale)
    d = f"M{tx},{ty} C{c1[0]:.0f},{c1[1]:.0f} {c2[0]:.0f},{c2[1]:.0f} {end[0]:.0f},{end[1]:.0f}"
    out.append(f'<path d="{d}" stroke="url(#trail)" stroke-width="{26*scale:.1f}" fill="none" stroke-linecap="round" filter="url(#soft)"/>')
    out.append(f'<path d="{d}" stroke="url(#trail)" stroke-width="{3*scale:.1f}" fill="none" stroke-linecap="round"/>')

    def bez(t):
        p = [(tx, ty), c1, c2, end]
        mt = 1 - t
        x = mt**3 * p[0][0] + 3 * mt**2 * t * p[1][0] + 3 * mt * t**2 * p[2][0] + t**3 * p[3][0]
        y = mt**3 * p[0][1] + 3 * mt**2 * t * p[1][1] + 3 * mt * t**2 * p[2][1] + t**3 * p[3][1]
        return x, y

    colors = ["#fff6d0", "#ffe08a", "#ffb3e6", "#c9a6ff", "#9fdcff"]
    for i in range(70):
        t = rng.random() ** 0.8
        x, y = bez(t)
        spread = (20 + 120 * t) * scale
        x += rng.gauss(0, spread)
        y += rng.gauss(0, spread)
        size = (26 * (1 - t) + 6) * scale * rng.uniform(0.4, 1.0)
        col = colors[min(len(colors) - 1, int(t * len(colors) + rng.uniform(-0.6, 0.6)))]
        op = 0.95 - 0.6 * t
        if rng.random() < 0.55:
            out.append(sparkle(x, y, size, col, op, rng.uniform(-0.3, 0.3)))
        else:
            out.append(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="{size*0.18:.1f}" fill="{col}" opacity="{op:.2f}"/>')
    return "\n".join(out)


def wallpaper(w=1920, h=1080, label=True):
    rng = random.Random(7)
    s = h / 1080
    tip = (w * 0.555, h * 0.40)
    base = (w * 0.30, h * 0.90)
    body = [
        f'<rect width="{w}" height="{h}" fill="url(#sky)"/>',
        f'<ellipse cx="{w*0.72}" cy="{h*0.30}" rx="{w*0.32}" ry="{h*0.30}" fill="url(#nebula)"/>',
        f'<ellipse cx="{w*0.25}" cy="{h*0.25}" rx="{w*0.28}" ry="{h*0.25}" fill="url(#nebula2)"/>',
        starfield(rng, w, h, 420),
    ]
    for _ in range(14):  # a few bright twinkling stars in the sky
        body.append(sparkle(rng.uniform(0, w), rng.uniform(0, h * 0.8), rng.uniform(5, 11) * s, "#ffffff", rng.uniform(0.5, 0.9)))
    body.append(f'<circle cx="{tip[0]}" cy="{tip[1]}" r="{380*s:.0f}" fill="url(#glow)" opacity="0.55"/>')
    body.append(trail(rng, tip, s))
    body.append(wand(base[0], base[1], tip[0], tip[1], 30 * s))
    body.append(f'<circle cx="{tip[0]}" cy="{tip[1]}" r="{120*s:.0f}" fill="url(#glow)"/>')
    body.append(sparkle(tip[0], tip[1], 95 * s, "#fffbe8", 1.0))
    body.append(sparkle(tip[0], tip[1], 55 * s, "#ffffff", 1.0, math.pi / 4))
    if label:
        body.append(f'<text x="{w-60*s:.0f}" y="{h-50*s:.0f}" text-anchor="end" font-family="DejaVu Sans, sans-serif" font-size="{34*s:.0f}" letter-spacing="{8*s:.0f}" fill="#e6d6ff" opacity="0.55">ANNIE OS</text>')
    return f'<svg xmlns="http://www.w3.org/2000/svg" width="{w}" height="{h}" viewBox="0 0 {w} {h}"><defs>{DEFS}</defs>\n' + "\n".join(body) + "\n</svg>\n"


def logo(size=256):
    rng = random.Random(3)
    c = size / 2
    body = [
        f'<circle cx="{c}" cy="{c}" r="{c}" fill="url(#sky)"/>',
        starfield(rng, size, size, 30).replace('r="2.2"', 'r="1.2"'),
        wand(size * 0.20, size * 0.82, size * 0.62, size * 0.38, size * 0.07),
        f'<circle cx="{size*0.62}" cy="{size*0.38}" r="{size*0.30}" fill="url(#glow)"/>',
        sparkle(size * 0.62, size * 0.38, size * 0.24, "#fffbe8"),
        sparkle(size * 0.80, size * 0.20, size * 0.07, "#ffb3e6"),
        sparkle(size * 0.84, size * 0.48, size * 0.05, "#9fdcff"),
    ]
    clip = f'<clipPath id="disc"><circle cx="{c}" cy="{c}" r="{c}"/></clipPath>'
    return (f'<svg xmlns="http://www.w3.org/2000/svg" width="{size}" height="{size}" viewBox="0 0 {size} {size}"><defs>{DEFS}{clip}</defs>\n'
            f'<g clip-path="url(#disc)">' + "\n".join(body) + "</g>\n</svg>\n")


if __name__ == "__main__":
    for name, svg in [("wallpaper.svg", wallpaper()), ("splash.svg", wallpaper(640, 480, label=False)), ("logo.svg", logo())]:
        with open(os.path.join(HERE, name), "w") as f:
            f.write(svg)
        print("wrote", name)
