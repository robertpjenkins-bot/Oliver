#!/bin/bash
# Build the Etros SD card image: Batocera for the Raspberry Pi 5 and the
# R36S on one card, with the Etros look and Ollie OS.
#
#   sudo ./build.sh [WORKDIR]
#
# Settings, all optional:
#   PI5_IMAGE, R36S_IMAGE   Batocera .img.gz files to use instead of
#                           downloading the latest stable release
#   OLLIE_SQUASHFS          a ready-built Ollie OS; SKIP_OLLIE=1 to leave it out
#   MY_ROMS                 folder of your own ROMs and hack patches to put
#                           on the card (default: etros/my-roms, not in git)
#   VERSION                 label for the image (default: git describe)
#
# Needs root (initramfs device nodes, mmdebstrap), mtools, squashfs-tools,
# u-boot-tools, lz4, cpio, fdisk, dosfstools, e2fsprogs, ffmpeg, python3
# with Pillow, and mmdebstrap (+ qemu-user-static off arm64) for Ollie OS.
set -euo pipefail

HERE=$(cd "$(dirname "$0")" && pwd)
S=$HERE/scripts
WORK=$(realpath -m "${1:-$HERE/build}")
VERSION=${VERSION:-$(git -C "$HERE" describe --tags --always --dirty 2>/dev/null || echo dev)}
MY_ROMS=${MY_ROMS:-$HERE/my-roms}
OUT="$WORK/etros-$VERSION.img"
export MTOOLS_SKIP_CHECK=1

MIB=$((1024 * 1024))
SHARE_MIB=512   # Batocera only grows and formats a 256 or 512 MiB share

step() { echo; echo "=== $*"; }

# Paths given relative to where we were started from.
for var in PI5_IMAGE R36S_IMAGE OLLIE_SQUASHFS MY_ROMS; do
    if [ -n "${!var:-}" ]; then
        printf -v "$var" '%s' "$(realpath -m "${!var}")"
    fi
done

mkdir -p "$WORK"
cd "$WORK"

# Where a partition starts, in bytes: partstart IMAGE NUMBER
partstart() {
    sfdisk -J "$1" | python3 -c "
import json, sys
t = json.load(sys.stdin)['partitiontable']
p = t['partitions'][int(sys.argv[1]) - 1]
print(p['start'] * t.get('sectorsize', 512))" "$2"
}

# Unpack a downloaded image: its boot partition into DIR, and for the R36S,
# the bootloader that sits in the gap in front of the first partition.
unpack() {
    local gz=$1 dir=$2 img=$2.img
    gunzip -c "$gz" | dd of="$img" bs=4M conv=sparse status=none
    local start
    start=$(partstart "$img" 1)
    rm -rf "$dir"
    mkdir -p "$dir"
    mcopy -s -p -n -i "$img@@$start" '::*' "$dir/"
    echo "$start" > "$dir.start"
}

step "Batocera images"
for board in pi5 r36s; do
    var=$( [ $board = pi5 ] && echo PI5_IMAGE || echo R36S_IMAGE )
    if [ -n "${!var:-}" ]; then
        cp "${!var}" "$board.img.gz"
    elif [ ! -f "$board.img.gz" ]; then
        if [ $board = pi5 ]; then
            "$S/fetch-batocera.sh" bcm2712 'bcm2712' pi5.img.gz
        else
            "$S/fetch-batocera.sh" rk3326 'r36s' r36s.img.gz
        fi
    fi
    unpack "$board.img.gz" "$board"
done

step "Moving each board's system into etros/<board>"
for board in pi5 r36s; do
    test -d "$board/boot" || { echo "$board image has no /boot directory" >&2; exit 1; }
    mkdir -p "$board/etros"
    mv "$board/boot" "$board/etros/$board"
done

