#!/bin/bash
# Build the Annie OS ISO.
#
# Runs as root inside an Arch Linux container (see .github/workflows/os-iso.yml):
#
#   docker run --rm --privileged -v "$PWD:/src" -w /src archlinux:latest bash os/build.sh
#
# The ISO and its checksum land in os/out/.
#
# The profile starts from archiso's own "releng" profile for the boot loader
# configuration, which tracks the installed archiso version, and layers
# os/profile on top for everything that makes the system Annie OS.
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd)"
os="$repo/os"
out="$os/out"
work=/tmp/annie-work
profile=/tmp/annie-profile

step() { printf '\n\e[1;36m==> %s\e[0m\n' "$*"; }
# Package lists: one name per line, # starts a comment.
list() { sed -e 's/#.*//' -e 's/[[:space:]]//g' -e '/^$/d' "$1"; }

step "Preparing the build container"
# The container image strips man pages and translations from everything it
# installs; the ISO must not inherit that.
sed -i '/^NoExtract/d' /etc/pacman.conf
# Steam, Wine and the 32-bit drivers live in multilib. Uncomment it, or add it
# if this pacman.conf has no multilib section at all.
sed -i '/^#\[multilib\]/,/^#Include/ s/^#//' /etc/pacman.conf
if ! grep -q '^\[multilib\]' /etc/pacman.conf; then
    printf '\n[multilib]\nInclude = /etc/pacman.d/mirrorlist\n' >>/etc/pacman.conf
fi
pacman-key --init >/dev/null
pacman -Sy --noconfirm --needed archlinux-keyring
pacman -Su --noconfirm
# python-pillow and ttf-dejavu draw the unicorn wallpaper and its "Annie OS"
# title (tools/make-wallpaper.py); without a font the letters are left out.
pacman -S --noconfirm --needed archiso git go python-pillow ttf-dejavu sudo

git config --global --add safe.directory "$repo"
version="$(git -C "$repo" describe --tags --always --dirty 2>/dev/null || echo dev)"

step "Assembling the profile"
rm -rf "$profile" "$work"
cp -a /usr/share/archiso/configs/releng "$profile"
rm -rf /tmp/releng-airootfs
mv "$profile/airootfs" /tmp/releng-airootfs
mkdir -p "$profile/airootfs"
# Annie OS ships its own live initramfs hooks (os/profile/airootfs/etc/
# mkinitcpio.conf.d); of releng's root filesystem it only keeps a kernel
# preset, in archiso versions that still provide one.
if [[ -f /tmp/releng-airootfs/etc/mkinitcpio.d/linux.preset ]]; then
    install -D -m 0644 /tmp/releng-airootfs/etc/mkinitcpio.d/linux.preset \
        "$profile/airootfs/etc/mkinitcpio.d/linux.preset"
fi
rm -rf /tmp/releng-airootfs
cp -a "$os/profile/airootfs/." "$profile/airootfs/"

# Name the ISO and boot menus Annie OS.
sed -i \
    -e 's/^iso_name=.*/iso_name="annie-os"/' \
    -e 's/^iso_label="ARCH_/iso_label="ANNIE_/' \
    -e 's|^iso_publisher=.*|iso_publisher="Annie OS"|' \
    -e 's|^iso_application=.*|iso_application="Annie OS live and installer"|' \
    "$profile/profiledef.sh"
# releng's permission list names files that Annie OS leaves out.
sed -i '/^file_permissions=(/,/^)/d' "$profile/profiledef.sh"
find "$profile/efiboot" "$profile/syslinux" "$profile/grub" -type f -name '*speech*' -delete 2>/dev/null || true
grep -rlZ 'Arch Linux' "$profile/efiboot" "$profile/syslinux" "$profile/grub" 2>/dev/null |
    xargs -0r sed -i -e 's/Arch Linux install medium/Annie OS/g' -e 's/Arch Linux/Annie OS/g'

