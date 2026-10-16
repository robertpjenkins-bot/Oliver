# Etros

A customised [Batocera](https://batocera.org) for the **Raspberry Pi 5** and
the **R36S** handheld, both on **one SD card**, styled after Super Metroid.
It also includes **Ollie OS**, a small desktop for downloading ROMs.

- Boots to a **WELCOME TO ETROS** splash: deep space, orange title lettering,
  and a typed-out *"The last cartridge is in captivity. The galaxy is at
  peace..."*
- The menus use the **Etros Metroid** theme: the in-game ENERGY bar and
  energy tanks along the top, the pink-and-blue pause-screen map in the
  corner, and a pixel font.
- An **ETROS** entry in the system list has **Boot into Ollie OS** and
  **Sort downloaded ROMs**.
- **Ollie OS** has Bobby as the wallpaper, a web browser, and a Downloads
  folder shared with Batocera. When you go back to Etros, the ROMs you
  downloaded are moved into the right console's folder.
- **Your own games**: copy them, and ROM-hack patches, onto the card from a
  computer. Etros adds them and applies the patches.

The same card works in either device. Saves and games are shared, so you
can start a game on the TV and carry on on the handheld.

## Getting the image

1. On GitHub, go to **Actions → Etros image → Run workflow**. A build takes
   about an hour.
2. When it finishes, open **Releases**. Download every
   `etros-….img.xz.partNN` file into one folder.
3. Join the parts into one file:
   - **Windows** (Command Prompt, in that folder):
     `copy /b etros-VERSION.img.xz.part00 + etros-VERSION.img.xz.part01 + etros-VERSION.img.xz.part02 etros-VERSION.img.xz`
     (list every part, in order)
   - **Mac / Linux**: `cat etros-*.img.xz.part* > etros.img.xz`
4. Write `etros….img.xz` to a microSD card (**32 GB or more**) with
   [Raspberry Pi Imager](https://www.raspberrypi.com/software/) ("Use
   custom") or [balenaEtcher](https://etcher.balena.io/).

GitHub can't keep a multi-GB image as a build artifact on a private
repository, which is why it comes as a release in parts.

## First boot

Put the card in either device and switch it on. On the first boot Batocera
grows its storage partition to fill the card, so give it a few minutes. The
other device skips this step when it first sees the card.

## Adding your games (and ROM hacks)

Etros doesn't ship any games. Super Metroid, Metroid Fusion and Metroid:
Zero Mission are Nintendo's, so you need your own copies. ROM hacks are
shared as patch files (`.bps` or `.ips`) that need the original game.

1. Put the card in a computer. It shows up as a drive called **BATOCERA**.
2. Open the **my-roms** folder and copy each game into the folder for its
   console:

   ```
   my-roms/snes/Super Metroid (Japan, USA).sfc
   my-roms/snes/<The hack>.bps            ← the hack's patch, next to Super Metroid
   my-roms/gba/Metroid Fusion (USA).gba
   my-roms/gba/Metroid - Zero Mission (USA).gba
   ```

3. Put the card back in the device and switch it on. The games appear in
   the menu. The hack shows up under the patch's name, e.g.
   `<The hack>.sfc`. Etros then empties `my-roms`.

A `.bps` patch knows which game it was made for, so it finds Super Metroid
by itself, even with other SNES games in the folder. An `.ips` patch
doesn't, so it must be the only game in its folder. If a patch doesn't fit
your copy, the files stay in `my-roms` and the reason is written to
`system/logs/etros.log` on the SHARE partition.

You can also bake games into the image at build time: put them in
`etros/my-roms/` (ignored by git, so they never get uploaded) and run the
build yourself (see *Building it yourself*).

## Ollie OS

From the system list, open **ETROS → Boot into Ollie OS**. The device
restarts into a desktop with:

| Icon | What it does |
| --- | --- |
| Web Browser | Chromium. Downloads go straight to the Downloads folder. |
| Downloads | The shared downloads folder (`/userdata/downloads`) |
| ROMs | Batocera's ROM folders, if you want to move something by hand |
| Back to Etros | Sorts your downloads into the ROM folders and restarts into Etros |
| Keyboard | On-screen keyboard |

**Bobby wallpaper:** copy a picture to `ollie/bobby.png` (or `.jpg`) on the
SHARE partition. Over the network that's `\\ETROS\share\ollie\bobby.png`.
Until you do, the wallpaper is an Etros-style placeholder that says BOBBY.

**Wi-Fi:** Ollie OS joins the networks you set up in Batocera (**Network
Settings**), so there's nothing to type. The R36S has no Wi-Fi of its own, so
it needs a USB Wi-Fi dongle on its OTG port.

**On the R36S** (no mouse or keyboard), the buttons drive the desktop:

| Control | Does |
| --- | --- |
| Left stick / d-pad | move the pointer |
| A | click |
| Y | right-click |
| B | browser back |
| X | on-screen keyboard |
| Right stick | scroll |
| L / R | page up / down |
| START / SELECT | Enter / Escape |
| hold SELECT + START | back to Etros |

Ollie OS only lasts one boot. Restarting it, or switching the device off and
on, always comes back to Etros. Your downloads, browser bookmarks and Bobby
picture are kept on the SHARE partition. Anything else you change in Ollie
OS is reset on every start.

**Sorting downloads:** ROMs are moved by file type (`.sfc` → snes, `.gba` →
gba, and so on). A `.zip` with one kind of ROM inside goes to that console,
still zipped. Some files can't be sorted from the name alone, because many
consoles use them: `.bin`, `.iso`, `.cue`, `.chd`, `.7z`, and arcade `.zip`.
Those stay in Downloads for you to move into `ROMs/<console>`.

## How it works

```
SD card
├─ 32 KiB–16 MiB  R36S bootloader (idbloader, U-Boot, trust), the Pi ignores it
├─ partition 1    BATOCERA (FAT32)
│   ├─ config.txt, firmware, bcm2712 *.dtb      ← Pi 5 boots from these
│   ├─ boot.ini, linux, uInitrd, rk3326 *.dtb   ← R36S boots from these
│   ├─ etros/pi5/    Batocera system image for the Pi 5 (+ its kernel)
│   ├─ etros/r36s/   Batocera system image for the R36S
│   ├─ etros/etros.squashfs   the Etros look and scripts, for both
│   ├─ etros/ollie.squashfs   Ollie OS
│   └─ my-roms/      drop-box for your games
└─ partition 2    SHARE (ext4): games, saves, settings, downloads, shared
```

The build takes the official Batocera images for both devices and patches
their boot script (initramfs) with [`initrd/etros-hook.sh`](initrd/etros-hook.sh):

- It reads the device tree to tell a Pi 5 (`bcm2712`) from an R36S
  (`rk3326`), and mounts that board's `etros/<board>` directory where
  Batocera expects its system. The rest of Batocera doesn't know it is
  sharing the card.
- It stacks `etros.squashfs` on top of the Batocera system. The theme, the
  splash, the ETROS menu entries and the start-up script
  ([`layer/`](layer/)) override Batocera's files without unpacking or
  rebuilding Batocera.
- If `etros/next-boot` says `ollie`, it resets that to `batocera` and boots
  Ollie OS instead. Ollie OS is Debian with Openbox, PCManFM and Chromium
  ([`ollie/`](ollie/)), running on the same Batocera kernel and borrowing
  its drivers.

All the artwork is drawn by [`scripts/make-art.py`](scripts/make-art.py),
using the Press Start 2P font (SIL Open Font License).

## Building it yourself

On Linux (Debian or Ubuntu), as root:

```sh
sudo apt-get install squashfs-tools mtools u-boot-tools lz4 cpio fdisk dosfstools \
    e2fsprogs ffmpeg python3-pil xz-utils mmdebstrap qemu-user-static binfmt-support
sudo ./build.sh            # → build/etros-<version>.img.xz
```

| Setting | Meaning |
| --- | --- |
| `PI5_IMAGE`, `R36S_IMAGE` | Batocera `.img.gz` files to use instead of downloading the latest stable release |
| `SKIP_OLLIE=1` | Leave Ollie OS out (much faster) |
| `OLLIE_SQUASHFS` | Use an Ollie OS you built earlier |
| `MY_ROMS` | Folder of games and patches to put in `my-roms` (default `etros/my-roms`) |

`python3 -m unittest discover -s test` runs the ROM-sorting and patching
tests. `test/make-fake-images.sh` plus `test/initrd-test.sh` build a card
from stand-in Batocera images and run Batocera's real boot script against
it, in a sandbox. CI does both on every pull request.

## Known limits

- **Not yet tried on real hardware.** The card layout and the boot script
  are tested, but only in a sandbox. Ollie OS's display, Wi-Fi and gamepad
  support on the R36S are the most likely to need fixes.
- **Batocera's built-in updater is turned off.** It would replace the
  boot files with a single-board layout and stop the other device booting.
  To update, build a new Etros image.
- **The ETROS entry is a system, not a button inside Settings.** Batocera's
  menus come from compiled code with no way to add entries. Changing that
  means rebuilding Batocera from source.
- **R36S clones** use different screens. Batocera picks the screen driver from
  `overlays/mipi-panel.dtbo`. If yours shows nothing, follow Batocera's R36S
  notes in `overlays/readme.txt` on the BATOCERA drive.
- Pi 5 and R36S share one `batocera.conf`. Most settings suit both. A video
  mode chosen for the TV can make the handheld's screen look wrong.
