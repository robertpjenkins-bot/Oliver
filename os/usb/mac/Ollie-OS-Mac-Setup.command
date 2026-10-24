#!/bin/bash
# Ollie OS on a Mac's own drive, next to macOS (Intel Macs).
#
# Double-click it. From the Ollie OS ISO it makes two folders:
#   OLLIE_BOOT   Ollie OS, set up to start from a partition named OLLIE_BOOT
#   OLLIE_SHARE  for files that both macOS and Ollie OS can open
# and, when those partitions already exist (made in Disk Utility), copies
# them straight onto them. Then: restart holding Option, pick "EFI Boot".
#
# It never partitions or erases anything itself: the partitions are made in
# Disk Utility, where you can see what you are doing.
#
# Written for the bash that comes with macOS (3.2).

bold=$'\e[1m' teal=$'\e[36m' red=$'\e[31m' green=$'\e[32m' reset=$'\e[0m'
say() { printf '\n%s==>%s %s%s%s\n' "$teal$bold" "$reset" "$bold" "$*" "$reset"; }
fail() {
    printf '\n%sError:%s %s\n\nPress Enter to close.' "$red$bold" "$reset" "$1"
    read -r _
    exit 1
}
ask() { # ask "question" -> REPLY
    printf '%s ' "$1"
    read -r REPLY || REPLY=""
}

mnt=""
cleanup() { [ -n "$mnt" ] && hdiutil detach -quiet "$mnt" >/dev/null 2>&1; }
trap cleanup EXIT

printf '\n%s  Ollie OS on this Mac%s  (two partitions next to macOS: OLLIE_BOOT and OLLIE_SHARE)\n' "$bold" "$reset"
[ "$(uname -m)" = x86_64 ] || fail "this is an Apple Silicon Mac (M1 or newer). Ollie OS needs an Intel Mac."

# --- 1. The ISO ------------------------------------------------------------------
iso=""
# The newest one in Downloads, on the Desktop, or next to this app.
for f in "$HOME"/Downloads/ollie-os*.iso "$HOME"/Desktop/ollie-os*.iso "$(dirname "$0")"/ollie-os*.iso; do
    [ -f "$f" ] || continue
    if [ -z "$iso" ] || [ "$f" -nt "$iso" ]; then iso="$f"; fi
done
if [ -z "$iso" ]; then
    say "Choose the Ollie OS ISO"
    iso="$(osascript -e 'POSIX path of (choose file with prompt "Choose the Ollie OS ISO (ollie-os-....iso)")' 2>/dev/null)"
    [ -n "$iso" ] || fail "no ISO chosen. Download it from the Ollie OS GitHub release first (ollie-os-....iso), then run this again."
fi
[ -f "$iso" ] || fail "no file at $iso."
say "ISO: $iso"

mnt="$(mktemp -d /tmp/ollie-iso.XXXXXX)"
hdiutil attach -readonly -nobrowse -noverify -mountpoint "$mnt" "$iso" >/dev/null 2>&1 ||
    fail "the Mac could not open the ISO. Is the download complete? (It is about 2 GB.)"
[ -f "$mnt/arch/x86_64/airootfs.sfs" ] && [ -d "$mnt/loader/entries" ] ||
    fail "this is not an Ollie OS ISO."
sfs_kb="$(du -k "$mnt/arch/x86_64/airootfs.sfs" | awk '{print $1}')"
[ "$sfs_kb" -lt 4194304 ] || fail "the system image in this ISO is over 4 GB, which an MS-DOS (FAT) partition cannot hold."

# --- 2. Where to put it --------------------------------------------------------------
boot_vol="" share_vol=""
[ -d /Volumes/OLLIE_BOOT ] && boot_vol=/Volumes/OLLIE_BOOT
[ -d /Volumes/OLLIE_SHARE ] && share_vol=/Volumes/OLLIE_SHARE

if [ -n "$boot_vol" ]; then
    fs="$(diskutil info "$boot_vol" 2>/dev/null | awk -F': *' '/File System Personality/ {print $2; exit}')"
    case "$fs" in
    *FAT*) ;;
    *) fail "the OLLIE_BOOT partition is \"$fs\", not MS-DOS (FAT). In Disk Utility, erase it as MS-DOS (FAT) named OLLIE_BOOT, then run this again." ;;
    esac
    say "Found the OLLIE_BOOT partition"
    diskutil info "$boot_vol" | awk -F': *' '/Device Node|Disk Size|Volume Free Space/ {print "    " $1 ": " $2}'
    printf '\n%sEverything on OLLIE_BOOT will be replaced by Ollie OS.%s Nothing else on the Mac is touched.\n' "$red$bold" "$reset"
    ask "Copy Ollie OS onto it now? [Y/n]"
    case "$REPLY" in [nN]*) boot_vol="" ;; esac
