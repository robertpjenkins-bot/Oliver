#!/usr/bin/env python3
"""OS Maker desktop.

Draws the desktop that was designed in OS Maker 2.0: the wallpaper, and every
search bar, clock, widget and app icon at the place it was put. The design was
made on a 1280x720 canvas; positions are scaled to whatever screen this runs on.
"""

import json
import math
import os
import subprocess
import sys
import time

import cairo
import gi

gi.require_version("Gtk", "3.0")
gi.require_version("Gdk", "3.0")
gi.require_version("Pango", "1.0")
from gi.repository import Gdk, GdkPixbuf, Gio, GLib, Gtk, Pango  # noqa: E402

LAYOUT = os.environ.get("OSMAKER_LAYOUT", "/etc/osmaker/layout.json")
NOTES_DIR = os.path.join(GLib.get_user_data_dir(), "osmaker", "notes")


def rgba(hex_color, alpha=1.0):
    h = hex_color.lstrip("#")
    r, g, b = (int(h[i:i + 2], 16) for i in (0, 2, 4))
    return r / 255, g / 255, b / 255, alpha


def css_rgba(hex_color, alpha=1.0):
    r, g, b, a = rgba(hex_color, alpha)
    return "rgba(%d,%d,%d,%.3f)" % (r * 255, g * 255, b * 255, a)


# --- launching apps --------------------------------------------------------

_all_apps = None


def all_apps():
    global _all_apps
    if _all_apps is None:
        _all_apps = [a for a in Gio.AppInfo.get_all() if a.should_show()]
    return _all_apps


def find_app(app):
    """Finds the installed application for a catalog entry.

    Desktop-file names change between releases, so fall back to matching the
    program the entry runs.
    """
    desktop = app.get("desktop")
    if desktop:
        try:
            info = Gio.DesktopAppInfo.new(desktop)
        except TypeError:
            info = None
        if info is not None:
            return info
    exe = app.get("exec")
    if exe:
        for info in Gio.AppInfo.get_all():
            if os.path.basename(info.get_executable() or "") == exe:
                return info
    return None


def launch(app):
    info = find_app(app)
    try:
        if info is not None:
            info.launch([], None)
            return
        # Command-line programs have no launcher; open them in a terminal.
        subprocess.Popen(["xfce4-terminal", "--hold", "-e", app["exec"]])
    except Exception as e:  # never let one bad app take the desktop down
        print("osmaker-desktop: could not start %s: %s" % (app.get("name"), e), file=sys.stderr)


# --- widgets ---------------------------------------------------------------

def make_app_icon(item, s):
    app = item["appInfo"]
    button = Gtk.Button()
    button.set_relief(Gtk.ReliefStyle.NONE)
    button.set_tooltip_text(app.get("description") or app["name"])
    box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=int(4 * s))
    box.set_valign(Gtk.Align.CENTER)
    image = Gtk.Image.new_from_icon_name(app.get("icon") or "application-x-executable", Gtk.IconSize.DIALOG)
    image.set_pixel_size(max(16, int(min(item["w"], item["h"] - item["fontSize"] * 1.6) * 0.6 * s)))
    label = Gtk.Label(label=app["name"])
    label.set_ellipsize(Pango.EllipsizeMode.END)
    label.set_max_width_chars(14)
    label.get_style_context().add_class("osm-label")
    box.pack_start(image, False, False, 0)
    box.pack_start(label, False, False, 0)
    button.add(box)
    button.connect("clicked", lambda *_: launch(app))
    return button


