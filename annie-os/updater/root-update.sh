#!/bin/bash
# The system half of the Annie OS Updater, run as root:
#   root-update.sh PAYLOAD_DIR kde|xfce [UPDATER_APPIMAGE]
set -uo pipefail
p="${1:?payload}"
desktop="${2:-kde}"
[[ "$(id -u)" -eq 0 ]] || { echo "ERROR: must run as root"; exit 1; }
fail() { echo "ERROR: $*"; }

echo "# Installing the newest Annie apps"
if [[ "$desktop" == kde ]]; then
    cp -a "$p/files/." / || fail "could not copy the Annie apps"
else
    # The first Annie OS (Xfce): only the parts that work there.
    while read -r f; do
        [[ -e "$p/files/$f" ]] || continue
        mkdir -p "/$(dirname "$f")"
        cp -a "$p/files/$f" "/$(dirname "$f")/" || fail "could not copy $f"
    done <"$p/xfce-files.txt"
    for t in application/vnd.appimage application/x-iso9660-appimage; do
        grep -qs "^$t=" /etc/xdg/mimeapps.list ||
            { grep -qs '^\[Default Applications\]' /etc/xdg/mimeapps.list || echo '[Default Applications]' >>/etc/xdg/mimeapps.list
              echo "$t=annie-run-appimage.desktop" >>/etc/xdg/mimeapps.list; }
    done
fi
chmod 755 /usr/local/bin/annie-* 2>/dev/null
if [[ -f "${3:-}" ]]; then
    install -D -m 0755 "$3" /usr/share/annie/Annie-OS-Updater-x86_64.AppImage
fi
mkdir -p /usr/share/icons/default
printf '[Icon Theme]\nInherits=AnnieRainbow\n' >/usr/share/icons/default/index.theme
gtk-update-icon-cache -q /usr/share/icons/hicolor 2>/dev/null
update-desktop-database -q /usr/share/applications 2>/dev/null

# Steam and the 32-bit drivers live in multilib.
if ! grep -q '^\[multilib\]' /etc/pacman.conf; then
    if grep -q '^#\[multilib\]' /etc/pacman.conf; then
        sed -i '/^#\[multilib\]/{s/^#//;n;s/^#//}' /etc/pacman.conf
    else
        printf '\n[multilib]\nInclude = /etc/pacman.d/mirrorlist\n' >>/etc/pacman.conf
    fi
fi

echo "# Unsticking the package installer"
if pgrep -x pacman >/dev/null; then
    fail "another install is running; wait for it to finish, then run the updater again"
    exit 1
fi
rm -f /var/lib/pacman/db.lck

echo "# Updating everything (this can take a while)"
updated=false
if pacman -Sy --noconfirm --needed archlinux-keyring && pacman -Su --noconfirm; then
    updated=true
else
    echo "# Repairing package keys, then trying again"
    rm -rf /etc/pacman.d/gnupg
    pacman-key --init && pacman-key --populate archlinux
    pacman -Sy --noconfirm archlinux-keyring && pacman -Su --noconfirm && updated=true
fi
$updated || fail "the update did not finish (is the internet on?)"

if $updated; then
    echo "# Adding anything Annie OS needs that is missing"
    list="$p/packages-essential.txt"
    [[ "$desktop" == kde ]] && list="$p/packages.txt"
    want=()
    while read -r pkg; do
        pkg="${pkg%%#*}"
        pkg="${pkg//[[:space:]]/}"
        [[ -z "$pkg" ]] && continue
        pacman -Qq "$pkg" >/dev/null 2>&1 && continue
        { pacman -Si "$pkg" || pacman -Sg "$pkg"; } >/dev/null 2>&1 && want+=("$pkg")
    done <"$list"
    if ((${#want[@]})); then
        pacman -S --needed --noconfirm "${want[@]}" || fail "could not add: ${want[*]}"
    fi
fi

echo "# Reinstalling damaged packages"
mapfile -t bad < <(pacman -Qkq 2>/dev/null | awk '{print $1}' | sort -u)
if ((${#bad[@]})); then
    pacman -S --noconfirm "${bad[@]}" || fail "could not reinstall ${bad[*]}"
fi

echo "# Freeing disk space"
paccache -rk1 >/dev/null 2>&1 || pacman -Sc --noconfirm >/dev/null
journalctl --vacuum-size=100M >/dev/null 2>&1

echo "# Wi-Fi and internet"
rfkill unblock all 2>/dev/null
systemctl enable --now NetworkManager >/dev/null 2>&1 || fail "NetworkManager would not start"
exit 0
