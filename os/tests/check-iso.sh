#!/bin/bash
# Check a built Ollie OS ISO without booting it: unpack it and confirm that
# everything Ollie OS promises is actually inside.
#
#   os/tests/check-iso.sh ollie-os-*.iso
#
# Needs xorriso and unsquashfs. Prints one line per check and exits non-zero
# if any check fails.
set -uo pipefail

iso="${1:?usage: check-iso.sh ISO}"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

pass=0 fail=0
ok() { printf '  \e[32mok\e[0m    %s\n' "$1"; pass=$((pass + 1)); }
bad() { printf '  \e[31mFAIL\e[0m  %s\n' "$1"; fail=$((fail + 1)); }
check() { # check "description" command...
    local what="$1"
    shift
    if "$@" >/dev/null 2>&1; then ok "$what"; else bad "$what"; fi
}

echo "== ISO layout"
xorriso -osirrox on -indev "$iso" -extract / "$work/iso" >/dev/null 2>&1
chmod -R u+w "$work/iso"
check "boot loader for UEFI (Steam Deck / Steam Machine)" test -f "$work/iso/EFI/BOOT/BOOTx64.EFI"
check "boot menu entries" compgen -G "$work/iso/loader/entries/*.conf"
check "kernel" test -f "$work/iso/arch/boot/x86_64/vmlinuz-linux"
check "initramfs" test -f "$work/iso/arch/boot/x86_64/initramfs-linux.img"
check "boot menu says Ollie OS" grep -rq "Ollie OS" "$work/iso/loader/entries"
check "boot menu finds the ISO by UUID (USB Maker rewrites this)" grep -rqE 'archisosearchuuid=|archisolabel=' "$work/iso/loader/entries"
sfs="$work/iso/arch/x86_64/airootfs.sfs"
check "system image present" test -f "$sfs"
size=$(stat -c %s "$sfs" 2>/dev/null || echo 0)
if ((size > 0 && size < 4 * 1024 * 1024 * 1024)); then
    ok "system image fits on FAT32 ($((size / 1024 / 1024)) MB < 4096 MB)"
else
    bad "system image fits on FAT32 ($((size / 1024 / 1024)) MB)"
fi
iso_mb=$(($(stat -c %s "$iso") / 1024 / 1024))
if ((iso_mb < 5 * 1024)); then ok "ISO fits the USB Maker's 5 GB boot partition ($iso_mb MB)"; else bad "ISO too big for 5 GB boot partition ($iso_mb MB)"; fi

echo "== System image"
unsquashfs -no-progress -quiet -d "$work/root" "$sfs" >/dev/null 2>&1 || bad "unpack system image"
r="$work/root"
pkg() { compgen -G "$r/var/lib/pacman/local/$1-[0-9]*" >/dev/null || compgen -G "$r/var/lib/pacman/local/$1-r[0-9]*" >/dev/null; }
for p in bibata-cursor-theme-bin pciutils usbutils vulkan-tools linux plasma-desktop sddm dolphin konsole steam wine wine-mono gamescope google-chrome proton-ge-custom-bin \
    networkmanager pipewire mesa vulkan-radeon lib32-vulkan-radeon zram-generator btrfs-progs arch-install-scripts; do
    check "package $p" pkg "$p"
done
check "Proton-GE where Oliver looks for it" compgen -G "$r/usr/share/steam/compatibilitytools.d/*"
check "kernel modules present" compgen -G "$r/usr/lib/modules/*/vmlinuz"

echo "== Ollie OS pieces"
for f in /usr/local/bin/oliver /usr/local/bin/ollie-install /usr/local/bin/ollie-write \
    /usr/local/bin/ollie-add-to-bar /usr/local/bin/ollie-live-setup \
    /usr/local/bin/ollie-dc /usr/local/bin/ollie-drivers; do
    check "$f is executable" test -x "$r$f"
