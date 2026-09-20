package device

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"os/exec"
	"strings"
)

// Everything in here is write protection written on the drive rather than held
// by the kernel — a flag in a partition table or a filesystem header that says
// "do not write to me", which every operating system then obeys. They are the
// causes people never find, because nothing they can run names them.

const (
	// gptReadOnly is bit 60 of a GPT entry's attributes. Windows sets it from
	// `diskpart` → `attributes disk set readonly`, and Linux honours it, so a
	// drive can come back from a Windows machine permanently read-only with
	// nothing on this side to show why.
	gptReadOnly = 1 << 60

	// FAT marks itself dirty when it was not unmounted cleanly, and drivers
	// mount a dirty volume read-only rather than risk it.
	fat32DirtyOffset = 0x41
	fat16DirtyOffset = 0x25
	fatCleanShutBit  = 0x08000000 // in FAT[1]; zero means dirty
	fatHardErrorBit  = 0x04000000

	// exFAT keeps the same idea in two bits of its boot sector.
	exfatFlagsOffset = 106
	exfatDirtyBit    = 1 << 1
	exfatFailureBit  = 1 << 2
)

// OnDiskProtection looks for write protection written on the drive itself.
// It reads and never writes.
func (s *Scanner) OnDiskProtection(d *Device) []Protection {
	var out []Protection

	if p, ok := gptReadOnlyPartitions(d.Path); ok {
		out = append(out, p)
	}
	// A filesystem's own flags live on the partition, or on the whole device
	// when there is no partition table.
	targets := []string{d.Path}
	for _, part := range d.Partitions {
		targets = append(targets, part.Path)
	}
	for _, target := range targets {
		if p, ok := filesystemProtection(target); ok {
			out = append(out, p)
		}
	}
	return out
}

// gptReadOnlyPartitions reports whether any GPT entry carries the read-only
// attribute.
func gptReadOnlyPartitions(path string) (Protection, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Protection{}, false
	}
	defer f.Close()

	entries, _, err := readGPT(f)
	if err != nil {
		return Protection{}, false
	}
	var locked []string
	for i, e := range entries {
		if e.attributes&gptReadOnly != 0 {
			locked = append(locked, fmt.Sprintf("partition %d", i+1))
		}
	}
	if len(locked) == 0 {
		return Protection{}, false
	}
	return Protection{
		Kind:    GPTAttribute,
		Summary: "the partition table marks " + strings.Join(locked, " and ") + " read-only",
		Detail: "A GPT entry carries a read-only attribute, and every operating system obeys it. " +
			"Windows sets it from diskpart's `attributes disk set readonly`, which is why a drive " +
			"can come back from a Windows machine locked with nothing on this side to explain it. " +
			"Clearing the attribute changes no data.",
		Clearable: true,
		Where:     path,
	}, true
}

// filesystemProtection reports a filesystem that has marked itself unsafe to
// write to.
func filesystemProtection(path string) (Protection, bool) {
	f, err := os.Open(path)
	if err != nil {
		return Protection{}, false
	}
	defer f.Close()

	head := make([]byte, 2048)
	if n, _ := f.ReadAt(head, 0); n < 512 {
		return Protection{}, false
	}

	switch {
	case string(head[3:11]) == "EXFAT   ":
		flags := binary.LittleEndian.Uint16(head[exfatFlagsOffset:])
		if flags&(exfatDirtyBit|exfatFailureBit) == 0 {
			return Protection{}, false
		}
		detail := "The volume was not unmounted cleanly, so it is marked dirty and drivers mount " +
			"it read-only rather than risk writing to it."
		if flags&exfatFailureBit != 0 {
			detail += " It is also marked as having had media errors, which is the drive reporting " +
				"trouble rather than the filesystem."
		}
		return Protection{Kind: DirtyFilesystem, Summary: "the exFAT volume on " + path + " is marked dirty",
			Detail: detail + " Clearing the mark tells the system the volume is clean; it does not " +
				"check that it is. Run a filesystem check first if what is on it matters.",
			Clearable: true, Where: path}, true

	case string(head[3:11]) == "NTFS    ":
		// NTFS keeps its dirty flag in metadata rather than the boot sector,
		// so this is reported rather than cleared.
		return Protection{
			Kind:    DirtyFilesystem,
			Summary: "an NTFS volume on " + path + ", which Linux mounts read-only when Windows left it dirty",
			Detail: "Windows leaves NTFS dirty when it hibernates, and Fast Startup hibernates every " +
				"time it shuts down. Linux then refuses to write to it. Oliver cannot clear that flag: " +
				"it lives in the filesystem's metadata, not its header. `sudo ntfsfix -d " + path + "` " +
				"clears it if ntfs-3g is installed; the lasting fix is to turn Fast Startup off in " +
				"Windows and shut down properly.",
			Clearable: false, Where: path}, true

	case string(head[82:90]) == "FAT32   ":
		if !fatDirty(head, fat32DirtyOffset) {
			return Protection{}, false
		}
		return fatProtection(path, "FAT32"), true

	case string(head[54:59]) == "FAT16", string(head[54:59]) == "FAT12":
		if !fatDirty(head, fat16DirtyOffset) {
			return Protection{}, false
		}
		return fatProtection(path, strings.TrimSpace(string(head[54:62]))), true

	case len(head) > 1082 && binary.LittleEndian.Uint16(head[1080:]) == 0xEF53:
		// ext2/3/4 keeps its state in the superblock: 1 is clean, anything
		// else means the kernel found errors and stopped writing.
		if state := binary.LittleEndian.Uint16(head[1082:]); state == 1 {
			return Protection{}, false
		}
		return Protection{
			Kind:    FilesystemErrors,
			Summary: "the ext filesystem on " + path + " is marked as having errors",
			Detail: "The kernel remounts an ext filesystem read-only the moment it finds a problem, " +
				"and records it so the next mount does the same. That is the filesystem protecting " +
				"itself, and clearing it without fixing anything would only hide it. " +
				"`sudo fsck -y " + path + "` is what puts it right.",
			Clearable: false, Where: path}, true
	}
	return Protection{}, false
}

