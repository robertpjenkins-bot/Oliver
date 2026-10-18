#!/bin/bash
# Build Ollie OS for ARM (aarch64): the same system as the PC Ollie OS, on
# Arch Linux ARM, as a disk image Etros Maker puts on the card for "Boot
# into Ollie OS".
#
#   sudo bash etros/ollie-arm/build.sh OLLIE_SRC OUT_DIR
#
# OLLIE_SRC is a checkout of the branch with Ollie OS's sources (os/, cmd/,
# packaging/, go.mod). Runs as root on Ubuntu; on a PC (x86_64) it runs the
# ARM programs through QEMU. Needs: qemu-user-static (on a PC), bsdtar,
# e2fsprogs, librsvg2-bin, xz-utils, go.
#
# The image has one Linux partition, labelled rootfs. There is no kernel in
# it: on the card, Ollie OS runs on Batocera's kernel for the board (see
# etros/initrd/etros-hook.sh).
set -euo pipefail

src=$(realpath "$1")
out=$(realpath -m "$2")
work=${WORK:-/mnt/ollie-arm}
root=$work/root
os=$src/os
# The sources belong to the CI user, and this runs as root.
git config --global --add safe.directory '*'
step() { printf '\n\e[1;36m==> %s\e[0m\n' "$*"; }
list() { sed -e 's/#.*//' -e 's/[[:space:]]//g' -e '/^$/d' "$1"; }

# PC-only parts of the PC package list: boot loaders and CPU microcode for
# PCs, the kernel (Batocera's is used), 32-bit PC libraries, PC graphics
# drivers, and Steam and Wine, which only run PC games.
skip='^(linux|amd-ucode|intel-ucode|mkinitcpio-archiso|syslinux|edk2-shell|memtest86\+.*|lib32-.*|vulkan-radeon|vulkan-intel|steam|gamescope|wine|wine-mono|umu-launcher|winetricks|gamemode)$'
# The Raspberry Pi's own graphics drivers.
extra=(vulkan-broadcom mesa-utils)

cleanup() {
    for m in run dev/pts dev sys proc; do
        umount -l "$root/$m" 2>/dev/null || true
    done
}
trap cleanup EXIT

step "Getting Arch Linux ARM"
rm -rf "$work"
mkdir -p "$root" "$out"
tarball=$work/alarm.tar.gz
for url in http://os.archlinuxarm.org/os/ArchLinuxARM-aarch64-latest.tar.gz \
           http://de3.mirror.archlinuxarm.org/os/ArchLinuxARM-aarch64-latest.tar.gz \
           http://fl.us.mirror.archlinuxarm.org/os/ArchLinuxARM-aarch64-latest.tar.gz; do
    curl -fL --retry 3 -o "$tarball" "$url" && break
done
bsdtar -xpf "$tarball" -C "$root"
rm "$tarball"

mount -t proc proc "$root/proc"
mount --rbind /sys "$root/sys"
mount --rbind /dev "$root/dev"
mount -t tmpfs tmpfs "$root/run"
rm -f "$root/etc/resolv.conf"
cp -L /etc/resolv.conf "$root/etc/resolv.conf"
in_root() { chroot "$root" /usr/bin/env -i HOME=/root TERM=dumb PATH=/usr/local/sbin:/usr/local/bin:/usr/bin LANG=C.UTF-8 "$@"; }

step "Setting up pacman"
# pacman's download sandbox doesn't work under QEMU.
if ! grep -q '^DisableSandbox' "$root/etc/pacman.conf"; then
    sed -i '/^\[options\]/a DisableSandbox' "$root/etc/pacman.conf"
fi
sed -i '/^NoExtract/d' "$root/etc/pacman.conf"
# Keep a little more room: no kernel, no Arch Linux ARM's board extras.
in_root pacman-key --init
in_root pacman-key --populate archlinuxarm
in_root pacman -Rdd --noconfirm linux-aarch64 2>/dev/null || true
in_root pacman -Syu --noconfirm

step "Installing Ollie OS's packages"
want=()
while read -r pkg; do
    [[ "$pkg" =~ $skip ]] && continue
    want+=("$pkg")
