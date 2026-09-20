package device

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gptDrive writes a drive with a GPT holding one partition, optionally marked
// read-only, and returns its path.
func gptDrive(t *testing.T, readOnly bool) string {
	t.Helper()
	const size = 64 << 20
	path := filepath.Join(t.TempDir(), "gpt.img")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := f.Truncate(size); err != nil {
		t.Fatal(err)
	}

	const entryLBA, entryCount, entrySize = 2, 128, 128
	lastLBA := int64(size/512 - 1)

	array := make([]byte, entryCount*entrySize)
	entry := array[0:entrySize]
	copy(entry[0:16], []byte("PARTITIONTYPEGUI")) // a non-empty type marks the slot used
	copy(entry[16:32], []byte("UNIQUEPARTGUID12"))
	binary.LittleEndian.PutUint64(entry[32:], 2048)
	binary.LittleEndian.PutUint64(entry[40:], uint64(lastLBA-33))
	if readOnly {
		binary.LittleEndian.PutUint64(entry[48:], gptReadOnly)
	}

	writeHeader := func(offset int64, myLBA, altLBA, entriesAt int64) {
		header := make([]byte, 512)
		copy(header[0:8], "EFI PART")
		binary.LittleEndian.PutUint32(header[8:], 0x00010000)
		binary.LittleEndian.PutUint32(header[12:], 92)
		binary.LittleEndian.PutUint64(header[24:], uint64(myLBA))
		binary.LittleEndian.PutUint64(header[32:], uint64(altLBA))
		binary.LittleEndian.PutUint64(header[72:], uint64(entriesAt))
		binary.LittleEndian.PutUint32(header[80:], entryCount)
		binary.LittleEndian.PutUint32(header[84:], entrySize)
		binary.LittleEndian.PutUint32(header[88:], crc32.ChecksumIEEE(array))
		binary.LittleEndian.PutUint32(header[16:], crc32.ChecksumIEEE(header[:92]))
		if _, err := f.WriteAt(header, offset); err != nil {
			t.Fatal(err)
		}
	}

	backupEntriesLBA := lastLBA - 32
	writeHeader(512, 1, lastLBA, entryLBA)
	writeHeader(lastLBA*512, lastLBA, 1, backupEntriesLBA)
	if _, err := f.WriteAt(array, entryLBA*512); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt(array, backupEntriesLBA*512); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTheReadOnlyAttributeInAPartitionTableIsFound(t *testing.T) {
	locked := gptDrive(t, true)
	p, ok := gptReadOnlyPartitions(locked)
	if !ok {
		t.Fatal("a read-only GPT entry was not found")
	}
	if p.Kind != GPTAttribute || !p.Clearable {
		t.Errorf("protection = %+v", p)
	}
	if !strings.Contains(p.Detail, "diskpart") {
		t.Errorf("the detail does not say where it comes from: %q", p.Detail)
	}

	// A drive without it must not be reported.
	if _, ok := gptReadOnlyPartitions(gptDrive(t, false)); ok {
		t.Error("an ordinary GPT was called read-only")
	}
}

func TestClearingTheGPTAttributeKeepsTheTableValid(t *testing.T) {
	drive := gptDrive(t, true)
	if err := clearGPTReadOnly(drive); err != nil {
		t.Fatalf("clearGPTReadOnly: %v", err)
	}
	if _, ok := gptReadOnlyPartitions(drive); ok {
		t.Fatal("the attribute survived")
	}

	f, err := os.Open(drive)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// A table whose checksums no longer match is one every tool will refuse
	// or quietly replace from the other copy — which would bring the flag
	// back. Both copies must still check out.
	for _, offset := range []int64{512, (64<<20)/512 - 1} {
		at := offset
		if at != 512 {
			at = offset * 512
		}
		header := make([]byte, 512)
		if _, err := f.ReadAt(header, at); err != nil {
			t.Fatal(err)
		}
		if string(header[0:8]) != "EFI PART" {
			t.Fatalf("no header at %d", at)
		}
		entryLBA := int64(binary.LittleEndian.Uint64(header[72:]))
		count := int64(binary.LittleEndian.Uint32(header[80:]))
		size := int64(binary.LittleEndian.Uint32(header[84:]))
		array := make([]byte, count*size)
		if _, err := f.ReadAt(array, entryLBA*512); err != nil {
			t.Fatal(err)
		}
		if got, want := binary.LittleEndian.Uint32(header[88:]), crc32.ChecksumIEEE(array); got != want {
			t.Errorf("entry checksum at %d is %#x, want %#x", at, got, want)
		}
		stored := binary.LittleEndian.Uint32(header[16:])
		binary.LittleEndian.PutUint32(header[16:], 0)
		if want := crc32.ChecksumIEEE(header[:92]); stored != want {
			t.Errorf("header checksum at %d is %#x, want %#x", at, stored, want)
		}
		// And the flag must be gone from this copy too, not only the first.
		if attrs := binary.LittleEndian.Uint64(array[48:56]); attrs&gptReadOnly != 0 {
			t.Errorf("the copy at %d still carries the read-only attribute", at)
		}
	}
}

// fatDrive writes a FAT32 volume, optionally marked dirty.
func fatDrive(t *testing.T, dirty bool) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fat.img")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.Truncate(64 << 20)

	boot := make([]byte, 512)
	copy(boot[82:90], "FAT32   ")
	binary.LittleEndian.PutUint16(boot[14:], 32) // reserved sectors
	binary.LittleEndian.PutUint16(boot[50:], 6)  // backup boot sector
	if dirty {
		boot[fat32DirtyOffset] = 0x01
	}
	f.WriteAt(boot, 0)

	// The allocation table's second entry, with the clean-shutdown bit off.
	entry := make([]byte, 4)
	binary.LittleEndian.PutUint32(entry, 0x0FFFFFFF&^fatCleanShutBit)
	f.WriteAt(entry, 32*512+4)
	return path
}

