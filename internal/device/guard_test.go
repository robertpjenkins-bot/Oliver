package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func removableDrive() *Device {
	return &Device{Path: "/dev/sdb", Name: "sdb", Size: 30 << 30, Removable: true,
		Model: "Cruzer", Readable: true, Writable: true}
}

func TestTheSystemDiskIsRefusedWithNoWayPast(t *testing.T) {
	d := removableDrive()
	d.Mounts = []Mount{{Source: "/dev/sdb1", Target: "/"}, {Source: "/dev/sdb2", Target: "/boot/efi"}}

	err := CheckWritable(d, "erase", true)
	if err == nil {
		t.Fatal("the system disk was allowed")
	}
	reason, ok := RefusedFor(err)
	if !ok || reason != SystemDisk {
		t.Fatalf("reason = %v, %v", reason, ok)
	}
	if reason.Overridable() {
		t.Error("the system disk must never be overridable")
	}
	if !strings.Contains(err.Error(), "/boot/efi") {
		t.Errorf("the refusal does not say which mounts made it one: %v", err)
	}
}

func TestAFixedDiskIsRefusedUntilPermitted(t *testing.T) {
	d := removableDrive()
	d.Removable = false

	err := CheckWritable(d, "write to", false)
	reason, ok := RefusedFor(err)
	if !ok || reason != FixedDisk {
		t.Fatalf("reason = %v, %v (err %v)", reason, ok, err)
	}
	if !reason.Overridable() {
		t.Error("a fixed disk should be overridable")
	}
	if !strings.Contains(err.Error(), "Cruzer") {
		t.Errorf("the refusal should describe the drive so it can be recognised: %v", err)
	}
	if err := CheckWritable(d, "write to", true); err != nil {
		t.Errorf("permission did not permit it: %v", err)
	}
}

func TestADriveTheUserCannotWriteToIsRefused(t *testing.T) {
	d := removableDrive()
	d.Writable = false

	reason, ok := RefusedFor(CheckWritable(d, "erase", false))
	if !ok || reason != NotWritable {
		t.Fatalf("reason = %v, %v", reason, ok)
	}
}

func TestMountedDrivesAreReportedSeparately(t *testing.T) {
	d := removableDrive()
	d.Mounts = []Mount{{Source: "/dev/sdb1", Target: "/media/user/DATA"}}

	// Being mounted does not make a drive unwritable in principle, only right
	// now: the caller may unmount it and carry on.
	if err := CheckWritable(d, "write to", false); err != nil {
		t.Errorf("a mounted removable drive was refused outright: %v", err)
	}
	err := CheckIdle(d, "write to")
	reason, ok := RefusedFor(err)
	if !ok || reason != Mounted {
		t.Fatalf("reason = %v, %v", reason, ok)
	}
	if !strings.Contains(err.Error(), "/media/user/DATA") {
		t.Errorf("the refusal does not say what is mounted: %v", err)
	}
	if err := CheckIdle(removableDrive(), "write to"); err != nil {
		t.Errorf("an idle drive was refused: %v", err)
	}
}

func TestAGoodDriveIsAllowed(t *testing.T) {
	if err := CheckWritable(removableDrive(), "write to", false); err != nil {
		t.Errorf("a plain removable drive was refused: %v", err)
	}
}

func TestRefusalSurvivesBeingWrapped(t *testing.T) {
	err := fmt.Errorf("could not start: %w", CheckWritable(removableDrive(), "erase", false))
	if _, ok := RefusedFor(err); ok {
		t.Error("a nil refusal was wrapped into something")
	}

	d := removableDrive()
	d.Removable = false
	wrapped := fmt.Errorf("%w.\nPass -allow-fixed", CheckWritable(d, "erase", false))
	reason, ok := RefusedFor(wrapped)
	if !ok || reason != FixedDisk {
		t.Errorf("a wrapped refusal lost its reason: %v", wrapped)
	}
}

