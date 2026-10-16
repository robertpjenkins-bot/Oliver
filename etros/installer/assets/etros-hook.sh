#!/bin/ash
# Etros additions to Batocera's initramfs init. Sourced by /init right after
# the boot partition is mounted on /boot_root, so everything here runs as
# part of do_root() and may use its helpers and variables.
#
# Batocera's initramfs busybox is small: no tr, grep, sync, chmod, tar or
# ln. Stick to cat, cp, mkdir, mount, umount, echo, test and shell builtins.
#
# One SD card carries two Batocera builds, one for the Raspberry Pi 5 and one
# for the R36S, plus Ollie OS. Their system images live side by side on the
# boot partition:
#
#   /etros/pi5/batocera     Pi 5 system image (and its saved overlay, version...)
#   /etros/r36s/batocera    R36S system image
#   /etros/etros.squashfs   the Etros layer, shared by both boards
#   /etros/ollie/           Etros's files for Ollie OS, copied in on each start
#   /etros/next-boot        "ollie" to start Ollie OS once on the next boot
#
# Ollie OS itself is a Raspberry Pi OS root file system on its own partition,
# labelled "rootfs".
#
# Batocera expects its images in /boot/boot, so the directory for the board
# we are running on is bound over /boot_root/boot and the rest of Batocera
# never notices it is sharing the card.

# shellcheck disable=SC2034  # used by the patched /init
ETROS_LAYER=

etros_board() {
    local compat
    # The file is a list of NUL-separated strings; the shell drops the NULs,
    # which leaves the names run together but still searchable.
    compat=$(cat "${ETROS_COMPATIBLE:-/proc/device-tree/compatible}" 2>/dev/null)
    case "${compat}" in
        *bcm2712*) echo pi5 ;;
        *rk3326*)  echo r36s ;;
    esac
}

etros_bind_board() {
    ETROS_BOARD=$(etros_board)
    if test -z "${ETROS_BOARD}" || ! test -d "/boot_root/etros/${ETROS_BOARD}"
    then
        echo "etros: no system image for this board, using /boot as is"
        return 0
    fi

    # A bind mount takes its read-only flag from the mount it was made from.
    # Make it from a read-write mount so that the later "remount,rw" Batocera
    # does on the whole partition reaches /boot/boot as well.
    mount -o remount,rw /boot_root                                 || return 1
    mount --bind "/boot_root/etros/${ETROS_BOARD}" /boot_root/boot || return 1
    mount -o remount,ro /boot_root                                 || return 1
    echo "etros: booting the ${ETROS_BOARD} system"
}

# Ollie OS is started once: the request is cleared before booting into it, so
# turning the device off and on again always comes back to Batocera.
etros_want_ollie() {
    local next
    read -r next < /boot_root/etros/next-boot 2>/dev/null || return 1
    test "${next}" = "ollie" || return 1

    mount -o remount,rw /boot_root || return 1
    echo batocera > /boot_root/etros/next-boot
    mount -o remount,ro /boot_root || return 1
    return 0
}

etros_boot_ollie() {
    echo "etros: starting Ollie OS"
    mount -o rw "${ETROS_OLLIE_ROOT:-LABEL=rootfs}" /new_root || return 1

    # Etros's settings and scripts for Ollie OS. Copied every start so a new
    # Etros brings new ones; Ollie OS's own setup service fixes permissions,
    # which the FAT partition cannot carry.
    if test -d /boot_root/etros/ollie
    then
        cp -r /boot_root/etros/ollie/. /new_root/ || return 1
    fi

    # Ollie OS runs on this board's Batocera kernel, so it uses that
    # kernel's modules and firmware, from the Batocera image.
    mkdir -p /new_root/.batocera /new_root/boot /new_root/lib/modules /new_root/lib/firmware || return 1
    if mount -o ro /boot_root/boot/batocera /new_root/.batocera
    then
        mount --bind /new_root/.batocera/lib/modules  /new_root/lib/modules
        mount --bind /new_root/.batocera/lib/firmware /new_root/lib/firmware
    else
        echo "etros: no Batocera image to take drivers from"
    fi

    mount --move /boot_root /new_root/boot || return 1
    mount --move /sys       /new_root/sys  || return 1
    mount --move /proc      /new_root/proc || return 1
    mount --move /dev       /new_root/dev  || return 1

    exec switch_root /new_root /sbin/init
}

# Called just before the Batocera root is assembled: stack the Etros layer
# above the Batocera image so its files replace Batocera's.
etros_mount_layer() {
    test -f /boot_root/etros/etros.squashfs || return 0
    mkdir -p /overlay_root/etros || return 1
    if mount /boot_root/etros/etros.squashfs /overlay_root/etros
    then
        # shellcheck disable=SC2034
        ETROS_LAYER="/overlay_root/etros:"
    fi
    return 0
}

etros_start() {
    etros_bind_board || return 1
    if etros_want_ollie
    then
        # Only returns if Ollie OS could not be started; carry on into
        # Batocera with a clean /new_root.
        etros_boot_ollie
        echo "etros: Ollie OS failed to start, falling back to Batocera"
        umount /new_root/lib/firmware /new_root/lib/modules /new_root/.batocera 2>/dev/null
        umount /new_root 2>/dev/null
    fi
    return 0
}
