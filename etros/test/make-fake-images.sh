#!/bin/bash
# Make stand-ins for the Batocera Pi 5 and R36S images (and their
# boot.tar.xz update archives), and for Raspberry Pi OS, laid out like the
# real ones (see board/batocera/broadcom/bcm2712 and rockchip/rk3326/r36s in
# batocera.linux), so the image assembly can be tested without downloading
# several gigabytes. The initramfs holds Batocera's real /init.
#
#   make-fake-images.sh BATOCERA_INIT OUTDIR
set -euo pipefail
INIT=$(realpath "$1")
OUT=$(realpath -m "$2")
export MTOOLS_SKIP_CHECK=1
mkdir -p "$OUT"
T=$(mktemp -d)
trap 'rm -rf "$T"' EXIT

initrd() {
    mkdir -p "$T/ir/bin" "$T/ir/dev"
    cp "$INIT" "$T/ir/init"
    echo busybox > "$T/ir/bin/busybox"
    mknod "$T/ir/dev/console" c 5 1 2>/dev/null || true
    (cd "$T/ir" && find . -mindepth 1 | cpio -o -H newc --quiet) | lz4 -l -q > "$1"
}

image() {  # image NAME FAT_OFFSET_MIB DIR
    local img="$OUT/$1.img"
    rm -f "$img" "$T/fat"
    mkfs.vfat -F 32 -n BATOCERA -C "$T/fat" $((64 * 1024)) > /dev/null
    mcopy -s -i "$T/fat" "$3"/* ::/
    truncate -s $(( ($2 + 64 + 8) * 1024 * 1024 )) "$img"
    printf 'label: dos\nstart=%d, size=%d, type=c, bootable\nstart=%d, type=83\n' \
        $(($2 * 2048)) $((64 * 2048)) $((($2 + 64) * 2048)) | sfdisk -q "$img"
    dd if="$T/fat" of="$img" bs=1M seek="$2" conv=notrunc status=none
}

# Raspberry Pi 5
P=$T/pi5; mkdir -p "$P/boot" "$P/overlays"
printf 'arm_64bit=1\nkernel=boot/linux\ninitramfs boot/initrd.lz4\ndtoverlay=vc4-kms-v3d\n' > "$P/config.txt"
echo 'console=tty3 dev=LABEL=BATOCERA rootwait' > "$P/cmdline.txt"
printf 'autoresize=true\nsharedevice=INTERNAL\nsplash.screen.enabled=1\n' > "$P/batocera-boot.conf"
echo dtb > "$P/bcm2712-rpi-5-b.dtb"; echo dtbo > "$P/overlays/vc4-kms-v3d-pi5.dtbo"
echo "pi5 kernel" > "$P/boot/linux"; initrd "$P/boot/initrd.lz4"
echo "pi5 squashfs" > "$P/boot/batocera.update"; echo bcm2712 > "$P/boot/batocera.board"
echo same > "$P/overlays/shared.txt"
: > "$P/overlays/README"   # real images have empty files, e.g. a README
: > "$P/boot/empty.marker"
image pi5 1 "$P"

# R36S
R=$T/r36s; mkdir -p "$R/boot" "$R/overlays"
printf 'odroidgoa-uboot-config\nload mmc 1:1 ${loadaddr} linux\n' > "$R/boot.ini"
echo script > "$R/boot.scr"
printf 'autoresize=true\nsharedevice=INTERNAL\nsplash.screen.enabled=0\nr36s.only=1\n' > "$R/batocera-boot.conf"
echo dtb > "$R/rk3326-gameconsole-r36s.dtb"; echo panel > "$R/overlays/mipi-panel.dtbo.r36s-panel1"
echo "r36s kernel" > "$R/linux"
initrd "$T/r36s-initrd.lz4"
mkimage -A arm64 -O linux -T ramdisk -C none -n uInitrd -d "$T/r36s-initrd.lz4" "$R/uInitrd" > /dev/null
echo "r36s squashfs" > "$R/boot/batocera.update"; echo r36s > "$R/boot/batocera.board"
echo same > "$R/overlays/shared.txt"
: > "$R/README"
image r36s 16 "$R"
# idbloader at 32K, U-Boot at 8M, trust at 12M
head -c 65536 /dev/urandom > "$OUT/loader.bin"
dd if="$OUT/loader.bin" of="$OUT/r36s.img" bs=1K seek=32 conv=notrunc status=none
dd if="$OUT/loader.bin" of="$OUT/r36s.img" bs=1M seek=8 conv=notrunc status=none
dd if="$OUT/loader.bin" of="$OUT/r36s.img" bs=1M seek=12 conv=notrunc status=none

# Batocera's boot.tar.xz is the boot partition's contents, with the
# system image still named batocera.update.
(cd "$P" && tar -cJf "$OUT/pi5-boot.tar.xz" ./*)
(cd "$R" && tar -cJf "$OUT/r36s-boot.tar.xz" ./*)
gzip -1 -f "$OUT/pi5.img" "$OUT/r36s.img"

# Raspberry Pi OS: a FAT firmware partition and an ext4 root labelled rootfs.
O=$T/rpios; mkdir -p "$O/etc" "$O/sbin" "$O/usr/lib/modules"
echo "raspios" > "$O/etc/rpi-issue"
mkfs.ext4 -q -L rootfs -d "$O" "$T/root.ext4" 48M
truncate -s $(( (4 + 16 + 48) * 1024 * 1024 )) "$OUT/rpios.img"
printf 'label: dos\nstart=8192, size=32768, type=c\nstart=40960, size=98304, type=83\n' | sfdisk -q "$OUT/rpios.img"
dd if="$T/root.ext4" of="$OUT/rpios.img" bs=1M seek=20 conv=notrunc status=none
xz -T0 -1 -f "$OUT/rpios.img"
ls -l "$OUT"
