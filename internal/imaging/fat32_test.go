package imaging

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// blankDrive makes a sparse file standing in for a drive of the given size.
// Sparse means a 32 GiB "drive" costs nothing until something is written to it.
func blankDrive(t *testing.T, size int64) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "drive.img")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := f.Truncate(size); err != nil {
		t.Fatal(err)
	}
	return path
}

func readAt(t *testing.T, path string, off int64, n int) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	buf := make([]byte, n)
	if _, err := f.ReadAt(buf, off); err != nil && err != io.EOF {
		t.Fatal(err)
	}
	return buf
}

func TestFormatWritesAPartitionTableAndAFAT32Volume(t *testing.T) {
	drive := blankDrive(t, 2<<30) // 2 GiB

	res, err := FormatFAT32(drive, FormatOptions{Label: "Rescue", Bootable: true})
	if err != nil {
		t.Fatalf("FormatFAT32: %v", err)
	}

	mbr := readAt(t, drive, 0, 512)
	if mbr[510] != 0x55 || mbr[511] != 0xAA {
		t.Errorf("partition table has no boot signature")
	}
	entry := mbr[446:462]
	if entry[0] != 0x80 {
		t.Errorf("partition is not marked active: %#x", entry[0])
	}
	if entry[4] != 0x0C {
		t.Errorf("partition type = %#x, want 0x0C (FAT32 LBA)", entry[4])
	}
	if got := binary.LittleEndian.Uint32(entry[8:]); got != partitionStart {
		t.Errorf("partition starts at sector %d, want %d", got, partitionStart)
	}
	sectors := int64(binary.LittleEndian.Uint32(entry[12:]))
	if want := int64(2<<30/sectorSize - partitionStart); sectors != want {
		t.Errorf("partition covers %d sectors, want %d", sectors, want)
	}
	if binary.LittleEndian.Uint32(mbr[440:]) == 0 {
		t.Errorf("no disk signature was written")
	}

	// The three copies of the boot sector must agree, since the point of the
	// backup is that it can be used when the first one is unreadable.
	base := int64(partitionStart) * sectorSize
	boot := readAt(t, drive, base, 512)
	backup := readAt(t, drive, base+backupBootSector*sectorSize, 512)
	if string(boot) != string(backup) {
		t.Errorf("the backup boot sector differs from the original")
	}

	if got := binary.LittleEndian.Uint16(boot[11:]); got != sectorSize {
		t.Errorf("bytes per sector = %d", got)
	}
	if got := int(boot[13]); got*sectorSize != res.ClusterBytes {
		t.Errorf("boot sector says %d sectors per cluster, result says %d bytes", got, res.ClusterBytes)
	}
	if got := binary.LittleEndian.Uint32(boot[28:]); got != partitionStart {
		t.Errorf("hidden sectors = %d, want the partition start %d", got, partitionStart)
	}
	if got := int64(binary.LittleEndian.Uint32(boot[32:])); got != sectors {
		t.Errorf("volume claims %d sectors, partition has %d", got, sectors)
	}
	if got := binary.LittleEndian.Uint32(boot[44:]); got != rootCluster {
		t.Errorf("root cluster = %d", got)
	}
	if string(boot[82:90]) != "FAT32   " {
		t.Errorf("filesystem type = %q", boot[82:90])
	}
	if string(boot[71:82]) != "RESCUE     " {
		t.Errorf("boot sector label = %q, want the name uppercased and padded", boot[71:82])
	}

	// The allocation table must reserve its first three entries, or a driver
	// will hand cluster 2 out twice and overwrite the root directory.
	fatStart := base + reservedSectors*sectorSize
	fat := readAt(t, drive, fatStart, 12)
	for i, want := range []uint32{0x0FFFFFF8, 0x0FFFFFFF, 0x0FFFFFFF} {
		if got := binary.LittleEndian.Uint32(fat[i*4:]); got != want {
			t.Errorf("FAT entry %d = %#x, want %#x", i, got, want)
		}
	}

	// Both tables are written, and the second is a full table away.
	fatSize := int64(binary.LittleEndian.Uint32(boot[36:])) * sectorSize
	second := readAt(t, drive, fatStart+fatSize, 12)
	if string(fat) != string(second) {
		t.Errorf("the second allocation table does not match the first")
	}

	// The volume label is also a directory entry, which is where Linux and
	// macOS look for it.
	dataStart := fatStart + 2*fatSize
	root := readAt(t, drive, dataStart, 32)
	if string(root[0:11]) != "RESCUE     " {
		t.Errorf("root directory label = %q", root[0:11])
	}
	if root[11] != 0x08 {
		t.Errorf("label entry attribute = %#x, want 0x08", root[11])
	}

	if res.Clusters < minFAT32Clusters {
		t.Errorf("volume has %d clusters, below the FAT32 minimum of %d", res.Clusters, minFAT32Clusters)
	}
	if res.Usable > sectors*sectorSize {
		t.Errorf("usable space %d exceeds the partition", res.Usable)
	}
}

