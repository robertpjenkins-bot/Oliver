#!/bin/bash
# Run an Etros-patched Batocera initramfs (as the installer writes it to the
# card) in a throwaway chroot and check where each kind of boot ends up. Needs root (mounts) and a kernel with
# squashfs, overlay and loop support; busybox stands in for the initramfs.
#
#   initrd-test.sh INITRD ETROS_SQUASHFS [BUSYBOX_CONFIG]
#
# INITRD is etros/pi5/initrd.lz4 from a card the installer made.
#
# BUSYBOX_CONFIG is Batocera's initramfs busybox.config; only the commands
# it enables are available in the sandbox, as on the real device.
#
# Only the boot partition is faked: a tmpfs instead of FAT, and the device
# tree's "compatible" string from a file. switch_root is replaced by a
# script that reports what the new root looks like instead of switching.
set -euo pipefail

INITRD=$(realpath "$1")
LAYER=$(realpath "$2")
BBCONFIG=${3:-}
T=$(mktemp -d)
trap 'umount -R "$T"/root/* 2>/dev/null || true; rm -rf "$T"' EXIT
fails=0

squash() {  # squash OUT WHOAMI [extra files as path=content...]
    local out=$1 dir
    dir=$(mktemp -d -p "$T")
    mkdir -p "$dir/sbin" "$dir/boot" "$dir/overlay" "$dir/sys" "$dir/proc" "$dir/dev"
    echo "$2" > "$dir/whoami"
    shift 2
    for kv in "$@"; do
        mkdir -p "$(dirname "$dir/${kv%%=*}")"
        echo "${kv#*=}" > "$dir/${kv%%=*}"
    done
    mksquashfs "$dir" "$out" -quiet -no-progress -noappend > /dev/null
}

# The boot partition as Etros lays it out.
FAT=$T/fat
mkdir -p "$FAT/boot" "$FAT/etros/pi5" "$FAT/etros/r36s"
squash "$FAT/etros/pi5/batocera.update" batocera-pi5 lib/modules/6.12-pi5/marker=pi5
squash "$FAT/etros/r36s/batocera.update" batocera-r36s lib/modules/6.12-r36s/marker=r36s
echo bcm2712 > "$FAT/etros/pi5/batocera.board"
echo r36s > "$FAT/etros/r36s/batocera.board"
cp "$LAYER" "$FAT/etros/etros.squashfs"
# Ollie OS: an ext4 root file system, plus Etros's files for it.
mkdir -p "$T/ollie/sbin" "$T/ollie/lib/modules" "$T/ollie/lib/firmware" "$T/ollie/boot" "$T/ollie/sys" "$T/ollie/proc" "$T/ollie/dev"
echo ollie > "$T/ollie/whoami"
ln -s ../lib/systemd/systemd "$T/ollie/sbin/init"  # as in Raspberry Pi OS
mkfs.ext4 -q -L rootfs -d "$T/ollie" "$T/ollie.img" 32M
# The same Ollie OS as a file on the storage partition, as cards put
# together without a password have it.
mkdir -p "$T/share/etros" "$T/share/roms"
echo "from share" > "$T/share/roms/marker"
cp "$T/ollie.img" "$T/share/etros/ollie-rootfs.img"
mkfs.ext4 -q -L SHARE -d "$T/share" "$T/share.img" 64M
mkdir -p "$FAT/etros/ollie/etc"
echo "from etros" > "$FAT/etros/ollie/etc/etros-marker"
echo batocera > "$FAT/etros/next-boot"

# A minimal initramfs: busybox, the patched init and the hook.
R=$T/root
mkdir -p "$R/bin" "$R/dev" "$R/fatsrc" "$R/dt"
cp "$(command -v busybox)" "$R/bin/busybox"
for lib in $(ldd "$R/bin/busybox" | grep -o '/[^ ]*'); do
    mkdir -p "$R$(dirname "$lib")"
    cp "$lib" "$R$lib"
done
if [ -n "$BBCONFIG" ]; then
    applets=$(sed -n 's/^CONFIG_\([A-Z0-9_]*\)=y$/\1/p' "$BBCONFIG" | tr '[:upper:]' '[:lower:]' | grep -vxE 'feature_.*')
else
    applets=$(busybox --list)
fi
for applet in $applets; do
    # names in busybox.config that are not commands are harmless links
    case "$applet" in switch_root|busybox) continue ;; esac
    ln -sf busybox "$R/bin/$applet"
done
ln -sf busybox "$R/bin/sh"
# The test's own tools, kept out of the PATH the boot script runs with.
mkdir -p "$R/hbin"
for applet in $(busybox --list); do
    [ "$applet" = switch_root ] || ln -sf ../bin/busybox "$R/hbin/$applet"
done
cp "$T/ollie.img" "$R/ollie.img"
cp "$T/share.img" "$R/share.img"
# The patched /init and the hook, from the initramfs itself.
(cd "$T" && mkdir unpacked && cd unpacked && lz4 -dc "$INITRD" | cpio -id --quiet init etros-hook.sh)
cp "$T/unpacked/init" "$T/unpacked/etros-hook.sh" "$R/"
grep -q 'etros_start' "$R/init"
# Boot "partition": a tmpfs filled from FAT and made read-only, like the
# real one. The kernel command line is not ours to set, so give the loop
# a device name and mount our tmpfs in its place.
sed -i -e 's|read -r cmdline < /proc/cmdline|cmdline="dev=fake"|' \
       -e 's|^    /bin/ash$|    exit 1|' \
       -e 's|if mount -o [^ ]* "${1}" *\/boot_root; then return 0; fi|mount -t tmpfs tmpfs /boot_root \&\& cp -a /fatsrc/. /boot_root/ \&\& mount -o remount,ro /boot_root \&\& return 0|' \
       "$R/init"
