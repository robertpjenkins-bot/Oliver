package imaging

import (
	"encoding/binary"
	"os"
	"strings"
	"testing"
)

// mbrWith writes a partition table describing one partition of the given size.
func mbrWith(t *testing.T, path string, startSector, sectors uint32) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	mbr := make([]byte, 512)
	e := mbr[446:462]
	e[4] = 0x0C
	binary.LittleEndian.PutUint32(e[8:], startSector)
	binary.LittleEndian.PutUint32(e[12:], sectors)
	mbr[510], mbr[511] = 0x55, 0xAA
	if _, err := f.WriteAt(mbr, 0); err != nil {
		t.Fatal(err)
	}
}

func faultOf(d *Diagnosis, kind FaultKind) *Fault {
	for i := range d.Faults {
		if d.Faults[i].Kind == kind {
			return &d.Faults[i]
		}
	}
	return nil
}

func TestAStickThatLooksSmallerThanItIsIsDiagnosed(t *testing.T) {
	// The everyday case: a 32 GiB stick written with a 1.4 GiB image, which
	// every tool then reports as a 1.4 GiB drive.
	drive := blankDrive(t, 32<<30)
	mbrWith(t, drive, 2048, 1400<<20/512)

	d, err := Diagnose(drive)
	if err != nil {
		t.Fatal(err)
	}
	if d.Healthy() {
		t.Fatal("a drive describing 4% of itself was called healthy")
	}
	f := faultOf(d, Shrunken)
	if f == nil {
		t.Fatalf("faults = %+v", d.Faults)
	}
	// The summary has to carry both numbers: the whole point is the gap.
	if !strings.Contains(f.Summary, "32.0 GB") || !strings.Contains(f.Summary, "1.4 GB") {
		t.Errorf("summary does not name both sizes: %q", f.Summary)
	}
	if d.Capacity != 32<<30 {
		t.Errorf("capacity = %d", d.Capacity)
	}
	if d.Described > d.Capacity/10 {
		t.Errorf("described = %d, expected a small fraction", d.Described)
	}
}

func TestAFreshlyFormattedDriveIsHealthy(t *testing.T) {
	drive := blankDrive(t, 2<<30)
	if _, err := FormatFAT32(drive, FormatOptions{Label: "FINE"}); err != nil {
		t.Fatal(err)
	}
	d, err := Diagnose(drive)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Healthy() {
		t.Errorf("a drive this program just made was called faulty: %+v", d.Faults)
	}
	if d.Described < d.Capacity/2 {
		t.Errorf("described %s of %s", HumanBytes(d.Described), HumanBytes(d.Capacity))
	}
}

func TestALeftoverDiscImageIsReported(t *testing.T) {
	drive := blankDrive(t, 8<<30)
	mbrWith(t, drive, 0, 700<<20/512)

	f, err := os.OpenFile(drive, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteAt([]byte("CD001"), 32769)
	f.Close()

	d, err := Diagnose(drive)
	if err != nil {
		t.Fatal(err)
	}
	fault := faultOf(d, LeftoverImage)
	if fault == nil {
		t.Fatalf("faults = %+v", d.Faults)
	}
	if !strings.Contains(fault.Detail, "read-only") {
		t.Errorf("the detail does not say what it causes: %q", fault.Detail)
	}
	// Both faults are true of this drive and both are worth saying.
	if faultOf(d, Shrunken) == nil {
		t.Error("the size fault was missed")
	}
}

func TestAPartitionPastTheEndOfTheDriveIsReported(t *testing.T) {
	drive := blankDrive(t, 1<<30)
	mbrWith(t, drive, 2048, 4<<30/512) // four times the drive

	d, err := Diagnose(drive)
	if err != nil {
		t.Fatal(err)
	}
	if faultOf(d, Overrun) == nil {
		t.Fatalf("faults = %+v", d.Faults)
	}
	// Overrunning is the more serious reading; it must not also be called
	// shrunken, which would be contradictory advice.
	if faultOf(d, Shrunken) != nil {
		t.Error("the same table was called both too large and too small")
	}
}

func TestAGPTWithoutItsBackupIsReported(t *testing.T) {
	drive := blankDrive(t, 4<<30)
	f, err := os.OpenFile(drive, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	// A protective MBR and a primary GPT header, with nothing at the far end.
	mbr := make([]byte, 512)
	mbr[446+4] = 0xEE
	mbr[510], mbr[511] = 0x55, 0xAA
	f.WriteAt(mbr, 0)
	header := make([]byte, 92)
	copy(header, "EFI PART")
	binary.LittleEndian.PutUint64(header[32:], uint64(8<<30/512-1)) // a larger drive's last sector
	f.WriteAt(header, 512)
	f.Close()

	d, err := Diagnose(drive)
	if err != nil {
		t.Fatal(err)
	}
	fault := faultOf(d, MissingBackupTable)
	if fault == nil {
		t.Fatalf("faults = %+v", d.Faults)
	}
	if !strings.Contains(fault.Detail, "refuse") {
		t.Errorf("the detail does not say what it costs: %q", fault.Detail)
	}
}

func TestDataWithNoTableAtAllIsReported(t *testing.T) {
	drive := blankDrive(t, 2<<30)
	f, _ := os.OpenFile(drive, os.O_RDWR, 0)
	f.WriteAt([]byte("NTFS    "), 3) // a filesystem written straight to the drive
	f.Close()

	d, err := Diagnose(drive)
	if err != nil {
		t.Fatal(err)
	}
	fault := faultOf(d, NoTable)
	if fault == nil {
		t.Fatalf("faults = %+v", d.Faults)
	}
	if !strings.Contains(fault.Detail, "NTFS") {
		t.Errorf("the detail does not say what is on it: %q", fault.Detail)
	}
}

func TestABlankDriveIsNotAFault(t *testing.T) {
	d, err := Diagnose(blankDrive(t, 2<<30))
	if err != nil {
		t.Fatal(err)
	}
	if !d.Healthy() {
		t.Errorf("an empty drive was called faulty: %+v", d.Faults)
	}
}

func TestRepairRestoresTheSpaceTheTableWasHiding(t *testing.T) {
	drive := blankDrive(t, 8<<30)
	mbrWith(t, drive, 2048, 700<<20/512)

	before, err := Diagnose(drive)
	if err != nil {
		t.Fatal(err)
	}
	if before.Healthy() {
		t.Fatal("expected a faulty drive to repair")
	}

	// A repair is an erase and a fresh full-size volume; what matters is that
	// the drive describes all of itself afterwards.
	if _, err := Wipe(t.Context(), drive, WipeOptions{}); err != nil {
		t.Fatal(err)
	}
	formatted, err := FormatFAT32(drive, FormatOptions{Label: "FIXED"})
	if err != nil {
		t.Fatal(err)
	}
	res := &RepairResult{Before: before, Format: formatted}

	after, err := Diagnose(drive)
	if err != nil {
		t.Fatal(err)
	}
	if !after.Healthy() {
		t.Errorf("still faulty after repair: %+v", after.Faults)
	}
	if after.Described < after.Capacity-(2048*sectorSize) {
		t.Errorf("describes %s of %s", HumanBytes(after.Described), HumanBytes(after.Capacity))
	}
	if res.Recovered() < 6<<30 {
		t.Errorf("recovered only %s", HumanBytes(res.Recovered()))
	}
}
