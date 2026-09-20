package device

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeMachine builds a synthetic /sys/block and /proc/mounts so device
// detection is tested without depending on the machine running the tests.
type fakeMachine struct {
	t    *testing.T
	root string
}

func newFakeMachine(t *testing.T) *fakeMachine {
	t.Helper()
	return &fakeMachine{t: t, root: t.TempDir()}
}

// disk adds a block device. sizeSectors is in 512-byte units, as /sys reports.
func (f *fakeMachine) disk(name string, sizeSectors int64, removable bool, model string) {
	f.t.Helper()
	base := filepath.Join(f.root, "sys", "block", name)
	if err := os.MkdirAll(filepath.Join(base, "device"), 0o755); err != nil {
		f.t.Fatal(err)
	}
	write := func(path, content string) {
		if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
			f.t.Fatal(err)
		}
	}
	write(filepath.Join(base, "size"), itoa(sizeSectors))
	rem := "0"
	if removable {
		rem = "1"
	}
	write(filepath.Join(base, "removable"), rem)
	if model != "" {
		write(filepath.Join(base, "device", "model"), model)
	}
	// The device node itself.
	if err := os.MkdirAll(filepath.Join(f.root, "dev"), 0o755); err != nil {
		f.t.Fatal(err)
	}
	write(filepath.Join(f.root, "dev", name), "")
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

func (f *fakeMachine) mounts(lines string) {
	f.t.Helper()
	if err := os.WriteFile(filepath.Join(f.root, "mounts"), []byte(lines), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fakeMachine) scanner() *Scanner {
	return &Scanner{
		SysBlock: filepath.Join(f.root, "sys", "block"),
		Dev:      filepath.Join(f.root, "dev"),
		Mounts:   filepath.Join(f.root, "mounts"),
		Access:   func(string) (bool, bool) { return true, true },
	}
}

func TestListReportsSizeRemovabilityAndModel(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sda", 60088320, true, "SanDisk Ultra")  // ~28.6 GiB stick
	f.disk("nvme0n1", 1000215216, false, "SSD 980") // ~477 GiB
	f.mounts("")

	devices, err := f.scanner().List()
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 2 {
		t.Fatalf("found %d devices, want 2", len(devices))
	}

	byName := map[string]*Device{}
	for _, d := range devices {
		byName[d.Name] = d
	}
	if got := byName["sda"]; !got.Removable || got.Model != "SanDisk Ultra" {
		t.Errorf("sda = %+v", got)
	}
	if got := byName["sda"].Size; got != 60088320*512 {
		t.Errorf("size = %d, want %d", got, 60088320*512)
	}
	if byName["nvme0n1"].Removable {
		t.Error("an internal SSD must not be reported as removable")
	}
}

func TestVirtualDevicesAreIgnored(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sda", 1000, true, "")
	for _, name := range []string{"loop0", "loop1", "ram0", "zram0", "dm-0", "sr0"} {
		f.disk(name, 1000, false, "")
	}
	f.mounts("")

	devices, err := f.scanner().List()
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 || devices[0].Name != "sda" {
		t.Errorf("expected only sda, got %+v", devices)
	}
}

func TestSystemDiskIsIdentifiedThroughItsPartitions(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("nvme0n1", 1000215216, false, "SSD")
	f.disk("sda", 60088320, true, "USB stick")
	// The root filesystem lives on a partition, not the whole device: the
	// guard has to attribute it to the parent disk.
	f.mounts(filepath.Join(f.root, "dev", "nvme0n1p3") + " / ext4 rw 0 0\n" +
		filepath.Join(f.root, "dev", "nvme0n1p1") + " /boot/efi vfat rw 0 0\n" +
		filepath.Join(f.root, "dev", "sda1") + " /run/media/deck/USB vfat rw 0 0\n")

	devices, err := f.scanner().List()
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]*Device{}
	for _, d := range devices {
		byName[d.Name] = d
	}

	sys := byName["nvme0n1"]
	if !sys.SystemCritical() {
		t.Fatalf("the disk holding / was not flagged as critical: %+v", sys)
	}
	crit := sys.CriticalMounts()
	if len(crit) != 2 || crit[0] != "/" || crit[1] != "/boot/efi" {
		t.Errorf("CriticalMounts = %v, want / and /boot/efi", crit)
	}

	usb := byName["sda"]
	if usb.SystemCritical() {
		t.Error("a mounted USB stick must not count as system-critical")
	}
	if len(usb.Mounts) != 1 {
		t.Errorf("expected the stick's mount to be recorded, got %+v", usb.Mounts)
	}
}

func TestMountPointEscapesAreDecoded(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sda", 1000, true, "")
	f.mounts(filepath.Join(f.root, "dev", "sda1") + ` /run/media/deck/My\040USB\040Drive vfat rw 0 0` + "\n")

	devices, _ := f.scanner().List()
	if len(devices[0].Mounts) != 1 {
		t.Fatal("mount not recorded")
	}
	if got := devices[0].Mounts[0].Target; got != "/run/media/deck/My USB Drive" {
		t.Errorf("target = %q, want spaces decoded", got)
	}
}

