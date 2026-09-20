package imaging

import (
	"bytes"
	"encoding/binary"
	"os"
	"strings"
	"unicode/utf16"
)

// Volume is what is on a drive or partition now, read straight from its
// superblock. Knowing this is what makes a confirmation prompt worth reading:
// "erase /dev/sdb" means little, "erase /dev/sdb — NTFS, \"Backup 2024\""
// means quite a lot.
type Volume struct {
	Type  string `json:"type,omitempty"`
	Label string `json:"label,omitempty"`
}

// Empty reports whether nothing recognisable was found.
func (v Volume) Empty() bool { return v.Type == "" }

// String renders the volume for a table cell.
func (v Volume) String() string {
	switch {
	case v.Type == "":
		return ""
	case v.Label == "":
		return v.Type
	}
	return v.Type + " " + `"` + v.Label + `"`
}

// Probe identifies the filesystem on a device or partition. A drive that
// cannot be read, or holds nothing recognisable, gives an empty Volume rather
// than an error: not knowing is a normal outcome here, and never a reason to
// stop what the caller was doing.
func Probe(path string) Volume {
	f, err := os.Open(path)
	if err != nil {
		return Volume{}
	}
	defer f.Close()
	return probeReader(f)
}

func probeReader(r readerAt) Volume {
	// One read covers every signature except the ISO descriptor, which lives
	// past the 64 KiB mark.
	head := make([]byte, 0x11000)
	n, _ := r.ReadAt(head, 0)
	head = head[:n]

	at := func(off int, s string) bool {
		return off+len(s) <= len(head) && string(head[off:off+len(s)]) == s
	}
	text := func(off, length int) string {
		if off+length > len(head) {
			return ""
		}
		return strings.TrimRight(strings.TrimRight(string(head[off:off+length]), "\x00"), " ")
	}

	switch {
	case at(3, "EXFAT   "):
		return Volume{Type: "exFAT", Label: exfatLabel(head)}
	case at(3, "NTFS    "):
		return Volume{Type: "NTFS"}
	case at(82, "FAT32   "):
		return Volume{Type: "FAT32", Label: text(71, 11)}
	case at(54, "FAT16   "), at(54, "FAT12   "), at(54, "FAT     "):
		return Volume{Type: strings.TrimSpace(text(54, 8)), Label: text(43, 11)}
	case at(0, "XFSB"):
		return Volume{Type: "XFS", Label: text(0x6C, 12)}
	case at(0, "hsqs"):
		return Volume{Type: "SquashFS"}
	case len(head) > 0x404 && string(head[0x400:0x404]) == "\xe2\xe1\xf5\xe0":
		return Volume{Type: "EROFS"}
	case len(head) > 0x10046 && string(head[0x10040:0x10048]) == "_BHRfS_M":
		return Volume{Type: "Btrfs", Label: text(0x1012B, 256)}
	case len(head) > 0x439 && head[0x438] == 0x53 && head[0x439] == 0xEF:
		return Volume{Type: "ext2/3/4", Label: text(0x478, 16)}
	case at(0, "LUKS\xba\xbe"):
		return Volume{Type: "LUKS encrypted"}
	case at(4086, "SWAPSPACE2"), at(4086, "SWAP-SPACE"):
		return Volume{Type: "swap", Label: text(0x41C, 16)}
	}

	// An image written to a drive leaves the ISO descriptor where it was.
	sig := make([]byte, 5)
	if _, err := r.ReadAt(sig, 32769); err == nil && string(sig) == "CD001" {
		label := make([]byte, 32)
		if _, err := r.ReadAt(label, 32768+40); err == nil {
			return Volume{Type: "ISO 9660", Label: strings.TrimSpace(string(label))}
		}
		return Volume{Type: "ISO 9660"}
	}

	// A partition table with nothing identifiable in front of it still tells
	// the user the drive is not blank.
	if len(head) >= 512 && head[510] == 0x55 && head[511] == 0xAA {
		if !bytes.Equal(head[446:510], make([]byte, 64)) {
			return Volume{Type: "partitioned"}
		}
	}
	return Volume{}
}

// exfatLabel reads the label from the exFAT root directory, which unlike every
// other format keeps it in a directory entry rather than the superblock.
func exfatLabel(head []byte) string {
	if len(head) < 512 {
		return ""
	}
	clusterHeap := int64(binary.LittleEndian.Uint32(head[88:92]))
	rootCluster := int64(binary.LittleEndian.Uint32(head[96:100]))
	sectorShift := int64(head[108])
	clusterShift := int64(head[109])
	if sectorShift > 12 || clusterShift > 25 || rootCluster < 2 {
		return ""
	}
	offset := (clusterHeap + (rootCluster-2)<<clusterShift) << sectorShift
	if offset < 0 || offset+32 > int64(len(head)) {
		return ""
	}
	entry := head[offset : offset+32]
	if entry[0] != 0x83 { // volume label entry
		return ""
	}
	length := int(entry[1])
	if length > 11 {
		length = 11
	}
	units := make([]uint16, length)
	for i := 0; i < length; i++ {
		units[i] = binary.LittleEndian.Uint16(entry[2+i*2:])
	}
	return string(utf16.Decode(units))
}

// Describe says what is on a drive in a few words, for a list somebody is
// about to pick from.
//
// It distinguishes three things a bare "empty" would run together: a drive
// that reads back as zeros, a drive holding something this program does not
// recognise, and a drive that cannot be read at all. Calling the second or
// third of those empty is how the wrong drive gets erased.
func Describe(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "unreadable"
	}
	defer f.Close()

	if v := probeReader(f); !v.Empty() {
		return v.String()
	}

	buf := make([]byte, 1<<20)
	n, _ := f.ReadAt(buf, 0)
	if n == 0 {
		return "unreadable"
	}
	for _, b := range buf[:n] {
		if b != 0 {
			return "unrecognised data"
		}
	}
	return "empty"
}

type readerAt interface {
	ReadAt(p []byte, off int64) (int, error)
}
