#!/usr/bin/env python3
"""The Games app's helper (ollie-games): finds every game, adds game files
dropped on the window, and starts a game.

  games.py list OUT.json      every game, for games.qml
  games.py add FILE...        copies game files (ROMs, disc images, Windows
                              games) into ~/Games/<console>/, finding out the
                              console from the file
  games.py launch KEY         starts a game (KEY from the list); a console's
                              emulator is installed the first time

Games: Steam's installed games, the apps menu's games (Minecraft too), and the
files in ~/Games. Only the person's own game files: nothing is downloaded but
the emulators, from the Arch Linux servers (PlayStation 2: from Flathub).
"""
import configparser
import glob
import json
import os
import re
import shutil
import subprocess
import sys
import zipfile

HOME = os.path.expanduser("~")
GAMES = os.path.join(HOME, "Games")

# console: (name, emoji, file endings, (Arch Linux package, RetroArch core) or a special way)
CONSOLES = {
    "nes": ("NES", "🍄", ["nes", "fds"], ("libretro-nestopia", "nestopia")),
    "snes": ("Super Nintendo", "🎮", ["sfc", "smc"], ("libretro-snes9x", "snes9x")),
    "gb": ("Game Boy", "🟩", ["gb", "gbc"], ("libretro-gambatte", "gambatte")),
    "gba": ("Game Boy Advance", "🟪", ["gba"], ("libretro-mgba", "mgba")),
    "sega": ("Sega", "🦔", ["md", "gen", "smd", "sms", "gg", "32x"], ("libretro-genesis-plus-gx", "genesis_plus_gx")),
    "n64": ("Nintendo 64", "⭐", ["n64", "z64", "v64"], ("libretro-mupen64plus-next", "mupen64plus_next")),
    "nds": ("Nintendo DS", "📱", ["nds"], ("libretro-melonds", "melonds")),
    "psp": ("PSP", "🕹️", ["cso"], ("libretro-ppsspp", "ppsspp")),
    "ps1": ("PlayStation", "💿", ["cue", "chd", "pbp", "m3u"], ("libretro-beetle-psx-hw", "mednafen_psx_hw")),
    "ps2": ("PlayStation 2", "📀", [], "pcsx2"),
    "gc": ("GameCube / Wii", "🟦", ["gcm", "rvz", "wbfs", "wad"], "dolphin"),
    "windows": ("Windows game", "🪟", ["exe", "msi"], "oliver"),
}
CORE_DIR = "/usr/lib/libretro"
STEAM_TOOLS = re.compile(r"Proton|Steam Linux Runtime|Steamworks Common|SteamVR", re.I)


def console_of(path):
    """The console a game file is for, from its ending or, for a disc image,
    from what is inside it; None when it cannot tell."""
    end = path.rsplit(".", 1)[-1].lower() if "." in os.path.basename(path) else ""
    for key, (_, _, ends, _) in CONSOLES.items():
        if end in ends:
            return key
    if end == "zip":
        try:
            with zipfile.ZipFile(path) as z:
                for name in z.namelist():
                    c = console_of(name)
                    if c and c not in ("ps1", "ps2", "gc", "psp", "windows"):
                        return c
        except (OSError, zipfile.BadZipFile):
            return None
        return None
    if end in ("iso", "bin", "img"):
        try:
            with open(path, "rb") as f:
                head = f.read(0x40)
                f.seek(0)
                start = f.read(16 * 1024 * 1024)
        except OSError:
            return None
        if head[0x1C:0x20] == b"\xc2\x33\x9f\x3d" or head[0x18:0x1C] == b"\x5d\x1c\x9e\xa3":
            return "gc"
        if b"PSP_GAME" in start or b"UMD_DATA.BIN" in start:
            return "psp"
        if b"BOOT2" in start:
            return "ps2"
        if b"BOOT =" in start or b"BOOT=" in start or b"PLAYSTATION" in start:
            return "ps1"
    return None


def nice_name(filename):
    name = os.path.splitext(os.path.basename(filename))[0]
    name = re.sub(r"[\[(][^\])]*[\])]", "", name)  # (USA), [!] and the like
    return re.sub(r"[_\s]+", " ", name).strip() or filename


def steam_root():
    for d in (".local/share/Steam", ".steam/steam", ".steam/root"):
        p = os.path.join(HOME, d)
        if os.path.isdir(os.path.join(p, "steamapps")):
            return p
    return None


