package device

import (
	"fmt"
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