func TestADirtyFATVolumeIsFoundAndCanBeMarkedClean(t *testing.T) {
	drive := fatDrive(t, true)

	p, ok := filesystemProtection(drive)
	if !ok {
		t.Fatal("a dirty FAT32 volume was not noticed")
	}
	if p.Kind != DirtyFilesystem || !p.Clearable {
		t.Errorf("protection = %+v", p)
	}
	// Clearing this says the volume is clean without checking, and the
	// wording has to admit that.
	if !strings.Contains(p.Detail, "does not check") {
		t.Errorf("the detail hides the cost: %q", p.Detail)
	}

	if err := clearDirtyFlag(drive); err != nil {
		t.Fatalf("clearDirtyFlag: %v", err)
	}
	if _, ok := filesystemProtection(drive); ok {
		t.Error("the volume is still marked dirty")
	}

	f, _ := os.Open(drive)
	defer f.Close()
	// Both copies of the state have to agree, or a driver that reads the
	// other one mounts read-only again.
	entry := make([]byte, 4)
	f.ReadAt(entry, 32*512+4)
	if binary.LittleEndian.Uint32(entry)&fatCleanShutBit == 0 {
		t.Error("the allocation table still says the volume is dirty")
	}
	backup := make([]byte, 512)
	f.ReadAt(backup, 6*512)
	if backup[fat32DirtyOffset]&0x03 != 0 {
		t.Error("the backup boot sector still says the volume is dirty")
	}
}

func TestACleanFATVolumeIsLeftAlone(t *testing.T) {
	if _, ok := filesystemProtection(fatDrive(t, false)); ok {
		t.Error("a clean volume was reported as dirty")
	}
}

func TestADirtyExFATVolumeIsFoundAndCleared(t *testing.T) {
	path := filepath.Join(t.TempDir(), "exfat.img")
	f, _ := os.Create(path)
	f.Truncate(64 << 20)
	boot := make([]byte, 512)
	copy(boot[3:11], "EXFAT   ")
	binary.LittleEndian.PutUint16(boot[exfatFlagsOffset:], exfatDirtyBit|exfatFailureBit)
	f.WriteAt(boot, 0)
	f.Close()

	p, ok := filesystemProtection(path)
	if !ok {
		t.Fatal("a dirty exFAT volume was not noticed")
	}
	if !strings.Contains(p.Detail, "media errors") {
		t.Errorf("the media-failure bit went unmentioned: %q", p.Detail)
	}
	if err := clearDirtyFlag(path); err != nil {
		t.Fatal(err)
	}
	if _, ok := filesystemProtection(path); ok {
		t.Error("the flags survived")
	}
}

func TestNTFSIsReportedHonestlyRatherThanTouched(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ntfs.img")
	f, _ := os.Create(path)
	f.Truncate(64 << 20)
	boot := make([]byte, 512)
	copy(boot[3:11], "NTFS    ")
	f.WriteAt(boot, 0)
	f.Close()

	p, ok := filesystemProtection(path)
	if !ok {
		t.Fatal("NTFS was not recognised")
	}
	if p.Clearable {
		t.Error("NTFS was claimed as clearable; its flag is not in the header")
	}
	for _, phrase := range []string{"Fast Startup", "ntfsfix"} {
		if !strings.Contains(p.Detail, phrase) {
			t.Errorf("the advice never mentions %q: %q", phrase, p.Detail)
		}
	}
}

func TestAnExtFilesystemWithErrorsIsReportedNotCleared(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ext4.img")
	f, _ := os.Create(path)
	f.Truncate(64 << 20)
	sb := make([]byte, 2048)
	binary.LittleEndian.PutUint16(sb[1080:], 0xEF53)
	binary.LittleEndian.PutUint16(sb[1082:], 2) // the kernel found errors
	f.WriteAt(sb, 0)
	f.Close()

	p, ok := filesystemProtection(path)
	if !ok {
		t.Fatal("an ext filesystem with errors was not noticed")
	}
	if p.Clearable {
		t.Error("clearing this would hide a real problem, not fix it")
	}
	if !strings.Contains(p.Detail, "fsck") {
		t.Errorf("the advice does not name the tool: %q", p.Detail)
	}

	// A clean one must not be reported.
	binary.LittleEndian.PutUint16(sb[1082:], 1)
	f2, _ := os.OpenFile(path, os.O_RDWR, 0)
	f2.WriteAt(sb, 0)
	f2.Close()
	if _, ok := filesystemProtection(path); ok {
		t.Error("a clean ext filesystem was reported as faulty")
	}
}

func TestTheWriteTestProvesItWithoutChangingAnything(t *testing.T) {
	drive := fatDrive(t, false)
	before, err := os.ReadFile(drive)
	if err != nil {
		t.Fatal(err)
	}
	if err := TestWrite(drive); err != nil {
		t.Fatalf("a writable drive failed the test: %v", err)
	}
	after, err := os.ReadFile(drive)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("the test changed the drive")
	}

	// And a drive that cannot be opened for writing fails it.
	if err := os.Chmod(drive, 0o444); err != nil {
		t.Skip("cannot make the file read-only here")
	}
	if os.Geteuid() == 0 {
		t.Skip("running as root, which ignores the permission")
	}
	if err := TestWrite(drive); err == nil {
		t.Error("a read-only drive passed the write test")
	}
}