func TestSomethingThatIsNotARefusalReportsAsSuch(t *testing.T) {
	if _, ok := RefusedFor(fmt.Errorf("disk on fire")); ok {
		t.Error("an ordinary error was read as a refusal")
	}
	if _, ok := RefusedFor(nil); ok {
		t.Error("nil was read as a refusal")
	}
}

// protected marks a device or partition read-only the way the kernel does.
func (f *fakeMachine) protected(name string) {
	f.t.Helper()
	path := filepath.Join(f.root, "sys", "block", name, "ro")
	if err := os.WriteFile(path, []byte("1\n"), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func TestTheKernelsReadOnlyFlagIsFoundAndCalledClearable(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sdb", 60088320, true, "Cruzer")
	f.protected("sdb")

	scanner := f.scanner()
	devices, err := scanner.List()
	if err != nil {
		t.Fatal(err)
	}
	d := devices[0]
	if !d.ReadOnly {
		t.Error("the flag was not read")
	}

	protections := scanner.WriteProtection(d)
	if len(protections) != 1 {
		t.Fatalf("found %d reasons, want 1: %+v", len(protections), protections)
	}
	p := protections[0]
	if p.Kind != KernelFlag || !p.Clearable {
		t.Errorf("protection = %+v", p)
	}
	if !strings.Contains(p.Detail, "changes nothing on the drive") {
		t.Errorf("the detail does not say it is safe: %q", p.Detail)
	}
}

func TestAReadOnlyMountIsToldApartFromALockedDrive(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sdb", 60088320, true, "Cruzer")
	f.partition("sdb", "sdb1", 1, 60000000)
	f.mounts(f.root + "/dev/sdb1 /media/user/STICK iso9660 ro,relatime 0 0\n")

	scanner := f.scanner()
	devices, _ := scanner.List()
	d := devices[0]

	if !d.Mounts[0].ReadOnly {
		t.Fatal("the ro mount option was not noticed")
	}
	protections := scanner.WriteProtection(d)
	if len(protections) != 1 {
		t.Fatalf("found %d reasons: %+v", len(protections), protections)
	}
	if protections[0].Kind != ReadOnlyMount {
		t.Errorf("kind = %q, want a mount rather than a locked drive", protections[0].Kind)
	}
	if !strings.Contains(protections[0].Summary, "/media/user/STICK") {
		t.Errorf("summary does not say where: %q", protections[0].Summary)
	}
}

func TestAWritableMountIsNotReportedAsProtection(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sdb", 60088320, true, "Cruzer")
	f.partition("sdb", "sdb1", 1, 60000000)
	f.mounts(f.root + "/dev/sdb1 /media/user/STICK vfat rw,relatime 0 0\n")

	scanner := f.scanner()
	devices, _ := scanner.List()
	if got := scanner.WriteProtection(devices[0]); len(got) != 0 {
		t.Errorf("a writable mount was called write protection: %+v", got)
	}
}

func TestAProtectedPartitionIsReportedSeparately(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sdb", 60088320, true, "Cruzer")
	f.partition("sdb", "sdb1", 1, 60000000)
	f.protected(filepath.Join("sdb", "sdb1"))

	scanner := f.scanner()
	devices, _ := scanner.List()
	protections := scanner.WriteProtection(devices[0])
	if len(protections) != 1 {
		t.Fatalf("found %d reasons: %+v", len(protections), protections)
	}
	if !strings.Contains(protections[0].Summary, "sdb1") {
		t.Errorf("it does not say which partition: %q", protections[0].Summary)
	}
}

func TestWhatCannotBeFixedSaysSoPlainly(t *testing.T) {
	p := HardwareProtection(&Device{Path: "/dev/sdb", Name: "sdb"})
	if p.Clearable {
		t.Error("a physical lock was reported as clearable")
	}
	// Both causes matter: one is a switch somebody can flip, the other means
	// the drive is finished, and the advice differs completely.
	for _, phrase := range []string{"lock switch", "worn-out flash", "copy anything you need off it"} {
		if !strings.Contains(p.Detail, phrase) {
			t.Errorf("the explanation never mentions %q: %q", phrase, p.Detail)
		}
	}
}
