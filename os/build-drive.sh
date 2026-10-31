#!/bin/bash
# Install Ollie OS straight onto a drive from the profile os/build.sh has put
# together, instead of building the ISO (OLLIE_BUILD_DISK; the Ollie OS Drive
# Writer, os/usb/ollie-install-drive):
#   1. Ollie OS's own installer (ollie-install) erases the drive and makes its
#      parts, as "Install Ollie OS" does from the USB.
#   2. On it goes what mkarchiso would put in the ISO's system: Ollie OS's own
#      files, then Arch Linux's packages, downloaded onto the drive itself.
#   3. The installer makes that an installed system that starts by itself on
#      new computers and old ones.
#   4. The drive is checked, what is wrong is fixed, and a report is written.
# Runs as root in the Arch Linux that os/build.sh runs in:
#
#   build-drive.sh PROFILE DISK REPORT
#
# Exit status: 0 installed and checked; 3 installed, but the check found
# problems it could not fix (in the report); anything else, not installed.
set -uo pipefail

profile=${1:?} disk=${2:?} report=${3:?}
step() { printf '\n\e[1;36m==> %s\e[0m\n' "$*"; }
list() { sed -e 's/#.*//' -e 's/[[:space:]]//g' -e '/^$/d' "$1"; }

# --- 1-3. Installing -------------------------------------------------------------------
# ollie-install runs this (OLLIE_INSTALL_FILL) with the drive's new parts
# mounted at TARGET, where it would copy the live USB's system. It is called
# as "fill_target ... || die", which set -e does not reach: every step checks.
fill_target() { # fill_target TARGET
    local target=$1 path log=/tmp/ollie-packages.log
    local -A file_permissions=()
    local -a perm pkgs
    # Ollie OS's own files first, the way mkarchiso puts them in the ISO, so
    # the packages' set-up (the start-up image, above all) already sees them.
    cp -af --no-preserve=ownership,mode -- "$OLLIE_PROFILE/airootfs/." "$target/" || return 1
    eval "$(sed -n '/^file_permissions=(/,/^)/p' "$OLLIE_PROFILE/profiledef.sh")"
    for path in "${!file_permissions[@]}"; do
        [[ -e "$target$path" ]] || continue
        IFS=: read -r -a perm <<<"${file_permissions[$path]}"
        if [[ $path == */ ]]; then
            chown -fhR -- "${perm[0]}:${perm[1]}" "$target$path" && chmod -fR -- "${perm[2]}" "$target$path"
        else
            chown -fh -- "${perm[0]}:${perm[1]}" "$target$path" && chmod -f -- "${perm[2]}" "$target$path"
        fi || return 1
    done
    # Arch Linux's packages. Without pacstrap's -c they are downloaded into
    # the drive's own package cache, not onto this computer. Shown as
    # downloading 5-45 and installing 45-75 of the whole install; all of
    # pacman's words go to the log.
    mapfile -t pkgs < <(list "$OLLIE_PROFILE/packages.x86_64")
    pacstrap -C "$OLLIE_PROFILE/pacman.conf" -G -M "$target" "${pkgs[@]}" 2>&1 | tee "$log" | awk '
        /^Packages \([0-9]+\)/ { total = substr($2, 2) + 0 }
        / downloading\.\.\.$/ { if (total) p = 5 + int(++got * 40 / total); text = "Downloading Arch Linux packages onto the drive" }
        /^\( *[0-9]+\/ *[0-9]+\) installing / { split($0, a, /[()\/]/); p = 45 + int(a[2] * 30 / a[3]); text = "Installing Ollie OS programs" }
        p && p != last { if (p > 75) p = 75; print "PROGRESS " p " " text; fflush(); last = p }'
    if ((PIPESTATUS[0] != 0)); then
        tail -n 25 "$log"
        return 1
    fi
}
export -f fill_target list
export OLLIE_PROFILE=$profile

OLLIE_INSTALL_DISK=$disk OLLIE_INSTALL_YES=1 OLLIE_INSTALL_PROGRESS=1 OLLIE_INSTALL_NO_EFI_VARS=1 \
    OLLIE_INSTALL_FILL=fill_target OLLIE_INSTALL_TIMEZONE="${OLLIE_INSTALL_TIMEZONE:-UTC}" \
    bash "$profile/airootfs/usr/local/bin/ollie-install" </dev/null || exit 1

