# Etros in a virtual ARM computer

Boots a card put together the way Etros Maker's no-password install makes
it, in QEMU's ARM "virt" machine, through Batocera's own initramfs `/init`
patched by Etros Maker's real code. VirtualBox can't do this: it only runs
PC (x86) systems, and Etros is for ARM.

Raspberry Pi OS and Batocera can't be downloaded from the build sandbox, so
the Ollie OS here is a stand-in: Ubuntu arm64 with LightDM and a PCManFM
desktop, like Raspberry Pi OS's. Batocera is the stand-in from
`../make-fake-images.sh`, which can't run, so only the Ollie OS side
boots. Everything of Etros's (the boot hook, Ollie OS's setup, My ROMs,
the start-up check, Back to Etros) is the real thing.

    sudo ./build-root.sh      # the stand-in Ollie OS (slow: ARM emulation)
    sudo ./make-initrd.sh     # Batocera's /init + busybox, patched by Etros Maker
    sudo ./make-card.sh       # the card; asks for Ollie OS on the next start
    sudo ./run-vm.sh 900      # boot it, screenshot every 30 s into run/
    python3 qmp.py run/qmp.sock dclick X Y    # double-click on the screen

Paths are under /var/tmp/ovm. Checked on 2026-10-18: Ollie OS starts, the
desktop comes up with every icon, SHARE is at /userdata, ROMs in My ROMs
(loose, and in the PlayStation folder) land in roms/snes and roms/psx,
folders open with a double-click without asking, and Back to Etros
restarts into Etros.
