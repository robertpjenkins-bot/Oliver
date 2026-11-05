#!/bin/bash
# Edit an existing, working Ollie OS ISO instead of building a new one:
# everything in it stays exactly as it is (same program versions), except:
#   - Google Chrome is removed and Firefox added, from the Arch Linux Archive
#     snapshot of the day the ISO was built, so it fits the programs already
#     in it;
#   - the Ollie OS files listed in CHANGED (from os/profile/airootfs) are
#     added or replaced.
# The ISO keeps its ID, so its boot menus still find it.
#
# Runs as root in an Arch Linux container (see os-iso-edit.yml):
#   edit-iso.sh IN.iso OUT.iso CHANGED-FILE-LIST
set -euo pipefail

in="$(realpath "$1")"
out="$(realpath -m "$2")"
changed="$(realpath "$3")"
src=/src/os/profile/airootfs
work=/work
root="$work/root"

step() { printf '\n==> %s\n' "$*"; }

step "Tools"
sed -i '/^NoExtract/d' /etc/pacman.conf
pacman -Sy --noconfirm --needed squashfs-tools libisoburn arch-install-scripts util-linux >/dev/null
pacman-key --init >/dev/null 2>&1
pacman-key --populate archlinux >/dev/null 2>&1

step "Reading the ISO"
uuid="$(blkid -o value -s UUID "$in")"     # e.g. 2026-10-07-23-26-55-00
label="$(blkid -o value -s LABEL "$in")"
echo "ISO ID $uuid, label $label"
rm -rf "$work"
mkdir -p "$work/iso"
xorriso -osirrox on -indev "$in" -extract /arch "$work/iso/arch" >/dev/null 2>&1
unsquashfs -no-progress -d "$root" "$work/iso/arch/x86_64/airootfs.sfs" >/dev/null

step "Chrome out, Firefox in"
y="${uuid:0:4}" m="${uuid:5:2}" d="${uuid:8:2}"
conf="$root/var/tmp/edit-pacman.conf"
sed -e "s|^Include *= */etc/pacman.d/mirrorlist|Server = https://archive.archlinux.org/repos/$y/$m/$d/\$repo/os/\$arch|" \
    -e 's|^#\?CacheDir.*|CacheDir = /var/tmp/edit-pkgcache/|' "$root/etc/pacman.conf" >"$conf"
grep -q '^CacheDir' "$conf" || sed -i '/^\[options\]/a CacheDir = /var/tmp/edit-pkgcache/' "$conf"
# arch-chroot wants the root to be a mount point.
mount --bind "$root" "$root"
had_gnupg=false
[[ -d "$root/etc/pacman.d/gnupg" ]] && had_gnupg=true
mkdir -p "$root/etc/pacman.d/gnupg"
mount --bind /etc/pacman.d/gnupg "$root/etc/pacman.d/gnupg"
trap 'umount "$root/etc/pacman.d/gnupg" 2>/dev/null || true; umount "$root" 2>/dev/null || true' EXIT
if arch-chroot "$root" pacman -Q google-chrome >/dev/null 2>&1; then
    arch-chroot "$root" pacman --config /var/tmp/edit-pacman.conf -Rns --noconfirm google-chrome
fi
arch-chroot "$root" pacman --config /var/tmp/edit-pacman.conf -Sy --noconfirm --needed firefox
umount "$root/etc/pacman.d/gnupg"
umount "$root"
trap - EXIT
rm -rf "$root/var/tmp/edit-pkgcache" "$conf" "$root/var/lib/pacman/sync"/*
$had_gnupg || rmdir "$root/etc/pacman.d/gnupg"

step "Ollie OS files"
while IFS= read -r f; do
    [[ -n "$f" ]] || continue
    rel="${f#os/profile/airootfs/}"
    if [[ -L "$src/$rel" ]]; then
        mkdir -p "$(dirname "$root/$rel")"
        ln -sfn "$(readlink "$src/$rel")" "$root/$rel"
    elif [[ -f "$src/$rel" ]]; then
        mode=0644
        [[ "$rel" == usr/local/bin/* ]] && mode=0755
        install -D -m "$mode" "$src/$rel" "$root/$rel"
    else
        rm -f "$root/$rel"
    fi
    echo "  $rel"
done <"$changed"
# Files the cursor package leaves out on purpose stay out after updates too.
grep -q '^NoExtract = usr/share/icons/Bibata' "$root/etc/pacman.conf" ||
    sed -i '/^\[options\]/a NoExtract = usr/share/icons/Bibata-* !usr/share/icons/Bibata-Modern-Ice/*' "$root/etc/pacman.conf"

step "Sealing the system image"
rm -f "$work/airootfs.sfs"
mksquashfs "$root" "$work/airootfs.sfs" -noappend -comp xz -Xbcj x86 -b 1M -Xdict-size 1M -no-progress >/dev/null
(cd "$work" && sha512sum airootfs.sfs >airootfs.sha512)
updates=(-update "$work/airootfs.sfs" /arch/x86_64/airootfs.sfs)
[[ -f "$work/iso/arch/x86_64/airootfs.sha512" ]] && updates+=(-update "$work/airootfs.sha512" /arch/x86_64/airootfs.sha512)
if [[ -f "$work/iso/arch/pkglist.x86_64.txt" ]]; then
    pacman --root "$root" -Q >"$work/pkglist.x86_64.txt"
    updates+=(-update "$work/pkglist.x86_64.txt" /arch/pkglist.x86_64.txt)
fi

step "Writing the edited ISO"
rm -f "$out"
xorriso -indev "$in" -outdev "$out" -boot_image any replay \
    -volid "$label" -volume_date uuid "${uuid//-/}" "${updates[@]}" 2>&1 | grep -iE 'error|fail|warn' || true
new_uuid="$(blkid -o value -s UUID "$out")"
echo "edited ISO: $(($(stat -c %s "$out") / 1048576)) MB, ID $new_uuid"
[[ "$new_uuid" == "$uuid" ]] || { echo "the ISO ID changed ($uuid -> $new_uuid); its boot menus would not find it" >&2; exit 1; }
(cd "$(dirname "$out")" && sha256sum "$(basename "$out")" >"$(basename "$out").sha256")
