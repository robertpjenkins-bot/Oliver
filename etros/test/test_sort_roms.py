import importlib.machinery
import importlib.util
import io
import os
import tempfile
import unittest
import zipfile

HERE = os.path.dirname(os.path.abspath(__file__))
SCRIPT = os.path.join(HERE, "..", "common", "etros-sort-roms")

loader = importlib.machinery.SourceFileLoader("etros_sort_roms", SCRIPT)
spec = importlib.util.spec_from_loader("etros_sort_roms", loader)
sorter = importlib.util.module_from_spec(spec)
loader.exec_module(sorter)


class SortTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.downloads = os.path.join(self.tmp.name, "downloads")
        self.roms = os.path.join(self.tmp.name, "roms")
        os.makedirs(self.downloads)

    def tearDown(self):
        self.tmp.cleanup()

    def touch(self, name, data=b"x"):
        path = os.path.join(self.downloads, name)
        with open(path, "wb") as f:
            f.write(data)
        return path

    def make_zip(self, name, members):
        path = os.path.join(self.downloads, name)
        with zipfile.ZipFile(path, "w") as z:
            for m in members:
                z.writestr(m, b"rom")
        return path

    def sort(self):
        return sorter.sort_downloads(self.downloads, self.roms, out=io.StringIO())

    def test_known_extensions_move(self):
        self.touch("Super Metroid (USA).sfc")
        self.touch("Metroid II.GB")
        moved, left = self.sort()
        self.assertEqual((moved, left), (2, 0))
        self.assertTrue(os.path.exists(os.path.join(self.roms, "snes", "Super Metroid (USA).sfc")))
        self.assertTrue(os.path.exists(os.path.join(self.roms, "gb", "Metroid II.GB")))

    def test_ambiguous_extensions_stay(self):
        self.touch("game.bin")
        self.touch("game.iso")
        self.assertEqual(self.sort(), (0, 2))
        self.assertTrue(os.path.exists(os.path.join(self.downloads, "game.bin")))

    def test_zip_with_one_system_moves_zipped(self):
        self.make_zip("Zero Mission.zip", ["Zero Mission.gba"])
        self.assertEqual(self.sort(), (1, 0))
        self.assertTrue(os.path.exists(os.path.join(self.roms, "gba", "Zero Mission.zip")))

    def test_mixed_or_unknown_zip_stays(self):
        self.make_zip("mixed.zip", ["a.gba", "b.nes"])
        self.make_zip("arcade.zip", ["mslug.p1"])
        self.touch("broken.zip", b"not a zip")
        self.assertEqual(self.sort(), (0, 3))

    def test_partial_downloads_ignored(self):
        self.touch("game.sfc.crdownload")
        self.touch(".hidden.sfc")
        self.assertEqual(self.sort(), (0, 0))

    def test_existing_file_not_overwritten(self):
        os.makedirs(os.path.join(self.roms, "nes"))
        with open(os.path.join(self.roms, "nes", "game.nes"), "w") as f:
            f.write("old")
        self.touch("game.nes", b"new")
        self.assertEqual(self.sort(), (0, 1))
        with open(os.path.join(self.roms, "nes", "game.nes")) as f:
            self.assertEqual(f.read(), "old")

    def test_pico8_cart(self):
        self.touch("celeste.p8.png")
        self.touch("photo.png")
        self.assertEqual(self.sort(), (1, 1))
        self.assertTrue(os.path.exists(os.path.join(self.roms, "pico8", "celeste.p8.png")))

    def test_missing_downloads_folder(self):
        self.assertEqual(sorter.sort_downloads("/nonexistent", self.roms), (0, 0))


class SettleTest(unittest.TestCase):
    def test_leaves_files_still_arriving(self):
        with tempfile.TemporaryDirectory() as d:
            dl, roms = os.path.join(d, "My ROMs"), os.path.join(d, "roms")
            os.makedirs(dl)
            old, new = os.path.join(dl, "Old.sfc"), os.path.join(dl, "New.gba")
            for p in (old, new):
                open(p, "wb").write(b"x")
            os.utime(old, (0, 0))
            moved, _ = sorter.sort_downloads(dl, roms, out=io.StringIO(), settle=5)
            self.assertEqual(moved, 1)
            self.assertTrue(os.path.exists(os.path.join(roms, "snes", "Old.sfc")))
            self.assertTrue(os.path.exists(new))


if __name__ == "__main__":
    unittest.main()
