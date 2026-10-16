# Etros

A customised [Batocera](https://batocera.org) for the **Raspberry Pi 5** and
the **R36S** handheld, both on **one microSD card**, styled after Super
Metroid. It includes **Ollie OS**, a desktop for downloading ROMs.

- Boots to **WELCOME TO ETROS**: deep space, orange title lettering, and
  *"The last cartridge is in captivity. The galaxy is at peace..."*
- The **Etros Metroid** menu theme has the ENERGY bar and energy tanks, the
  pink-and-blue pause-screen map, and a pixel font.
- An **ETROS** entry in the system list has **Boot into Ollie OS** and
  **Sort downloaded ROMs**.
- **Ollie OS** has Bobby on the wallpaper, a web browser, and a Downloads
  folder shared with Batocera. Downloaded ROMs are moved into the right
  console's folder when you go back to Etros.
- **Your own games and ROM hacks** can be added from a computer.

## Installing

Download the installer for your computer and run it:

| Computer | File | Notes |
| --- | --- | --- |
| Windows | `Etros Installer.exe` | Choose **Yes** when Windows asks for administrator rights. If SmartScreen says "Windows protected your PC", choose **More info → Run anyway**: the installer isn't signed. |
| Mac (M1 and later) | `etros-installer-mac-apple-silicon.zip` | Unzip, then **right-click → Open** the first time (it isn't signed). It asks for your password. |
| Mac (Intel) | `etros-installer-mac-intel.zip` | As above. |
| Linux | `etros-installer-linux-x86_64.zip` | Unzip and run `sudo ./etros-installer-linux-x86_64`. |

The installer opens in your web browser:

1. Put a **32 GB or bigger** microSD card in your computer and pick it.
2. Leave **Include Ollie OS** ticked (or untick it for a smaller, quicker
   install).
3. Optionally tick **Add my games** and give it your games folder (see below).
4. Tick the box confirming the card will be erased, and choose
   **INSTALL ETROS**.

It downloads Batocera for both devices from Batocera's own servers (about
6 GB) and Raspberry Pi OS for Ollie OS (about 1.3 GB). Then it builds the
card and writes it. Downloads are kept, so a second card is much quicker.
It can also save an image file to write later with balenaEtcher.

Then put the card in either device and switch it on. The first start takes
a few minutes while Batocera sets up its storage.

Command line, for the curious: `etros-installer -list` shows the cards, and
`etros-installer -target disk:ID` or `-target file:etros.img` installs
without the window. `-help` lists the rest.

## Your games (and ROM hacks)

Etros doesn't come with games. Super Metroid, Metroid Fusion and Metroid:
Zero Mission are Nintendo's, so you need your own copies. ROM hacks are
shared as patch files (`.bps` or `.ips`) that need the original game.

Put them in a folder with one folder per console:

```
My Games/snes/Super Metroid (Japan, USA).sfc
My Games/snes/<The hack>.bps           ← next to the game it patches
My Games/gba/Metroid Fusion (USA).gba
My Games/gba/Metroid - Zero Mission (USA).gba
```

Either give that folder to the installer, or, after installing, put the
card in your computer and copy them into **my-roms** on the **BATOCERA**
drive. On the next start Etros adds the games, applies the patches (the
hack shows up as `<The hack>.sfc`), and empties `my-roms`. A `.bps` patch
finds its game by itself. An `.ips` patch must be the only game in its
folder.

## Ollie OS

Choose **ETROS → Boot into Ollie OS** in the system list. The device
restarts into a desktop with **Web Browser**, **Downloads**, **ROMs**,
**Keyboard** and **Back to Etros**. Downloads go straight into the shared
Downloads folder, and **Back to Etros** sorts them into your ROM folders
before restarting.

- **Bobby:** copy a picture to `ollie/bobby.png` (or `.jpg`) on the SHARE
  partition. Over the network that's `\\ETROS\share\ollie\`. Until then the
  wallpaper is a placeholder that says BOBBY.
- **Wi-Fi:** Ollie OS joins the networks you set up in Batocera. The R36S
  has no Wi-Fi of its own, so it needs a USB Wi-Fi dongle.
- **R36S controls:** the stick or d-pad moves the pointer. **A** clicks,
  **Y** right-clicks, **B** goes back, **X** shows the keyboard, the right
  stick scrolls, **L/R** page up and down, and **START/SELECT** are Enter
  and Escape. Hold **SELECT+START** to go back to Etros.
- Ollie OS lasts one boot. Restarting, or switching off, always comes back
  to Etros.
- Some files can't be sorted from the name alone, because many consoles
  use them: `.bin`, `.iso`, `.cue`, `.chd`, `.7z` and arcade `.zip`. Those
  stay in Downloads for you to move into **ROMs** yourself.

Ollie OS is Raspberry Pi OS, running on the Batocera kernel of whichever
device it's on. The first time it starts, it sets up its user, desktop and
wallpaper, and grows into its partition, so that start is slower.

## How it works

```
microSD card
├─ 32 KiB–16 MiB   R36S bootloader (the Pi ignores it)
├─ partition 1     BATOCERA (FAT32)
│   ├─ config.txt, firmware, bcm2712 *.dtb       ← the Pi 5 boots from these
│   ├─ boot.ini, linux, uInitrd, rk3326 *.dtb    ← the R36S boots from these
│   ├─ etros/pi5/, etros/r36s/   each device's Batocera system
│   ├─ etros/etros.squashfs      the Etros look and scripts, for both
│   ├─ etros/ollie/              Etros's settings for Ollie OS
│   └─ my-roms/                  drop-box for your games
├─ partition 3     rootfs (ext4): Ollie OS
└─ partition 2     SHARE (ext4): games, saves, settings, downloads (shared)
```

The installer ([`installer/`](installer/)) takes Batocera's official
releases for both devices and adds a hook ([`initrd/etros-hook.sh`](initrd/etros-hook.sh))
to each one's boot script (initramfs). At start-up the hook:

- tells a Pi 5 (`bcm2712`) from an R36S (`rk3326`) by the device tree, and
  mounts that device's `etros/<board>` where Batocera expects its system;
- stacks `etros.squashfs` ([`layer/`](layer/)) over Batocera, so the theme,
  splash, menu entries and start-up script replace Batocera's own files;
- boots Ollie OS instead if `etros/next-boot` says `ollie`, resetting it
  first, and lends Ollie OS that device's drivers.

The installer is a single Go program with no dependencies. It writes the
FAT32 partition itself, in one pass, straight onto the card. All the
artwork is drawn by [`scripts/make-art.py`](scripts/make-art.py), with the
Press Start 2P font (SIL Open Font License).

## Developing

- `installer/gen-assets.sh` regenerates what the installer carries inside
  it, after you change `layer/`, `ollie/`, `initrd/` or the artwork.
- `installer/build-dist.sh` builds the installer for every platform into
  `installer/dist/`.
- Tests:
  - `go test ./...` in `installer/`. Set `BATOCERA_INIT` to Batocera's real
    `init` to test against it.
  - `python3 -m unittest discover -s test` for the ROM sorting and patching.
  - `test/make-fake-images.sh`, then the installer with `-pi5-boot`,
    `-r36s-boot`, `-r36s-image` and `-rpios` pointing at the fakes, then
    `test/initrd-test.sh` on the resulting initramfs. This runs Batocera's
    boot script in a sandbox with only the commands Batocera's own
    initramfs has. `.github/workflows/etros.yml` shows the full sequence.

## Known limits

- **Not yet tried on real hardware.** The card layout, the boot partition
  and the boot script are tested, but only in a sandbox. Ollie OS on the
  R36S (display, Wi-Fi dongle, gamepad) is the most likely to need fixes.
- **Batocera's built-in updater is turned off.** It would replace the
  boot files with a single-device layout and stop the other device
  booting. To update, run the installer again.
- **The ETROS entry is a system, not a button inside Settings.** Batocera's
  settings menus are compiled code with no way to add entries.
- **R36S clones** use different screens. If yours shows nothing, follow
  Batocera's R36S notes in `overlays/readme.txt` on the BATOCERA drive.
- The Pi 5 and R36S share one `batocera.conf`. A video mode picked for the
  TV can look wrong on the handheld.