def make_search(item, s):
    entry = Gtk.SearchEntry()
    entry.set_placeholder_text(item.get("text") or "Search apps and the web…")
    store = Gtk.ListStore(str, object)
    for info in sorted(all_apps(), key=lambda a: a.get_display_name().lower()):
        store.append([info.get_display_name(), info])
    completion = Gtk.EntryCompletion()
    completion.set_model(store)
    completion.set_text_column(0)
    completion.set_minimum_key_length(1)
    completion.set_match_func(lambda c, key, it, _: key.lower() in store[it][0].lower(), None)

    def picked(_c, model, it):
        model[it][1].launch([], None)
        entry.set_text("")
        return True

    completion.connect("match-selected", picked)
    entry.set_completion(completion)

    def activate(_e):
        q = entry.get_text().strip()
        if not q:
            return
        for name, info in store:
            if q.lower() in name.lower():
                info.launch([], None)
                break
        else:
            uri = "https://duckduckgo.com/?q=" + GLib.uri_escape_string(q, None, False)
            Gio.AppInfo.launch_default_for_uri(uri, None)
        entry.set_text("")

    entry.connect("activate", activate)
    return entry


def make_clock(item, s):
    box = Gtk.Box(orientation=Gtk.Orientation.VERTICAL)
    box.set_valign(Gtk.Align.CENTER)
    t = Gtk.Label()
    t.get_style_context().add_class("osm-big")
    d = Gtk.Label()
    d.get_style_context().add_class("osm-small")
    box.pack_start(t, False, False, 0)
    box.pack_start(d, False, False, 0)
    fmt = "%I:%M %p" if item.get("format") == "12h" else "%H:%M"

    def tick():
        now = time.localtime()
        t.set_text(time.strftime(fmt, now).lstrip("0") if item.get("format") == "12h" else time.strftime(fmt, now))
        d.set_text(time.strftime("%A %d %B", now))
        return True

    tick()
    GLib.timeout_add_seconds(1, tick)
    return box


def make_calendar(item, s):
    cal = Gtk.Calendar()
    cal.set_display_options(Gtk.CalendarDisplayOptions.SHOW_HEADING | Gtk.CalendarDisplayOptions.SHOW_DAY_NAMES)
    return cal


class _CPU:
    def __init__(self):
        self.prev = self.read()

    @staticmethod
    def read():
        with open("/proc/stat") as f:
            parts = [int(x) for x in f.readline().split()[1:]]
        idle = parts[3] + (parts[4] if len(parts) > 4 else 0)
        return idle, sum(parts)

    def usage(self):
        idle, total = self.read()
        pi, pt = self.prev
        self.prev = (idle, total)
        dt = total - pt
        return 0.0 if dt <= 0 else 1.0 - (idle - pi) / dt


def mem_usage():
    info = {}
    with open("/proc/meminfo") as f:
        for line in f:
            k, v = line.split(":", 1)
            info[k] = int(v.split()[0])
    total = info.get("MemTotal", 1)
    return 1.0 - info.get("MemAvailable", total) / total


def make_sysmon(item, s):
    grid = Gtk.Grid(column_spacing=int(8 * s), row_spacing=int(6 * s))
    grid.set_valign(Gtk.Align.CENTER)
    cpu = _CPU()
    bars = {}
    for row, name in enumerate(("CPU", "Memory")):
        label = Gtk.Label(label=name, xalign=0)
        bar = Gtk.LevelBar()
        bar.set_hexpand(True)
        pct = Gtk.Label(xalign=1)
        grid.attach(label, 0, row, 1, 1)
        grid.attach(bar, 1, row, 1, 1)
        grid.attach(pct, 2, row, 1, 1)
        bars[name] = (bar, pct)

    def tick():
        for name, value in (("CPU", cpu.usage()), ("Memory", mem_usage())):
            bar, pct = bars[name]
            bar.set_value(value)
            pct.set_text("%d%%" % round(value * 100))
        return True

    tick()
    GLib.timeout_add_seconds(2, tick)
    return grid