func fatProtection(path, kind string) Protection {
	return Protection{
		Kind:    DirtyFilesystem,
		Summary: fmt.Sprintf("the %s volume on %s is marked dirty", kind, path),
		Detail: "The volume was not unmounted cleanly — usually a stick pulled out while it was " +
			"still being written to — so drivers mount it read-only rather than risk it. Clearing " +
			"the mark tells the system the volume is clean; it does not check that it is. Run a " +
			"filesystem check first if what is on it matters.",
		Clearable: true,
		Where:     path,
	}
}

// fatDirty reports whether a FAT boot sector says the volume is dirty.
//
// FAT keeps the same state twice, here and in the allocation table, and the
// two can disagree. Only the boot sector is read: it is the copy Windows
// writes and Linux reads, and clearing sets both regardless.
func fatDirty(head []byte, offset int) bool {
	return int(head[offset])&0x03 != 0
}

// gptEntry is one partition table entry, in the two fields that matter here.
type gptEntry struct {
	offset     int64 // where the entry sits on the drive
	attributes uint64
}

// readGPT returns the entries of both copies of a GPT.
func readGPT(f *os.File) ([]gptEntry, []int64, error) {
	header := make([]byte, 92)
	if _, err := f.ReadAt(header, 512); err != nil {
		return nil, nil, err
	}
	if string(header[0:8]) != "EFI PART" {
		return nil, nil, fmt.Errorf("no GPT")
	}
	entryLBA := int64(binary.LittleEndian.Uint64(header[72:]))
	count := int64(binary.LittleEndian.Uint32(header[80:]))
	size := int64(binary.LittleEndian.Uint32(header[84:]))
	if count <= 0 || count > 256 || size < 128 || size > 4096 {
		return nil, nil, fmt.Errorf("unreasonable GPT")
	}

	var entries []gptEntry
	for i := int64(0); i < count; i++ {
		off := entryLBA*512 + i*size
		buf := make([]byte, 56)
		if _, err := f.ReadAt(buf, off); err != nil {
			break
		}
		// An entry with no type GUID is an empty slot.
		if isZero(buf[0:16]) {
			continue
		}
		entries = append(entries, gptEntry{offset: off, attributes: binary.LittleEndian.Uint64(buf[48:])})
	}
	return entries, []int64{512}, nil
}

func isZero(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

// clearGPTReadOnly clears the read-only attribute on every entry that has it,
// in both copies of the table, and repairs the checksums that describe them.
func clearGPTReadOnly(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return openHint(path, err)
	}
	defer f.Close()

	capacity, err := f.Seek(0, 2)
	if err != nil {
		return err
	}
	headers := []int64{512} // the primary, at LBA 1
	if alt := altHeaderOffset(f); alt > 0 && alt < capacity {
		headers = append(headers, alt)
	}
	for _, header := range headers {
		if err := clearGPTAt(f, header); err != nil {
			return err
		}
	}
	return nil
}

func altHeaderOffset(f *os.File) int64 {
	header := make([]byte, 92)
	if _, err := f.ReadAt(header, 512); err != nil {
		return 0
	}
	return int64(binary.LittleEndian.Uint64(header[32:])) * 512
}

