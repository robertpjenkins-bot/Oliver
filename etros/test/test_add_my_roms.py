import importlib.util
import os
import tempfile
import unittest
import zlib

HERE = os.path.dirname(os.path.abspath(__file__))
spec = importlib.util.spec_from_file_location("add_my_roms", os.path.join(HERE, "..", "scripts", "add-my-roms.py"))
mod = importlib.util.module_from_spec(spec)
spec.loader.exec_module(mod)


def number(n):
    out = bytearray()
    while True:
        x = n & 0x7F
        n >>= 7
        if n == 0:
            out.append(0x80 | x)
            return bytes(out)
        out.append(x)
        n -= 1


def make_bps(source, target):
    """A simple valid BPS: copy the unchanged prefix, then write the rest."""
    same = 0
    while same < min(len(source), len(target)) and source[same] == target[same]:
        same += 1
    body = b"BPS1" + number(len(source)) + number(len(target)) + number(0)
    if same:
        body += number(((same - 1) << 2) | 0)
    if len(target) > same:
        body += number(((len(target) - same - 1) << 2) | 1) + target[same:]
    body += zlib.crc32(source).to_bytes(4, "little") + zlib.crc32(target).to_bytes(4, "little")
    return body + zlib.crc32(body).to_bytes(4, "little")


class PatchTest(unittest.TestCase):
    def test_ips(self):
        rom = bytes(16)
        patch = b"PATCH" + (2).to_bytes(3, "big") + (2).to_bytes(2, "big") + b"AB" \
            + (8).to_bytes(3, "big") + (0).to_bytes(2, "big") + (3).to_bytes(2, "big") + b"Z" + b"EOF"
        out = mod.apply_ips(rom, patch)
        self.assertEqual(out[2:4], b"AB")
        self.assertEqual(out[8:11], b"ZZZ")
        self.assertEqual(len(out), 16)

    def test_bps(self):
        src = b"SUPER METROID" + bytes(100)
        tgt = b"SUPER METROID" + b"hack!" * 30
        self.assertEqual(mod.apply_bps(src, make_bps(src, tgt)), tgt)

    def test_bps_wrong_rom(self):
        patch = make_bps(b"a" * 20, b"b" * 20)
        with self.assertRaises(ValueError):
            mod.apply_bps(b"c" * 20, patch)

    def test_folder(self):
        with tempfile.TemporaryDirectory() as d:
            src, out = os.path.join(d, "in", "snes"), os.path.join(d, "out")
            os.makedirs(src)
            base = b"SM" * 1024
            with open(os.path.join(src, "Super Metroid.sfc"), "wb") as f:
                f.write(b"\0" * 512 + base)  # headered dump
            with open(os.path.join(src, "Other.sfc"), "wb") as f:
                f.write(b"other")
            with open(os.path.join(src, "My Hack.bps"), "wb") as f:
                f.write(make_bps(base, base + b"NEW"))
            mod.add_system(src, os.path.join(out, "snes"), lambda m: None)
            self.assertEqual(sorted(os.listdir(os.path.join(out, "snes"))),
                             ["My Hack.sfc", "Other.sfc", "Super Metroid.sfc"])
            with open(os.path.join(out, "snes", "My Hack.sfc"), "rb") as f:
                self.assertEqual(f.read(), base + b"NEW")


if __name__ == "__main__":
    unittest.main()
