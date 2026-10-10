#!/bin/bash
# Boot the ISO with annie.selftest=apps: once the desktop is up, Annie OS
# starts Firefox, Dolphin, Konsole, Settings, Annie Paint, Annie Disk Writer
# and an AppImage, and prints to the serial console which of them started
# (ANNIE-APPS-TEST: PASS or FAIL). A screenshot of the end is kept.
#
#   annie-os/tests/apps-test.sh ISO OUTDIR
set -uo pipefail

iso="$(realpath "${1:?usage: apps-test.sh ISO OUTDIR}")"
out="$(realpath -m "${2:?}")"
mkdir -p "$out"
boot="$out/bootfiles"
rm -rf "$boot"
mkdir -p "$boot"
xorriso -osirrox on -indev "$iso" \
    -extract /arch/boot/x86_64/vmlinuz-linux "$boot/vmlinuz-linux" \
    -extract /arch/boot/x86_64/initramfs-linux.img "$boot/initramfs-linux.img" \
    -extract /loader/entries "$boot/entries" >/dev/null 2>&1
options="$(awk '$1 == "options" {$1 = ""; print substr($0, 2)}' "$(ls "$boot"/entries/*.conf | head -n1)")"
accel=tcg cpu=max
if [[ -w /dev/kvm ]]; then accel=kvm cpu=host; fi

qemu-system-x86_64 -machine q35,accel=$accel -cpu $cpu -smp 2 -m 4096 \
    -kernel "$boot/vmlinuz-linux" -initrd "$boot/initramfs-linux.img" \
    -append "$options console=tty0 console=ttyS0,115200 annie.selftest=apps" \
    -cdrom "$iso" -device virtio-vga -display none \
    -serial file:"$out/serial.log" -monitor unix:"$out/monitor.sock",server,nowait \
    -nic user,model=virtio-net-pci &
qemu=$!
trap 'kill $qemu 2>/dev/null || true' EXIT

result=""
for ((t = 0; t < 1500; t += 10)); do
    sleep 10
    if grep -aq 'ANNIE-APPS-TEST: PASS' "$out/serial.log"; then result=PASS; break; fi
    if grep -aq 'ANNIE-APPS-TEST: FAIL' "$out/serial.log"; then result=FAIL; break; fi
done
printf 'screendump %s\n' "$out/end.ppm" | socat - UNIX-CONNECT:"$out/monitor.sock" >/dev/null 2>&1 &&
    sleep 2 && convert "$out/end.ppm" "$out/end.png" 2>/dev/null && rm -f "$out/end.ppm"
grep -a 'ANNIE-APPS-TEST' "$out/serial.log" | tr -d '\r' | awk '!seen[$0]++'
echo "Apps self-test: ${result:-did not finish in 25 minutes}"
[[ "$result" == PASS ]]
