#!/usr/bin/env python3
"""Add your own ROMs to Etros, applying ROM-hack patches on the way.

    add-my-roms.py MY_ROMS ROMS_DIR

MY_ROMS holds one folder per Batocera system, named as in /userdata/roms:

    my-roms/snes/Super Metroid (Japan, USA).sfc
    my-roms/snes/Some Hack.bps
    my-roms/gba/Metroid Fusion (USA).gba
    my-roms/gba/Metroid - Zero Mission (USA).gba

ROMs are copied to ROMS_DIR/<system>/. Each .bps or .ips patch is applied to
a ROM in the same folder and the result is saved under the patch's name
with that ROM's extension ("Some Hack.sfc"). A BPS patch records the
checksum of the ROM it was made for, so the right ROM is found by itself.
An IPS patch does not, so its folder must hold exactly one ROM.

On the card, MY_ROMS is the my-roms folder on the BATOCERA drive; Etros runs
this at boot and then empties it.
"""
import os
import shutil
import sys
import zlib

PATCHES = (".bps", ".ips")
SKIP = {"README.txt", "Thumbs.db", "desktop.ini"}


def apply_ips(rom, patch):
    if patch[:5] != b"PATCH":
        raise ValueError("not an IPS patch")
    out = bytearray(rom)
    i = 5
    while True:
        if patch[i:i + 3] == b"EOF":
            i += 3
            break
        offset = int.from_bytes(patch[i:i + 3], "big")
        size = int.from_bytes(patch[i + 3:i + 5], "big")
        i += 5
        if size:
            data = patch[i:i + size]
            i += size
        else:  # run-length record
            count = int.from_bytes(patch[i:i + 2], "big")
            data = patch[i + 2:i + 3] * count
            i += 3
        if len(out) < offset + len(data):
            out.extend(b"\0" * (offset + len(data) - len(out)))
        out[offset:offset + len(data)] = data
    if len(patch) - i == 3:  # optional truncation size
        out = out[:int.from_bytes(patch[i:i + 3], "big")]
    return bytes(out)


def _bps_number(data, i):
    value, shift = 0, 1
    while True:
        b = data[i]
        i += 1
        value += (b & 0x7F) * shift
        if b & 0x80:
            return value, i
        shift <<= 7
        value += shift


def bps_source_crc(patch):
    return int.from_bytes(patch[-12:-8], "little")


def apply_bps(rom, patch):
    if patch[:4] != b"BPS1":
        raise ValueError("not a BPS patch")
    if zlib.crc32(patch[:-4]) != int.from_bytes(patch[-4:], "little"):
        raise ValueError("BPS patch is damaged (checksum mismatch)")
    if zlib.crc32(rom) != bps_source_crc(patch):
        raise ValueError("this BPS patch was made for a different ROM")
    i = 4
    _, i = _bps_number(patch, i)
    target_size, i = _bps_number(patch, i)
    meta_size, i = _bps_number(patch, i)
    i += meta_size
    out = bytearray()
    src_rel = tgt_rel = 0
    end = len(patch) - 12
    while i < end:
        n, i = _bps_number(patch, i)
        action, length = n & 3, (n >> 2) + 1
        if action == 0:  # SourceRead
            out += rom[len(out):len(out) + length]
        elif action == 1:  # TargetRead
            out += patch[i:i + length]
            i += length
        else:
            d, i = _bps_number(patch, i)
            delta = -(d >> 1) if d & 1 else d >> 1
            if action == 2:  # SourceCopy
                src_rel += delta
                out += rom[src_rel:src_rel + length]
                src_rel += length
            else:  # TargetCopy, may overlap what it is writing
                tgt_rel += delta
                for _ in range(length):
                    out.append(out[tgt_rel])
                    tgt_rel += 1
    out = bytes(out)
    if len(out) != target_size or zlib.crc32(out) != int.from_bytes(patch[-8:-4], "little"):
        raise ValueError("patching produced the wrong result")
    return out


def add_system(src, dest, log):
    names = sorted(n for n in os.listdir(src)
                   if os.path.isfile(os.path.join(src, n)) and not n.startswith(".") and n not in SKIP)
    roms = [n for n in names if not n.lower().endswith(PATCHES)]
    patches = [n for n in names if n.lower().endswith(PATCHES)]
    os.makedirs(dest, exist_ok=True)
    for name in roms:
        shutil.copy2(os.path.join(src, name), os.path.join(dest, name))
        log(f"added {name}")
    for name in patches:
        with open(os.path.join(src, name), "rb") as f:
            patch = f.read()
        if name.lower().endswith(".bps"):
            wanted = bps_source_crc(patch)
            base = rom = None
            for r in roms:
                with open(os.path.join(src, r), "rb") as f:
                    data = f.read()
                # Some SNES dumps carry a 512-byte copier header that the
                # patch author's ROM did not have.
                for candidate in (data, data[512:] if len(data) % 1024 == 512 else None):
                    if candidate is not None and zlib.crc32(candidate) == wanted:
                        base, rom = r, candidate
                        break
                if base:
                    break
            if base is None:
                raise SystemExit(f"{name}: none of the ROMs next to it is the one it was made for "
                                 f"(it wants a ROM with CRC32 {wanted:08X})")
        else:
            if len(roms) != 1:
                raise SystemExit(f"{name}: an IPS patch needs exactly one ROM in its folder, found {len(roms)}")
            base = roms[0]
            with open(os.path.join(src, base), "rb") as f:
                rom = f.read()
        apply = apply_bps if name.lower().endswith(".bps") else apply_ips
        result = apply(rom, patch)
        out = os.path.splitext(name)[0] + os.path.splitext(base)[1]
        with open(os.path.join(dest, out), "wb") as f:
            f.write(result)
        log(f"patched {base} with {name} -> {out}")


def main():
    if len(sys.argv) != 3:
        raise SystemExit("usage: add-my-roms.py MY_ROMS ROMS_DIR")
    src, out = sys.argv[1:]
    if not os.path.isdir(src):
        return
    for system in sorted(os.listdir(src)):
        path = os.path.join(src, system)
        if os.path.isdir(path) and not system.startswith("."):
            add_system(path, os.path.join(out, system), lambda m, s=system: print(f"{s}: {m}"))


if __name__ == "__main__":
    main()
