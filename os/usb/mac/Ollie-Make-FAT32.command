#!/bin/bash
# Make a USB drive FAT32 on a Mac, ready for Ollie OS USB Maker.
# Double-click it. It only offers external (plugged-in) drives, and nothing
# is erased until you type ERASE. No password is needed.

bold=$'\e[1m' red=$'\e[31m' reset=$'\e[0m'
fail() { printf '\n%sError:%s %s\n\nPress Enter to close.' "$red$bold" "$reset" "$1"; read -r; exit 1; }

printf '\n%s  Ollie OS: make a USB drive FAT32%s\n\n' "$bold" "$reset"

disks=()
while read -r d; do
    [[ -n "$d" ]] && disks+=("$d")
done < <(diskutil list external physical | awk '/^\/dev\/disk/ {print $1}')

((${#disks[@]} > 0)) || fail "no USB drive found. Plug it in and try again."

echo "USB drives plugged in:"
echo
for i in "${!disks[@]}"; do
    d="${disks[$i]}"
    size="$(diskutil info "$d" | awk -F': *' '/Disk Size/ {print $2; exit}' | cut -d'(' -f1)"
    name="$(diskutil info "$d" | awk -F': *' '/Media Name/ {print $2; exit}')"
    printf '  %d) %-12s %-12s %s\n' $((i + 1)) "$d" "$size" "$name"
done
echo
read -rp "Which drive? [1-${#disks[@]}] " pick
[[ "$pick" =~ ^[0-9]+$ ]] && ((pick >= 1 && pick <= ${#disks[@]})) || fail "no such drive."
disk="${disks[pick - 1]}"

echo
diskutil list "$disk"
printf '%sEVERYTHING on %s will be erased.%s\n' "$red$bold" "$disk" "$reset"
read -rp "Type ERASE to continue: " answer
[[ "$answer" == ERASE ]] || fail "cancelled, nothing was changed."

echo
diskutil eraseDisk FAT32 OLLIE MBRFormat "$disk" || fail "the Mac could not format the drive."
diskutil eject "$disk" >/dev/null 2>&1

printf '\n%sDone.%s Unplug the drive, plug it into the Steam Machine, and run\n' "$bold" "$reset"
echo "Ollie OS USB Maker. Pick the drive and press Enter: no password needed."
echo
echo "Press Enter to close."
read -r