# The Pi firmware loads the kernel and initramfs itself, from config.txt.
grep -q '^kernel=boot/linux' pi5/config.txt
grep -q '^initramfs boot/initrd.lz4' pi5/config.txt
sed -i -e 's|^kernel=boot/linux|kernel=etros/pi5/linux|' \
       -e 's|^initramfs boot/initrd.lz4|initramfs etros/pi5/initrd.lz4|' pi5/config.txt
"$S/patch-initrd.sh" pi5/etros/pi5/initrd.lz4

# The R36S U-Boot loads /linux and /uInitrd, which stay where they are.
"$S/patch-initrd.sh" r36s/uInitrd

step "Merging the boot partitions"
rm -rf fat
"$S/merge-boot.py" pi5 r36s fat
mkdir -p fat/boot    # each board's etros/<board> is bound here at boot
echo batocera > fat/etros/next-boot

step "Etros layer"
python3 "$S/make-art.py" art
"$S/make-layer.sh" art fat/etros/etros.squashfs "$VERSION"

if [ "${SKIP_OLLIE:-0}" != 1 ]; then
    step "Ollie OS"
    if [ -n "${OLLIE_SQUASHFS:-}" ]; then
        cp "$OLLIE_SQUASHFS" fat/etros/ollie.squashfs
    else
        "$HERE/ollie/build-ollie.sh" art fat/etros/ollie.squashfs
    fi
fi

step "my-roms folder"
# Copy games here from a computer after flashing; Etros adds them at boot.
for sys in snes gba gb gbc nes n64 nds megadrive psx; do
    mkdir -p "fat/my-roms/$sys"
done
cp "$HERE/my-roms-README.txt" fat/my-roms/README.txt
if [ -d "$MY_ROMS" ]; then
    # Try the patches now, so a wrong ROM shows up here and not on the device.
    python3 "$S/add-my-roms.py" "$MY_ROMS" "$WORK/my-roms-check"
    rm -rf "$WORK/my-roms-check"
    cp -r "$MY_ROMS"/. fat/my-roms/
fi

step "Building the card image"
big=$(find fat -type f -size +4095M)
if [ -n "$big" ]; then
    echo "too large for the FAT boot partition (4 GiB per file): $big" >&2
    exit 1
fi
used_mib=$(( $(du -sb fat | cut -f1) / MIB ))
fat_mib=$(( used_mib + used_mib / 10 + 1024 ))   # room for Batocera's saved overlays
# Both boards' bootloaders start the boot partition at 16 MiB or later; the
# R36S's own bootloader fills the space before it.
start=$(cat r36s.start)
if [ "$start" -lt $((16 * MIB)) ] || [ $((start % MIB)) -ne 0 ]; then
    echo "unexpected R36S layout: partition 1 at byte $start" >&2
    exit 1
fi

rm -f fat.img share.img "$OUT"
mkfs.vfat -F 32 -n BATOCERA -C fat.img $((fat_mib * 1024)) > /dev/null
mcopy -s -p -i fat.img fat/* ::/
mkfs.ext4 -q -L SHARE -m 0 share.img ${SHARE_MIB}M

share_start=$(( start + fat_mib * MIB ))
truncate -s $(( share_start + SHARE_MIB * MIB + MIB )) "$OUT"
sfdisk -q "$OUT" <<EOF
label: dos
start=$((start / 512)), size=$((fat_mib * MIB / 512)), type=c, bootable
start=$((share_start / 512)), size=$((SHARE_MIB * MIB / 512)), type=83
EOF
# R36S idbloader, U-Boot and trust image, between the partition table and
# the first partition. The Pi ignores this space.
dd if=r36s.img of="$OUT" bs=512 skip=64 seek=64 count=$((start / 512 - 64)) conv=notrunc,sparse status=none
dd if=fat.img of="$OUT" bs=1M seek=$((start / MIB)) conv=notrunc,sparse status=none
dd if=share.img of="$OUT" bs=1M seek=$((share_start / MIB)) conv=notrunc,sparse status=none

step "Compressing"
xz -T0 -6 -f -k "$OUT"
ls -lh "$OUT.xz"
echo "done: $OUT.xz"