// clearGPTAt rewrites one copy of the table.
func clearGPTAt(f *os.File, headerOffset int64) error {
	header := make([]byte, 92)
	if _, err := f.ReadAt(header, headerOffset); err != nil {
		return err
	}
	if string(header[0:8]) != "EFI PART" {
		return nil // this copy is not there; the other one is what matters
	}
	entryLBA := int64(binary.LittleEndian.Uint64(header[72:]))
	count := int64(binary.LittleEndian.Uint32(header[80:]))
	size := int64(binary.LittleEndian.Uint32(header[84:]))
	if count <= 0 || count > 256 || size < 128 || size > 4096 {
		return fmt.Errorf("the partition table at %d does not make sense", headerOffset/512)
	}

	array := make([]byte, count*size)
	if _, err := f.ReadAt(array, entryLBA*512); err != nil {
		return err
	}
	changed := false
	for i := int64(0); i < count; i++ {
		entry := array[i*size : (i+1)*size]
		attrs := binary.LittleEndian.Uint64(entry[48:56])
		if attrs&gptReadOnly == 0 {
			continue
		}
		binary.LittleEndian.PutUint64(entry[48:56], attrs&^gptReadOnly)
		changed = true
	}
	if !changed {
		return nil
	}

	// The table describes itself with two checksums, and a table whose
	// checksums do not match is one every tool will refuse or silently
	// replace with the other copy — which still has the flag.
	if _, err := f.WriteAt(array, entryLBA*512); err != nil {
		return err
	}
	binary.LittleEndian.PutUint32(header[88:], crc32.ChecksumIEEE(array))

	headerSize := int64(binary.LittleEndian.Uint32(header[12:]))
	if headerSize < 92 || headerSize > 512 {
		headerSize = 92
	}
	full := make([]byte, headerSize)
	if _, err := f.ReadAt(full, headerOffset); err != nil {
		return err
	}
	copy(full, header)
	binary.LittleEndian.PutUint32(full[16:], 0) // the field is zeroed while it is computed
	binary.LittleEndian.PutUint32(full[16:], crc32.ChecksumIEEE(full))

	_, err := f.WriteAt(full, headerOffset)
	return err
}

// clearDirtyFlag marks a FAT or exFAT volume clean.
func clearDirtyFlag(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return openHint(path, err)
	}
	defer f.Close()

	head := make([]byte, 512)
	if _, err := f.ReadAt(head, 0); err != nil {
		return err
	}

	switch {
	case string(head[3:11]) == "EXFAT   ":
		flags := binary.LittleEndian.Uint16(head[exfatFlagsOffset:])
		binary.LittleEndian.PutUint16(head[exfatFlagsOffset:], flags&^(exfatDirtyBit|exfatFailureBit))
		if _, err := f.WriteAt(head, 0); err != nil {
			return err
		}
		// exFAT keeps a backup of the boot region twelve sectors in.
		if _, err := f.WriteAt(head, 12*512); err != nil {
			return err
		}
		return nil

	case string(head[82:90]) == "FAT32   ":
		head[fat32DirtyOffset] &^= 0x03
		if _, err := f.WriteAt(head, 0); err != nil {
			return err
		}
		if backup := int64(binary.LittleEndian.Uint16(head[50:])); backup > 0 {
			f.WriteAt(head, backup*512)
		}
		return clearFATTableFlags(f, head, true)

	case string(head[54:59]) == "FAT16", string(head[54:59]) == "FAT12":
		head[fat16DirtyOffset] &^= 0x03
		if _, err := f.WriteAt(head, 0); err != nil {
			return err
		}
		return clearFATTableFlags(f, head, false)
	}
	return fmt.Errorf("%s holds no filesystem with a flag to clear", path)
}

// clearFATTableFlags sets the clean and no-error bits in the allocation table,
// which is where a FAT driver looks second.
func clearFATTableFlags(f *os.File, boot []byte, fat32 bool) error {
	reserved := int64(binary.LittleEndian.Uint16(boot[14:]))
	if reserved <= 0 {
		return nil
	}
	offset := reserved * 512
	if fat32 {
		entry := make([]byte, 4)
		if _, err := f.ReadAt(entry, offset+4); err != nil {
			return err
		}
		value := binary.LittleEndian.Uint32(entry) | fatCleanShutBit | fatHardErrorBit
		binary.LittleEndian.PutUint32(entry, value)
		_, err := f.WriteAt(entry, offset+4)
		return err
	}
	entry := make([]byte, 2)
	if _, err := f.ReadAt(entry, offset+2); err != nil {
		return err
	}
	value := binary.LittleEndian.Uint16(entry) | 0x8000 | 0x4000
	binary.LittleEndian.PutUint16(entry, value)
	_, err := f.WriteAt(entry, offset+2)
	return err
}

// TestWrite proves whether a drive takes writes, by writing back the sector it
// just read. Nothing on the drive changes: the bytes written are the bytes
// that were already there.
func TestWrite(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		return openHint(path, err)
	}
	defer f.Close()

	sector := make([]byte, 512)
	if _, err := f.ReadAt(sector, 0); err != nil {
		return fmt.Errorf("could not read the drive: %w", err)
	}
	if _, err := f.WriteAt(sector, 0); err != nil {
		return fmt.Errorf("the drive refused the write: %w", err)
	}
	return f.Sync()
}

// haveTool reports whether an external repair tool is installed.
func haveTool(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