def make_note(item, s):
    os.makedirs(NOTES_DIR, exist_ok=True)
    path = os.path.join(NOTES_DIR, item["id"] + ".txt")
    view = Gtk.TextView()
    view.set_wrap_mode(Gtk.WrapMode.WORD_CHAR)
    buf = view.get_buffer()
    try:
        with open(path) as f:
            buf.set_text(f.read())
    except OSError:
        buf.set_text(item.get("text") or "")
    pending = [0]

    def save():
        pending[0] = 0
        start, end = buf.get_bounds()
        with open(path, "w") as f:
            f.write(buf.get_text(start, end, False))
        return False

    def changed(_b):
        if pending[0]:
            GLib.source_remove(pending[0])
        pending[0] = GLib.timeout_add(800, save)

    buf.connect("changed", changed)
    scroll = Gtk.ScrolledWindow()
    scroll.set_policy(Gtk.PolicyType.NEVER, Gtk.PolicyType.AUTOMATIC)
    scroll.add(view)
    return scroll


def make_text(item, s):
    label = Gtk.Label(label=item.get("text") or "")
    label.set_line_wrap(True)
    label.set_justify(Gtk.Justification.CENTER)
    return label


MAKERS = {
    "app": make_app_icon,
    "search": make_search,
    "clock": make_clock,
    "calendar": make_calendar,
    "sysmon": make_sysmon,
    "note": make_note,
    "text": make_text,
}


# --- the desktop window ----------------------------------------------------

