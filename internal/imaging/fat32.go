package imaging

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Laying down a filesystem by hand needs a handful of constants from the FAT
// specification. They are named rather than written inline because every one
// of them appears in two places: the boot sector and its backup.
const (
	sectorSize       = 512
	reservedSectors  = 32 // boot sector, FSInfo, backups, room to grow
	numFATs          = 2  // one spare copy, as every FAT32 volume has
	rootCluster      = 2  // the first cluster is always the root directory
	fsInfoSector     = 1  //
	backupBootSector = 6  // where Windows looks when sector 0 is unreadable
	// A partition starting at 1 MiB is what every current tool does: it keeps
	// the filesystem aligned with the erase blocks of flash media, which is
	// worth several times the throughput of the old 63-sector convention.
	partitionStart = 2048
	// FAT32 is only valid above this many clusters; below it the driver is
	// entitled to read the volume as FAT16 and see nothing.
	minFAT32Clusters = 65525
)

// FormatOptions control a fresh FAT32 volume.
type FormatOptions struct {
	// Label names the volume. It is what a file manager shows when the drive
	// is plugged in. Empty leaves the volume unnamed.
	Label string

	// Bootable sets the active flag on the partition. It does not install a
	// boot loader; it only marks the partition the way firmware expects.
	Bootable bool
}

// FormatResult describes the volume that was created.
type FormatResult struct {
	Label            string `json:"label,omitempty"`
	ClusterBytes     int    `json:"cluster_bytes"`
	Clusters         uint32 `json:"clusters"`
	PartitionStart   int64  `json:"partition_start_sector"`
	PartitionSectors int64  `json:"partition_sectors"`
	Usable           int64  `json:"usable_bytes"`
}

// FormatFAT32 puts a single FAT32 partition on a device, filling it.
//
// FAT32 is the deliberate choice: it is the one filesystem that Linux, macOS,
// Windows and the firmware of a machine being installed can all read. The
// price is a four gigabyte limit on any single file, which matters for exactly
// one common case — the install.wim inside a modern Windows image.
func FormatFAT32(devicePath string, opt FormatOptions) (*FormatResult, error) {
	label, err := NormaliseLabel(opt.Label)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(devicePath, os.O_RDWR, 0)
	if err != nil {
		return nil, openError(devicePath, err)
	}
	defer f.Close()

	capacity, err := Capacity(f)
	if err != nil {
		return nil, err
	}
	totalSectors := capacity / sectorSize
	if totalSectors <= partitionStart {
		return nil, fmt.Errorf("%s is too small to hold a partition (%s)", devicePath, HumanBytes(capacity))
	}

	partSectors := totalSectors - partitionStart
	layout, err := planFAT32(partSectors)
	if err != nil {
		return nil, fmt.Errorf("%s (%s): %w", devicePath, HumanBytes(capacity), err)
	}
	layout.label = label
	layout.partStart = partitionStart

	if err := writeMBR(f, partitionStart, partSectors, opt.Bootable); err != nil {
		return nil, err
	}
	if err := layout.write(f); err != nil {
		return nil, err
	}
	if err := f.Sync(); err != nil {
		return nil, fmt.Errorf("flushing %s: %w", devicePath, err)
	}
	rereadPartitions(f)

	return &FormatResult{
		Label:            label,
		ClusterBytes:     layout.secPerClus * sectorSize,
		Clusters:         layout.clusters,
		PartitionStart:   partitionStart,
		PartitionSectors: partSectors,
		Usable:           int64(layout.clusters) * int64(layout.secPerClus) * sectorSize,
	}, nil
}

// NormaliseLabel turns a name the user typed into the eleven bytes a FAT
// volume label is, or explains why it cannot.
func NormaliseLabel(name string) (string, error) {
	s := strings.ToUpper(strings.TrimSpace(name))
	if s == "" {
		return "", nil
	}
	if len(s) > 11 {
		return "", fmt.Errorf("a FAT32 volume name is at most 11 characters; %q is %d", name, len(s))
	}
	for _, r := range s {
		if r < 0x20 || r > 0x7E {
			return "", fmt.Errorf("a FAT32 volume name cannot contain %q", r)
		}
		if strings.ContainsRune(`*?/\|,;:+=<>[]".`, r) {
			return "", fmt.Errorf("a FAT32 volume name cannot contain %q", r)
		}
	}
	return s, nil
}

// fat32Layout is the arithmetic of a volume, kept apart from the writing so it
// can be checked without a device.
type fat32Layout struct {
	totalSectors int64
	secPerClus   int
	fatSectors   int64
	clusters     uint32
	partStart    int64
	label        string
	volumeID     uint32
}

