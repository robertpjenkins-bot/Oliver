package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

type nopReporter struct{}

func (nopReporter) Stage(string)          {}
func (nopReporter) Log(string)            {}
func (nopReporter) Progress(int64, int64) {}

// testImage is a small disk image: a partition table with one partition,
// and random contents.
func testImage(t *testing.T) []byte {
	img := make([]byte, 6*mib)
	rand.Read(img[512:])
	e := img[446:]
	e[4] = 0x0C
	binary.LittleEndian.PutUint32(e[8:], 2048)
	binary.LittleEndian.PutUint32(e[12:], 6*2048-2048)
	img[510], img[511] = 0x55, 0xAA
	return img
}

func TestWriteImageAllPackings(t *testing.T) {
	img := testImage(t)
	dir := t.TempDir()
	raw := filepath.Join(dir, "etros.img")
	os.WriteFile(raw, img, 0o644)

	var gz bytes.Buffer
	w := gzip.NewWriter(&gz)
	w.Write(img)
	w.Close()
	gzPath := filepath.Join(dir, "etros.img.gz")
	os.WriteFile(gzPath, gz.Bytes(), 0o644)

	zipPath := filepath.Join(dir, "etros.zip")
	zf, _ := os.Create(zipPath)
	zw := zip.NewWriter(zf)
	zw.Create("Etros Installer.exe")
	f, _ := zw.CreateHeader(&zip.FileHeader{Name: "etros.img.gz", Method: zip.Store})
	f.Write(gz.Bytes())
	zw.Close()
	zf.Close()

	for _, src := range []string{raw, gzPath, zipPath} {
		outPath := filepath.Join(dir, "card-"+filepath.Base(src))
		out, _ := os.Create(outPath)
		out.Truncate(8 * mib) // a "card" bigger than the image
		if err := writeImage(src, fileTarget{out}, 8*mib, nopReporter{}); err != nil {
			t.Fatalf("%s: %v", src, err)
		}
		got, _ := os.ReadFile(outPath)
		if !bytes.Equal(got[:len(img)], img) {
			t.Errorf("%s: card differs from the image", filepath.Base(src))
		}
	}
}

func TestWriteImageRefusesSmallCard(t *testing.T) {
	dir := t.TempDir()
	raw := filepath.Join(dir, "etros.img")
	os.WriteFile(raw, testImage(t), 0o644)
	out, _ := os.Create(filepath.Join(dir, "card"))
	if err := writeImage(raw, fileTarget{out}, 4*mib, nopReporter{}); err == nil {
		t.Error("wrote a 6 MiB image to a 4 MiB card")
	}
}

func TestClassify(t *testing.T) {
	cases := map[string]string{
		"etros-2026-10-17.img.gz":                     "etros",
		"Etros.zip":                                   "etros-zip",
		"Etros Maker - Linux (11).zip":                "",
		"Etros Installer - Windows.zip":               "",
		"batocera-bcm2712-43-20261001.img.gz":         "pi5",
		"batocera-rk3326-r36s-43-20261001.img.gz":     "r36s",
		"boot-r36s-43.tar.xz":                         "r36s",
		"2026-10-01-raspios-trixie-arm64.img.xz":      "ollie",
		"2026-10-01-raspios-trixie-arm64-lite.img.xz": "",
		"ollie-os.img":                                "ollie",
		"batocera-bcm2712.img.gz.crdownload":          "",
		"holiday.jpg":                                 "",
	}
	for name, want := range cases {
		if got := classify(name); got != want {
			t.Errorf("classify(%q) = %q, want %q", name, got, want)
		}
	}
}
