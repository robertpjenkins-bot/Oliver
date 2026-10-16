#!/usr/bin/env python3
"""Merge the Pi 5 and R36S boot partitions into one directory tree.

    merge-boot.py PI5_DIR R36S_DIR OUT_DIR

Each board's /boot directory (its system image and friends) must already
have been moved to etros/<board>/. Everything else is merged: the Pi
firmware and the R36S boot script don't share names, and where both have a
file with the same contents one copy is kept. A real clash stops the merge
instead of silently letting one board break the other. batocera-boot.conf,
which both have, is merged setting by setting.

The FAT partition ignores case, so names are compared case-insensitively.
"""
import filecmp
import os
import shutil
import sys

BOOT_CONF = "batocera-boot.conf"


def files(root):
    out = {}
    for dirpath, _, names in os.walk(root):
        for n in names:
            path = os.path.join(dirpath, n)
            out[os.path.relpath(path, root).lower()] = path
    return out


def conf_settings(path):
    out = {}
    with open(path) as f:
        for line in f:
            s = line.strip()
            if s and not s.startswith("#") and "=" in s:
                k, v = s.split("=", 1)
                out[k.strip()] = v.strip()
    return out


def merge_conf(pi, r36s, dest):
    """Keep the Pi 5 file as the base and add any setting only the R36S has."""
    a, b = conf_settings(pi), conf_settings(r36s)
    extra = [f"{k}={v}" for k, v in b.items() if k not in a]
    for k in sorted(set(a) & set(b)):
        if a[k] != b[k]:
            print(f"note: {BOOT_CONF}: {k} is {a[k]!r} on the Pi 5 and {b[k]!r} on the R36S; "
                  f"keeping {a[k]!r}")
    shutil.copy2(pi, dest)
    if extra:
        with open(dest, "a") as f:
            f.write("\n# from the R36S image\n" + "\n".join(extra) + "\n")


def merge(pi_dir, r36s_dir, out):
    pi, r36s = files(pi_dir), files(r36s_dir)
    clashes = []
    for key, src in list(pi.items()) + list(r36s.items()):
        rel = os.path.relpath(src, pi_dir if src.startswith(pi_dir + os.sep) else r36s_dir)
        dest = os.path.join(out, rel)
        if key in pi and key in r36s:
            if os.path.basename(key) == BOOT_CONF:
                if src == pi[key]:
                    os.makedirs(os.path.dirname(dest), exist_ok=True)
                    merge_conf(pi[key], r36s[key], dest)
                continue
            if not filecmp.cmp(pi[key], r36s[key], shallow=False):
                if src == pi[key]:
                    clashes.append(rel)
                continue
            if src == r36s[key]:
                continue  # identical, the Pi copy is already there
        os.makedirs(os.path.dirname(dest), exist_ok=True)
        shutil.copy2(src, dest)
    if clashes:
        raise SystemExit("both images have different files at:\n  " + "\n  ".join(sorted(clashes)))


def main():
    if len(sys.argv) != 4:
        raise SystemExit("usage: merge-boot.py PI5_DIR R36S_DIR OUT_DIR")
    pi, r36s, out = (os.path.abspath(p) for p in sys.argv[1:])
    os.makedirs(out, exist_ok=True)
    merge(pi, r36s, out)


if __name__ == "__main__":
    main()