// planFAT32 sizes the clusters and the allocation tables for a partition.
func planFAT32(totalSectors int64) (*fat32Layout, error) {
	secPerClus := clusterSectors(totalSectors)
	l := &fat32Layout{totalSectors: totalSectors, secPerClus: secPerClus}

	// The table has to describe the data area, but the table itself takes
	// sectors out of that area. This is the resolution of that circularity
	// given in the FAT specification, rounded so the table is never short.
	for {
		dataSectors := totalSectors - reservedSectors
		per := (256*int64(l.secPerClus) + numFATs) / 2
		l.fatSectors = (dataSectors + per - 1) / per
		clusters := (totalSectors - reservedSectors - numFATs*l.fatSectors) / int64(l.secPerClus)
		if clusters >= minFAT32Clusters {
			l.clusters = uint32(clusters)
			break
		}
		// Too few clusters for a valid FAT32 volume: halve them and try again.
		if l.secPerClus <= 1 {
			return nil, fmt.Errorf("too small for FAT32: it needs at least %s",
				HumanBytes(minFAT32Clusters*sectorSize+reservedSectors*sectorSize))
		}
		l.secPerClus /= 2
	}
	// A cluster number is 28 bits, and the last few values are reserved.
	if l.clusters > 0x0FFFFFF5 {
		return nil, fmt.Errorf("too large for FAT32 at this cluster size")
	}
	return l, nil
}

// clusterSectors picks a cluster size from the volume size, following the
// table Windows itself uses so that a drive formatted here is indistinguishable
// from one formatted there.
func clusterSectors(totalSectors int64) int {
	switch {
	case totalSectors <= 532480: // up to 260 MiB
		return 1
	case totalSectors <= 16777216: // up to 8 GiB
		return 8
	case totalSectors <= 33554432: // up to 16 GiB
		return 16
	case totalSectors <= 67108864: // up to 32 GiB
		return 32
	default:
		return 64
	}
}

// write lays the volume down inside the partition.
func (l *fat32Layout) write(f *os.File) error {
	if l.volumeID == 0 {
		l.volumeID = randomUint32()
	}
	base := l.partStart * sectorSize

	boot := l.bootSector()
	info := l.fsInfoSector()

	// The boot sector and FSInfo are written twice: a FAT32 volume keeps a
	// backup set at sector 6 precisely so a damaged first sector is
	// recoverable rather than fatal.
	writes := []struct {
		sector int64
		data   []byte
	}{
		{0, boot},
		{fsInfoSector, info},
		{backupBootSector, boot},
		{backupBootSector + fsInfoSector, info},
	}
	for _, w := range writes {
		if _, err := f.WriteAt(w.data, base+w.sector*sectorSize); err != nil {
			return fmt.Errorf("writing boot sector: %w", err)
		}
	}

	if err := l.writeFATs(f, base); err != nil {
		return err
	}
	return l.writeRootDirectory(f, base)
}

func (l *fat32Layout) bootSector() []byte {
	b := make([]byte, sectorSize)

	// A jump instruction and an OEM name: no firmware reads them here, but a
	// volume without them is rejected by some Windows versions outright.
	copy(b[0:], []byte{0xEB, 0x58, 0x90})
	copy(b[3:], "MSWIN4.1")

	binary.LittleEndian.PutUint16(b[11:], sectorSize)
	b[13] = byte(l.secPerClus)
	binary.LittleEndian.PutUint16(b[14:], reservedSectors)
	b[16] = numFATs
	binary.LittleEndian.PutUint16(b[17:], 0) // no fixed root directory on FAT32
	binary.LittleEndian.PutUint16(b[19:], 0) // the 16-bit sector count is unused
	b[21] = 0xF8                             // fixed disk
	binary.LittleEndian.PutUint16(b[22:], 0) // as is the 16-bit FAT size
	binary.LittleEndian.PutUint16(b[24:], 63)
	binary.LittleEndian.PutUint16(b[26:], 255)
	binary.LittleEndian.PutUint32(b[28:], uint32(l.partStart))
	binary.LittleEndian.PutUint32(b[32:], uint32(l.totalSectors))
	binary.LittleEndian.PutUint32(b[36:], uint32(l.fatSectors))
	binary.LittleEndian.PutUint16(b[40:], 0) // both tables are live mirrors
	binary.LittleEndian.PutUint16(b[42:], 0) // filesystem version 0.0
	binary.LittleEndian.PutUint32(b[44:], rootCluster)
	binary.LittleEndian.PutUint16(b[48:], fsInfoSector)
	binary.LittleEndian.PutUint16(b[50:], backupBootSector)

	b[64] = 0x80 // BIOS drive number
	b[66] = 0x29 // an extended boot signature follows
	binary.LittleEndian.PutUint32(b[67:], l.volumeID)
	copy(b[71:82], labelBytes(l.label))
	copy(b[82:90], "FAT32   ")

	b[510], b[511] = 0x55, 0xAA
	return b
}

func (l *fat32Layout) fsInfoSector() []byte {
	b := make([]byte, sectorSize)
	binary.LittleEndian.PutUint32(b[0:], 0x41615252)   // "RRaA"
	binary.LittleEndian.PutUint32(b[484:], 0x61417272) // "rrAa"
	// One cluster is already spent on the root directory.
	binary.LittleEndian.PutUint32(b[488:], l.clusters-1)
	binary.LittleEndian.PutUint32(b[492:], rootCluster+1)
	binary.LittleEndian.PutUint32(b[508:], 0xAA550000)
	return b
}

