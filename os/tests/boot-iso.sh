#!/bin/bash
# Boot an Ollie OS ISO in a virtual machine (UEFI, like a Steam Deck) and
# take screenshots, to see that it reaches the desktop.
#
#   os/tests/boot-iso.sh ISO OUTDIR [MINUTES]
#
# Writes OUTDIR/serial.log and OUTDIR/screen-NN.png every 2 minutes.
# Without KVM the machine is emulated, so give it 20+ minutes.
set -euo pipefail

iso="$(realpath "${1:?usage: boot-iso.sh ISO OUTDIR [MINUTES]}")"
out="$(realpath -m "${2:?}")"
minutes="${3:-25}"
mkdir -p "$out"

vars="$out/OVMF_VARS.fd"
cp /usr/share/OVMF/OVMF_VARS_4M.fd "$vars"
accel=tcg
[[ -w /dev/kvm ]] && accel=kvm

mon="$out/monitor.sock"
rm -f "$mon"
qemu-system-x86_64 \
    -machine q35,accel=$accel -cpu max -smp 4 -m 6144 \
    -drive if=pflash,format=raw,readonly=on,file=/usr/share/OVMF/OVMF_CODE_4M.fd \
    -drive if=pflash,format=raw,file="$vars" \
    -cdrom "$iso" -boot d \
    -device virtio-vga -display none \
    -serial file:"$out/serial.log" \
    -monitor unix:"$mon",server,nowait \
    -nic user,model=virtio-net-pci &
qemu=$!
trap 'kill $qemu 2>/dev/null || true' EXIT

for ((i = 1; i <= minutes / 2; i++)); do
    sleep 120
    kill -0 $qemu 2>/dev/null || { echo "VM stopped"; break; }
    printf 'screendump %s/screen-%02d.ppm\n' "$out" "$i" | socat - UNIX-CONNECT:"$mon" >/dev/null 2>&1 || true
    sleep 2
    if [[ -f "$out/screen-$(printf %02d "$i").ppm" ]]; then
        convert "$out/screen-$(printf %02d "$i").ppm" "$out/screen-$(printf %02d "$i").png" && rm -f "$out/screen-$(printf %02d "$i").ppm"
    fi
    echo "screenshot $i taken"
done
