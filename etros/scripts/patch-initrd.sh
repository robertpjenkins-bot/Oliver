#!/bin/bash
# Add the Etros boot hook to a Batocera initramfs, in place.
#
#   patch-initrd.sh initrd.lz4    Raspberry Pi: a bare lz4 (legacy) archive
#   patch-initrd.sh uInitrd       R36S: the same, wrapped in a U-Boot header
#
# Must run as root so device nodes in the archive survive the round trip.
set -euo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
ETROS=$(dirname "$HERE")
IN=$(realpath "$1")

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

uboot=0
if [ "$(head -c4 "$IN" | od -An -tx1 | tr -d ' ')" = "27051956" ]; then
    uboot=1
    dd if="$IN" of="$WORK/initrd.lz4" bs=64 skip=1 status=none
else
    cp "$IN" "$WORK/initrd.lz4"
fi

mkdir "$WORK/root"
lz4 -dc "$WORK/initrd.lz4" | (cd "$WORK/root" && cpio -id --quiet --no-absolute-filenames)
test -f "$WORK/root/init" || { echo "no /init in $IN" >&2; exit 1; }

python3 "$HERE/patch-init.py" "$WORK/root/init"
install -m 0755 "$ETROS/initrd/etros-hook.sh" "$WORK/root/etros-hook.sh"

# The kernel only understands the legacy lz4 frame format.
(cd "$WORK/root" && find . -mindepth 1 | LC_ALL=C sort | cpio -o -H newc --quiet) |
    lz4 -l -9 -q > "$WORK/new.lz4"

if [ "$uboot" = 1 ]; then
    mkimage -A arm64 -O linux -T ramdisk -C none -n uInitrd \
        -d "$WORK/new.lz4" "$IN" > /dev/null
else
    cp "$WORK/new.lz4" "$IN"
fi
echo "patched $(basename "$IN")"
