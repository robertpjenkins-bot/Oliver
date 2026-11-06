#!/bin/bash
# Make an Ollie OS ISO start with Secure Boot on (after mkarchiso, by build.sh):
#
#   bash os/secureboot-iso.sh ISO KEYDIR
#
# Its start-up files get shim, MokManager, and systemd-boot and Linux signed
# with the key in KEYDIR (made there if it has none; os/out/secureboot by
# default: the same Builder keeps the same key, so a computer that knows it once
# knows every USB made there):
#   - the FAT image the ISO carries as its EFI partition (what computers start
#     from when the ISO is written to a USB drive whole, or burnt to a DVD);
#   - the same files in the ISO's own folders (what USB Maker copies onto its
#     FAT32 drive).
# The key also goes on the ISO, in /ollie/secureboot, for the installer to sign
# the Ollie OS it installs. See os/profile/airootfs/usr/local/lib/ollie/secureboot.
#
# xorriso is used in small steps, each checked (its partition table, and the
# ISO's dates, which its UUID is made from: the live system finds its USB by
# it): changing many files at once can make it add a stray partition. If the
# folders cannot be done, only the EFI partition is (the important part).
set -euo pipefail
iso="$(readlink -f "${1:?ISO}")"
keydir="${2:?KEYDIR}"
os="$(cd "$(dirname "$0")" && pwd)"
sb="$os/profile/airootfs/usr/local/lib/ollie/secureboot"
export MTOOLS_SKIP_CHECK=1
for tool in xorriso mcopy mkfs.fat sbsign openssl; do
    command -v "$tool" >/dev/null || { echo "Secure Boot: $tool is missing" >&2; exit 1; }
done

bash "$sb" key "$keydir"
work="$(mktemp -d)"
[[ -n "${OLLIE_SB_DEBUG:-}" ]] && echo "Secure Boot: work folder $work" || trap 'rm -rf -- "$work"' EXIT

dates() { xorriso -indev "$1" -pvd_info 2>/dev/null | sed -n 's/^\(Creation\|Modif\.\) Time *: *//p'; }
read -r created modified < <(dates "$iso" | paste -sd' ')
# step IN OUT XORRISO-ARGS...: one checked xorriso step
step() {
    local in="$1" out="$2" parts
    shift 2
    rm -f "$out"
    # (What is changed comes after "replay", so the new partition wins.)
    xorriso -indev "$in" -outdev "$out" \
        -volume_date c "$created" -volume_date m "$modified" -volume_date uuid "$modified" \
        -boot_image any replay "$@" >/dev/null 2>&1 || { echo "  (xorriso failed)" >&2; return 1; }
    parts="$(xorriso -indev "$out" -report_el_torito as_mkisofs 2>/dev/null | grep -c '^-append_partition')"
    [[ $parts == 1 ]] || { echo "  (a stray partition)" >&2; return 1; }
    [[ "$(dates "$out")" == "$(dates "$iso")" ]] || { echo "  (the dates changed)" >&2; return 1; }
}

# 1. The EFI partition image, signed, as a new FAT image with room for shim.
mkdir -p "$work/boot" "$work/esp" "$work/key" "$work/tree"
xorriso -osirrox on -indev "$iso" -extract_boot_images "$work/boot/" >/dev/null 2>&1
img="$(find "$work/boot" -name '*part*2*' -o -name '*uefi*' | grep -v -i bios | head -n 1)"
[[ -n $img && -s $img ]] || { echo "Secure Boot: the ISO has no EFI start-up image" >&2; exit 1; }
mcopy -s -p -n -i "$img" '::*' "$work/esp/"
bash "$sb" esp "$work/esp" "$keydir"
# (Whole MiB: other sizes make xorriso pad it with a stray partition. The name,
# OLLIE_OS, is what MokManager shows.)
kib=$(((($(du -sk "$work/esp" | cut -f1) + 4096 + 1023) / 1024) * 1024))
mkfs.fat -C -n OLLIE_OS "$work/esp.img" "$kib" >/dev/null
mcopy -s -p -i "$work/esp.img" "$work/esp/"* ::/

# 2. The key, for the installer.
cp "$keydir/ollie.key" "$keydir/ollie.crt" "$keydir/ollie.cer" "$work/key/"
echo "Secure Boot key for this Ollie OS USB: $(openssl x509 -in "$keydir/ollie.crt" -noout -fingerprint -sha256 | cut -d= -f2)" >"$work/key/README"

# 3. The same files in the ISO's folders (for USB Maker), signed.
tree=""
if xorriso -osirrox on -indev "$iso" -extract /EFI "$work/tree/EFI" -extract /loader "$work/tree/loader" >/dev/null 2>&1 &&
    [[ -d $work/tree/EFI && -d $work/tree/loader/entries ]]; then
    kernels=()
    while read -r k; do
        [[ -n $k ]] || continue
        mkdir -p "$work/tree$(dirname "$k")"
        xorriso -osirrox on -indev "$iso" -extract "$k" "$work/tree$k" >/dev/null 2>&1 && kernels+=("$k")
    done < <(sed -n 's/^linux[[:space:]]\+\(.*[^[:space:]]\)[[:space:]]*$/\1/p' "$work/tree/loader/entries"/*.conf | sort -u)
    chmod -R u+w "$work/tree"
    if bash "$sb" esp "$work/tree" "$keydir"; then
        kmaps=()
        for k in "${kernels[@]}"; do kmaps+=(-map "$work/tree$k" "$k"); done
        if step "$iso" "$work/a.iso" -map "$work/tree/EFI" /EFI -map "$work/tree/OLLIE-OS-KEY.cer" /OLLIE-OS-KEY.cer &&
            { ((${#kmaps[@]} == 0)) && mv "$work/a.iso" "$work/b.iso" || step "$work/a.iso" "$work/b.iso" "${kmaps[@]}"; }; then
            tree="$work/b.iso"
        else
            echo "Secure Boot: the ISO's own folders were left as they are (only USB Maker uses them)." >&2
        fi
        rm -f "$work/a.iso"
    fi
fi

# 4. The EFI partition and the key.
if ! step "${tree:-$iso}" "$iso.sb" -map "$work/key" /ollie/secureboot -append_partition 2 0xef "$work/esp.img"; then
    rm -f "$iso.sb"
    echo "Secure Boot: the ISO could not be made again" >&2
    exit 1
fi
mv -f "$iso.sb" "$iso"
echo "Secure Boot: the ISO starts with Secure Boot on (key: $(openssl x509 -in "$keydir/ollie.crt" -noout -subject | sed 's/.*CN *= *//'))."