step "Building Oliver $version"
(cd "$repo" && CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X main.version=$version" \
    -o "$profile/airootfs/usr/local/bin/oliver" ./cmd/oliver)
install -D -m 0644 "$repo/packaging/oliver.desktop" "$profile/airootfs/usr/share/applications/oliver.desktop"
install -D -m 0644 "$repo/packaging/oliver.svg" "$profile/airootfs/usr/share/icons/hicolor/scalable/apps/oliver.svg"
install -D -m 0755 "$os/annie-write" "$profile/airootfs/usr/local/bin/annie-write"

step "Building the Annie OS Updater"
# It is in the menu as "Update Annie OS", and the apps self-test runs it.
bash "$os/updater/build-appimage.sh"
install -D -m 0755 "$os/out/Annie-OS-Updater-x86_64.AppImage" \
    "$profile/airootfs/usr/share/annie/Annie-OS-Updater-x86_64.AppImage"

step "Drawing the logo and wallpaper"
for icon in annie-os annie-dc annie-drivers; do  # art/*.svg
    install -D -m 0644 "$os/art/$icon.svg" "$profile/airootfs/usr/share/icons/hicolor/scalable/apps/$icon.svg"
done
images="$profile/airootfs/usr/share/wallpapers/AnnieOS/contents/images"
mkdir -p "$images"
for size in 1280x720 1920x1080 2560x1440 3840x2160; do
    python3 "$os/tools/make-wallpaper.py" "$images/$size.png" "${size%x*}" "${size#*x}"
done
# The BIOS boot menu background; syslinux wants exactly 640x480.
python3 "$os/tools/make-wallpaper.py" "$profile/syslinux/splash.png" 640 480

step "Writing package lists and pacman configuration"
# Firmware for consumer hardware only. Arch splits linux-firmware into parts;
# the NVIDIA part (large, and for cards the Steam Machine and Deck do not
# have), Qualcomm phone and laptop chips (qcom) and the server network cards
# are left out. Without the split, the
# whole package.
firmware=(linux-firmware)
if pacman -Si linux-firmware-amdgpu &>/dev/null; then
    firmware=()
    for part in amdgpu radeon intel realtek mediatek atheros broadcom cirrus other; do
        pacman -Si "linux-firmware-$part" &>/dev/null && firmware+=("linux-firmware-$part")
    done
fi
echo "firmware: ${firmware[*]}"
{
    printf '%s\n' "${firmware[@]}"
    list "$os/profile/packages.x86_64"
    while read -r pkg; do
        if pacman -Si "$pkg" &>/dev/null; then
            echo "$pkg"
        else
            echo "note: optional package $pkg is not in the repositories, leaving it out" >&2
        fi
    done < <(list "$os/profile/packages-optional.x86_64")
} | sort -u >"$profile/packages.x86_64"

# The installed system gets the container's pacman setup (with multilib) and
# its working mirror list. Everything comes from the official repositories.
install -D -m 0644 /etc/pacman.conf "$profile/airootfs/etc/pacman.conf"
install -D -m 0644 /etc/pacman.d/mirrorlist "$profile/airootfs/etc/pacman.d/mirrorlist"
cp /etc/pacman.conf "$profile/pacman.conf"

# Every program Annie OS adds must stay allowed to run: list them all
# (mkarchiso otherwise copies them without that mark).
{
    echo 'file_permissions=('
    echo '  ["/etc/sudoers.d"]="0:0:750"'
    echo '  ["/etc/sudoers.d/10-wheel"]="0:0:440"'
    for f in "$profile"/airootfs/usr/local/bin/* "$profile"/airootfs/usr/share/annie/*.AppImage; do
        [[ -f "$f" ]] && printf '  ["%s"]="0:0:755"\n' "${f#"$profile/airootfs"}"
    done
    echo ')'
} >>"$profile/profiledef.sh"
grep -c '0:0:755' "$profile/profiledef.sh"

step "Building the ISO"
mkdir -p "$out"
rm -f "$out"/annie-os-*.iso "$out"/annie-os-*.iso.sha256
mkarchiso -v -w "$work" -o "$out" "$profile"
rm -rf "$work"

cd "$out"
for iso in annie-os-*.iso; do
    sha256sum "$iso" >"$iso.sha256"
done
step "Done"
ls -lh "$out"
