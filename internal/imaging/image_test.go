package imaging

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// isoBuilder assembles just enough of an ISO 9660 image for the inspector to
// read: the descriptors, a boot catalogue and a root directory. Real images
// are megabytes of filesystem around exactly these structures.
type isoBuilder struct {
	label     string
	entries   []string
	platforms []byte // El Torito platform ids, nil for a non-bootable image
	hybrid    bool
	truncate  bool
}

func (b *isoBuilder) build(t *testing.T) string {
	t.Helper()
	const sectors = 24
	img := make([]byte, sectors*isoSectorSize)

	const (
		pvdSector     = 16
		bootSector    = 17
		endSector     = 18
		catalogSector = 19
		rootSector    = 20
	)

	// Primary volume descriptor.
	pvd := img[pvdSector*isoSectorSize:]
	pvd[0] = 1
	copy(pvd[1:], "CD001")
	pvd[6] = 1
	copy(pvd[40:72], pad(b.label, 32))
	binary.LittleEndian.PutUint32(pvd[80:], sectors)
	binary.LittleEndian.PutUint16(pvd[128:], isoSectorSize)

	// The root directory record lives inside the descriptor.
	root := pvd[156:190]
	root[0] = 34
	binary.LittleEndian.PutUint32(root[2:], rootSector)
	dir := b.directory()
	binary.LittleEndian.PutUint32(root[10:], uint32(len(dir)))
	root[25] = 0x02 // a directory
	copy(img[rootSector*isoSectorSize:], dir)

	if b.platforms != nil {
		boot := img[bootSector*isoSectorSize:]
		boot[0] = 0
		copy(boot[1:], "CD001")
		boot[6] = 1
		copy(boot[7:], "EL TORITO SPECIFICATION")
		binary.LittleEndian.PutUint32(boot[71:], catalogSector)

		cat := img[catalogSector*isoSectorSize:]
		// Validation entry, naming the platform of the default entry.
		cat[0] = 1
		cat[1] = b.platforms[0]
		cat[30], cat[31] = 0x55, 0xAA
		cat[32] = 0x88 // the default entry is bootable
		// Any further platforms arrive as sections.
		off := 64
		for i, p := range b.platforms[1:] {
			header := byte(0x90)
			if i == len(b.platforms)-2 {
				header = 0x91 // final section header
			}
			cat[off] = header
			cat[off+1] = p
			binary.LittleEndian.PutUint16(cat[off+2:], 1)
			cat[off+32] = 0x88
			off += 64
		}
	}

	end := img[endSector*isoSectorSize:]
	end[0] = 255
	copy(end[1:], "CD001")

	if b.hybrid {
		// An isohybrid image carries a partition table in its system area,
		// which is the whole of why it can be booted from a USB stick.
		entry := img[446:462]
		entry[0] = 0x80
		entry[4] = 0x17
		binary.LittleEndian.PutUint32(entry[8:], 0)
		binary.LittleEndian.PutUint32(entry[12:], sectors*4)
		img[510], img[511] = 0x55, 0xAA
	}

	if b.truncate {
		img = img[:len(img)-777]
	}

	path := filepath.Join(t.TempDir(), "test.iso")
	if err := os.WriteFile(path, img, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// directory renders the root directory: the two self-referential records every
// ISO has, then one per named entry.
func (b *isoBuilder) directory() []byte {
	var out []byte
	record := func(name string, flags byte) {
		length := 33 + len(name)
		if length%2 == 1 {
			length++
		}
		rec := make([]byte, length)
		rec[0] = byte(length)
		binary.LittleEndian.PutUint32(rec[2:], 21)
		binary.LittleEndian.PutUint32(rec[10:], 2048)
		rec[25] = flags
		rec[32] = byte(len(name))
		copy(rec[33:], name)
		out = append(out, rec...)
	}
	record("\x00", 0x02)
	record("\x01", 0x02)
	for _, e := range b.entries {
		record(e, 0)
	}
	return out
}

func pad(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

func TestInspectReadsAnISOsNameAndBootMethods(t *testing.T) {
	iso := (&isoBuilder{
		label:     "UBUNTU 24.04 AMD64",
		entries:   []string{"CASPER", "EFI", "BOOT"},
		platforms: []byte{0x00, 0xEF},
		hybrid:    true,
	}).build(t)

	img, err := Inspect(iso)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if img.Format != "ISO 9660 image" {
		t.Errorf("format = %q", img.Format)
	}
	if img.Label != "UBUNTU 24.04 AMD64" {
		t.Errorf("label = %q", img.Label)
	}
	if !img.HybridUSB {
		t.Error("a hybrid image was not recognised as one")
	}
	if got := strings.Join(img.BootMethods, "+"); got != "BIOS+UEFI" {
		t.Errorf("boot methods = %q, want BIOS+UEFI", got)
	}
	if !img.Bootable() {
		t.Error("image with a boot catalogue reported as not bootable")
	}
	if len(img.Warnings) != 0 {
		t.Errorf("a well-formed bootable hybrid image drew warnings: %v", img.Warnings)
	}
}

func TestInspectWarnsAboutAnISOThatCannotBootFromAStick(t *testing.T) {
	iso := (&isoBuilder{
		label:     "DEBIAN",
		platforms: []byte{0x00},
		hybrid:    false,
	}).build(t)

	img, err := Inspect(iso)
	if err != nil {
		t.Fatal(err)
	}
	if img.HybridUSB {
		t.Fatal("an image without a partition table was called hybrid")
	}
	if !warned(img, "no partition table") {
		t.Errorf("no warning about the missing partition table: %v", img.Warnings)
	}
}

func TestInspectRecognisesAWindowsInstaller(t *testing.T) {
	iso := (&isoBuilder{
		label:     "CCCOMA_X64FRE_EN-US_DV9",
		entries:   []string{"BOOTMGR", "SOURCES", "EFI", "SETUP.EXE"},
		platforms: []byte{0x00, 0xEF},
	}).build(t)

	img, err := Inspect(iso)
	if err != nil {
		t.Fatal(err)
	}
	if !img.WindowsInstaller {
		t.Fatal("a Windows installation layout was not recognised")
	}
	if !warned(img, "Windows installation image") {
		t.Errorf("the warning that matters most was not raised: %v", img.Warnings)
	}
	// The generic disc warning must not pile on top of the specific one.
	if len(img.Warnings) != 1 {
		t.Errorf("want one warning, got %d: %v", len(img.Warnings), img.Warnings)
	}
}

func TestInspectSpotsATruncatedDownload(t *testing.T) {
	iso := (&isoBuilder{label: "ARCH", platforms: []byte{0x00}, hybrid: true, truncate: true}).build(t)
	img, err := Inspect(iso)
	if err != nil {
		t.Fatal(err)
	}
	if !warned(img, "truncated") {
		t.Errorf("a short image drew no warning: %v", img.Warnings)
	}
}

func TestInspectSaysWhenAnISOHoldsNoBootCatalogue(t *testing.T) {
	iso := (&isoBuilder{label: "PHOTOS", entries: []string{"IMG_0001.JPG"}}).build(t)
	img, err := Inspect(iso)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bootable() {
		t.Error("a data ISO was reported bootable")
	}
	if !warned(img, "data rather than installation media") {
		t.Errorf("warnings = %v", img.Warnings)
	}
}

func TestInspectAcceptsAnImageItCannotIdentify(t *testing.T) {
	path := filepath.Join(t.TempDir(), "firmware.bin")
	if err := os.WriteFile(path, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	img, err := Inspect(path)
	if err != nil {
		t.Fatalf("an unknown image should still be described: %v", err)
	}
	if img.Format != "unrecognised image" {
		t.Errorf("format = %q", img.Format)
	}
	if img.Size != 4096 {
		t.Errorf("size = %d", img.Size)
	}
}

func TestInspectRejectsAnEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.iso")
	os.WriteFile(path, nil, 0o644)
	if _, err := Inspect(path); err == nil {
		t.Fatal("an empty file was accepted as an image")
	}
}

func TestADiskImageIsRecognisedByItsPartitionTable(t *testing.T) {
	drive := blankDrive(t, 2<<30)
	if _, err := FormatFAT32(drive, FormatOptions{Label: "Data"}); err != nil {
		t.Fatal(err)
	}
	img, err := Inspect(drive)
	if err != nil {
		t.Fatal(err)
	}
	if img.Format != "raw disk image" {
		t.Errorf("format = %q, want a raw disk image", img.Format)
	}
	if !img.HybridUSB {
		t.Error("a partitioned image was not seen to have a partition table")
	}
}

func warned(img *Image, phrase string) bool {
	for _, w := range img.Warnings {
		if strings.Contains(w, phrase) {
			return true
		}
	}
	return false
}