fi

if [ -n "$boot_vol" ]; then
    boot_dir="$boot_vol"
    share_dir="${share_vol:-}"
else
    out="$HOME/Downloads/Ollie OS for Mac"
    boot_dir="$out/OLLIE_BOOT"
    share_dir="$out/OLLIE_SHARE"
    say "Making the two folders in Downloads: Ollie OS for Mac"
    rm -rf "$boot_dir"
    mkdir -p "$boot_dir" "$share_dir" || fail "could not make the folders in Downloads."
fi

# --- 3. Copy Ollie OS and point its start menu at OLLIE_BOOT -----------------------------
say "Copying Ollie OS (2 GB, a few minutes)"
if [ "$boot_dir" = "$boot_vol" ]; then
    # Only Ollie OS's own folders, so leftovers of an older copy go.
    for d in arch boot EFI loader shellx64.efi; do rm -rf "${boot_dir:?}/$d"; done
fi
ditto --norsrc --noextattr --noacl "$mnt" "$boot_dir" || fail "copying failed (is the partition or disk full?)."
chmod -R u+w "$boot_dir" 2>/dev/null

# The ISO finds itself by its own ID; on the Mac it is the partition named OLLIE_BOOT.
n=0
for cfg in "$boot_dir"/loader/entries/*.conf "$boot_dir"/boot/syslinux/*.cfg; do
    [ -f "$cfg" ] || continue
    if grep -qE 'archisosearchuuid=|archisolabel=|archisodevice=' "$cfg"; then
        sed -i '' -E -e 's/ cow_label=[^ ]*//g' \
            -e 's/(archisosearchuuid|archisolabel|archisodevice)=[^ ]*/archisolabel=OLLIE_BOOT/' "$cfg"
        n=$((n + 1))
    fi
done
[ "$n" -gt 0 ] || fail "could not find Ollie OS's start menu to set up."
[ -f "$boot_dir/loader/loader.conf" ] && sed -i '' -E 's/^timeout.*/timeout 3/' "$boot_dir/loader/loader.conf"
# macOS's hidden ._ and .DS_Store files are not needed on it.
find "$boot_dir" \( -name '._*' -o -name .DS_Store \) -delete 2>/dev/null

if [ -n "$share_dir" ]; then
    cat >"$share_dir/About OLLIE_SHARE.txt" <<'EOF'
OLLIE_SHARE is for files you want in both macOS and Ollie OS.
In Ollie OS it shows up in the file manager (Dolphin) as OLLIE_SHARE.
EOF
fi
sync

# --- 4. What next ------------------------------------------------------------------------
printf '\n%s  Done.%s\n' "$green$bold" "$reset"
if [ "$boot_dir" = "$boot_vol" ]; then
    cat <<EOF

  Ollie OS is on the OLLIE_BOOT partition.

  To start it: restart, hold the Option (alt) key until the start-up disks
  appear, and pick "EFI Boot". To go back to macOS, restart normally (or
  hold Option and pick Macintosh HD).
EOF
else
    cat <<EOF

  The two folders are in Downloads > "Ollie OS for Mac":
    OLLIE_BOOT    Ollie OS
    OLLIE_SHARE   for files you want in both macOS and Ollie OS

  To put Ollie OS on this Mac's drive:
   1. Open Disk Utility. Choose View > Show All Devices, select the Mac's
      drive at the top (not "Macintosh HD"), and click Partition.
   2. Click +, name it OLLIE_BOOT, format MS-DOS (FAT), size 4 GB.
   3. Click + again, name it OLLIE_SHARE, format ExFAT, any size.
   4. Click Apply (macOS shrinks to make room; your files stay).
   5. Run this app again: it copies Ollie OS straight onto OLLIE_BOOT.
      (Or drag everything INSIDE the OLLIE_BOOT folder onto the OLLIE_BOOT
      partition yourself.)
   6. Restart holding Option and pick "EFI Boot".

  Back up the Mac first (Time Machine): changing partitions is safe in
  Disk Utility, but a backup is the safe way. If the Mac also has Windows
  (Boot Camp), it may stop appearing after partitioning.
EOF
fi
cat <<EOF

  Ollie OS starts fresh from OLLIE_BOOT each time: keep your files on
  OLLIE_SHARE. Wi-Fi on Macs from about 2015 needs a Broadcom driver Ollie OS
  does not have yet; use a cable or a USB Wi-Fi stick.

Press Enter to close.
EOF
read -r _