def steam_games():
    root = steam_root()
    if not root:
        return []
    libraries = {os.path.join(root, "steamapps")}
    try:  # other Steam library folders (other drives)
        text = open(os.path.join(root, "steamapps", "libraryfolders.vdf"), encoding="utf-8", errors="replace").read()
        for p in re.findall(r'"path"\s+"([^"]+)"', text):
            libraries.add(os.path.join(p.replace("\\\\", "/"), "steamapps"))
    except OSError:
        pass
    games = []
    for lib in libraries:
        for acf in glob.glob(os.path.join(lib, "appmanifest_*.acf")):
            try:
                text = open(acf, encoding="utf-8", errors="replace").read()
            except OSError:
                continue
            appid = re.search(r'"appid"\s+"(\d+)"', text)
            name = re.search(r'"name"\s+"([^"]+)"', text)
            if not appid or not name or STEAM_TOOLS.search(name.group(1)):
                continue
            appid = appid.group(1)
            cover = ""
            for c in (f"appcache/librarycache/{appid}_library_600x900.jpg", f"appcache/librarycache/{appid}/library_600x900.jpg"):
                if os.path.isfile(os.path.join(root, c)):
                    cover = os.path.join(root, c)
                    break
            games.append({"key": "steam:" + appid, "name": name.group(1), "kind": "steam", "system": "Steam",
                          "emoji": "🎮", "cover": cover})
    return games


def icon_path(name):
    if not name:
        return ""
    if os.path.isabs(name):
        return name if os.path.isfile(name) else ""
    for pattern in ("/usr/share/icons/hicolor/256x256/apps/{}.png", "/usr/share/icons/hicolor/128x128/apps/{}.png",
                    "/usr/share/icons/hicolor/scalable/apps/{}.svg", "/usr/share/pixmaps/{}.png", "/usr/share/pixmaps/{}.svg",
                    "/var/lib/flatpak/exports/share/icons/hicolor/128x128/apps/{}.png",
                    HOME + "/.local/share/flatpak/exports/share/icons/hicolor/128x128/apps/{}.png",
                    HOME + "/.local/share/icons/hicolor/128x128/apps/{}.png"):
        p = pattern.format(name)
        if os.path.isfile(p):
            return p
    return ""


def app_games():
    """Games in the apps menu (Minecraft, Flathub games, SuperTuxKart...)."""
    seen, games = set(), []
    dirs = [HOME + "/.local/share/applications", "/usr/local/share/applications", "/usr/share/applications",
            HOME + "/.local/share/flatpak/exports/share/applications", "/var/lib/flatpak/exports/share/applications"]
    for d in dirs:
        for f in sorted(glob.glob(os.path.join(d, "*.desktop"))):
            base = os.path.basename(f)
            if base in seen:
                continue
            seen.add(base)
            p = configparser.ConfigParser(interpolation=None, strict=False)
            try:
                p.read(f, encoding="utf-8")
                e = p["Desktop Entry"]
            except (configparser.Error, KeyError, UnicodeDecodeError):
                continue
            if "Game" not in e.get("Categories", "").split(";") or e.get("NoDisplay", "").lower() == "true" \
                    or e.get("Hidden", "").lower() == "true" or e.get("Type", "Application") != "Application":
                continue
            if base.startswith(("steam", "ollie-games", "ollie-typing", "ollie-pet")):
                continue
            games.append({"key": "app:" + f, "name": e.get("Name", base), "kind": "app", "system": "App",
                          "emoji": "⛏️" if "minecraft" in base.lower() else "🕹️", "cover": icon_path(e.get("Icon", ""))})
    return games


def file_games():
    games = []
    for key, (cname, emoji, _, _) in CONSOLES.items():
        d = os.path.join(GAMES, key)
        if not os.path.isdir(d):
            continue
        files = sorted(os.listdir(d))
        cued = {b.lower() for f in files if f.lower().endswith(".cue") for b in cue_files(os.path.join(d, f))}
        for f in files:
            path = os.path.join(d, f)
            if not os.path.isfile(path) or f.startswith(".") or f.lower() in cued:
                continue  # (a disc's .bin parts play through its .cue)
            cover = ""
            for ext in (".png", ".jpg"):
                if os.path.isfile(os.path.splitext(path)[0] + ext) and not path.endswith(ext):
                    cover = os.path.splitext(path)[0] + ext
            if f.lower().endswith((".png", ".jpg", ".txt")):
                continue
            games.append({"key": "file:" + path, "name": nice_name(f), "kind": "file", "system": cname,
                          "emoji": emoji, "cover": cover})
    return games


def cue_files(cue):
    try:
        text = open(cue, encoding="utf-8", errors="replace").read()
    except OSError:
        return []
    return [m for m in re.findall(r'FILE\s+"([^"]+)"', text)]


def cmd_list(out):
    games = steam_games() + app_games() + file_games()
    games.sort(key=lambda g: g["name"].lower())
    with open(out, "w", encoding="utf-8") as f:
        json.dump({"games": games, "folder": GAMES}, f)


def ask_console(path):
    """kdialog: which console a file is for, when the file does not say."""
    items = []
    for key, (cname, _, _, _) in CONSOLES.items():
        items += [key, cname]
    r = subprocess.run(["kdialog", "--title", "Games", "--menu",
                        f"Which console is this game for?\n{os.path.basename(path)}"] + items,
                       capture_output=True, text=True)
    return r.stdout.strip() or None


