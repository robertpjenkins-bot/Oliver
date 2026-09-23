#!/usr/bin/env python3
"""First-run setup window for the small JS Academy AppImage.

Downloads the Electron engine with a progress bar and unpacks it.
Usage: engine-setup.py URL DEST_DIR
Exit status 0 on success, 1 on failure, 2 if the user cancelled.
"""
import os
import shutil
import sys
import tempfile
import threading
import urllib.request
import zipfile
import tkinter as tk
from tkinter import ttk

URL, DEST = sys.argv[1], sys.argv[2]
BG, FG, MUTED, ACCENT = "#0f1420", "#dbe2f0", "#8a96b0", "#f7df1e"

state = {"done": 0, "total": 0, "phase": "download", "error": None, "cancel": False, "finished": False}


def work():
    tmp = tempfile.mkdtemp(prefix="download.", dir=os.path.dirname(DEST))
    try:
        zpath = os.path.join(tmp, "electron.zip")
        with urllib.request.urlopen(URL, timeout=60) as resp, open(zpath, "wb") as out:
            state["total"] = int(resp.headers.get("Content-Length") or 0)
            while True:
                if state["cancel"]:
                    return
                chunk = resp.read(256 * 1024)
                if not chunk:
                    break
                out.write(chunk)
                state["done"] += len(chunk)
        state["phase"] = "unpack"
        engine = os.path.join(tmp, "engine")
        with zipfile.ZipFile(zpath) as z:
            for info in z.infolist():
                path = z.extract(info, engine)
                mode = info.external_attr >> 16
                if mode:
                    os.chmod(path, mode & 0o777)
        for name in ("electron", "chrome_crashpad_handler", "chrome-sandbox"):
            p = os.path.join(engine, name)
            if os.path.exists(p):
                os.chmod(p, 0o755)
        shutil.rmtree(DEST, ignore_errors=True)
        os.rename(engine, DEST)
    except Exception as e:  # shown in the window
        state["error"] = str(e)
    finally:
        shutil.rmtree(tmp, ignore_errors=True)
        state["finished"] = True


root = tk.Tk()
root.title("JS Academy Setup")
root.configure(bg=BG)
root.resizable(False, False)
w, h = 460, 210
root.geometry(f"{w}x{h}+{(root.winfo_screenwidth() - w) // 2}+{(root.winfo_screenheight() - h) // 3}")

style = ttk.Style()
style.theme_use("clam")
style.configure("Bar.Horizontal.TProgressbar", troughcolor="#1c2436", background=ACCENT, bordercolor=BG, lightcolor=ACCENT, darkcolor=ACCENT)

tk.Label(root, text="Setting up JS Academy", font=("Sans", 16, "bold"), bg=BG, fg=FG).pack(anchor="w", padx=24, pady=(22, 4))
tk.Label(root, text="Downloading the app engine. This only happens once.", font=("Sans", 10), bg=BG, fg=MUTED).pack(anchor="w", padx=24)
bar = ttk.Progressbar(root, style="Bar.Horizontal.TProgressbar", length=w - 48, maximum=1000)
bar.pack(padx=24, pady=(18, 6))
status = tk.Label(root, text="Connecting…", font=("Sans", 10), bg=BG, fg=MUTED)
status.pack(anchor="w", padx=24)


def cancel():
    state["cancel"] = True
    root.destroy()
    sys.exit(2)


btn = tk.Button(root, text="Cancel", command=cancel, bg="#1c2436", fg=FG, activebackground="#273149", activeforeground=FG, relief="flat", padx=14, pady=3)
btn.pack(anchor="e", padx=24, pady=(10, 0))
root.protocol("WM_DELETE_WINDOW", cancel)


def tick():
    if state["finished"]:
        if state["error"]:
            status.config(text="Download failed. Check your internet connection and try again.", fg="#ff6b6b")
            btn.config(text="Close", command=lambda: (root.destroy(), sys.exit(1)))
            root.protocol("WM_DELETE_WINDOW", lambda: (root.destroy(), sys.exit(1)))
            return
        root.destroy()
        return
    if state["phase"] == "unpack":
        bar.config(mode="indeterminate")
        bar.start(12)
        status.config(text="Unpacking…")
    elif state["total"]:
        bar["value"] = 1000 * state["done"] / state["total"]
        status.config(text=f"{state['done'] / 1048576:.0f} of {state['total'] / 1048576:.0f} MB")
    root.after(150, tick)


threading.Thread(target=work, daemon=True).start()
root.after(150, tick)
root.mainloop()
sys.exit(0 if os.path.exists(os.path.join(DEST, "electron")) else 1)
