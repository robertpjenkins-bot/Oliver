#!/bin/bash
# Make Ollie OS from Arch Linux ARM on this computer, for Etros Updater and
# Etros Maker (archbuild_linux.go), so the Pi 5 doesn't have to.
#
# Runs as "root" of a user namespace of its own: really the person running
# Etros, so no password, and everything it makes belongs to them. Pi
# programs (pacman and the rest) run through qemu, registered for this
# namespace only. Ollie OS ends up as an ext4 file system in IMAGE, owned by
# root as it should be (mkfs.ext4 -d sees the namespace's owners) and with
# the few files packages give to other users set from pacman's records.
#
#   archbuild.sh TARBALL WORK IMAGE [QEMU]
#
# Lines starting "STAGE " name the step it's on; "FAILED " says why it
# stopped. Exit code 3: this computer can't run Pi programs at all.

set -u
tarball=$1 work=$2 img=$3 qemu=${4:-}
root=$work/root
export PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# The same as etros-setup-arch, which does this on the Pi instead.
packages=(
    plasma-desktop plasma-nm plasma-pa kscreen breeze breeze-gtk kde-gtk-config powerdevil
    sddm sddm-kcm xdg-desktop-portal-kde xorg-server
    dolphin konsole kate ark gwenview spectacle kdialog
    firefox
    networkmanager bluez bluez-utils pipewire pipewire-pulse wireplumber
    mesa vulkan-broadcom noto-fonts noto-fonts-emoji ttf-liberation
    python sudo nano bash-completion dosfstools e2fsprogs exfatprogs ntfs-3g
)

say() { echo "STAGE $1"; }
undo() {
    [ -d "$root" ] || return 0
    in_root gpgconf --homedir /etc/pacman.d/gnupg --kill all >/dev/null 2>&1
    umount -R "$root" 2>/dev/null
    while mountpoint -q "$root" 2>/dev/null; do umount -l "$root" || break; done
}
fail() {
    echo "FAILED $1"
    undo
    rm -f "$img"
    exit "${2:-1}"
}
in_root() { chroot "$root" /usr/bin/env -i HOME=/root TERM=dumb PATH=/usr/local/sbin:/usr/local/bin:/usr/bin LANG=C.UTF-8 "$@"; }

# Nothing mounted here reaches the rest of the computer.
mount --make-rprivate / 2>/dev/null

for tool in tar mkfs.ext4 debugfs chroot; do
    command -v "$tool" >/dev/null || fail "this computer has no $tool" 3
done

# Pi programs through qemu, for this namespace only.
if [ -n "$qemu" ]; then
    mkdir -p "$work/binfmt"
    mount -t binfmt_misc binfmt_misc "$work/binfmt" ||
        fail "this computer can't run Pi programs (no binfmt_misc in a user namespace)" 3
    [ -e "$work/binfmt/etros-aarch64" ] && echo -1 > "$work/binfmt/etros-aarch64"
    printf '%s' ':etros-aarch64:M::\x7f\x45\x4c\x46\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00\x02\x00\xb7\x00:\xff\xff\xff\xff\xff\xff\xff\x00\xff\xff\xff\xff\xff\xff\xff\xff\xfe\xff\xff\xff:'"$qemu"':F' \
        > "$work/binfmt/register" || fail "couldn't set up qemu for Pi programs" 3
fi

say "Unpacking Arch Linux"
undo
rm -rf "$root"
mkdir -p "$root"
tar -xpzf "$tarball" -C "$root" --numeric-owner --no-same-owner || fail "couldn't unpack $(basename "$tarball")"
in_root /usr/bin/true || fail "this computer can't run Pi programs" 3

# Its own mount, so pacman can tell how much room there is.
mount --bind "$root" "$root" || fail "couldn't mount Arch Linux's folder"
if ! { mount --rbind /proc "$root/proc" && mount --rbind /sys "$root/sys" &&
    mount --rbind /dev "$root/dev" && mount -t tmpfs tmpfs "$root/run"; }; then
    fail "couldn't mount /proc, /sys, /dev and /run for Arch Linux"