class Desktop(Gtk.Window):
    def __init__(self, layout, monitor):
        super().__init__(title="Desktop")
        self.layout = layout
        geo = monitor.get_geometry()
        self.geo = geo
        self.set_type_hint(Gdk.WindowTypeHint.DESKTOP)
        self.set_decorated(False)
        self.set_skip_taskbar_hint(True)
        self.set_skip_pager_hint(True)
        self.set_keep_below(True)
        self.stick()
        self.move(geo.x, geo.y)
        self.set_default_size(geo.width, geo.height)
        self.set_size_request(geo.width, geo.height)

        cw, ch = layout["canvas"]["w"], layout["canvas"]["h"]
        self.sx, self.sy = geo.width / cw, geo.height / ch
        self.s = min(self.sx, self.sy)

        self.wallpaper = None
        bg = layout["background"]
        if bg["kind"] == "image" and bg.get("image"):
            try:
                self.wallpaper = GdkPixbuf.Pixbuf.new_from_file(bg["image"])
            except GLib.Error as e:
                print("osmaker-desktop: wallpaper: %s" % e.message, file=sys.stderr)

        overlay = Gtk.Overlay()
        area = Gtk.DrawingArea()
        area.connect("draw", self.draw_background)
        overlay.add(area)
        fixed = Gtk.Fixed()
        overlay.add_overlay(fixed)
        self.add(overlay)

        css = []
        for n, item in enumerate(layout["items"]):
            maker = MAKERS.get(item["type"])
            if maker is None:
                continue
            try:
                widget = maker(item, self.s)
            except Exception as e:
                print("osmaker-desktop: %s %s: %s" % (item["type"], item["id"], e), file=sys.stderr)
                continue
            name = "osm%d" % n
            frame = Gtk.EventBox()
            frame.set_name(name)
            frame.add(widget)
            w, h = max(1, int(item["w"] * self.sx)), max(1, int(item["h"] * self.sy))
            frame.set_size_request(w, h)
            fixed.put(frame, int(item["x"] * self.sx), int(item["y"] * self.sy))
            css.append(self.item_css(name, item))

        provider = Gtk.CssProvider()
        provider.load_from_data("\n".join(css).encode())
        Gtk.StyleContext.add_provider_for_screen(
            Gdk.Screen.get_default(), provider, Gtk.STYLE_PROVIDER_PRIORITY_APPLICATION)

    def item_css(self, name, item):
        s = self.s
        fs = max(6, round(item["fontSize"] * s))
        bg_alpha = 0.0 if item["type"] == "app" else item["opacity"]
        pad = 0 if item["type"] in ("app", "search") else round(10 * s)
        rules = [
            "#%s { background-color: %s; border-radius: %dpx; padding: %dpx; }" % (
                name, css_rgba(item["color"], bg_alpha), round(item["radius"] * s), pad),
            "#%s, #%s * { color: %s; font-size: %dpx; }" % (name, name, item["textColor"], fs),
            "#%s .osm-big { font-size: %dpx; font-weight: bold; }" % (name, fs),
            "#%s .osm-small { font-size: %dpx; }" % (name, max(6, round(fs * 0.38))),
        ]
        if item["type"] == "app":
            rules.append("#%s button { padding: 0; border-radius: %dpx; }" % (name, round(item["radius"] * s)))
            rules.append("#%s button:hover { background: rgba(255,255,255,0.18); }" % name)
            rules.append("#%s .osm-label { text-shadow: 0 1px 3px rgba(0,0,0,0.8); }" % name)
        if item["type"] == "search":
            rules.append("#%s entry { background: %s; color: %s; border-radius: %dpx; border: none; "
                         "box-shadow: none; min-height: %dpx; padding: 0 %dpx; }" % (
                             name, css_rgba(item["color"], item["opacity"]), item["textColor"],
                             round(item["radius"] * s), int(item["h"] * self.sy), round(14 * s)))
            rules.append("#%s { background-color: transparent; }" % name)
        if item["type"] in ("note", "calendar"):
            rules.append("#%s textview, #%s textview text, #%s calendar { background-color: transparent; }" % (
                name, name, name))
        return "\n".join(rules)

    def draw_background(self, widget, cr):
        w, h = widget.get_allocated_width(), widget.get_allocated_height()
        bg = self.layout["background"]
        if self.wallpaper is not None:
            pw, ph = self.wallpaper.get_width(), self.wallpaper.get_height()
            scale = max(w / pw, h / ph)  # cover the screen, like the designer does
            sw, sh = int(pw * scale), int(ph * scale)
            scaled = self.wallpaper.scale_simple(sw, sh, GdkPixbuf.InterpType.BILINEAR)
            Gdk.cairo_set_source_pixbuf(cr, scaled, (w - sw) / 2, (h - sh) / 2)
            cr.paint()
            return False
        if bg["kind"] == "gradient":
            # Same geometry as CSS linear-gradient(<angle>deg, ...).
            a = math.radians(bg.get("angle", 180))
            dx, dy = math.sin(a), -math.cos(a)
            half = (abs(w * dx) + abs(h * dy)) / 2
            cx, cy = w / 2, h / 2
            grad = cairo.LinearGradient(cx - dx * half, cy - dy * half, cx + dx * half, cy + dy * half)
            grad.add_color_stop_rgb(0, *rgba(bg["color1"])[:3])
            grad.add_color_stop_rgb(1, *rgba(bg["color2"])[:3])
            cr.set_source(grad)
        else:
            cr.set_source_rgb(*rgba(bg["color1"])[:3])
        cr.paint()
        return False


def main():
    with open(LAYOUT) as f:
        layout = json.load(f)

    # The designed desktop replaces Xfce's own; make sure it is not running.
    subprocess.run(["xfdesktop", "--quit"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)

    settings = Gtk.Settings.get_default()
    settings.set_property("gtk-application-prefer-dark-theme", layout.get("theme") == "dark")

    display = Gdk.Display.get_default()
    monitor = display.get_primary_monitor() or display.get_monitor(0)
    win = Desktop(layout, monitor)
    win.connect("destroy", Gtk.main_quit)
    win.show_all()

    # Rebuild at the new size when the screen resolution changes, e.g. when a
    # virtual machine window is resized.
    def resized(*_):
        os.execv(sys.executable, [sys.executable] + sys.argv)

    Gdk.Screen.get_default().connect("size-changed", resized)
    Gtk.main()


if __name__ == "__main__":
    main()