# --- 4. Checking it ----------------------------------------------------------------------
step "Checking Ollie OS on the drive (and fixing what is wrong)"
echo "PROGRESS 99 Checking Ollie OS on the drive"
if [[ "$disk" =~ [0-9]$ ]]; then sep=p; else sep=; fi
boot_part=${disk}${sep}1 root_part=${disk}${sep}2
t=/mnt/ollie-check
mkdir -p "$t"
mount -o subvol=@ "$root_part" "$t" && mount -o subvol=@home "$root_part" "$t/home" && mount "$boot_part" "$t/boot" ||
    { echo "The drive could not be opened to check it."; exit 1; }
trap 'umount -R "$t" 2>/dev/null' EXIT
in_target() { arch-chroot "$t" "$@"; }
# Packages again (to put back what is damaged or missing): with the build's
# keys and settings, from the drive's own package cache where it has them.
reinstall() { pacstrap -C "$profile/pacman.conf" -G -M "$t" "$@" >/tmp/ollie-reinstall.log 2>&1; }
uuid="$(blkid -s UUID -o value "$root_part")"
problems=0
{
    echo "Ollie OS Drive Writer: the check of $disk ($(lsblk -dno SIZE,MODEL "$disk" | xargs)), $(date '+%Y-%m-%d %H:%M')"
    echo
} >"$report"
# check NAME TEST FIX: TEST (a function, run apart: what it sets is not kept)
# passes, or FIX runs and TEST must pass after it. What was wrong is noted.
check() {
    local name=$1 test=$2 fix=$3 why was
    if why="$($test 2>&1)"; then
        printf '  OK       %s\n' "$name" | tee -a "$report"
    elif $fix >/dev/null 2>&1 && was=$why && why="$($test 2>&1)"; then
        printf '  FIXED    %s (%s)\n' "$name" "$(head -c 200 <<<"$was" | tr -s '\n ' '  ' | sed 's/ $//')" | tee -a "$report"
    else
        printf '  PROBLEM  %s%s\n' "$name" "${why:+: $(head -c 300 <<<"$why" | tr '\n' ' ')}" | tee -a "$report"
        problems=$((problems + 1))
    fi
}

# Every program's files are there (not damaged or missing). (The cursor
# colour sets Ollie OS leaves out on purpose do not count.)
broken_pkgs() { in_target pacman -Qkq 2>/dev/null | grep -v ' /usr/share/icons/Bibata-' | awk '{print $1}' | sort -u; }
test_files() {
    local broken
    broken="$(broken_pkgs)"
    [[ -z $broken ]] || { echo "missing files in" $broken; return 1; }
}
fix_files() {
    local -a broken
    mapfile -t broken < <(broken_pkgs)
    reinstall "${broken[@]}"
}
check "All of Ollie OS's programs are there, nothing damaged ($(in_target pacman -Qq | wc -l) programs)" test_files fix_files

# Steam: the program, its 32-bit graphics drivers (Steam itself is 32-bit),
# the multilib packages it updates from, and the language setting it needs
# to start.
steam_pkgs=(steam lib32-mesa lib32-vulkan-radeon lib32-vulkan-intel lib32-vulkan-icd-loader lib32-pipewire ttf-liberation)
missing_steam() { local p; for p in "${steam_pkgs[@]}"; do in_target pacman -Q "$p" >/dev/null 2>&1 || echo "$p"; done; }
test_steam() {
    local missing
    missing="$(missing_steam)"
    [[ -z $missing ]] || { echo "not installed:" $missing; return 1; }
    [[ -f "$t/usr/lib/steam/bootstraplinux_ubuntu12_32.tar.xz" && -e "$t/usr/lib32/libGL.so.1" ]] || { echo "Steam's files are incomplete"; return 1; }
}
fix_steam() {
    local -a missing
    mapfile -t missing < <(missing_steam)
    reinstall "${missing[@]:-steam}"
}
check "Steam and its 32-bit graphics drivers" test_steam fix_steam
test_multilib() { grep -q '^\[multilib\]' "$t/etc/pacman.conf"; }
fix_multilib() { printf '\n[multilib]\nInclude = /etc/pacman.d/mirrorlist\n' >>"$t/etc/pacman.conf"; }
check "Steam's updates (the multilib packages are switched on)" test_multilib fix_multilib
test_locale() { in_target locale -a 2>/dev/null | grep -qix 'en_US.utf8'; }
fix_locale() {
    sed -i 's/^#\(en_US.UTF-8 UTF-8\)/\1/' "$t/etc/locale.gen"
    echo LANG=en_US.UTF-8 >"$t/etc/locale.conf"
    in_target locale-gen
}
check "The language setting Steam needs to start (en_US.UTF-8)" test_locale fix_locale

