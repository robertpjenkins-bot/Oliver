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

Download the zip for your computer, unzip it, and run the installer inside:

| Computer | Zip | Notes |
| --- | --- | --- |
| Windows | `Etros Installer - Windows.zip` | Double-click **Etros Installer.exe**. If SmartScreen says "Windows protected your PC", choose **More info → Run anyway** (it isn't signed). Choose **Yes** for administrator rights. |
| Mac | `Etros Installer - Mac (Apple chip).zip` or `(Intel).zip` | Right-click **etros-installer → Open** the first time (it isn't signed). It asks for your password. |
| Linux | `Etros Installer - Linux.zip` | Double-click **Etros Installer** (choose **Run**, or right-click → **Allow Launching** first), or run `./etros-installer` in a terminal. It asks for your password. On a Chromebook, use the Terminal app. |

It runs in a terminal window. First it looks in your **Downloads** folder
(and the folder it was unzipped to) for:

- **an Etros image** (`etros-….img.gz`, `.img`, `.img.xz`, or a `.zip`
  with one inside). If there is one, it offers to write it straight to a
  card.
- **Batocera** for each device, as downloaded from batocera.org:
  `batocera-bcm2712-….img.gz` for the Pi 5 and `batocera-…-r36s-….img.gz`
  for the R36S. A `boot.tar.xz` works too.
- **Ollie OS**: an `ollie….img` of your own, or a Raspberry Pi OS 64-bit
  desktop image (`…raspios…arm64.img.xz`).
- **your games**: a folder called `my-roms`.

Anything it doesn't find, it downloads. The first time, it makes the Etros
image and saves it to Downloads as `etros-<date>.img.gz` (about 5 GB). Then
it writes it to the card you pick, after you type **ERASE** to confirm.
After that the image is found straight away, so more cards are quick. The
image also works with balenaEtcher or Raspberry Pi Imager.

Then put the card in either device and switch it on. The first start takes
a few minutes while Batocera sets up its storage.

For scripts: `etros-installer -list` shows the cards. Use
`etros-installer -target file:etros.img.gz` to make an image,
`-write etros.img.gz -target disk:ID` to write one, and `-help` for the
rest.

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

Name that folder `my-roms` and put it in Downloads, and the installer adds
it. Or, after installing, put the card in your computer and copy them into **my-roms** on the **BATOCERA**
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

Ollie OS is Raspberry Pi OS (or your own Ollie OS image, if you put one in
Downloads), running on the Batocera kernel of whichever device it's on. The first time it starts, it sets up its user, desktop and
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
├─ partition 3     Ollie OS (ext4)
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

The installer is a single Go program with no dependencies. It reads the
Batocera images and writes the FAT32 partition itself, in one pass each,
without unpacking anything to disk. All the
artwork is drawn by [`scripts/make-art.py`](scripts/make-art.py), with the
Press Start 2P font (SIL Open Font License).

## Developing

- `installer/gen-assets.sh` regenerates what the installer carries inside
  it, after you change `layer/`, `ollie/`, `initrd/` or the artwork.
- `installer/build-dist.sh` builds the zips for every platform into
  `installer/dist/`.
- Tests:
  - `go test ./...` in `installer/`. Set `BATOCERA_INIT` to Batocera's real
    `init` to test against it.
  - `python3 -m unittest discover -s test` for the ROM sorting and patching.
  - `test/make-fake-images.sh`, then the installer with `-pi5-boot`,
    `-r36s-boot` and `-rpios` pointing at the fakes, then
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