fi
rm -f "$root/etc/resolv.conf"
cp -L /etc/resolv.conf "$root/etc/resolv.conf"
grep -q '^DisableSandbox' "$root/etc/pacman.conf" || sed -i '/^\[options\]/a DisableSandbox' "$root/etc/pacman.conf"
# Downloads as another user and the space check can't work in a namespace.
sed -i 's/^CheckSpace/#CheckSpace/; s/^DownloadUser/#DownloadUser/' "$root/etc/pacman.conf"

say "Setting up pacman's keys"
in_root pacman-key --init || fail "pacman-key --init failed"
in_root pacman-key --populate archlinuxarm || fail "pacman-key --populate failed"
# Ollie OS runs on Batocera's kernel, so Arch's isn't needed.
in_root pacman -Rdd --noconfirm linux-aarch64 linux-firmware 2>/dev/null

say "Updating Arch Linux"
ok=no
for _ in 1 2 3; do
    in_root pacman -Syu --noconfirm && { ok=yes; break; }
    sleep 20
done
[ "$ok" = yes ] || fail "couldn't update Arch Linux (is this computer connected to the internet?)"

say "Installing KDE Plasma, Firefox and the rest (the longest part)"
ok=no
for _ in 1 2 3; do
    in_root pacman -S --noconfirm --needed "${packages[@]}" && { ok=yes; break; }
    sleep 20
done
[ "$ok" = yes ] || fail "couldn't install KDE Plasma and the rest (is this computer connected to the internet?)"

say "Making it Ollie OS"
sed -i 's/^#\(en_US.UTF-8 UTF-8\)/\1/' "$root/etc/locale.gen"
in_root locale-gen
echo LANG=en_US.UTF-8 > "$root/etc/locale.conf"
echo ollie-os > "$root/etc/hostname"
# Nothing left for its first start to ask (on a console nobody can see).
echo KEYMAP=us > "$root/etc/vconsole.conf"
[ -e "$root/etc/localtime" ] || ln -sf /usr/share/zoneinfo/UTC "$root/etc/localtime"
in_root systemctl mask systemd-firstboot.service
in_root systemctl enable sddm NetworkManager bluetooth
in_root systemctl set-default graphical.target
in_root userdel -r alarm 2>/dev/null
echo "Ollie OS (Arch Linux ARM), set up $(date '+%Y-%m-%d')" > "$root/etc/ollie-os-arch"
in_root pacman -Scc --noconfirm >/dev/null 2>&1
sed -i 's/^#CheckSpace/CheckSpace/; s/^#DownloadUser/DownloadUser/' "$root/etc/pacman.conf"
rm -f "$root/etc/resolv.conf"
ln -sf /run/NetworkManager/resolv.conf "$root/etc/resolv.conf"
undo
mountpoint -q "$root/proc" && fail "couldn't unmount /proc from Arch Linux's folder"

say "Putting Ollie OS on the card"
rm -f "$img"
truncate -s 24G "$img" || fail "couldn't make Ollie OS's file on the card"
mkfs.ext4 -q -F -L ollie -d "$root" "$img" || fail "couldn't copy Ollie OS onto the card (is the card full?)"

# Files packages give to users other than root (pacman's records of them).
cmds=$work/owners.txt
for m in "$root"/var/lib/pacman/local/*/mtree; do
    [ -f "$m" ] && gzip -dc "$m"
done | awk '
    /^\/set / { for (i = 2; i <= NF; i++) { split($i, kv, "="); if (kv[1] == "uid") du = kv[2]; if (kv[1] == "gid") dg = kv[2] } next }
    /^\.\// {
        u = du; g = dg
        for (i = 2; i <= NF; i++) { split($i, kv, "="); if (kv[1] == "uid") u = kv[2]; if (kv[1] == "gid") g = kv[2] }
        if ($1 ~ /\\/ || (u + 0 == 0 && g + 0 == 0)) next
        p = substr($1, 2)
        printf "sif \"%s\" uid %d\nsif \"%s\" gid %d\n", p, u, p, g
    }' > "$cmds"
if [ -s "$cmds" ]; then
    debugfs -w -f "$cmds" "$img" >/dev/null 2>&1 || fail "couldn't give Ollie OS's files their owners"
fi
sync "$img" 2>/dev/null || sync
rm -rf "$root" "$cmds"
echo "DONE"
