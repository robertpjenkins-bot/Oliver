#!/usr/bin/env python3
"""Draw the Etros artwork: theme images, boot logo, splash video frames and
the Ollie OS wallpaper. Everything is generated so the repository holds no
large binaries and the look can be changed in one place.

    make-art.py OUTDIR

writes OUTDIR/theme/*.png, OUTDIR/boot-logo.png, OUTDIR/splash-16x9/*.png,
OUTDIR/splash-4x3/*.png and OUTDIR/ollie/bobby.png. Needs Pillow.
"""
import os
import random
import sys

from PIL import Image, ImageDraw, ImageFilter, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
FONT = os.path.join(HERE, "..", "layer", "usr", "share", "emulationstation",
                    "themes", "etros-metroid", "fonts", "PressStart2P-Regular.ttf")

# Super Metroid's palette: deep space, the orange/yellow of the title and
# HUD, the pink and blue of the pause-screen map.
SPACE = (2, 4, 20)
NAVY = (8, 20, 64)
ORANGE = (248, 120, 0)
YELLOW = (248, 216, 64)
RED = (200, 24, 0)
PINK = (232, 72, 168)
BLUE = (40, 112, 232)
CYAN = (88, 216, 248)
WHITE = (240, 240, 248)

FPS = 30


def font(size):
    return ImageFont.truetype(FONT, size)


