// Package flash writes disk images, such as a Linux installer ISO, to a drive
// or a partition, and reads them back to prove the drive holds what was sent.
package flash

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Format is what kind of image a file is.
type Format int

const (
	// Raw is a file with no recognisable structure. It is written as it is.
	Raw Format = iota
	// ISO is an ISO 9660 image with no partition table: a CD or DVD image.
	// Written to a USB drive byte for byte, it will not boot.
	ISO
	// HybridISO is an ISO 9660 image that also carries a partition table, so
	// the same file boots from a DVD or a USB drive. Most Linux ISOs are.
	HybridISO
	// DiskImage is a picture of a whole drive, partition table included, such
	// as a Raspberry Pi image.
	DiskImage
)

// Image describes a file about to be written.
type Image struct {
	Path string
	Size int64
	// Format is what the file turned out to be when read.
	Format Format
	// Label is the volume name an ISO declares, such as "Fedora-WS-Live-40".
	Label string
	// PartitionTable is "MBR" or "GPT" when the image carries one.
	PartitionTable string
	// Compression names the compressor when the file is compressed, which
	// means it must be unpacked before it can be written.
	Compression string
}

// Name is the file's name without its directory.
func (i *Image) Name() string { return filepath.Base(i.Path) }

// Bootable reports whether writing the image byte for byte to a drive yields
// something a PC can start from.
func (i *Image) Bootable() bool { return i.Format == HybridISO || i.Format == DiskImage }

// Describe says what the image is in a few words.
func (i *Image) Describe() string {
	switch i.Format {
	case HybridISO:
		return "ISO image that can boot from a USB drive"
	case ISO:
		return "ISO image for a CD or DVD, with no partition table"
	case DiskImage:
		return "disk image (" + i.PartitionTable + " partition table)"
	}
	return "raw image"
}

// compressors are the signatures of the formats images are commonly shipped
// in. Writing one of these as it is would put a compressed blob on the drive.
var compressors = []struct {
	name   string
	magic  []byte
	unpack string
}{
	{"xz", []byte{0xFD, '7', 'z', 'X', 'Z', 0x00}, "xz -dk"},
	{"gzip", []byte{0x1F, 0x8B}, "gunzip -k"},
	{"bzip2", []byte("BZh"), "bunzip2 -k"},
	{"zstd", []byte{0x28, 0xB5, 0x2F, 0xFD}, "unzstd"},
	{"zip", []byte("PK\x03\x04"), "unzip"},
	{"7-Zip", []byte{'7', 'z', 0xBC, 0xAF, 0x27, 0x1C}, "7z x"},
}

// UnpackCommand returns the command that decompresses the image, for advice.
func (i *Image) UnpackCommand() string {
	for _, c := range compressors {
		if c.name == i.Compression {
			return c.unpack + " " + shellQuote(i.Path)
		}
	}
	return ""
}

func shellQuote(s string) string {
	if !strings.ContainsAny(s, " '\"\\$`!*?&;|<>()[]{}#~") {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Inspect reads enough of a file to say what it is.
func Inspect(path string) (*Image, error) {
	st, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s does not exist", path)
		}
		return nil, err
	}
	if st.IsDir() {
		return nil, fmt.Errorf("%s is a folder, not an image file", path)
	}
	if !st.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not an ordinary file", path)
	}
	if st.Size() == 0 {
		return nil, fmt.Errorf("%s is empty; the download may not have finished", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// The ISO 9660 primary volume descriptor sits at 32 KiB; read past it.
	head := make([]byte, 0x8000+2048)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	head = head[:n]

	img := &Image{Path: path, Size: st.Size()}
	for _, c := range compressors {
		if bytes.HasPrefix(head, c.magic) {
			img.Compression = c.name
			return img, nil
		}
	}

	switch {
	case len(head) >= 520 && string(head[512:520]) == "EFI PART":
		img.PartitionTable = "GPT"
	case len(head) >= 4104 && string(head[4096:4104]) == "EFI PART":
		img.PartitionTable = "GPT" // an image made for 4K-sector drives
	case len(head) >= 512 && head[510] == 0x55 && head[511] == 0xAA && hasMBRPartition(head):
		img.PartitionTable = "MBR"
	}

	iso := len(head) >= 0x8006 && string(head[0x8001:0x8006]) == "CD001"
	switch {
	case iso && img.PartitionTable != "":
		img.Format = HybridISO
	case iso:
		img.Format = ISO
	case img.PartitionTable != "":
		img.Format = DiskImage
	}
	if iso && len(head) >= 0x8000+72 {
		img.Label = strings.TrimSpace(strings.TrimRight(string(head[0x8028:0x8048]), "\x00"))
	}
	return img, nil
}

// hasMBRPartition reports whether an MBR's partition table has an entry. The
// 0x55AA signature alone also appears on a bare FAT filesystem's boot sector.
func hasMBRPartition(sector []byte) bool {
	for i := 0; i < 4; i++ {
		entry := sector[446+16*i : 446+16*(i+1)]
		if entry[4] != 0 { // partition type
			return true
		}
	}
	return false
}