func TestFormattedDriveIsRecognisedAsFAT32(t *testing.T) {
	drive := blankDrive(t, 4<<30)
	if _, err := FormatFAT32(drive, FormatOptions{Label: "Install"}); err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(drive)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// The whole drive reads as partitioned...
	if got := probeReader(f); got.Type != "partitioned" {
		t.Errorf("whole drive probed as %q, want \"partitioned\"", got.Type)
	}
	// ...and the partition itself as the volume that was just made.
	part := io.NewSectionReader(f, partitionStart*sectorSize, 1<<30)
	got := probeReader(part)
	if got.Type != "FAT32" || got.Label != "INSTALL" {
		t.Errorf("partition probed as %+v, want FAT32 labelled INSTALL", got)
	}
}

func TestClusterSizesFollowTheVolumeSize(t *testing.T) {
	cases := []struct {
		name    string
		bytes   int64
		sectors int
	}{
		{"256 MiB", 256 << 20, 1},
		{"2 GiB", 2 << 30, 8},
		{"12 GiB", 12 << 30, 16},
		{"24 GiB", 24 << 30, 32},
		{"64 GiB", 64 << 30, 64},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			l, err := planFAT32(c.bytes / sectorSize)
			if err != nil {
				t.Fatal(err)
			}
			if l.secPerClus != c.sectors {
				t.Errorf("cluster = %d sectors, want %d", l.secPerClus, c.sectors)
			}
			if l.clusters < minFAT32Clusters {
				t.Errorf("only %d clusters, below the FAT32 minimum", l.clusters)
			}
			// The tables and the data they describe must fit in the volume.
			used := reservedSectors + numFATs*l.fatSectors + int64(l.clusters)*int64(l.secPerClus)
			if used > l.totalSectors {
				t.Errorf("layout needs %d sectors of %d available", used, l.totalSectors)
			}
			// Every cluster must be addressable by the table that was sized.
			capacity := l.fatSectors * sectorSize / 4
			if int64(l.clusters)+2 > capacity {
				t.Errorf("%d clusters do not fit in a table holding %d entries", l.clusters, capacity)
			}
		})
	}
}

func TestFormatRefusesADriveTooSmallForFAT32(t *testing.T) {
	drive := blankDrive(t, 16<<20) // 16 MiB: far below 65525 clusters
	_, err := FormatFAT32(drive, FormatOptions{})
	if err == nil {
		t.Fatal("expected a refusal, formatted anyway")
	}
	if !strings.Contains(err.Error(), "too small") {
		t.Errorf("unhelpful error: %v", err)
	}
}

func TestVolumeNamesAreCheckedBeforeTheDriveIsTouched(t *testing.T) {
	for _, c := range []struct {
		in      string
		want    string
		invalid bool
	}{
		{in: "backup", want: "BACKUP"},
		{in: "  win11  ", want: "WIN11"},
		{in: "", want: ""},
		{in: "TWELVE CHARS", invalid: true},
		{in: "bad/name", invalid: true},
		{in: "dotted.name", invalid: true},
		{in: "café", invalid: true},
	} {
		got, err := NormaliseLabel(c.in)
		if c.invalid {
			if err == nil {
				t.Errorf("NormaliseLabel(%q) = %q, expected a refusal", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormaliseLabel(%q): %v", c.in, err)
		} else if got != c.want {
			t.Errorf("NormaliseLabel(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFormatLeavesNoTraceOfThePreviousFilesystem(t *testing.T) {
	drive := blankDrive(t, 2<<30)

	// An old volume, with stale allocation tables where the new ones go.
	f, err := os.OpenFile(drive, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	junk := make([]byte, 8<<20)
	for i := range junk {
		junk[i] = 0xA5
	}
	if _, err := f.WriteAt(junk, partitionStart*sectorSize); err != nil {
		t.Fatal(err)
	}
	f.Close()

	if _, err := FormatFAT32(drive, FormatOptions{Label: "Fresh"}); err != nil {
		t.Fatal(err)
	}

	boot := readAt(t, drive, partitionStart*sectorSize, 512)
	fatSize := int64(binary.LittleEndian.Uint32(boot[36:])) * sectorSize
	fatStart := int64(partitionStart)*sectorSize + reservedSectors*sectorSize

	// Past the three reserved entries, both tables must be empty: a single
	// stale entry would point a driver at the previous volume's data.
	for table := int64(0); table < numFATs; table++ {
		rest := readAt(t, drive, fatStart+table*fatSize+16, int(fatSize-16))
		for i, b := range rest {
			if b != 0 {
				t.Fatalf("allocation table %d still holds %#x at offset %d", table, b, i+16)
			}
		}
	}
}
