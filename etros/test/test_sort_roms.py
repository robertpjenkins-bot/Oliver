import importlib.machinery
import importlib.util
import io
import os
import shutil
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



class MyRomsTest(unittest.TestCase):
    def test_console_folders_loose_files_and_not_sorted(self):
        with tempfile.TemporaryDirectory() as d:
            base, roms = os.path.join(d, "My ROMs"), os.path.join(d, "roms")
            os.makedirs(base)
            sorter.sort_my_roms(base, roms, out=io.StringIO())  # makes the folders
            self.assertTrue(os.path.isdir(os.path.join(base, "PlayStation")))
            self.assertTrue(os.path.isfile(os.path.join(base, "READ ME.txt")))

            def put(rel):
                p = os.path.join(base, rel)
                os.makedirs(os.path.dirname(p), exist_ok=True)
                open(p, "wb").write(b"x")
            put("Super Metroid (USA).sfc")                  # loose: by extension
            put("Downloaded pack/Metroid Fusion.gba")       # inside a downloaded folder
            put("Crash.bin")                                # loose but ambiguous
            put("PlayStation/Spyro (USA)/Spyro.cue")        # a multi-file game
            put("PlayStation/Spyro (USA)/Spyro.bin")
            put("Arcade/sf2.zip")                           # arcade zip
            put("Partial.sfc.crdownload")                   # still downloading

            moved = sorter.sort_my_roms(base, roms, out=io.StringIO())
            self.assertEqual(moved, 4)
            for rel in ("snes/Super Metroid (USA).sfc", "gba/Metroid Fusion.gba",
                        "psx/Spyro (USA)/Spyro.cue", "psx/Spyro (USA)/Spyro.bin", "mame/sf2.zip"):
                self.assertTrue(os.path.exists(os.path.join(roms, rel)), rel)
            self.assertTrue(os.path.exists(os.path.join(base, sorter.NOT_SORTED, "Crash.bin")))
            self.assertTrue(os.path.exists(os.path.join(base, "Partial.sfc.crdownload")))
            self.assertFalse(os.path.exists(os.path.join(base, "Downloaded pack")))
            self.assertTrue(os.path.isdir(os.path.join(base, "PlayStation")))



class ArchiveTest(unittest.TestCase):
    def test_zip_with_a_readme_beside_the_game(self):
        with tempfile.TemporaryDirectory() as d:
            p = os.path.join(d, "Metroid Fusion (USA).zip")
            with zipfile.ZipFile(p, "w") as z:
                z.writestr("Metroid Fusion (USA).gba", b"x" * 100)
                z.writestr("readme.txt", b"hi")
                z.writestr("box.jpg", b"img")
            self.assertEqual(sorter.system_for(p), "gba")

    @unittest.skipUnless(shutil.which("7z"), "needs 7z to make the test archives")
    def test_7z_read_without_a_7z_program(self):
        import subprocess
        with tempfile.TemporaryDirectory() as d:
            rom = os.path.join(d, "Super Metroid (USA).sfc")
            open(rom, "wb").write(os.urandom(50000))
            open(os.path.join(d, "readme.txt"), "w").write("hi")
            for method in ("-m0=LZMA2", "-m0=LZMA", "-m0=Copy", "-mhc=off"):
                arc = os.path.join(d, "game%s.7z" % method)
                subprocess.run(["7z", "a", "-bd", "-bso0", method, arc, rom, os.path.join(d, "readme.txt")], check=True)
                self.assertEqual(sorter.sevenzip_names(arc), ["Super Metroid (USA).sfc", "readme.txt"], method)
                self.assertEqual(sorter.system_for(arc), "snes", method)


if __name__ == "__main__":
    unittest.main()