# Starting: the kernel, and a start-up image for it with every computer's
# drivers (USB drives, NVMe, btrfs) and none of the live USB's.
kver="$(ls "$t/usr/lib/modules" 2>/dev/null | head -n 1)"
test_image() {
    [[ -n $kver && -s "$t/boot/vmlinuz-linux" && -s "$t/boot/initramfs-linux.img" ]] || { echo "the start-up files are missing"; return 1; }
    cmp -s "$t/boot/vmlinuz-linux" "$t/usr/lib/modules/$kver/vmlinuz" || { echo "the kernel does not match its drivers"; return 1; }
    local files
    files="$(in_target lsinitcpio /boot/initramfs-linux.img 2>/dev/null)"
    grep -q 'hooks/archiso' <<<"$files" && { echo "the start-up image is the live USB's"; return 1; }
    local m
    for m in uas usb-storage xhci-pci nvme btrfs; do
        grep -q "/$m\.ko" <<<"$files" || grep -q "/$m\.ko\$" "$t/usr/lib/modules/$kver/modules.builtin" ||
            { echo "the start-up image lacks $m"; return 1; }
    done
}
fix_image() {
    rm -f "$t/etc/mkinitcpio.conf.d/archiso.conf"
    cp -f "$t/usr/lib/modules/$kver/vmlinuz" "$t/boot/vmlinuz-linux"
    in_target mkinitcpio -P
}
check "Start-up image with the drivers for every computer (kernel $kver)" test_image fix_image

entry() { # entry TITLE OPTIONS: a systemd-boot menu entry
    printf 'title   %s\nlinux   /vmlinuz-linux\ninitrd  /initramfs-linux.img\noptions root=UUID=%s rootflags=subvol=@ rw%s\n' "$1" "$uuid" "$2"
}
test_uefi() {
    local f
    for f in EFI/BOOT/BOOTX64.EFI EFI/systemd/systemd-bootx64.efi loader/loader.conf loader/entries/ollie.conf; do
        [[ -s "$t/boot/$f" ]] || { echo "$f is missing"; return 1; }
    done
    grep -q "root=UUID=$uuid " "$t/boot/loader/entries/ollie.conf" || { echo "the menu does not point at Ollie OS"; return 1; }
}
fix_uefi() {
    in_target bootctl install --esp-path=/boot --no-variables 2>/dev/null ||
        in_target bootctl update --esp-path=/boot --no-variables 2>/dev/null
    printf 'default ollie.conf\ntimeout 2\neditor no\n' >"$t/boot/loader/loader.conf"
    entry "Ollie OS" " quiet" >"$t/boot/loader/entries/ollie.conf"
    entry "Ollie OS (show start-up messages)" "" >"$t/boot/loader/entries/ollie-fallback.conf"
}
check "Starting on new computers: Steam Machine, Dells, Intel Macs (UEFI)" test_uefi fix_uefi

