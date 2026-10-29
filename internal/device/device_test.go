package device

import (
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

// partition adds a partition to a disk created with disk.
func (f *fakeMachine) partition(disk, name string, number int, sizeSectors int64) {
	f.t.Helper()
	dir := filepath.Join(f.root, "sys", "block", disk, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		f.t.Fatal(err)
	}
	for file, content := range map[string]string{
		"partition": itoa(int64(number)),
		"size":      itoa(sizeSectors),
	} {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content+"\n"), 0o644); err != nil {
			f.t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(f.root, "dev", name), nil, 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// sysFile writes a file under a device's /sys directory, such as "ro" or
// "holders/dm-0".
func (f *fakeMachine) sysFile(content string, parts ...string) {
	f.t.Helper()
	p := filepath.Join(append([]string{f.root, "sys", "block"}, parts...)...)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content+"\n"), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// udev records what udev knows about a device, keyed by its major:minor.
func (f *fakeMachine) udev(id string, props ...string) {
	f.t.Helper()
	dir := filepath.Join(f.root, "udev")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		f.t.Fatal(err)
	}
	var b []byte
	b = append(b, "S:disk/by-id/whatever\n"...)
	for _, p := range props {
		b = append(b, "E:"+p+"\n"...)
	}
	if err := os.WriteFile(filepath.Join(dir, "b"+id), b, 0o644); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fakeMachine) fullScanner() *Scanner {
	s := f.scanner()
	s.Udev = filepath.Join(f.root, "udev")
	s.Swaps = filepath.Join(f.root, "swaps")
	return s
}

func (f *fakeMachine) dev(name string) string { return filepath.Join(f.root, "dev", name) }

func TestPartitionsAreListedWithTheirFilesystems(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sdb", 60088320, false, "Ultra")
	f.partition("sdb", "sdb2", 2, 1000)
	f.partition("sdb", "sdb1", 1, 60000000)
	f.sysFile("8:17", "sdb", "sdb1", "dev")
	f.sysFile("8:16", "sdb", "dev")
	f.udev("8:17", "ID_FS_TYPE=exfat", "ID_FS_LABEL=Ventoy")
	f.udev("8:16", "ID_BUS=usb")
	f.mounts(f.dev("sdb1") + " /run/media/u/Ventoy exfat rw 0 0\n")

	devices, err := f.fullScanner().List()
	if err != nil {
		t.Fatal(err)
	}
	d := devices[0]
	if !d.USB {
		t.Error("a drive udev puts on the USB bus must count as USB, whatever its removable flag says")
	}
	if len(d.Partitions) != 2 {
		t.Fatalf("found %d partitions, want 2", len(d.Partitions))
	}
	p := d.Partitions[0]
	if p.Number != 1 || p.Path != f.dev("sdb1") || p.Size != 60000000*512 {
		t.Errorf("partitions should be in order with path and size: %+v", p)
	}
	if p.FSType != "exfat" || p.Label != "Ventoy" {
		t.Errorf("filesystem = %q %q, want exfat Ventoy", p.FSType, p.Label)
	}
	if len(p.Mounts) != 1 || len(d.Partitions[1].Mounts) != 0 {
		t.Errorf("mounts should be attributed to the partition they come from: %+v / %+v", p.Mounts, d.Partitions[1].Mounts)
	}
}

func TestTargetsOfferDrivesAndPartitionsPluggedInOnesFirst(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("nvme0n1", 1000215216, false, "SSD")
	f.partition("nvme0n1", "nvme0n1p1", 1, 1000000)
	f.disk("sdb", 60088320, true, "Stick")
	f.partition("sdb", "sdb1", 1, 60000000)
	f.disk("sdc", 0, true, "Card Reader") // no card inserted
	f.mounts("")

	targets, err := f.fullScanner().Targets()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, tg := range targets {
		got = append(got, filepath.Base(tg.Path))
	}
	want := []string{"sdb", "sdb1", "nvme0n1", "nvme0n1p1"}
	if len(got) != len(want) {
		t.Fatalf("targets = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("targets = %v, want %v", got, want)
		}
	}
	if !targets[0].Whole() || targets[1].Whole() {
		t.Error("the drive and its partition should be told apart")
	}
	if targets[1].Name() != "partition 1 of Stick" {
		t.Errorf("Name = %q", targets[1].Name())
	}
}

func TestTargetProblemsProtectWhatIsInUse(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("nvme0n1", 1000215216, false, "SSD")
	f.partition("nvme0n1", "nvme0n1p1", 1, 1000)
	f.partition("nvme0n1", "nvme0n1p2", 2, 1000)
	f.partition("nvme0n1", "nvme0n1p3", 3, 1000) // a spare partition
	f.disk("sdb", 1000, true, "")
	f.partition("sdb", "sdb1", 1, 900) // an opened encrypted volume
	f.sysFile("", "sdb", "sdb1", "holders", "dm-0")
	f.disk("sdc", 1000, true, "")
	f.partition("sdc", "sdc1", 1, 900)
	f.disk("sdd", 1000, true, "") // a locked SD card
	f.sysFile("1", "sdd", "ro")
	f.mounts(f.dev("nvme0n1p1") + " /boot/efi vfat rw 0 0\n" + f.dev("nvme0n1p2") + " / ext4 rw 0 0\n")
	if err := os.WriteFile(filepath.Join(f.root, "swaps"),
		[]byte("Filename Type Size Used Priority\n"+f.dev("sdc1")+" partition 100 0 -2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := f.fullScanner()
	cases := map[string]string{
		"nvme0n1":   "running system",
		"nvme0n1p1": "running system (/boot/efi)",
		"nvme0n1p2": "running system (/)",
		"nvme0n1p3": "",
		"sdb":       "dm-0",
		"sdb1":      "dm-0",
		"sdc":       "swap",
		"sdc1":      "swap",
		"sdd":       "write-protected",
	}
	for name, want := range cases {
		tg, err := s.FindTarget(name)
		if err != nil {
			t.Fatalf("FindTarget(%s): %v", name, err)
		}
		got := tg.Problem()
		if want == "" && got != "" {
			t.Errorf("%s: unexpected problem %q", name, got)
		}
		if want != "" && !strings.Contains(got, want) {
			t.Errorf("%s: problem = %q, want it to mention %q", name, got, want)
		}
	}
}

func TestMountedUSBPartitionIsAllowedAndItsMountsListed(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sdb", 1000, true, "")
	f.partition("sdb", "sdb1", 1, 900)
	f.mounts(f.dev("sdb1") + " /run/media/u/STICK vfat rw 0 0\n")

	s := f.fullScanner()
	for _, name := range []string{"sdb", "sdb1", f.dev("sdb1")} {
		tg, err := s.FindTarget(name)
		if err != nil {
			t.Fatalf("FindTarget(%s): %v", name, err)
		}
		if p := tg.Problem(); p != "" {
			t.Errorf("%s: a mounted stick can be unmounted, so it is no problem: %q", name, p)
		}
		if len(tg.Mounts()) != 1 {
			t.Errorf("%s: mounts to unmount = %+v", name, tg.Mounts())
		}
		if !tg.Holds(f.dev("sdb1")) {
			t.Errorf("%s should hold sdb1", name)
		}
	}
	if _, err := s.FindTarget("sdz"); err == nil {
		t.Error("expected an error for a drive that is not there")
	}

	// A stable name is a link to the device node.
	byID := filepath.Join(f.root, "dev", "disk", "by-id")
	if err := os.MkdirAll(byID, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(byID, "usb-SanDisk_Ultra-0:0-part1")
	if err := os.Symlink("../../sdb1", link); err != nil {
		t.Fatal(err)
	}
	if tg, err := s.FindTarget(link); err != nil || tg.Path != f.dev("sdb1") {
		t.Errorf("FindTarget(%s) = %+v, %v; want sdb1", link, tg.Path, err)
	}
}

func TestMountForFindsTheFilesystemAFileIsOn(t *testing.T) {
	f := newFakeMachine(t)
	f.disk("sdb", 1000, true, "")
	media := filepath.Join(f.root, "media", "stick")
	f.mounts(f.dev("nvme0n1p2") + " / ext4 rw 0 0\n" + f.dev("sdb1") + " " + media + " vfat rw 0 0\n")

	s := f.fullScanner()
	if m, ok := s.MountFor(filepath.Join(media, "linux.iso")); !ok || m.Source != f.dev("sdb1") {
		t.Errorf("MountFor inside the stick = %+v, %v", m, ok)
	}
	if m, ok := s.MountFor(filepath.Join(f.root, "media", "stickier", "x.iso")); !ok || m.Target != "/" {
		t.Errorf("a sibling directory with a longer name is not inside the mount: %+v", m)
	}
}
