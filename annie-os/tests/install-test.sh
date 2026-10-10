#!/bin/bash
# Install Annie OS onto an empty disk and start the installed system, in a
# virtual machine (UEFI, like the Steam Machine):
#   1. boot the ISO with annie.selftest=install: it runs the installer onto
#      the empty 30 GB disk, answering its questions, and switches off;
#   2. start the installed system from that disk, with its own kernel,
#      initramfs and boot options, and check it gets past mounting its root
#      filesystem to the login screen.
#
#   os/tests/install-test.sh ISO OUTDIR
set -uo pipefail

iso="$(realpath "${1:?usage: install-test.sh ISO OUTDIR}")"
out="$(realpath -m "${2:?}")"
mkdir -p "$out"
disk="$out/disk.img"
rm -f "$disk" "$out"/*.log
truncate -s 30G "$disk"

accel=tcg cpu=max
if [[ -w /dev/kvm ]]; then accel=kvm cpu=host; fi
ovmf=""
for f in /usr/share/ovmf/OVMF.fd /usr/share/OVMF/OVMF_CODE.fd /usr/share/qemu/OVMF.fd; do
    [[ -f "$f" ]] && { ovmf="$f"; break; }
done
[[ -n "$ovmf" ]] || { echo "no OVMF (UEFI firmware) found"; exit 1; }

boot="$out/bootfiles"
rm -rf "$boot"
mkdir -p "$boot"
xorriso -osirrox on -indev "$iso" \
    -extract /arch/boot/x86_64/vmlinuz-linux "$boot/vmlinuz-linux" \
    -extract /arch/boot/x86_64/initramfs-linux.img "$boot/initramfs-linux.img" \
    -extract /loader/entries "$boot/entries" >/dev/null 2>&1
options="$(awk '$1 == "options" {$1 = ""; print substr($0, 2)}' "$(ls "$boot"/entries/*.conf | head -n1)")"

ok=true
echo "== 1. Installing onto the empty disk"
timeout 2400 qemu-system-x86_64 -machine q35,accel=$accel -cpu $cpu -smp 2 -m 4096 -bios "$ovmf" \
    -kernel "$boot/vmlinuz-linux" -initrd "$boot/initramfs-linux.img" \
    -append "$options console=tty0 console=ttyS0,115200 annie.selftest=install" \
    -cdrom "$iso" -drive "file=$disk,format=raw,if=virtio" -device virtio-vga -display none \
    -serial file:"$out/install.log" -nic user,model=virtio-net-pci
grep -a 'ANNIE-INSTALL-TEST' "$out/install.log" | tr -d '\r' | sed 's/^.*ANNIE-INSTALL-TEST: //' | awk '!seen[$0]++'
grep -aq 'ANNIE-INSTALL-TEST: installer finished' "$out/install.log" || { echo "FAIL  the installer did not finish"; exit 1; }
echo "ok    the installer finished"

echo "== 2. Starting the installed system"
# Its kernel, initramfs and boot options, from its own boot partition.
loop="$(sudo losetup -fP --show "$disk")"
inst="$out/installed-boot"
rm -rf "$inst"
mkdir -p "$inst" "$out/mnt"
sudo mount -o ro "${loop}p1" "$out/mnt"
sudo cp "$out/mnt/vmlinuz-linux" "$out/mnt/initramfs-linux.img" "$out/mnt/loader/entries/annie.conf" "$inst/"
sudo umount "$out/mnt"
sudo losetup -d "$loop"
sudo chmod a+r "$inst"/*
cat "$inst/annie.conf"
ioptions="$(awk '$1 == "options" {$1 = ""; print substr($0, 2)}' "$inst/annie.conf")"
ioptions="${ioptions/ quiet/}"

qemu-system-x86_64 -machine q35,accel=$accel -cpu $cpu -smp 2 -m 4096 -bios "$ovmf" \
    -kernel "$inst/vmlinuz-linux" -initrd "$inst/initramfs-linux.img" \
    -append "$ioptions console=tty0 console=ttyS0,115200" \
    -drive "file=$disk,format=raw,if=virtio" -device virtio-vga -display none \
    -serial file:"$out/installed.log" -nic user,model=virtio-net-pci &
qemu=$!
trap 'kill $qemu 2>/dev/null || true' EXIT

result=""
for ((t = 0; t < 600; t += 10)); do
    sleep 10
    if grep -aqE 'Failed to mount .*sysroot|emergency mode|Cannot open access to console' "$out/installed.log"; then result=FAIL; sleep 5; break; fi
    if grep -aqE 'Started .*Simple Desktop Display Manager|Reached target .*Graphical' "$out/installed.log"; then result=PASS; break; fi
done
kill $qemu 2>/dev/null || true
echo "-- what the installed system printed while starting (problems):"
sed 's/\x1b\[[0-9;]*m//g' "$out/installed.log" | tr -d '\r' |
    grep -aiE 'sysroot|btrfs|mount|fail|error|emergency|root=|initrd|Reached target|sddm' | head -n 80
case "$result" in
PASS) echo "ok    the installed system started and reached the login screen" ;;
FAIL) echo "FAIL  the installed system could not start (see above)"; ok=false ;;
*) echo "FAIL  the installed system did not reach the login screen in 10 minutes"; ok=false ;;
esac
$ok
