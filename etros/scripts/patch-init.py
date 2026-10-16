#!/usr/bin/env python3
"""Insert the Etros hooks into Batocera's initramfs /init script.

Every edit is anchored on a line of Batocera's script. If an anchor is missing
or appears more than once, Batocera changed its init and the patch refuses to
guess: it exits with an error naming the anchor, and the build stops.
"""
import sys

EDITS = [
    # Right after /boot_root is mounted and before any image is touched.
    ("insert-before", "    # update the squashfs\n",
     "    . /etros-hook.sh && etros_start || return 1\n\n"),
    # Just before the root overlay is assembled.
    ("insert-before", "    # mount the future root in read write\n",
     "    etros_mount_layer || return 1\n\n"),
    # Stack the Etros layer above the Batocera image.
    ("replace", "lowerdir=/overlay_root/base$EXTRALAYERS,",
     "lowerdir=${ETROS_LAYER}/overlay_root/base$EXTRALAYERS,"),
]


def patch(text):
    if "etros-hook.sh" in text:
        raise SystemExit("init is already patched for Etros")
    for kind, anchor, new in EDITS:
        count = text.count(anchor)
        if count != 1:
            raise SystemExit(
                f"Batocera's init has changed: expected one {anchor.strip()!r}, found {count}")
        if kind == "insert-before":
            text = text.replace(anchor, new + anchor)
        else:
            text = text.replace(anchor, new)
    return text


def main():
    if len(sys.argv) != 2:
        raise SystemExit("usage: patch-init.py INIT")
    path = sys.argv[1]
    with open(path) as f:
        text = f.read()
    with open(path, "w") as f:
        f.write(patch(text))


if __name__ == "__main__":
    main()
