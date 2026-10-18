#!/bin/bash
# Put together a virtual Etros SD card the way Etros Maker's no-password
# install does (BATOCERA FAT + SHARE ext4 holding etros/ollie-rootfs.img),
# from the real Etros Maker output, plus the initramfs to start it with.
set -euo pipefail
V=/var/tmp/ovm
R=$V/root
EI=$V/ei
cd "$V"

# 1. Ollie OS stand-in as a "Raspberry Pi system image": one Linux partition.
if [ ! -f ollie-src.img ]; then
    used=$(du -sxm "$R" | cut -f1)
    size=$((used * 115 / 100 + 1024))
    mkfs.ext4 -q -F -L rootfs -d "$R" rootfs.ext4 "${size}M"
    truncate -s $(((size + 1) * 1024 * 1024)) ollie-src.img
    echo 'start=2048, type=83' | sfdisk -q ollie-src.img
    dd if=rootfs.ext4 of=ollie-src.img bs=1M seek=1 conv=notrunc,sparse status=none
    rm rootfs.ext4
fi

# 2. The real Etros Maker build, with stand-in Batocera files.
rm -f etros.img
"$EI" -target file:etros.img -pi5-boot /tmp/claude-0/fake3/pi5.img.gz \
    -r36s-boot /tmp/claude-0/fake3/r36s.img.gz -rpios ollie-src.img > maker.log 2>&1 || { tail -5 maker.log; exit 1; }
read -r fstart fsize ostart osize < <(sfdisk -J etros.img | python3 -c '
import json,sys; p=json.load(sys.stdin)["partitiontable"]["partitions"]
print(p[0]["start"], p[0]["size"], p[2]["start"], p[2]["size"])')

# 3. The card, as the no-password install makes it.
rm -f card.img
truncate -s 24G card.img
printf 'start=%s, size=%s, type=c, bootable\nstart=%s, type=83\n' "$fstart" "$fsize" $((fstart + fsize)) | sfdisk -q card.img
dd if=etros.img of=card.img bs=512 skip="$fstart" seek="$fstart" count="$fsize" conv=notrunc,sparse status=none
rm -rf share && mkdir -p share/etros "share/downloads/My ROMs/PlayStation"
dd if=etros.img of=share/etros/ollie-rootfs.img bs=512 skip="$ostart" count="$osize" conv=sparse status=none
# Test ROMs waiting to be sorted.
echo snes > "share/downloads/My ROMs/Super Metroid (USA).sfc"
echo psx > "share/downloads/My ROMs/PlayStation/Tekken 3.chd"
sharesize=$(( (24 * 1024 * 1024 * 1024 / 512 - fstart - fsize) / 2048 ))
mkfs.ext4 -q -F -L SHARE -d share share.ext4 "${sharesize}M"
dd if=share.ext4 of=card.img bs=512 seek=$((fstart + fsize)) conv=notrunc,sparse status=none
rm -rf share share.ext4

# 4. Ask for Ollie OS on the next start, as ETROS -> Boot into Ollie OS does.
echo ollie > next-boot
MTOOLS_SKIP_CHECK=1 mcopy -o -i "card.img@@$((fstart * 512))" next-boot ::/etros/next-boot
echo "card ready: FAT at $fstart, SHARE after it"
