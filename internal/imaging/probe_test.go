package imaging

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

// superblock builds a sparse volume header: pairs of offset and content, in a
// buffer big enough for every signature the prober looks at.
func superblock(parts ...any) *bytes.Reader {
	buf := make([]byte, 0x11000)
	for i := 0; i+1 < len(parts); i += 2 {
		off := parts[i].(int)
		switch v := parts[i+1].(type) {
		case string:
			copy(buf[off:], v)
		case []byte:
			copy(buf[off:], v)
		}
	}
	return bytes.NewReader(buf)
}

func TestProbeNamesWhatIsOnADriveAlready(t *testing.T) {
	cases := []struct {
		name  string
		block *bytes.Reader
		want  Volume
	}{
		{
			name:  "NTFS",
			block: superblock(3, "NTFS    "),
			want:  Volume{Type: "NTFS"},
		},
		{
			name:  "FAT32 with a label",
			block: superblock(82, "FAT32   ", 71, "MY STICK   "),
			want:  Volume{Type: "FAT32", Label: "MY STICK"},
		},
		{
			name:  "FAT16",
			block: superblock(54, "FAT16   ", 43, "DOS        "),
			want:  Volume{Type: "FAT16", Label: "DOS"},
		},
		{
			name:  "ext4",
			block: superblock(0x438, []byte{0x53, 0xEF}, 0x478, "root\x00"),
			want:  Volume{Type: "ext2/3/4", Label: "root"},
		},
		{
			name:  "btrfs",
			block: superblock(0x10040, "_BHRfS_M", 0x1012B, "pool\x00"),
			want:  Volume{Type: "Btrfs", Label: "pool"},
		},
		{
			name:  "XFS",
			block: superblock(0, "XFSB", 0x6C, "data\x00"),
			want:  Volume{Type: "XFS", Label: "data"},
		},
		{
			name:  "an encrypted volume",
			block: superblock(0, "LUKS\xba\xbe"),
			want:  Volume{Type: "LUKS encrypted"},
		},
		{
			name:  "swap",
			block: superblock(4086, "SWAPSPACE2"),
			want:  Volume{Type: "swap"},
		},
		{
			name:  "nothing at all",
			block: superblock(),
			want:  Volume{},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := probeReader(c.block)
			if got != c.want {
				t.Errorf("probed as %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestProbeReadsAnExFATLabelFromItsDirectory(t *testing.T) {
	buf := make([]byte, 0x11000)
	copy(buf[3:], "EXFAT   ")
	binary.LittleEndian.PutUint32(buf[88:], 8) // cluster heap starts at sector 8
	binary.LittleEndian.PutUint32(buf[96:], 2) // root directory is cluster 2
	buf[108] = 9                               // 512-byte sectors
	buf[109] = 3                               // 8-sector clusters

	entry := buf[8*512:]
	entry[0] = 0x83
	entry[1] = 5
	for i, r := range "PHOTO" {
		binary.LittleEndian.PutUint16(entry[2+i*2:], uint16(r))
	}

	got := probeReader(bytes.NewReader(buf))
	if got.Type != "exFAT" || got.Label != "PHOTO" {
		t.Errorf("probed as %+v, want exFAT labelled PHOTO", got)
	}
}

func TestProbeSeesAnImageThatWasWrittenToADrive(t *testing.T) {
	buf := make([]byte, 0x12000)
	copy(buf[32769:], "CD001")
	copy(buf[32768+40:], "FEDORA-WS-LIVE-40               ")

	got := probeReader(bytes.NewReader(buf))
	if got.Type != "ISO 9660" || got.Label != "FEDORA-WS-LIVE-40" {
		t.Errorf("probed as %+v", got)
	}
}

func TestProbeDescribesItselfForATable(t *testing.T) {
	if got := (Volume{Type: "NTFS", Label: "Backup"}).String(); got != `NTFS "Backup"` {
		t.Errorf("String() = %q", got)
	}
	if got := (Volume{Type: "NTFS"}).String(); got != "NTFS" {
		t.Errorf("String() = %q", got)
	}
	if got := (Volume{}).String(); got != "" {
		t.Errorf("String() = %q", got)
	}
}

func TestProbeOfAnUnreadableDriveIsNotAnError(t *testing.T) {
	if got := Probe("/nonexistent/sdz"); !got.Empty() {
		t.Errorf("probing a missing drive gave %+v", got)
	}
}

func TestDescribeTellsBlankFromUnrecognisedFromUnreadable(t *testing.T) {
	dir := t.TempDir()

	blank := filepath.Join(dir, "blank")
	if err := os.WriteFile(blank, make([]byte, 4096), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := Describe(blank); got != "empty" {
		t.Errorf("a zeroed drive described as %q", got)
	}

	junk := filepath.Join(dir, "junk")
	if err := os.WriteFile(junk, bytes.Repeat([]byte{0x17}, 4096), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := Describe(junk); got != "unrecognised data" {
		t.Errorf("a drive holding data described as %q", got)
	}

	if got := Describe(filepath.Join(dir, "missing")); got != "unreadable" {
		t.Errorf("a drive that cannot be opened described as %q", got)
	}

	formatted := blankDrive(t, 512<<20)
	if _, err := FormatFAT32(formatted, FormatOptions{Label: "DISK"}); err != nil {
		t.Fatal(err)
	}
	if got := Describe(formatted); got != "partitioned" {
		t.Errorf("a partitioned drive described as %q", got)
	}
}