test_bios() {
    [[ -s "$t/boot/syslinux/ldlinux.sys" ]] || { echo "SYSLINUX is not installed"; return 1; }
    grep -q "root=UUID=$uuid " "$t/boot/syslinux/syslinux.cfg" || { echo "the menu does not point at Ollie OS"; return 1; }
    cmp -s -n 440 "$disk" /usr/lib/syslinux/bios/gptmbr.bin || { echo "the drive's start-up code is missing"; return 1; }
    sgdisk -A 1:get:2 "$disk" 2>/dev/null | grep -q ':2:1' || { echo "the start-up part is not marked for BIOS"; return 1; }
}
fix_bios() {
    extlinux --install "$t/boot/syslinux" &&
        dd if=/usr/lib/syslinux/bios/gptmbr.bin of="$disk" bs=440 count=1 conv=notrunc status=none &&
        sgdisk --attributes=1:set:2 "$disk"
}
check "Starting on older computers set to BIOS / Legacy" test_bios fix_bios

test_fstab() {
    grep -q "^UUID=${uuid}[[:space:]]*/[[:space:]]" "$t/etc/fstab" && grep -q "[[:space:]]/boot[[:space:]]" "$t/etc/fstab" ||
        { echo "the list of the drive's parts is wrong"; return 1; }
}
fix_fstab() { genfstab -U "$t" >"$t/etc/fstab"; }
check "The drive's parts are found at start-up (fstab)" test_fstab fix_fstab

# (Service links point at /usr/lib/... inside Ollie OS: -L, as this Arch
# Linux's own /usr is not Ollie OS's.)
test_first_start() {
    grep -q '^ollie:' "$t/etc/passwd" || { echo "the account is missing"; return 1; }
    [[ -s "$t/etc/xdg/autostart/ollie-first-setup.desktop" ]] || { echo "the first-start setup is missing"; return 1; }
    [[ ! -L "$t/etc/systemd/system/multi-user.target.wants/ollie-live-setup.service" ]] || { echo "the live USB's set-up is still on"; return 1; }
    [[ -L "$t/etc/systemd/system/display-manager.service" && -e "$t$(readlink "$t/etc/systemd/system/display-manager.service")" ]] ||
        { echo "the desktop does not start"; return 1; }
}
fix_first_start() {
    rm -f "$t/etc/systemd/system/multi-user.target.wants/ollie-live-setup.service"
    grep -q '^ollie:' "$t/etc/passwd" || in_target useradd -m -G wheel,video,audio,input,storage -s /bin/bash -c "Ollie" ollie
    in_target passwd -d ollie
    [[ -s "$t/etc/xdg/autostart/ollie-first-setup.desktop" ]] ||
        install -D -m 0644 "$profile/airootfs/etc/xdg/autostart/ollie-first-setup.desktop" "$t/etc/xdg/autostart/ollie-first-setup.desktop"
    [[ -e "$t/usr/lib/systemd/system/sddm.service" ]] || reinstall sddm
    ln -sfn /usr/lib/systemd/system/sddm.service "$t/etc/systemd/system/display-manager.service"
}
check "The desktop, and the first start that asks for your name and password" test_first_start fix_first_start

# The downloaded packages are not needed any more: their room goes to Steam
# and everything you save. (The Fixer downloads what it needs to put back.)
rm -rf -- "${t:?}"/var/cache/pacman/pkg/*
free_gb=$(($(df -B1 --output=avail "$t" | tail -n 1) / 1000000000))
used_gb=$(($(df -B1 --output=used "$t" | tail -n 1) / 1000000000))
printf '  ROOM     %s GB free for Steam, games and everything you save (Ollie OS uses %s GB)\n' "$free_gb" "$used_gb" | tee -a "$report"

# Last, the drive's file systems themselves, closed.
umount -R "$t"
trap - EXIT
sync
test_fs() {
    fsck.fat -n "$boot_part" >/dev/null 2>&1 || { echo "the start-up part has errors"; return 1; }
    btrfs check --readonly "$root_part" >/dev/null 2>&1 || { echo "Ollie OS's part has errors"; return 1; }
}
fix_fs() { fsck.fat -a "$boot_part"; }
check "The drive's file systems have no errors" test_fs fix_fs
sync

echo >>"$report"
if ((problems == 0)); then
    echo "Result: Ollie OS on this drive is ready to start." | tee -a "$report"
    exit 0
fi
echo "Result: $problems problem(s) left, above. Running the Drive Writer again usually fixes them." | tee -a "$report"
exit 3
