package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/robertpjenkins-bot/oliver/internal/device"
	"github.com/robertpjenkins-bot/oliver/internal/imaging"
	"github.com/robertpjenkins-bot/oliver/internal/ui"
)

// machine is a synthetic set of drives: a /sys/block describing them and a
// /dev holding a file per drive that stands in for the device node. Because
// every operation here works on a path, writing to one of those files exercises
// the same code that writes to a real stick.
type machine struct {
	t           *testing.T
	root        string
	out         *bytes.Buffer
	err         *bytes.Buffer
	input       string
	interactive bool
}

func newMachine(t *testing.T) *machine {
	t.Helper()
	m := &machine{t: t, root: t.TempDir(), out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	if err := os.MkdirAll(filepath.Join(m.root, "dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	m.mounts("")
	return m
}

// drive adds a block device of the given size and returns the path of the file
// standing in for its device node.
func (m *machine) drive(name string, size int64, removable bool, model string) string {
	m.t.Helper()
	base := filepath.Join(m.root, "sys", "block", name)
	if err := os.MkdirAll(filepath.Join(base, "device"), 0o755); err != nil {
		m.t.Fatal(err)
	}
	write := func(path, content string) {
		if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
			m.t.Fatal(err)
		}
	}
	write(filepath.Join(base, "size"), itoa(size/512))
	removableFlag := "0"
	if removable {
		removableFlag = "1"
	}
	write(filepath.Join(base, "removable"), removableFlag)
	write(filepath.Join(base, "device", "model"), model)

	node := filepath.Join(m.root, "dev", name)
	f, err := os.Create(node)
	if err != nil {
		m.t.Fatal(err)
	}
	defer f.Close()
	if err := f.Truncate(size); err != nil {
		m.t.Fatal(err)
	}
	return node
}

func (m *machine) mounts(lines string) {
	m.t.Helper()
	if err := os.WriteFile(filepath.Join(m.root, "mounts"), []byte(lines), 0o644); err != nil {
		m.t.Fatal(err)
	}
}

// mount records a filesystem from one of the drives as mounted.
func (m *machine) mount(devName, target string) {
	m.t.Helper()
	line := filepath.Join(m.root, "dev", devName) + " " + target + " ext4 rw 0 0\n"
	existing, _ := os.ReadFile(filepath.Join(m.root, "mounts"))
	m.mounts(string(existing) + line)
}

// answers queues what a user types at the prompts.
func (m *machine) answers(lines ...string) {
	m.interactive = true
	m.input = strings.Join(lines, "\n") + "\n"
}

func (m *machine) run(args ...string) (string, error) {
	m.t.Helper()
	m.out.Reset()
	m.err.Reset()

	origUI, origScanner := newUI, newScanner
	newUI = func() *ui.UI {
		return &ui.UI{
			Out:         m.out,
			Err:         m.err,
			In:          strings.NewReader(m.input),
			Interactive: m.interactive,
		}
	}
	newScanner = func() *device.Scanner {
		return &device.Scanner{
			SysBlock: filepath.Join(m.root, "sys", "block"),
			Dev:      filepath.Join(m.root, "dev"),
			Mounts:   filepath.Join(m.root, "mounts"),
			Access:   func(string) (bool, bool) { return true, true },
		}
	}
	defer func() { newUI, newScanner = origUI, origScanner }()

	m.t.Setenv("OLIVER_CONFIG", filepath.Join(m.root, "config.json"))
	err := run(args)
	return m.out.String() + m.err.String(), err
}

func itoa(n int64) string { return strings.TrimSpace(strings.Join([]string{intToString(n)}, "")) }

func intToString(n int64) string {
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

// bootableISO writes a small image that identifies as a bootable hybrid ISO,
// so the inspector has something real to read.
func bootableISO(t *testing.T, name string, size int64) string {
	t.Helper()
	if size < 24*2048 {
		size = 24 * 2048
	}
	img := make([]byte, size)
	for i := range img {
		img[i] = byte(i * 7)
	}

	// A partition table in the system area: this is what isohybrid adds.
	for i := 0; i < 512; i++ {
		img[i] = 0
	}
	img[446] = 0x80
	img[450] = 0x17
	binary.LittleEndian.PutUint32(img[458:], uint32(size/512))
	img[510], img[511] = 0x55, 0xAA

	// The primary volume descriptor, at the usual sector 16.
	pvd := img[16*2048:]
	for i := 0; i < 2048; i++ {
		pvd[i] = 0
	}
	pvd[0] = 1
	copy(pvd[1:], "CD001")
	pvd[6] = 1
	copy(pvd[40:72], "TESTIMAGE                       ")

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, img, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func driveContents(t *testing.T, path string, n int) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	buf := make([]byte, n)
	f.ReadAt(buf, 0)
	return buf
}

func TestWriteCopiesAnImageToTheChosenDriveAndVerifiesIt(t *testing.T) {
	m := newMachine(t)
	node := m.drive("sdb", 64<<20, true, "Cruzer Blade")
	iso := bootableISO(t, "test.iso", 2<<20)

	out, err := m.run("write", iso, "sdb", "-f")
	if err != nil {
		t.Fatalf("write failed: %v\n%s", err, out)
	}

	want, _ := os.ReadFile(iso)
	if got := driveContents(t, node, len(want)); !bytes.Equal(got, want) {
		t.Error("the drive does not hold the image")
	}
	for _, phrase := range []string{"ISO 9660", "TESTIMAGE", "read back and compared", "sha256"} {
		if !strings.Contains(out, phrase) {
			t.Errorf("output never mentions %q:\n%s", phrase, out)
		}
	}
}

func TestWriteRefusesTheDiskTheSystemIsRunningFrom(t *testing.T) {
	m := newMachine(t)
	node := m.drive("sda", 64<<20, false, "SSD")
	m.mount("sda", "/")
	iso := bootableISO(t, "test.iso", 1<<20)

	before := driveContents(t, node, 4096)
	out, err := m.run("write", iso, "sda", "-f", "-allow-fixed")
	if err == nil {
		t.Fatal("wrote to the system disk")
	}
	if !strings.Contains(err.Error(), "running system") || !strings.Contains(err.Error(), "no flag for this") {
		t.Errorf("refusal does not explain itself: %v\n%s", err, out)
	}
	if !bytes.Equal(driveContents(t, node, 4096), before) {
		t.Error("the system disk was written to despite the refusal")
	}
}

func TestWriteRefusesAFixedDiskUnlessToldOtherwise(t *testing.T) {
	m := newMachine(t)
	m.drive("sdb", 64<<20, false, "Internal disk")
	iso := bootableISO(t, "test.iso", 1<<20)

	_, err := m.run("write", iso, "sdb", "-f")
	if err == nil {
		t.Fatal("a fixed disk was written to without -allow-fixed")
	}
	if !strings.Contains(err.Error(), "-allow-fixed") {
		t.Errorf("refusal does not name the way past it: %v", err)
	}

	if _, err := m.run("write", iso, "sdb", "-f", "-allow-fixed"); err != nil {
		t.Errorf("-allow-fixed did not permit the write: %v", err)
	}
}

func TestWriteRefusesADriveThatIsMounted(t *testing.T) {
	m := newMachine(t)
	node := m.drive("sdb", 64<<20, true, "stick")
	m.mount("sdb", "/media/user/DATA")
	iso := bootableISO(t, "test.iso", 1<<20)

	before := driveContents(t, node, 4096)
	_, err := m.run("write", iso, "sdb", "-f")
	if err == nil {
		t.Fatal("wrote to a mounted drive")
	}
	if !strings.Contains(err.Error(), "in use") || !strings.Contains(err.Error(), "-unmount") {
		t.Errorf("refusal is not actionable: %v", err)
	}
	if !bytes.Equal(driveContents(t, node, 4096), before) {
		t.Error("the mounted drive was written to anyway")
	}
}

func TestWriteRefusesAnImageTooBigForTheDrive(t *testing.T) {
	m := newMachine(t)
	m.drive("sdb", 1<<20, true, "tiny")
	iso := bootableISO(t, "big.iso", 8<<20)

	_, err := m.run("write", iso, "sdb", "-f")
	if err == nil || !strings.Contains(err.Error(), "smaller than") {
		t.Fatalf("error = %v, want a refusal naming the size", err)
	}
}

func TestWriteAsksWhichDriveAndNeverOffersTheSystemDisk(t *testing.T) {
	m := newMachine(t)
	m.drive("sda", 64<<20, false, "System SSD")
	m.drive("sdb", 64<<20, true, "Kingston")
	node := m.drive("sdc", 64<<20, true, "SanDisk")
	m.mount("sda", "/")
	iso := bootableISO(t, "test.iso", 1<<20)

	// The system disk is shown but unnumbered, so the removable drives are
	// choices 1 and 2. Then the drive's name is typed to confirm.
	m.answers("2", "sdc")
	out, err := m.run("write", iso)
	if err != nil {
		t.Fatalf("write failed: %v\n%s", err, out)
	}

	want, _ := os.ReadFile(iso)
	if got := driveContents(t, node, len(want)); !bytes.Equal(got, want) {
		t.Error("the image did not go to the drive that was chosen")
	}
	if !strings.Contains(out, "the running system") {
		t.Errorf("the system disk was not shown as excluded:\n%s", out)
	}
}

func TestConfirmationMustNameTheDrive(t *testing.T) {
	m := newMachine(t)
	node := m.drive("sdb", 64<<20, true, "stick")
	iso := bootableISO(t, "test.iso", 1<<20)

	before := driveContents(t, node, 4096)
	m.answers("yes") // not the drive's name
	_, err := m.run("write", iso, "sdb")
	if err == nil {
		t.Fatal("a write went ahead on \"yes\"")
	}
	if !bytes.Equal(driveContents(t, node, 4096), before) {
		t.Error("the drive was written to without confirmation")
	}
}

func TestWriteWithoutATerminalWillNotGuess(t *testing.T) {
	m := newMachine(t)
	m.drive("sdb", 64<<20, true, "stick")
	iso := bootableISO(t, "test.iso", 1<<20)

	_, err := m.run("write", iso, "sdb")
	if err == nil || !strings.Contains(err.Error(), "-f") {
		t.Fatalf("error = %v, want a refusal that names -f", err)
	}
}

func TestEraseLeavesAnEmptyNamedVolume(t *testing.T) {
	m := newMachine(t)
	node := m.drive("sdb", 512<<20, true, "stick")

	// Something that looks like an installation image is on it already.
	f, err := os.OpenFile(node, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteAt(bytes.Repeat([]byte{0xA5}, 8<<20), 0)
	f.Close()

	out, err := m.run("erase", "sdb", "-f", "-name", "work")
	if err != nil {
		t.Fatalf("erase failed: %v\n%s", err, out)
	}

	head := driveContents(t, node, 512)
	if head[510] != 0x55 || head[511] != 0xAA {
		t.Error("no partition table was left behind")
	}
	if head[450] != 0x0C {
		t.Errorf("partition type = %#x, want FAT32", head[450])
	}
	part := driveContents(t, node, (2048+1)*512)[2048*512:]
	if string(part[82:90]) != "FAT32   " {
		t.Errorf("no FAT32 volume at the partition: %q", part[82:90])
	}
	if string(part[71:82]) != "WORK       " {
		t.Errorf("volume name = %q, want WORK", part[71:82])
	}
	if !strings.Contains(out, "FAT32") {
		t.Errorf("output does not say what was made:\n%s", out)
	}
}

func TestEraseRawLeavesNoFilesystem(t *testing.T) {
	m := newMachine(t)
	node := m.drive("sdb", 512<<20, true, "stick")
	if _, err := m.run("erase", "sdb", "-f", "-raw"); err != nil {
		t.Fatal(err)
	}
	if got := imaging.Probe(node); !got.Empty() {
		t.Errorf("a raw erase left %v behind", got)
	}
}

func TestABadVolumeNameStopsBeforeTheDriveIsTouched(t *testing.T) {
	m := newMachine(t)
	node := m.drive("sdb", 512<<20, true, "stick")
	f, _ := os.OpenFile(node, os.O_RDWR, 0)
	f.WriteAt(bytes.Repeat([]byte{0x5A}, 1<<20), 0)
	f.Close()
	before := driveContents(t, node, 4096)

	_, err := m.run("erase", "sdb", "-f", "-name", "far too long a name")
	if err == nil {
		t.Fatal("an impossible volume name was accepted")
	}
	if !bytes.Equal(driveContents(t, node, 4096), before) {
		t.Error("the drive was erased before the name was checked")
	}
}

func TestFormatMakesAVolumeTheSystemCanRead(t *testing.T) {
	m := newMachine(t)
	node := m.drive("sdb", 512<<20, true, "stick")

	if _, err := m.run("format", "sdb", "-name", "BACKUP", "-f"); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(node)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	// Read the partition the way the kernel would, then identify it.
	part := make([]byte, 0x11000)
	if _, err := f.ReadAt(part, 2048*512); err != nil {
		t.Fatal(err)
	}
	if got := imaging.Probe(node); got.Type != "partitioned" {
		t.Errorf("the whole drive probes as %v", got)
	}
	if string(part[82:90]) != "FAT32   " || string(part[71:82]) != "BACKUP     " {
		t.Errorf("volume = %q labelled %q", part[82:90], part[71:82])
	}
}

func TestVerifySaysWhenADriveHasLostTheImage(t *testing.T) {
	m := newMachine(t)
	node := m.drive("sdb", 64<<20, true, "stick")
	iso := bootableISO(t, "test.iso", 2<<20)

	if _, err := m.run("write", iso, "sdb", "-f", "-no-verify"); err != nil {
		t.Fatal(err)
	}
	if _, err := m.run("verify", iso, "sdb"); err != nil {
		t.Fatalf("a good drive failed verification: %v", err)
	}

	f, _ := os.OpenFile(node, os.O_RDWR, 0)
	f.WriteAt([]byte{0xDE, 0xAD}, 1<<20)
	f.Close()

	_, err := m.run("verify", iso, "sdb")
	if err == nil {
		t.Fatal("a corrupted drive passed verification")
	}
	if !strings.Contains(err.Error(), "drive holds") {
		t.Errorf("unhelpful error: %v", err)
	}
}

func TestWriteWarnsBeforeWritingAWindowsImage(t *testing.T) {
	m := newMachine(t)
	m.drive("sdb", 64<<20, true, "stick")

	// An ISO with the Microsoft layout and no partition table.
	iso := bootableISO(t, "windows.iso", 2<<20)
	raw, _ := os.ReadFile(iso)
	for i := 0; i < 512; i++ {
		raw[i] = 0
	}
	pvd := raw[16*2048:]
	binary.LittleEndian.PutUint32(pvd[158:], 20)  // root directory extent
	binary.LittleEndian.PutUint32(pvd[166:], 148) // and its length
	dir := raw[20*2048:]
	off := 0
	for _, name := range []string{"\x00", "\x01", "BOOTMGR", "SOURCES"} {
		length := 33 + len(name)
		if length%2 == 1 {
			length++
		}
		dir[off] = byte(length)
		dir[off+32] = byte(len(name))
		copy(dir[off+33:], name)
		off += length
	}
	os.WriteFile(iso, raw, 0o644)

	out, err := m.run("write", iso, "sdb", "-f")
	if err != nil {
		t.Fatalf("write failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Windows installation image") {
		t.Errorf("no warning about a Windows image:\n%s", out)
	}
	if !strings.Contains(out, "oliver format") {
		t.Errorf("the warning does not say what to do instead:\n%s", out)
	}
}

func TestDrivesListingShowsWhatIsOnEachDrive(t *testing.T) {
	m := newMachine(t)
	m.drive("sdb", 512<<20, true, "Cruzer")
	m.drive("sdc", 512<<20, true, "Kingston")
	if _, err := m.run("format", "sdb", "-name", "TOOLS", "-f"); err != nil {
		t.Fatal(err)
	}

	out, err := m.run("drives")
	if err != nil {
		t.Fatal(err)
	}
	for _, phrase := range []string{"sdb", "sdc", "removable", "Cruzer", "oliver write"} {
		if !strings.Contains(out, phrase) {
			t.Errorf("listing never mentions %q:\n%s", phrase, out)
		}
	}
}

// sectorSize sets the logical block size a drive reports.
func (m *machine) sectorSize(name string, n int) {
	m.t.Helper()
	dir := filepath.Join(m.root, "sys", "block", name, "queue")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "logical_block_size"), []byte(itoa(int64(n))+"\n"), 0o644); err != nil {
		m.t.Fatal(err)
	}
}

func TestFormatRefusesADriveWithUnusualSectors(t *testing.T) {
	m := newMachine(t)
	node := m.drive("sdb", 512<<20, true, "4Kn drive")
	m.sectorSize("sdb", 4096)
	before := driveContents(t, node, 4096)

	_, err := m.run("format", "sdb", "-f", "-name", "DATA")
	if err == nil {
		t.Fatal("formatted a drive whose sectors are not 512 bytes")
	}
	if !strings.Contains(err.Error(), "4096-byte sectors") || !strings.Contains(err.Error(), "mkfs.vfat") {
		t.Errorf("refusal does not explain the problem or the way round it: %v", err)
	}
	if !bytes.Equal(driveContents(t, node, 4096), before) {
		t.Error("the drive was written to despite the refusal")
	}

	// Erasing it is still allowed; only laying down the filesystem is not.
	if _, err := m.run("erase", "sdb", "-f", "-raw"); err != nil {
		t.Errorf("a raw erase should still work: %v", err)
	}
}

func TestAMountedDriveIsRefusedBeforeBeingOfferedForDestruction(t *testing.T) {
	m := newMachine(t)
	m.drive("sdb", 64<<20, true, "stick")
	m.mount("sdb", "/media/user/DATA")
	iso := bootableISO(t, "test.iso", 1<<20)

	m.answers("sdb") // would confirm, if it were ever asked
	out, err := m.run("write", iso, "sdb")
	if err == nil {
		t.Fatal("wrote to a mounted drive")
	}
	if strings.Contains(out, "will be destroyed") {
		t.Errorf("the drive was offered for destruction before the refusal:\n%s", out)
	}
}

func TestVerifyFailsLoudlyEvenWhenAskedForJSON(t *testing.T) {
	m := newMachine(t)
	node := m.drive("sdb", 64<<20, true, "stick")
	iso := bootableISO(t, "test.iso", 1<<20)

	if _, err := m.run("write", iso, "sdb", "-f", "-no-verify"); err != nil {
		t.Fatal(err)
	}
	f, _ := os.OpenFile(node, os.O_RDWR, 0)
	f.WriteAt([]byte{0x99}, 4096)
	f.Close()

	out, err := m.run("verify", iso, "sdb", "-json")
	if err == nil {
		t.Fatal("a corrupt drive was reported as a success because -json was given")
	}
	if !strings.Contains(out, `"verified": false`) {
		t.Errorf("the JSON does not record the failure:\n%s", out)
	}
}

func TestJSONOutputIsNotMixedWithTheHumanReadableReport(t *testing.T) {
	m := newMachine(t)
	m.drive("sdb", 64<<20, true, "stick")
	iso := bootableISO(t, "test.iso", 1<<20)

	if _, err := m.run("write", iso, "sdb", "-f", "-json"); err != nil {
		t.Fatal(err)
	}
	// Only stdout is the answer; the image summary and progress belong on
	// stderr, where a pipe does not see them.
	var res struct {
		Bytes    int64  `json:"bytes_written"`
		Verified bool   `json:"verified"`
		Source   string `json:"source_sha256"`
	}
	if err := json.Unmarshal(m.out.Bytes(), &res); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, m.out.String())
	}
	if res.Bytes != 1<<20 || !res.Verified || res.Source == "" {
		t.Errorf("JSON result = %+v", res)
	}
	if !strings.Contains(m.err.String(), "ISO 9660") {
		t.Errorf("the human-readable summary went missing from stderr:\n%s", m.err.String())
	}
}
