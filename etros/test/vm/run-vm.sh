#!/bin/bash
# Start the virtual ARM computer with the card, for SECONDS seconds, taking a
# screenshot every 30 s (shot-NNN.png) and keeping the serial console log.
set -euo pipefail
V=/var/tmp/ovm
secs=${1:-900}
out=$V/run
rm -rf "$out" && mkdir -p "$out"
qemu-system-aarch64 -M virt -cpu cortex-a72 -smp 4 -m 3072 \
    -kernel "$V/vmlinuz" -initrd "$V/initrd.lz4" \
    -append "rdinit=/preinit dev=/dev/vda1 console=ttyAMA0 loglevel=6 systemd.show_status=1" \
    -drive file="$V/card.img",if=none,id=card,format=raw -device virtio-blk-pci,drive=card \
    -device virtio-gpu-pci -device qemu-xhci -device usb-kbd -device usb-tablet \
    -netdev user,id=n -device virtio-net-pci,netdev=n \
    -display none -serial "file:$out/serial.log" \
    -qmp "unix:$out/qmp.sock,server,nowait" -daemonize -pidfile "$out/qemu.pid"
shot() {
    python3 - "$out/qmp.sock" "$1" <<'P'
import json, socket, sys
s = socket.socket(socket.AF_UNIX); s.connect(sys.argv[1]); f = s.makefile("rw")
f.readline()
for cmd in ({"execute": "qmp_capabilities"},
            {"execute": "screendump", "arguments": {"filename": sys.argv[2], "format": "png"}}):
    f.write(json.dumps(cmd) + "\n"); f.flush(); f.readline()
P
}
n=0
while [ $((n * 30)) -lt "$secs" ] && kill -0 "$(cat "$out/qemu.pid")" 2>/dev/null; do
    sleep 30
    n=$((n + 1))
    shot "$out/shot-$(printf %03d $n).png" 2>/dev/null || true
done
kill "$(cat "$out/qemu.pid")" 2>/dev/null || true
echo "ran $((n * 30)) s"