// writeFATs writes both allocation tables. Only their first sector holds
// anything on a fresh volume, but the rest must be zero rather than whatever
// the drive held before, or the driver will follow the old chains.
func (l *fat32Layout) writeFATs(f *os.File, base int64) error {
	first := make([]byte, sectorSize)
	// Entry 0 repeats the media byte, entry 1 is the end-of-chain marker, and
	// entry 2 closes the root directory's one-cluster chain.
	binary.LittleEndian.PutUint32(first[0:], 0x0FFFFFF8)
	binary.LittleEndian.PutUint32(first[4:], 0x0FFFFFFF)
	binary.LittleEndian.PutUint32(first[8:], 0x0FFFFFFF)

	zeros := make([]byte, 1<<20)
	for n := int64(0); n < numFATs; n++ {
		start := base + (reservedSectors+n*l.fatSectors)*sectorSize
		if _, err := f.WriteAt(first, start); err != nil {
			return fmt.Errorf("writing allocation table: %w", err)
		}
		if err := zeroFill(f, start+sectorSize, (l.fatSectors-1)*sectorSize, zeros); err != nil {
			return fmt.Errorf("clearing allocation table: %w", err)
		}
	}
	return nil
}

// writeRootDirectory clears the root directory's cluster and, if the volume is
// named, puts the label in it. The name lives in two places on a FAT32 volume
// and tools disagree about which to believe, so both are written.
func (l *fat32Layout) writeRootDirectory(f *os.File, base int64) error {
	dataStart := base + (reservedSectors+numFATs*l.fatSectors)*sectorSize
	clusterBytes := int64(l.secPerClus) * sectorSize

	entry := make([]byte, clusterBytes)
	if l.label != "" {
		copy(entry[0:11], labelBytes(l.label))
		entry[11] = 0x08 // volume label attribute
		date, time := fatTimestamp(time.Now())
		binary.LittleEndian.PutUint16(entry[22:], time)
		binary.LittleEndian.PutUint16(entry[24:], date)
	}
	if _, err := f.WriteAt(entry, dataStart); err != nil {
		return fmt.Errorf("writing root directory: %w", err)
	}
	return nil
}

// zeroFill writes length zero bytes at offset using a reusable buffer.
func zeroFill(f *os.File, offset, length int64, zeros []byte) error {
	for written := int64(0); written < length; {
		n := int64(len(zeros))
		if left := length - written; left < n {
			n = left
		}
		if _, err := f.WriteAt(zeros[:n], offset+written); err != nil {
			return err
		}
		written += n
	}
	return nil
}

// writeMBR puts a single-partition table on the front of the device.
func writeMBR(w io.WriterAt, start, sectors int64, bootable bool) error {
	mbr := make([]byte, sectorSize)

	// A disk signature: Windows uses it to tell drives apart, and a drive
	// without one gets assigned a random value the first time it is seen.
	binary.LittleEndian.PutUint32(mbr[440:], randomUint32())

	e := mbr[446:462]
	if bootable {
		e[0] = 0x80
	}
	copy(e[1:4], chs(start))
	// 0x0C is FAT32 addressed by LBA, which is what anything made this century
	// expects; the older 0x0B says the same thing through cylinder geometry.
	e[4] = 0x0C
	copy(e[5:8], chs(start+sectors-1))
	binary.LittleEndian.PutUint32(e[8:], uint32(start))
	binary.LittleEndian.PutUint32(e[12:], uint32(sectors))

	mbr[510], mbr[511] = 0x55, 0xAA
	_, err := w.WriteAt(mbr, 0)
	if err != nil {
		return fmt.Errorf("writing partition table: %w", err)
	}
	return nil
}

// chs converts a sector number to the cylinder/head/sector triple the table
// still carries. Anything past what the encoding can hold is pinned to its
// maximum, which is the convention every operating system understands as
// "read the LBA fields instead".
func chs(lba int64) []byte {
	const heads, sectorsPerTrack = 255, 63
	c := lba / (heads * sectorsPerTrack)
	h := (lba / sectorsPerTrack) % heads
	s := lba%sectorsPerTrack + 1
	if c > 1023 {
		return []byte{0xFE, 0xFF, 0xFF}
	}
	return []byte{byte(h), byte(s) | byte((c>>2)&0xC0), byte(c)}
}

// labelBytes pads a volume name to the eleven bytes the format stores.
func labelBytes(label string) []byte {
	b := []byte("           ")
	copy(b, label)
	return b
}

// fatTimestamp packs a time into the date and time words FAT uses.
func fatTimestamp(t time.Time) (date, clock uint16) {
	year := t.Year() - 1980
	if year < 0 {
		year = 0
	}
	date = uint16(year<<9) | uint16(int(t.Month())<<5) | uint16(t.Day())
	clock = uint16(t.Hour()<<11) | uint16(t.Minute()<<5) | uint16(t.Second()/2)
	return date, clock
}

func randomUint32() uint32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		// A predictable identifier is a cosmetic problem, not a reason to
		// refuse to format a drive.
		return uint32(time.Now().UnixNano())
	}
	v := binary.LittleEndian.Uint32(b[:])
	if v == 0 {
		v = 1
	}
	return v
}