def starfield(w, h, seed=7, density=1.0, shift=0.0):
    """A dark sky with three layers of stars; shift scrolls them sideways."""
    img = Image.new("RGB", (w, h), SPACE)
    d = ImageDraw.Draw(img)
    # a faint blue band, like the planet glow behind the title screen
    for y in range(h):
        t = max(0.0, 1 - abs(y - h * 0.62) / (h * 0.5))
        c = tuple(int(SPACE[i] + (NAVY[i] - SPACE[i]) * t * 0.8) for i in range(3))
        d.line([(0, y), (w, y)], fill=c)
    rng = random.Random(seed)
    for layer, (count, size, bright, speed) in enumerate(
            [(260, 1, 120, 0.2), (120, 2, 190, 0.5), (40, 3, 255, 1.0)]):
        for _ in range(int(count * density * w * h / (1920 * 1080))):
            x = int((rng.uniform(0, w) - shift * speed * w) % w)
            y = int(rng.uniform(0, h))
            b = rng.randint(bright // 2, bright)
            tint = rng.choice([(b, b, b), (b, b, min(255, b + 40)), (min(255, b + 30), b, b)])
            d.rectangle([x, y, x + size - 1, y + size - 1], fill=tint)
    return img


def title_text(text, size, outline=None):
    """Chunky title lettering: a yellow-to-red gradient with a dark outline
    and a blue drop shadow, after the Super Metroid logo."""
    f = font(size)
    outline = outline or max(2, size // 12)
    shadow = max(3, size // 8)
    l, t, r, b = f.getbbox(text)
    w, h = r - l + outline * 2 + shadow, b - t + outline * 2 + shadow
    pad = outline
    mask = Image.new("L", (w, h), 0)
    ImageDraw.Draw(mask).text((pad - l, pad - t), text, font=f, fill=255)

    grad = Image.new("RGB", (w, h))
    gd = ImageDraw.Draw(grad)
    stops = [(0.0, YELLOW), (0.45, ORANGE), (1.0, RED)]
    for y in range(h):
        v = y / max(1, h - 1)
        for (p0, c0), (p1, c1) in zip(stops, stops[1:]):
            if p0 <= v <= p1:
                k = (v - p0) / (p1 - p0)
                gd.line([(0, y), (w, y)], fill=tuple(int(c0[i] + (c1[i] - c0[i]) * k) for i in range(3)))
                break

    out = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    edge = mask.filter(ImageFilter.MaxFilter(outline * 2 + 1))
    shadow_mask = Image.new("L", (w, h), 0)
    shadow_mask.paste(edge, (shadow, shadow))
    out.paste((16, 40, 136, 255), (0, 0), shadow_mask)
    out.paste((10, 6, 24, 255), (0, 0), edge)
    out.paste(grad, (0, 0), mask)
    # a highlight stripe across the upper third, for the metallic look
    hi = Image.new("L", (w, h), 0)
    ImageDraw.Draw(hi).rectangle([0, int(h * 0.2), w, int(h * 0.28)], fill=110)
    hi = Image.composite(hi, Image.new("L", (w, h), 0), mask)
    out.paste((255, 255, 255, 255), (0, 0), hi)
    return out


def plain_text(text, size, color, shadow=True):
    f = font(size)
    l, t, r, b = f.getbbox(text)
    s = max(1, size // 8) if shadow else 0
    img = Image.new("RGBA", (r - l + s, b - t + s), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    if shadow:
        d.text((s - l, s - t), text, font=f, fill=(0, 0, 0, 255))
    d.text((-l, -t), text, font=f, fill=color + (255,))
    return img


def energy_tank(size, full=True):
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    b = max(1, size // 8)
    d.rectangle([0, 0, size - 1, size - 1], fill=(10, 6, 24, 255))
    d.rectangle([b, b, size - 1 - b, size - 1 - b], fill=(PINK if full else NAVY) + (255,))
    if full:
        d.rectangle([b, b, size - 1 - b, b + b], fill=(255, 190, 230, 255))
    return img


def hud_bar(w, h, tanks=6):
    """The energy readout that sits along the top of the screen in game."""
    img = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    d.rectangle([0, 0, w, h], fill=(0, 0, 0, 200))
    d.line([(0, h - 2), (w, h - 2)], fill=ORANGE + (255,), width=2)
    label = plain_text("ENERGY", int(h * 0.32), WHITE)
    img.alpha_composite(label, (int(h * 0.4), int(h * 0.52)))
    num = plain_text("99", int(h * 0.5), WHITE)
    img.alpha_composite(num, (int(h * 0.4) + label.width + int(h * 0.3), int(h * 0.36)))
    ts = int(h * 0.28)
    x0 = int(h * 0.4)
    for i in range(tanks):
        img.alpha_composite(energy_tank(ts), (x0 + i * (ts + ts // 3), int(h * 0.14)))
    return img


def minimap(cols=5, rows=3, cell=24):
    """The pink-and-blue explored-rooms map from the top right of the HUD."""
    w, h = cols * cell + 2, rows * cell + 2
    img = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    d = ImageDraw.Draw(img)
    rng = random.Random(3)
    for cy in range(rows):
        for cx in range(cols):
            x, y = cx * cell + 1, cy * cell + 1
            explored = rng.random() < 0.6 or (cx, cy) == (cols // 2, rows // 2)
            d.rectangle([x, y, x + cell - 1, y + cell - 1],
                        fill=(PINK if explored else NAVY) + (255,), outline=BLUE + (255,))
    cx, cy = cols // 2, rows // 2
    d.rectangle([cx * cell + cell // 3, cy * cell + cell // 3,
                 cx * cell + 2 * cell // 3, cy * cell + 2 * cell // 3], fill=WHITE + (255,))
    return img


def centered(canvas, item, y):
    canvas.alpha_composite(item, ((canvas.width - item.width) // 2, int(y)))


def scale_to_width(img, width):
    return img.resize((width, int(img.height * width / img.width)), Image.NEAREST)


def welcome_screen(w, h, shift=0.0, reveal=1.0, sub_chars=None):
    """WELCOME TO ETROS, as both the still splash and each video frame."""
    canvas = starfield(w, h, shift=shift).convert("RGBA")
    unit = min(w, h * 4 // 3)
    welcome = plain_text("WELCOME TO", max(8, unit // 40), CYAN)
    logo = scale_to_width(title_text("ETROS", 96), int(unit * 0.62))
    top = h * 0.30
    if reveal < 1.0:
        # the title wipes in from the left, like a power-on scan
        cut = int(logo.width * max(0.0, reveal))
        part = Image.new("RGBA", logo.size, (0, 0, 0, 0))
        part.paste(logo.crop((0, 0, cut, logo.height)), (0, 0))
        logo = part
    centered(canvas, welcome, top)
    centered(canvas, logo, top + welcome.height * 2.2)
    lines = ["THE LAST CARTRIDGE IS IN CAPTIVITY.", "THE GALAXY IS AT PEACE..."]
    size = max(6, unit // 70)
    y = top + welcome.height * 2.2 + logo.height + size * 4
    budget = sum(len(s) for s in lines) if sub_chars is None else sub_chars
    for line in lines:
        shown = line[:max(0, budget)]
        budget -= len(line)
        if shown:
            canvas.alpha_composite(plain_text(shown, size, WHITE),
                                   (int((w - font(size).getlength(line)) // 2), int(y)))
        y += size * 2.4
    return canvas.convert("RGB")


def splash_frames(outdir, w, h, seconds=6):
    os.makedirs(outdir, exist_ok=True)
    total = FPS * seconds
    sub_total = len("THE LAST CARTRIDGE IS IN CAPTIVITY.") + len("THE GALAXY IS AT PEACE...")
    for i in range(total):
        t = i / FPS
        reveal = min(1.0, max(0.0, (t - 0.4) / 1.2))
        chars = int(max(0.0, t - 2.0) * 22)
        frame = welcome_screen(w, h, shift=t * 0.01, reveal=reveal,
                               sub_chars=min(chars, sub_total))
        if t > seconds - 0.7:  # fade to black into the menu
            k = (seconds - t) / 0.7
            frame = Image.eval(frame, lambda v: int(v * k))
        frame.save(os.path.join(outdir, f"{i:04d}.png"))


def item_card(w, h, title, sub, accent):
    img = starfield(w, h, seed=len(title)).convert("RGBA")
    d = ImageDraw.Draw(img)
    d.rectangle([4, 4, w - 5, h - 5], outline=accent + (255,), width=4)
    t = plain_text(title, h // 9, accent)
    centered(img, t, h * 0.30)
    s = plain_text(sub, h // 24, WHITE)
    centered(img, s, h * 0.58)
    return img


def bobby_placeholder(w, h):
    img = starfield(w, h, seed=42, density=1.4).convert("RGBA")
    centered(img, scale_to_width(title_text("BOBBY", 96), int(w * 0.45)), h * 0.34)
    centered(img, plain_text("PUT BOBBY'S PICTURE IN", h // 60, WHITE), h * 0.62)
    centered(img, plain_text("/userdata/ollie/bobby.png", h // 60, CYAN), h * 0.66)
    return img.convert("RGB")


def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: make-art.py OUTDIR")
    out = sys.argv[1]
    theme = os.path.join(out, "theme")
    os.makedirs(theme, exist_ok=True)
    os.makedirs(os.path.join(out, "ollie"), exist_ok=True)

    starfield(1920, 1080).save(os.path.join(theme, "background.png"))
    title_text("ETROS", 96).save(os.path.join(theme, "logo.png"))
    hud_bar(1920, 96).save(os.path.join(theme, "hud.png"))
    minimap().save(os.path.join(theme, "minimap.png"))
    energy_tank(32).save(os.path.join(theme, "etank.png"))
    welcome_screen(1920, 1080).save(os.path.join(theme, "splash.png"))
    item_card(640, 480, "OLLIE OS", "BROWSE + DOWNLOAD ROMS", ORANGE).save(os.path.join(theme, "ollie.png"))
    item_card(640, 480, "SORT ROMS", "DOWNLOADS -> ROM FOLDERS", PINK).save(os.path.join(theme, "sort.png"))

    # The early boot logo is drawn by fbv, stretched to the whole screen, so
    # keep it 4:3 to look right on the R36S and acceptable on a TV.
    welcome_screen(640, 480).save(os.path.join(out, "boot-logo.png"))

    splash_frames(os.path.join(out, "splash-16x9"), 1280, 720)
    splash_frames(os.path.join(out, "splash-4x3"), 640, 480)

    bobby_placeholder(1920, 1080).save(os.path.join(out, "ollie", "bobby.png"))


if __name__ == "__main__":
    main()
