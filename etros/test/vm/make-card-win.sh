#!/bin/bash
# Like make-card.sh, with Etros's Windows 11 layout: p3 WINDOWS (empty),
# p4 WINBOOT (FAT, the Pi 5 UEFI firmware and the marker), and a stand-in
# Windows 11 Arm64 ISO in Downloads.
set -euo pipefail
V=/var/tmp/ovm
cd "$V"
read -r fstart fsize ostart osize < <(sfdisk -J etros.img | python3 -c '
import json,sys; p=json.load(sys.stdin)["partitiontable"]["partitions"]
print(p[0]["start"], p[0]["size"], p[2]["start"], p[2]["size"])')
total=$((24 * 1024 * 1024 * 2))          # sectors in 24 GiB
win=$((2 * 1024 * 1024 * 2))             # 2 GiB
wb=$((256 * 1024 * 2))                   # 256 MiB
wstart=$((total - win - wb - 2048))
sstart=$((fstart + fsize))
rm -f card.img && truncate -s 24G card.img
printf 'start=%s, size=%s, type=c, bootable\nstart=%s, size=%s, type=83\nstart=%s, size=%s, type=7\nstart=%s, size=%s, type=c\n' \
    "$fstart" "$fsize" "$sstart" $((wstart - sstart)) "$wstart" "$win" $((wstart + win)) "$wb" | sfdisk -q card.img
dd if=etros.img of=card.img bs=512 skip="$fstart" seek="$fstart" count="$fsize" conv=notrunc,sparse status=none
rm -rf share && mkdir -p share/etros share/downloads
dd if=etros.img of=share/etros/ollie-rootfs.img bs=512 skip="$ostart" count="$osize" conv=sparse status=none
cp /tmp/claude-0/fakewin/Win11_25H2_English_Arm64.iso share/downloads/
mkfs.ext4 -q -F -L SHARE -d share share.ext4 $(( (wstart - sstart) / 2048 ))M
dd if=share.ext4 of=card.img bs=512 seek="$sstart" conv=notrunc,sparse status=none
rm -rf share share.ext4
# WINBOOT, as Etros Updater makes it.
rm -f wb.img && truncate -s 256M wb.img && mkfs.vfat -F 32 -n WINBOOT wb.img >/dev/null
for f in /home/user/Oliver/etros/installer/assets/uefi/*; do MTOOLS_SKIP_CHECK=1 mcopy -i wb.img "$f" ::/; done
echo "This is Etros's Windows 11 start-up partition (WINBOOT)." > marker && MTOOLS_SKIP_CHECK=1 mcopy -i wb.img marker ::/etros-winboot.txt
dd if=wb.img of=card.img bs=512 seek=$((wstart + win)) conv=notrunc status=none
rm -f wb.img marker
echo ollie > next-boot
MTOOLS_SKIP_CHECK=1 mcopy -o -i "card.img@@$((fstart * 512))" next-boot ::/etros/next-boot
echo "windows card ready"
