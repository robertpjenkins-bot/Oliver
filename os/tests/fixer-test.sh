#!/bin/bash
# Boot the ISO with ollie.selftest=fixer: Ollie OS breaks a program and an
# Ollie OS file on purpose, The Fixer repairs them, and the result is
# printed to the serial console (OLLIE-FIXER-TEST: PASS or FAIL).
#
#   os/tests/fixer-test.sh ISO OUTDIR
set -uo pipefail

iso="$(realpath "${1:?usage: fixer-test.sh ISO OUTDIR}")"
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
    -append "$options console=tty0 console=ttyS0,115200 ollie.selftest=fixer" \
    -cdrom "$iso" -device virtio-vga -display none \
    -serial file:"$out/serial.log" -nic user,model=virtio-net-pci &
qemu=$!
trap 'kill $qemu 2>/dev/null || true' EXIT

result=""
for ((t = 0; t < 900; t += 10)); do
    sleep 10
    if grep -aq 'OLLIE-FIXER-TEST: PASS' "$out/serial.log"; then result=PASS; break; fi
    if grep -aq 'OLLIE-FIXER-TEST: FAIL' "$out/serial.log"; then result=FAIL; break; fi
done
grep -a 'OLLIE-FIXER-TEST' "$out/serial.log" | tr -d '\r' | sort -u
echo "The Fixer self-test: ${result:-did not finish in 15 minutes}"
[[ "$result" == PASS ]]
