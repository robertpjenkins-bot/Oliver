#!/usr/bin/env bash
# Builds the Annie OS live ISO.
#
# Run on Arch Linux as root:   sudo ./build.sh
# Needs:                       pacman -S archiso python-pillow
#
# Everything is fetched from the Arch package mirrors; nothing comes from
# GitHub or the AUR.
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
out="${OUTDIR:-${here}/out}"
work="${WORKDIR:-${here}/work}"
profile="${work}/profile"
max_bytes=$((25 * 1000 * 1000 * 1000))   # the ISO must be 25 GB or under

say() { printf '\033[1;35m==>\033[0m %s\n' "$*"; }
die() { printf '\033[1;31merror:\033[0m %s\n' "$*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "run as root: sudo $0"
[ -d /usr/share/archiso/configs/releng ] || die "archiso is not installed: pacman -S archiso"

say "preparing the profile"
rm -rf "$work"
mkdir -p "$work" "$out"
cp -a /usr/share/archiso/configs/releng "$profile"

# Name and brand the ISO.
sed -i \
	-e 's/^iso_name=.*/iso_name="annie-os"/' \
	-e 's/^iso_label=.*/iso_label="ANNIE_$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y%m)"/' \
	-e 's/^iso_publisher=.*/iso_publisher="Annie OS"/' \
	-e 's/^iso_application=.*/iso_application="Annie OS Live"/' \
	"$profile/profiledef.sh"
grep -rlZ 'Arch Linux' "$profile"/{grub,syslinux,efiboot} 2>/dev/null \
	| xargs -0 -r sed -i 's/Arch Linux/Annie OS/g'

# Steam and its 32-bit libraries live in [multilib].
if ! grep -q '^\[multilib\]' "$profile/pacman.conf"; then
	printf '\n[multilib]\nInclude = /etc/pacman.d/mirrorlist\n' >>"$profile/pacman.conf"
fi

# Packages.
grep -Ev '^\s*(#|$)' "$here/packages.txt" >>"$profile/packages.x86_64"
sort -u -o "$profile/packages.x86_64" "$profile/packages.x86_64"

# NetworkManager replaces the live ISO's iwd + systemd-networkd.
rm -f "$profile"/airootfs/etc/systemd/system/multi-user.target.wants/{iwd,systemd-networkd}.service \
      "$profile"/airootfs/etc/systemd/system/sockets.target.wants/systemd-networkd.socket \
      "$profile"/airootfs/etc/systemd/system/network-online.target.wants/systemd-networkd-wait-online.service

# The Annie files.
cp -a "$here/airootfs/." "$profile/airootfs/"
install -Dm755 "$here/appimage/AnnieDiskWriter/annie-disk-writer" "$profile/airootfs/usr/local/bin/annie-disk-writer"
install -Dm755 "$here/appimage/AnnieDiskWriter/annie-disk-helper" "$profile/airootfs/usr/lib/annie/annie-disk-helper"
install -Dm644 "$here/appimage/AnnieDiskWriter/annie-disk-writer.desktop" "$profile/airootfs/usr/share/applications/annie-disk-writer.desktop"
install -Dm644 "$here/appimage/AnnieDiskWriter/annie-disk-writer.svg" "$profile/airootfs/usr/share/icons/hicolor/scalable/apps/annie-disk-writer.svg"
install -Dm644 "$here/packages.txt" "$profile/airootfs/usr/share/annie/packages.txt"

# The disk writer also ships as a portable AppImage, when appimagetool is here.
OUTDIR="$work/appimage" "$here/appimage/build-appimage.sh"
if [ -f "$work/appimage/AnnieDiskWriter-x86_64.AppImage" ]; then
	install -Dm755 "$work/appimage/AnnieDiskWriter-x86_64.AppImage" \
		"$profile/airootfs/usr/share/annie/AnnieDiskWriter-x86_64.AppImage"
	cp "$work/appimage/AnnieDiskWriter-x86_64.AppImage" "$out/"
fi

# Oliver (in the folder above) runs Windows programs with Wine or Proton.
if command -v go >/dev/null 2>&1 && [ -f "$here/../go.mod" ]; then
	say "building Oliver"
	(cd "$here/.." && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" \
		-o "$profile/airootfs/usr/local/bin/oliver" ./cmd/oliver)
fi

# mkarchiso resets file modes, so list everything that must stay executable.
{
	for f in "$profile"/airootfs/usr/local/bin/* "$profile"/airootfs/usr/lib/annie/* \
	         "$profile"/airootfs/usr/share/annie/*.AppImage; do
		[ -f "$f" ] || continue
		printf '  ["%s"]="0:0:755"\n' "${f#"$profile/airootfs"}"
	done
} >"$work/perms"
sed -i "/^file_permissions=(/r $work/perms" "$profile/profiledef.sh"

say "building the ISO (this takes a while)"
mkarchiso -v -w "$work/mkarchiso" -o "$out" "$profile"

iso=$(ls -t "$out"/annie-os-*.iso | head -n1)
size=$(stat -c %s "$iso")
if [ "$size" -gt "$max_bytes" ]; then
	die "$iso is $((size / 1000000)) MB, over the 25 GB limit"
fi
say "done: $iso ($((size / 1000000)) MB, under the 25 GB limit)"
