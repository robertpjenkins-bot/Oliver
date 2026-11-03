#!/bin/bash
# Ollie OS Installer for a Mac: downloads the newest Ollie OS from its GitHub
# releases and writes it onto a USB drive, which then starts an Intel Mac
# (hold Option), the Steam Machine or a PC into Ollie OS. The drive is erased.
# Writing to a drive asks for this Mac's password (macOS's own rule).
# (Ollie OS on this Mac's own drive, next to macOS: Ollie OS Mac Setup.)
set -uo pipefail
REPO=robertpjenkins-bot/Ollie-OS TAG=ollie-os-newest
bold=$'\e[1m' red=$'\e[31m' green=$'\e[32m' reset=$'\e[0m'
die() {
    printf '\n%sError:%s %s\n\nPress Enter to close.' "$red$bold" "$reset" "$*"
    read -r _
    exit 1
}
printf '\n%s  Ollie OS Installer%s  (puts Ollie OS on a USB drive)\n\n' "$bold" "$reset"

echo "==> Finding the newest Ollie OS"
json="$(curl -fsSL "https://api.github.com/repos/$REPO/releases/tags/$TAG")" ||
    die "could not reach GitHub. Is the internet connected?"
# The ISO's links (one file, or its parts in order), its size and checksum link.
info="$(osascript -l JavaScript -e '
function run(argv) {
  const a = JSON.parse(argv[0]).assets;
  let iso = a.filter(x => /\.iso$/.test(x.name));
  if (!iso.length) iso = a.filter(x => /\.iso\.part\d+$/.test(x.name))
    .sort((x, y) => +x.name.match(/\d+$/)[0] - +y.name.match(/\d+$/)[0]);
  const sum = a.filter(x => /\.iso\.sha256$/.test(x.name));
  const size = iso.reduce((n, x) => n + x.size, 0);
  return [size, sum.length ? sum[0].browser_download_url : "-"].concat(iso.map(x => x.browser_download_url)).join("\n");
}' "$json")" || die "could not read GitHub's answer."
size="$(sed -n 1p <<<"$info")" sumurl="$(sed -n 2p <<<"$info")"
urls=()
while IFS= read -r u; do [[ -n $u ]] && urls+=("$u"); done < <(sed -n '3,$p' <<<"$info")
((${#urls[@]})) || die "the newest Ollie OS release has no ISO in it yet. Try again later."
echo "  Ollie OS: $((size / 1000000)) MB"

echo
echo "==> USB drives (everything on the one you pick is erased)"
disks=()
while read -r d; do
    [[ $d == /dev/disk* ]] || continue
    disks+=("$d")
    printf '  %d) %-12s %s\n' "${#disks[@]}" "$d" "$(diskutil info "$d" | awk -F': *' '/Media Name|Disk Size/ {printf "%s  ", $2}' | cut -c1-70)"
done < <(diskutil list external physical | awk '/^\/dev\/disk/ {print $1}')
((${#disks[@]})) || die "no USB drive was found. Plug in a USB drive (8 GB or more), then run this again."
echo
read -rp "Put Ollie OS on which drive? [1-${#disks[@]}] " pick
[[ $pick =~ ^[0-9]+$ ]] && ((pick >= 1 && pick <= ${#disks[@]})) || die "no such drive; nothing was changed."
disk="${disks[pick - 1]}"
bytes="$(diskutil info "$disk" | sed -n 's/.*Disk Size:.*(\([0-9]*\) Bytes).*/\1/p')"
[[ -z $bytes ]] || ((bytes >= size)) || die "$disk is too small for Ollie OS (it needs $((size / 1000000)) MB)."
printf '\n%sEVERYTHING on %s will be erased.%s\n' "$red$bold" "$disk" "$reset"
read -rp "Type ERASE to put Ollie OS on it: " answer
[[ $answer == ERASE ]] || die "cancelled; nothing was changed."

echo
echo "==> Writing to a drive needs this Mac's password (the one you log in with)"
sudo -v || die "that password did not work."
diskutil unmountDisk force "$disk" >/dev/null || die "the drive could not be closed: quit what has files open on it."
echo
echo "==> Downloading Ollie OS onto the drive (10 to 30 minutes)"
sumfile="$(mktemp)"
for u in "${urls[@]}"; do curl -fL --retry 3 "$u" || exit 1; done |
    tee >(shasum -a 256 | awk '{print $1}' >"$sumfile") |
    sudo dd of="${disk/disk/rdisk}" bs=4m 2>/dev/null
codes=("${PIPESTATUS[@]}")
((codes[0] == 0 && codes[2] == 0)) || die "the download or the writing stopped. Run this again."
sleep 2
if [[ $sumurl != - ]]; then
    want="$(curl -fsSL "$sumurl" | awk '{print tolower($1)}')"
    [[ -z $want || "$(cat "$sumfile")" == "$want" ]] || die "the download was damaged on the way. Run this again."
fi
sync
diskutil eject "$disk" >/dev/null 2>&1
printf '\n%s  Done: Ollie OS is on the USB drive, and checked.%s\n' "$green$bold" "$reset"
cat <<'MSG'

  Start a computer from it: Intel Mac, hold Option while it starts and pick
  "EFI Boot"; Steam Machine, hold Volume Down; Dell and most PCs, tap F12.
  (If this Mac says the drive is not readable, click Ignore: that is Ollie OS.)

Press Enter to close.
MSG
read -r _
