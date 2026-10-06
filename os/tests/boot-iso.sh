#!/bin/bash
# Boot an Ollie OS ISO in a virtual machine and record what happens.
#
#   os/tests/boot-iso.sh ISO OUTDIR [MINUTES]
#
# Boots the ISO's own kernel and boot options with the console copied to a
# serial log, takes a screenshot every 30 seconds, and stops once the desktop
# has had a minute to settle (or after MINUTES). Leaves in OUTDIR:
#   serial.log            everything the system printed while starting
#   screen-NNN.png        the screenshots
#   summary.txt           what started, what failed
# Exits non-zero if the system never reached the desktop.
set -euo pipefail

iso="$(realpath "${1:?usage: boot-iso.sh ISO OUTDIR [MINUTES]}")"
out="$(realpath -m "${2:?}")"
minutes="${3:-15}"
mkdir -p "$out"
rm -f "$out"/screen-*.png "$out/serial.log" "$out/summary.txt"

# The kernel, initramfs and boot options, exactly as the ISO's UEFI menu has them.
boot="$out/bootfiles"
rm -rf "$boot"
mkdir -p "$boot"
xorriso -osirrox on -indev "$iso" \
    -extract /arch/boot/x86_64/vmlinuz-linux "$boot/vmlinuz-linux" \
    -extract /arch/boot/x86_64/initramfs-linux.img "$boot/initramfs-linux.img" \
    -extract /loader/entries "$boot/entries" >/dev/null 2>&1
entry="$(ls "$boot"/entries/*.conf | head -n1)"
options="$(awk '$1 == "options" {$1 = ""; print substr($0, 2)}' "$entry")"
echo "boot options: $options"

# With KVM the guest gets the host's CPU as it is; "max" can expose features
# KVM cannot back, which hangs the guest kernel early.
accel=tcg cpu=max
if [[ -w /dev/kvm ]]; then accel=kvm cpu=host; fi
echo "acceleration: $accel"

mon="$out/monitor.sock"
rm -f "$mon"
qemu-system-x86_64 \
    -machine q35,accel=$accel -cpu $cpu -smp 2 -m 4096 \
    -kernel "$boot/vmlinuz-linux" -initrd "$boot/initramfs-linux.img" \
    -append "$options console=tty0 console=ttyS0,115200 loglevel=6 systemd.show_status=1" \
    -cdrom "$iso" \
    -device virtio-vga -display none \
    -serial file:"$out/serial.log" \
    -monitor unix:"$mon",server,nowait \
    -nic user,model=virtio-net-pci &
qemu=$!
trap 'kill $qemu 2>/dev/null || true' EXIT

shot() {
    local ppm="$out/screen-$1.ppm"
    printf 'screendump %s\n' "$ppm" | socat - UNIX-CONNECT:"$mon" >/dev/null 2>&1 || return 0
    sleep 1
    [[ -f "$ppm" ]] && convert "$ppm" "$out/screen-$1.png" && rm -f "$ppm"
}

reached=""
for ((s = 30, n = 1; s <= minutes * 60; s += 30, n++)); do
    sleep 30
    kill -0 $qemu 2>/dev/null || {
        echo "the virtual machine stopped"
        break
    }
    shot "$(printf %03d "$n")"
    if [[ -z "$reached" ]] && grep -q 'Reached target.*Graphical Interface' "$out/serial.log"; then
        reached=$s
        echo "reached the desktop target after ${s}s"
    fi
    # Give the desktop a minute to appear after the system says it is ready.
    if [[ -n "$reached" ]] && ((s >= reached + 60)); then
        break
    fi
done
kill $qemu 2>/dev/null || true
wait $qemu 2>/dev/null || true

# Each grep may find nothing; that must not end the script before it reports.
{
    echo "== Reached"
    grep -aoE 'Reached target [^.]*' "$out/serial.log" | sed 's/\x1b\[[0-9;]*m//g' | sort -u || true
    echo
    echo "== Failed"
    grep -aE 'FAILED|Failed to start|failed with result|Dependency failed|ERROR|emergency mode|Kernel panic' "$out/serial.log" |
        sed 's/\x1b\[[0-9;]*m//g' | sort -u | head -n 60 || true
    echo
    echo "== Ollie OS services"
    grep -aE 'ollie-|Prepare the Ollie OS|package signing keys|Simple Desktop Display Manager|sddm' "$out/serial.log" |
        sed 's/\x1b\[[0-9;]*m//g' | sort -u | head -n 30 || true
    echo
    echo "== Last lines the system printed"
    sed 's/\x1b\[[0-9;]*m//g' "$out/serial.log" | tr -d '\r' | tail -n 60 || true
} >"$out/summary.txt"
cat "$out/summary.txt"

[[ -n "$reached" ]]