grep -q 'cp -a /fatsrc' "$R/init"
cat > "$R/bin/switch_root" <<'EOF'
#!/bin/busybox ash
# Report the root we would have switched to.
PATH=/hbin
new=$1
echo "root=$(cat $new/whoami)"
echo "board=$(cat $new/boot/boot/batocera.board 2>/dev/null)"
echo "layer=$(test -e $new/etc/init.d/S13etros && echo yes || echo no)"
echo "modules=$(cat $new/lib/modules/*/marker 2>/dev/null)"
echo "etros-files=$(cat $new/etc/etros-marker 2>/dev/null)"
echo "next=$(cat $new/boot/etros/next-boot)"
echo "userdata=$(cat $new/userdata/roms/marker 2>/dev/null)"
# Batocera remounts /boot read-write to save settings; /boot/boot must follow.
mkdir -p /proc && mount -t proc proc /proc
if mount -o remount,rw $new/boot && touch $new/boot/boot/written; then echo "boot-writable=yes"; else echo "boot-writable=no"; fi
mount -o remount,ro $new/boot
EOF
chmod +x "$R/bin/switch_root"

boot() {  # boot COMPATIBLE  -> report on stdout, boot partition saved back
    rm -rf "${R:?}/fatsrc"
    cp -a "$FAT" "$R/fatsrc"
    printf '%b\0' "$1" > "$R/dt/compatible"
    timeout 60 unshare -m chroot "$R" /bin/busybox env ETROS_COMPATIBLE=/dt/compatible "${BOOT_ENV:-ETROS_OLLIE_ROOT=/ollie.img}" PATH=/hbin /bin/busybox ash -c '
        PATH=/bin /init > /log 2>&1
        cat /log
        # keep what the boot changed on the boot partition for the next boot
        mkdir -p /proc && mount -t proc proc /proc
        umount -l /new_root/boot/boot
        rm -rf /fatsrc/*; cp -a /new_root/boot/. /fatsrc/' > "$T/log" 2>&1 || true
    grep -E '^[a-z-]+=' "$T/log" || true
    rm -rf "$FAT"
    cp -a "$R/fatsrc" "$FAT"
}

expect() {  # expect REPORT KEY VALUE
    if grep -qx "$2=$3" <<< "$1"; then
        echo "  ok   $2=$3"
    else
        echo "  FAIL $2: wanted $3, got $(grep "^$2=" <<< "$1" || echo nothing)"
        fails=$((fails + 1))
    fi
}

echo "Pi 5, first boot"
out=$(boot brcm,bcm2712)
expect "$out" root batocera-pi5
expect "$out" board bcm2712
expect "$out" layer yes
expect "$out" boot-writable yes
if test -f "$FAT/etros/pi5/batocera"; then
    echo "  ok   batocera.update installed"
else
    echo "  FAIL batocera.update not installed"
    fails=$((fails + 1))
fi

echo "R36S, same card"
out=$(boot "gameconsole,r36s\0rockchip,rk3326")
expect "$out" root batocera-r36s
if grep -q "reached Etros's start-up" "$FAT/etros/last-start-r36s.txt" 2>/dev/null; then
    echo "  ok   R36S start noted on the card"
else
    echo "  FAIL no etros/last-start-r36s.txt"
    fails=$((fails + 1))
fi
expect "$out" board r36s
expect "$out" layer yes

echo "R36S, Boot into Ollie OS"
echo ollie > "$FAT/etros/next-boot"
out=$(boot "gameconsole,r36s\0rockchip,rk3326")
expect "$out" root ollie
expect "$out" modules r36s
expect "$out" etros-files "from etros"
expect "$out" next batocera

echo "Pi 5, Ollie OS kept as a file on the storage partition"
echo ollie > "$FAT/etros/next-boot"
out=$(BOOT_ENV="ETROS_SHARE=/share.img" boot "raspberrypi,5-model-b\0brcm,bcm2712")
expect "$out" root ollie
expect "$out" modules pi5
expect "$out" userdata "from share"
expect "$out" next batocera

echo "Pi 5, Ollie OS missing: back to Batocera, with the reason left on the card"
echo ollie > "$FAT/etros/next-boot"
rm -f "$FAT/etros/last-ollie-boot.txt"
out=$(BOOT_ENV="ETROS_SHARE=/no-such-share.img" boot "raspberrypi,5-model-b\0brcm,bcm2712")
expect "$out" root batocera-pi5
expect "$out" next batocera
if grep -q "did not start" "$FAT/etros/last-ollie-boot.txt" 2>/dev/null; then
    echo "  ok   reason saved: $(cat "$FAT/etros/last-ollie-boot.txt")"
else
    echo "  FAIL no etros/last-ollie-boot.txt"
    fails=$((fails + 1))
fi

echo "R36S, the boot after Ollie OS"
out=$(boot "gameconsole,r36s\0rockchip,rk3326")
expect "$out" root batocera-r36s

if [ "$fails" -ne 0 ]; then
    echo "$fails check(s) failed; last init log:"
    cat "$T/log"
    exit 1
fi
echo "all boot checks passed"
