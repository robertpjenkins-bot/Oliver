#!/usr/bin/env bash
# shellcheck disable=SC2034
# archiso profile for Annie OS. Build with ./build.sh (see README.md).

iso_name="annie-os"
iso_label="ANNIE_$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y%m)"
iso_publisher="Annie OS"
iso_application="Annie OS Live/Install DVD"
iso_version="$(date --date="@${SOURCE_DATE_EPOCH:-$(date +%s)}" +%Y.%m.%d)"
install_dir="arch"
buildmodes=('iso')
bootmodes=('bios.syslinux'
           'uefi.systemd-boot')
pacman_conf="pacman.conf"
# The live session keeps its changes in RAM; give it room for a Steam login.
kernel_params_x86_64="cow_spacesize=2G"
airootfs_image_type="squashfs"
airootfs_image_tool_options=('-comp' 'zstd' '-Xcompression-level' '15' '-b' '1M')
file_permissions=(
  ["/etc/sudoers.d/10-wheel"]="0:0:440"
  ["/etc/skel/Desktop/steam.desktop"]="0:0:755"
  ["/etc/skel/Desktop/firefox.desktop"]="0:0:755"
  ["/usr/local/bin/annie-installer"]="0:0:755"
  ["/usr/local/bin/annie-install-backend"]="0:0:755"
  ["/usr/local/bin/annie-live-setup"]="0:0:755"
  ["/usr/local/bin/annie-set-wallpaper"]="0:0:755"
  ["/usr/local/bin/annie-fit-screen"]="0:0:755"
  ["/usr/local/bin/oliver"]="0:0:755"
)