done
check "oliver runs" "$r/usr/local/bin/oliver" help
check "Oliver handles .exe files" grep -q 'application/x-ms-dos-executable=oliver.desktop' "$r/etc/xdg/mimeapps.list"
check "oliver.desktop installed" test -f "$r/usr/share/applications/oliver.desktop"
check "Chrome is the default browser" grep -q 'https=google-chrome.desktop' "$r/etc/xdg/mimeapps.list"
check "google-chrome.desktop exists" test -f "$r/usr/share/applications/google-chrome.desktop"
check "steam.desktop exists" test -f "$r/usr/share/applications/steam.desktop"
check "DC - Debug Computer in the menu" test -f "$r/usr/share/applications/ollie-dc.desktop"
check "Driver Installer in the menu" test -f "$r/usr/share/applications/ollie-drivers.desktop"
check "DC icon" test -f "$r/usr/share/icons/hicolor/scalable/apps/ollie-dc.svg"
check "Driver Installer icon" test -f "$r/usr/share/icons/hicolor/scalable/apps/ollie-drivers.svg"
check "Bibata cursor installed" test -d "$r/usr/share/icons/Bibata-Modern-Ice/cursors"
check "Bibata cursor is the default" grep -q 'cursorTheme=Bibata-Modern-Ice' "$r/etc/xdg/kcminputrc"
check "lspci for the Driver Installer" test -x "$r/usr/bin/lspci"
check "vulkaninfo for DC" test -x "$r/usr/bin/vulkaninfo"
check "cube logo icon" test -f "$r/usr/share/icons/hicolor/scalable/apps/ollie-os.svg"
for s in 1280x720 1920x1080 2560x1440 3840x2160; do
    check "wallpaper $s" test -s "$r/usr/share/wallpapers/OllieOS/contents/images/$s.png"
done
check "Google search box widget" test -f "$r/usr/share/plasma/plasmoids/org.ollieos.googlesearch/contents/ui/main.qml"
check "bar layout" test -f "$r/usr/share/plasma/look-and-feel/org.ollieos.desktop/contents/layouts/org.kde.plasma.desktop-layout.js"
check "look-and-feel is the default" grep -q 'LookAndFeelPackage=org.ollieos.desktop' "$r/etc/xdg/kdeglobals"
check "'Set as wallpaper' right-click menu" test -f "$r/usr/share/kio/servicemenus/ollie-set-wallpaper.desktop"
check "'Add to bar' right-click menu" test -f "$r/usr/share/kio/servicemenus/ollie-add-to-bar.desktop"
check "plasma-apply-wallpaperimage exists" test -x "$r/usr/bin/plasma-apply-wallpaperimage"
check "busctl exists (Add to bar)" test -x "$r/usr/bin/busctl"
check "Steam Game Mode in the menu" test -f "$r/usr/share/applications/steam-game-mode.desktop"
check "Plasma Wayland session" test -f "$r/usr/share/wayland-sessions/plasma.desktop"
check "login screen background" test -f "$r/usr/share/sddm/themes/breeze/theme.conf.user"
check "breeze login theme" test -d "$r/usr/share/sddm/themes/breeze"
check "kwin_wayland for the login screen" test -x "$r/usr/bin/kwin_wayland"

echo "== Services"
check "no first-boot questions (systemd-firstboot masked)" test "$(readlink "$r/etc/systemd/system/systemd-firstboot.service")" = /dev/null
check "time zone set" test -L "$r/etc/localtime"
check "keyboard layout set" test -f "$r/etc/vconsole.conf"
check "boots to the desktop" test "$(readlink "$r/etc/systemd/system/default.target")" = /usr/lib/systemd/system/graphical.target
check "login screen enabled" test -e "$r/usr/lib/systemd/system/sddm.service"
check "networking enabled" test -e "$r/usr/lib/systemd/system/NetworkManager.service"
check "bluetooth unit exists" test -e "$r/usr/lib/systemd/system/bluetooth.service"
check "power profiles unit exists" test -e "$r/usr/lib/systemd/system/power-profiles-daemon.service"
check "paccache timer exists" test -e "$r/usr/lib/systemd/system/paccache.timer"
check "pacman has multilib" grep -q '^\[multilib\]' "$r/etc/pacman.conf"
check "pacman has no build-only local repo" bash -c "! grep -q 'ollie-local' '$r/etc/pacman.conf'"
check "pacman has working mirrors" grep -q '^Server' "$r/etc/pacman.d/mirrorlist"
check "live initramfs hooks" grep -q archiso "$r/etc/mkinitcpio.conf.d/archiso.conf"
check "admin tasks need no password (sudo)" grep -q 'NOPASSWD' "$r/etc/sudoers.d/10-wheel"
check "admin tasks need no password (polkit)" test -f "$r/etc/polkit-1/rules.d/49-ollie-no-password.rules"
check "straight to the desktop (autologin)" grep -q '^User=ollie' "$r/etc/sddm.conf.d/20-autologin.conf"
check "screen never locks" grep -q '^Autolock=false' "$r/etc/xdg/kscreenlockerrc"
check "installer asks for no password" bash -c "! grep -q 'read -rsp' '$r/usr/local/bin/ollie-install'"
check "translations kept (NoExtract removed)" compgen -G "$r/usr/share/locale/de/LC_MESSAGES/*"

echo
echo "$pass passed, $fail failed"
((fail == 0))
