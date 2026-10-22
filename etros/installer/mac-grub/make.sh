#!/bin/bash
# Makes assets/mac-bootx64.efi: GRUB for Intel Macs, with embedded.cfg.
# Needs grub-mkimage and GRUB's x86_64-efi modules (Debian/Ubuntu:
# grub-common and grub-efi-amd64-bin, or unpacked: GRUB_ROOT=/that/dir).
set -e
cd "$(dirname "$0")"
root=${GRUB_ROOT:-}
"$root/usr/bin/grub-mkimage" -d "$root/usr/lib/grub/x86_64-efi" -O x86_64-efi \
    -o ../assets/mac-bootx64.efi -p /EFI/BOOT -c embedded.cfg \
    part_gpt part_msdos fat ext2 normal linux configfile regexp test echo \
    all_video efi_gop efi_uga gfxterm gzio search search_label
