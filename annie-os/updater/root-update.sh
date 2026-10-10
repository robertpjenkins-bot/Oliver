#!/bin/bash
# The system half of the Annie OS Updater, run as root:
#   root-update.sh PAYLOAD_DIR
set -uo pipefail
p="${1:?payload}"
[[ "$(id -u)" -eq 0 ]] || { echo "ERROR: must run as root"; exit 1; }
fail() { echo "ERROR: $*"; }

echo "# Installing the newest Annie apps"
cp -a "$p/files/." / || fail "could not copy the Annie apps"
chmod 755 /usr/local/bin/annie-* /usr/lib/annie/* 2>/dev/null
mkdir -p /usr/share/icons/default
printf '[Icon Theme]\nInherits=AnnieRainbow\n' >/usr/share/icons/default/index.theme
gtk-update-icon-cache -q /usr/share/icons/hicolor 2>/dev/null
update-desktop-database -q /usr/share/applications 2>/dev/null
/usr/lib/annie/enable-multilib

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

if $updated && [[ -f "$p/packages.txt" ]]; then
    echo "# Adding anything new to Annie OS"
    want=()
    while read -r pkg; do
        [[ -z "$pkg" || "$pkg" == \#* ]] && continue
        pacman -Qq "$pkg" >/dev/null 2>&1 && continue
        { pacman -Si "$pkg" || pacman -Sg "$pkg"; } >/dev/null 2>&1 && want+=("$pkg")
    done <"$p/packages.txt"
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