def cmd_add(paths):
    added, skipped = [], []
    for path in paths:
        if path.startswith("file://"):
            from urllib.parse import unquote
            path = unquote(path[7:])
        if not os.path.isfile(path):
            skipped.append(os.path.basename(path))
            continue
        c = console_of(path) or ask_console(path)
        if not c:
            skipped.append(os.path.basename(path))
            continue
        d = os.path.join(GAMES, c)
        os.makedirs(d, exist_ok=True)
        todo = [path] + [os.path.join(os.path.dirname(path), b) for b in (cue_files(path) if path.lower().endswith(".cue") else [])]
        for p in todo:
            if os.path.isfile(p) and os.path.dirname(os.path.abspath(p)) != os.path.abspath(d):
                shutil.copy2(p, d)
        added.append(f"{nice_name(path)} ({CONSOLES[c][0]})")
    msg = ""
    if added:
        msg += "Added to Games:\n" + "\n".join("  " + a for a in added)
    if skipped:
        msg += ("\n\n" if msg else "") + "Not added (not a game file it knows):\n" + "\n".join("  " + s for s in skipped)
    if msg:
        subprocess.run(["kdialog", "--title", "Games", "--msgbox", msg])


def in_terminal(script):
    """Runs a few shell lines in a window that shows them (installing)."""
    for term in (["konsole", "-e"], ["xterm", "-e"]):
        if shutil.which(term[0]):
            return subprocess.run(term + ["bash", "-c", script]).returncode
    return subprocess.run(["bash", "-c", script]).returncode


def install_packages(pkgs, what):
    script = (f'printf "\\n\\e[1m  Getting the {what} player (only this once)\\e[0m\\n\\n"; '
              f'sudo -n pacman -S --needed --noconfirm {" ".join(pkgs)} || sudo -n pacman -Syu --needed --noconfirm {" ".join(pkgs)}; '
              'echo; echo "Press Enter to close."; read -r _')
    in_terminal(script)


def cmd_launch(key):
    kind, _, what = key.partition(":")
    if kind == "steam":
        subprocess.Popen(["steam", f"steam://rungameid/{what}"], start_new_session=True)
        return
    if kind == "app":
        for tool in (["gio", "launch", what], ["kioclient", "exec", what], ["kioclient5", "exec", what]):
            if shutil.which(tool[0]):
                subprocess.Popen(tool, start_new_session=True)
                return
        return
    if kind != "file":
        return
    path = what
    c = os.path.basename(os.path.dirname(path))
    if c not in CONSOLES:
        return
    cname, _, _, how = CONSOLES[c]
    if how == "oliver":
        runner = "/usr/local/bin/oliver" if os.path.exists("/usr/local/bin/oliver") else "wine"
        subprocess.Popen([runner, "run", path] if runner.endswith("oliver") else [runner, path], start_new_session=True)
        return
    if how == "dolphin":
        if not shutil.which("dolphin-emu"):
            install_packages(["dolphin-emu"], cname)
        subprocess.Popen(["dolphin-emu", "-b", "-e", path], start_new_session=True)
        return
    if how == "pcsx2":
        app = "net.pcsx2.PCSX2"
        if subprocess.run(["flatpak", "info", "--user", app], capture_output=True).returncode != 0:
            in_terminal('printf "\\n\\e[1m  Getting the PlayStation 2 player (only this once)\\e[0m\\n\\n"; '
                        "flatpak remote-add --user --if-not-exists flathub https://dl.flathub.org/repo/flathub.flatpakrepo; "
                        f"flatpak install --user -y flathub {app}; echo; echo 'Press Enter to close.'; read -r _")
        subprocess.Popen(["flatpak", "run", "--user", app, "-fullscreen", path], start_new_session=True)
        return
    pkg, core = how
    core_file = os.path.join(CORE_DIR, core + "_libretro.so")
    if not shutil.which("retroarch") or not os.path.exists(core_file):
        install_packages(["retroarch", pkg], cname)
    if not os.path.exists(core_file):
        subprocess.run(["kdialog", "--title", "Games", "--sorry", f"The {cname} player could not be installed. Is the internet connected?"])
        return
    if c == "ps1" and not glob.glob(os.path.join(HOME, ".config/retroarch/system/scph*")):
        subprocess.run(["kdialog", "--title", "Games", "--msgbox",
                        "PlayStation games need the PlayStation's BIOS file, copied from your own console "
                        "(for example scph5501.bin). Put it in the folder .config/retroarch/system in your home folder."])
    subprocess.Popen(["retroarch", "-f", "-L", core_file, path], start_new_session=True)


if __name__ == "__main__":
    if len(sys.argv) >= 3 and sys.argv[1] == "list":
        cmd_list(sys.argv[2])
    elif len(sys.argv) >= 3 and sys.argv[1] == "add":
        cmd_add(sys.argv[2:])
    elif len(sys.argv) >= 3 and sys.argv[1] == "launch":
        cmd_launch(sys.argv[2])
    else:
        print(__doc__)
        sys.exit(2)