func TestFindAcceptsPathsAndRejectsPartitions(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sda", 1000, true, "")
	f.mounts("")
	s := f.scanner()

	for _, q := range []string{"sda", filepath.Join(f.root, "dev", "sda")} {
		if d, err := s.Find(q); err != nil || d.Name != "sda" {
			t.Errorf("Find(%q) = %v, %v", q, d, err)
		}
	}
	// A partition is not a drive: writing a boot record to sda1 is a mistake.
	if _, err := s.Find("sda1"); err == nil {
		t.Error("expected Find to reject a partition")
	}
	if _, err := s.Find("nonexistent"); err == nil {
		t.Error("expected an error for an unknown device")
	}
	if _, err := s.Find(""); err == nil {
		t.Error("expected an error for an empty query")
	}
}

func TestAccessIsReportedAndAdvised(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sda", 1000, true, "")
	f.mounts("")

	s := f.scanner()
	s.Access = func(string) (bool, bool) { return true, false } // readable, not writable
	devices, _ := s.List()
	d := devices[0]
	if !d.Readable || d.Writable {
		t.Errorf("access = %v/%v, want readable but not writable", d.Readable, d.Writable)
	}
	advice := AccessAdvice(d)
	if advice == "" {
		t.Fatal("expected advice when the device is not writable")
	}

	s.Access = func(string) (bool, bool) { return true, true }
	devices, _ = s.List()
	if AccessAdvice(devices[0]) != "" {
		t.Error("no advice needed when the device is already writable")
	}
}

// partition adds a partition to a device that already exists.
func (f *fakeMachine) partition(disk, name string, number int, sizeSectors int64) {
	f.t.Helper()
	base := filepath.Join(f.root, "sys", "block", disk, name)
	if err := os.MkdirAll(base, 0o755); err != nil {
		f.t.Fatal(err)
	}
	write := func(path, content string) {
		if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
			f.t.Fatal(err)
		}
	}
	write(filepath.Join(base, "partition"), itoa(int64(number)))
	write(filepath.Join(base, "size"), itoa(sizeSectors))
	write(filepath.Join(f.root, "dev", name), "")
}

func TestPartitionsAreListedInOrderWithTheirMounts(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sdb", 30310400, true, "Cruzer Blade")
	// Deliberately created out of order: /sys does not promise an order, and a
	// numbered list shown to a user has to be stable.
	f.partition("sdb", "sdb2", 2, 20000000)
	f.partition("sdb", "sdb1", 1, 1048576)
	f.mounts(f.root + "/dev/sdb2 /media/user/DATA ext4 rw 0 0\n")

	devices, err := f.scanner().List()
	if err != nil {
		t.Fatal(err)
	}
	d := devices[0]
	if len(d.Partitions) != 2 {
		t.Fatalf("found %d partitions, want 2", len(d.Partitions))
	}
	if d.Partitions[0].Name != "sdb1" || d.Partitions[1].Name != "sdb2" {
		t.Errorf("partitions out of order: %s, %s", d.Partitions[0].Name, d.Partitions[1].Name)
	}
	if got := d.Partitions[0].Size; got != 1048576*512 {
		t.Errorf("sdb1 size = %d bytes", got)
	}
	if len(d.Partitions[1].Mounts) != 1 || d.Partitions[1].Mounts[0].Target != "/media/user/DATA" {
		t.Errorf("sdb2 mounts = %+v", d.Partitions[1].Mounts)
	}
	if len(d.Partitions[0].Mounts) != 0 {
		t.Errorf("sdb1 should not have inherited its neighbour's mount")
	}
	if !d.Busy() {
		t.Error("a device with a mounted partition is not idle")
	}
}

func TestDeviceWithoutPartitionsReportsNone(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sdc", 2048, true, "blank")
	devices, err := f.scanner().List()
	if err != nil {
		t.Fatal(err)
	}
	if len(devices[0].Partitions) != 0 {
		t.Errorf("an unpartitioned drive reported %d partitions", len(devices[0].Partitions))
	}
	if devices[0].Busy() {
		t.Error("an unmounted drive reported as busy")
	}
}

func TestUnmountReleasesEveryMountAndReportsWhatItCannot(t *testing.T) {
	var called [][]string
	original := runCommand
	t.Cleanup(func() { runCommand = original })
	runCommand = func(name string, args ...string) ([]byte, error) {
		called = append(called, append([]string{name}, args...))
		if args[len(args)-1] == "/dev/sdb2" {
			return []byte("target is busy"), errBusy
		}
		return nil, nil
	}
	// Only the commands present on this machine are tried, so the test drives
	// the lookup through a directory it controls.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "umount"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	d := &Device{Path: "/dev/sdb", Mounts: []Mount{
		{Source: "/dev/sdb1", Target: "/media/a"},
		{Source: "/dev/sdb2", Target: "/media/b"},
	}}
	err := Unmount(d)
	if err == nil {
		t.Fatal("a busy mount was reported as released")
	}
	if !strings.Contains(err.Error(), "/dev/sdb2") || !strings.Contains(err.Error(), "busy") {
		t.Errorf("error does not say what failed or why: %v", err)
	}
	if len(called) != 2 {
		t.Errorf("ran %d commands, want one per mount: %v", len(called), called)
	}
}

var errBusy = errors.New("exit status 32")
