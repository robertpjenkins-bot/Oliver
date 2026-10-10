#!/usr/bin/env bash
# Builds a bootable ISO of an OS designed in OS Maker 2.0.
#
# It starts from archiso's "releng" profile (the one the official Arch Linux
# ISO is built from), adds the apps you picked and lays your desktop over it.
#
# Run it on Arch Linux as root:   sudo bash build.sh
# OS Maker can also run it for you inside an Arch Linux container (Docker or
# Podman), which is how it works on other distributions.
set -euo pipefail

# --- filled in by OS Maker ---
OS_NAME=@OS_NAME@
ISO_NAME=@ISO_NAME@
MULTILIB=@MULTILIB@
# -----------------------------

here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
work="${WORK:-${here}/work}"
out="${here}/out"
profile="${work}/profile"
releng=/usr/share/archiso/configs/releng

say() { printf '\n\033[1m==> %s\033[0m\n' "$*"; }
die() { printf '\nerror: %s\n' "$*" >&2; exit 1; }

[ "$(id -u)" -eq 0 ] || die "this needs to run as root: sudo bash $0"

if ! command -v mkarchiso >/dev/null 2>&1; then
	[ -f /etc/arch-release ] || die "this needs Arch Linux (or let OS Maker use Docker/Podman)"
	say "installing archiso"
	if [ "${OSMAKER_CONTAINER:-0}" = 1 ]; then
		# A throwaway container: bring everything up to date first.
		pacman -Syu --noconfirm --needed archiso
	else
		pacman -S --noconfirm --needed archiso
	fi
fi
[ -d "${releng}" ] || die "archiso is installed but ${releng} is missing"

say "preparing ${OS_NAME}"
rm -rf "${work}"
mkdir -p "${work}" "${out}"
cp -a "${releng}" "${profile}"

say "adding your apps"
if [ "${MULTILIB}" = 1 ]; then
	# Steam, Wine and other 32-bit programs live in the multilib repository.
	sed -i '/^#\[multilib\]/,/^#Include/ s/^#//' "${profile}/pacman.conf"
fi
# Check every package against Arch's repositories first. Packages get renamed
# or dropped now and then; one missing app should be skipped, not stop the
# whole build at the very end.
db="${work}/pacman-db"
known="${work}/known-packages"
mkdir -p "${db}"
if pacman --config "${profile}/pacman.conf" --dbpath "${db}" -Sy >/dev/null; then
	{
		pacman --config "${profile}/pacman.conf" --dbpath "${db}" -Slq
		pacman --config "${profile}/pacman.conf" --dbpath "${db}" -Sg | awk '{ print $1 }'
	} | sort -u > "${known}"
else
	echo "could not check the package list; trying them all"
fi
skipped=()
while IFS= read -r pkg; do
	[ -n "${pkg}" ] || continue
	if [ ! -s "${known}" ] || grep -qxF "${pkg}" "${known}"; then
		echo "${pkg}" >> "${profile}/packages.x86_64"
	else
		skipped+=("${pkg}")
	fi
done < <(grep -v '^\s*#' "${here}/packages.txt")
if [ "${#skipped[@]}" -gt 0 ]; then
	say "skipping these, Arch Linux no longer has them: ${skipped[*]}"
fi
sort -u -o "${profile}/packages.x86_64" "${profile}/packages.x86_64"

say "applying your desktop"
# releng logs root straight into a console and runs its own networking; the
# designed OS logs its user into the desktop and uses NetworkManager instead.
rm -rf "${profile}/airootfs/etc/systemd/system/getty@tty1.service.d"
rm -f "${profile}/airootfs/etc/systemd/system/multi-user.target.wants/systemd-networkd.service" \
      "${profile}/airootfs/etc/systemd/system/multi-user.target.wants/iwd.service" \
      "${profile}/airootfs/etc/systemd/system/network-online.target.wants/systemd-networkd-wait-online.service"
cp -a "${here}/airootfs/." "${profile}/airootfs/"

sed -i \
	-e "s|^iso_name=.*|iso_name=\"${ISO_NAME}\"|" \
	-e "s|^iso_publisher=.*|iso_publisher=\"Made with OS Maker 2.0\"|" \
	-e "s|^iso_application=.*|iso_application=\"${OS_NAME} Live\"|" \
	"${profile}/profiledef.sh"
cat "${here}/permissions.sh" >> "${profile}/profiledef.sh"

# Put the OS name on the boot menu.
for d in grub syslinux efiboot; do
	[ -d "${profile}/${d}" ] || continue
	grep -rlZ 'Arch Linux' "${profile}/${d}" | xargs -0 -r sed -i \
		-e "s|Arch Linux install medium|${OS_NAME}|g" \
		-e "s|Arch Linux|${OS_NAME}|g"
done

say "building the ISO (this downloads every package and takes a while)"
mkarchiso -v -w "${work}/build" -o "${out}" "${profile}"

# Hand the ISO back to whoever asked for it rather than leaving it owned by root.
owner="${HOST_UID:-${SUDO_UID:-${PKEXEC_UID:-}}}"
if [ -n "${owner}" ]; then
	chown -R "${owner}:${HOST_GID:-${SUDO_GID:-${owner}}}" "${out}"
fi
[ "${KEEP_WORK:-0}" = 1 ] || rm -rf "${work}"

iso="$(ls -t "${out}"/*.iso | head -n 1)"
say "done: ${iso}"