done < <(list "$os/profile/packages.x86_64"; list "$os/profile/packages-optional.x86_64"; printf '%s\n' "${extra[@]}")
have=()
for pkg in "${want[@]}"; do
    if in_root pacman -Si "$pkg" >/dev/null 2>&1; then
        have+=("$pkg")
    else
        echo "note: $pkg isn't available for ARM, leaving it out"
    fi
done
in_root pacman -S --noconfirm --needed "${have[@]}"
# Not on ARM: the Bibata cursor from the AUR. Breeze's cursor is used.

step "Adding Ollie OS's own files"
cp -a "$os/profile/airootfs/." "$root/"
# The live USB's set-up is for the ISO only (it checks for archiso itself);
# Etros's ollie-setup makes the account on the card.
rm -f "$root/etc/mkinitcpio.conf.d/archiso.conf"
# Breeze's cursor instead of Bibata, which isn't built for ARM.
grep -rl Bibata "$root/etc" 2>/dev/null | xargs -r sed -i 's/Bibata-Modern-Ice/breeze_cursors/g'
chmod 0750 "$root/etc/sudoers.d"
chmod 0440 "$root/etc/sudoers.d/10-wheel"
chmod 0755 "$root"/usr/local/bin/*
install -D -m 0755 "$os/ollie-write" "$root/usr/local/bin/ollie-write"

step "Building Oliver for ARM"
version=$(git -C "$src" describe --tags --always 2>/dev/null || echo dev)
(cd "$src" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -buildvcs=false \
    -ldflags "-s -w -X main.version=$version" -o "$root/usr/local/bin/oliver" ./cmd/oliver)
install -D -m 0644 "$src/packaging/oliver.desktop" "$root/usr/share/applications/oliver.desktop"
install -D -m 0644 "$src/packaging/oliver.svg" "$root/usr/share/icons/hicolor/scalable/apps/oliver.svg"

step "Drawing the logo and wallpaper"
for icon in ollie-os ollie-dc ollie-drivers; do
    install -D -m 0644 "$os/art/$icon.svg" "$root/usr/share/icons/hicolor/scalable/apps/$icon.svg"
done
images=$root/usr/share/wallpapers/OllieOS/contents/images
mkdir -p "$images"
for size in 1280x720 1920x1080 2560x1440 3840x2160; do
    rsvg-convert -w "${size%x*}" -h "${size#*x}" -o "$images/$size.png" "$os/art/wallpaper.svg"
done

step "Finishing the system"
sed -i 's/^#\(en_US.UTF-8 UTF-8\)/\1/' "$root/etc/locale.gen"
in_root locale-gen
in_root systemctl enable NetworkManager bluetooth sddm 2>/dev/null || true
in_root systemctl set-default graphical.target
# Arch Linux ARM's own "alarm" account isn't Ollie OS's; Etros makes "ollie".
in_root userdel -r alarm 2>/dev/null || true
echo "Ollie OS for ARM $version, $(date -u +%Y-%m-%d)" > "$root/etc/ollie-os-arm"
rm -f "$root/etc/machine-id" && touch "$root/etc/machine-id"
in_root pacman -Scc --noconfirm >/dev/null || true
rm -rf "$root/var/cache/pacman/pkg/"* "$root/var/lib/pacman/sync/"*
cleanup
rm -f "$root/etc/resolv.conf"
ln -sf /run/systemd/resolve/stub-resolv.conf "$root/etc/resolv.conf"

step "Making the disk image"
# Room for the system plus 2 GB free, on a partition starting at 1 MiB.
used=$(du -sxm "$root" | cut -f1)
size=$(( used * 115 / 100 + 2048 ))
fs=$work/rootfs.ext4
mkfs.ext4 -q -L rootfs -d "$root" "$fs" "${size}M"
name=ollie-os-arm64.img
img=$work/$name
truncate -s $(( (size + 1) * 1024 * 1024 )) "$img"
echo 'start=2048, type=83' | sfdisk -q "$img"
dd if="$fs" of="$img" bs=1M seek=1 conv=notrunc,sparse status=none
rm "$fs"
xz -T0 -6 -c "$img" > "$out/$name.xz"
rm "$img"
(cd "$out" && sha256sum "$name.xz" > "$name.xz.sha256")
step "Done"
ls -lh "$out"
