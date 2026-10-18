#!/bin/bash
# The virtual computer's initramfs: Batocera's own /init, patched by Etros
# Maker's real code (-patch-initrd), with an ARM busybox. A small /preinit
# loads the FAT driver first (the Ubuntu kernel has it as a module; the Pi's
# Batocera kernel has it built in).
set -euo pipefail
V=/var/tmp/ovm
R=$V/root
I=$V/initrd
rm -rf "$I" && mkdir -p "$I/bin" "$I/dev" "$I/lib/modules"
cp "$R/usr/bin/busybox" "$I/bin/busybox"
ln -s busybox "$I/bin/ash"; ln -s busybox "$I/bin/sh"
cp /home/user/batocera-linux/batocera.linux/package/batocera/boot/batocera-initramfs/init "$I/init"
chmod +x "$I/init"
kver=$(ls "$R/lib/modules" | head -1)
for m in fat vfat nls_cp437 nls_iso8859-1 nls_utf8 loop; do
    f=$(find "$R/lib/modules/$kver" -name "$m.ko*" | head -1)
    case "$f" in
        "") ;;
        *.zst) zstd -q -dc "$f" > "$I/lib/modules/$m.ko" ;;
        *) cp "$f" "$I/lib/modules/$m.ko" ;;
    esac
done
cat > "$I/preinit" <<'P'
#!/bin/busybox ash
/bin/busybox --install -s /bin
export PATH=/bin
mount -t proc proc /proc 2>/dev/null || { mkdir -p /proc && mount -t proc proc /proc; }
for m in fat vfat nls_cp437 nls_iso8859-1 nls_utf8 loop; do
    test -f /lib/modules/$m.ko && insmod /lib/modules/$m.ko
done
umount /proc
exec /init
P
chmod +x "$I/preinit"
ls "$I/lib/modules"
(cd "$I" && find . | cpio -o -H newc --quiet | lz4 -l -q -9 > "$V/initrd-plain.lz4")
"$V/ei" -patch-initrd "$V/initrd-plain.lz4" -target "file:$V/initrd.lz4"
echo "initrd ready"
