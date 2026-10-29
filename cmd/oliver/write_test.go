package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/robertpjenkins-bot/oliver/internal/device"
	"github.com/robertpjenkins-bot/oliver/internal/flash"
	"github.com/robertpjenkins-bot/oliver/internal/ui"
)

// drives gives the synthetic system a set of disks: the internal one the
// system runs from, and a USB stick with one partition. Device nodes are
// ordinary files, so what is written to them can be read back.
func (s *system) drives() {
	s.t.Helper()
	s.disk("nvme0n1", 1000215216, false, "SSD 980")
	s.part("nvme0n1", "nvme0n1p1", 1, 1000000)
	s.part("nvme0n1", "nvme0n1p2", 2, 900000000)
	s.disk("sdb", 60088320, true, "SanDisk Ultra")
	s.part("sdb", "sdb1", 1, 60000000)
	s.mounted(s.dev("nvme0n1p1")+" /boot/efi", s.dev("nvme0n1p2")+" /")

	orig := newScanner
	s.t.Cleanup(func() { newScanner = orig })
	newScanner = func() *device.Scanner {
		return &device.Scanner{
			SysBlock: s.path("sys", "block"),
			Dev:      s.path("dev"),
			Mounts:   s.path("proc", "mounts"),
			Access:   func(string) (bool, bool) { return true, true },
		}
	}
}

func (s *system) disk(name string, sectors int64, removable bool, model string) {
	rem := "0"
	if removable {
		rem = "1"
	}
	s.write(strconv.FormatInt(sectors, 10), "sys", "block", name, "size")
	s.write(rem, "sys", "block", name, "removable")
	s.write(model, "sys", "block", name, "device", "model")
	s.write("", "dev", name)
}

func (s *system) part(disk, name string, number int, sectors int64) {
	s.write(strconv.Itoa(number), "sys", "block", disk, name, "partition")
	s.write(strconv.FormatInt(sectors, 10), "sys", "block", disk, name, "size")
	s.write("", "dev", name)
}

func (s *system) dev(name string) string { return s.path("dev", name) }

// mounted replaces the mount table with "source target" pairs.
func (s *system) mounted(pairs ...string) {
	var b strings.Builder
	for _, p := range pairs {
		b.WriteString(p + " ext4 rw 0 0\n")
	}
	s.write(b.String(), "proc", "mounts")
}

// image writes a hybrid ISO of the given size into the user's downloads.
func (s *system) image(name string, size int) (string, []byte) {
	s.t.Helper()
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i*7 + i/4096)
	}
	data[446+4], data[510], data[511] = 0x17, 0x55, 0xAA
	copy(data[0x8001:], "CD001")
	p := s.path("home", "user", "Downloads", name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		s.t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		s.t.Fatal(err)
	}
	return p, data
}

func (s *system) contents(name string) []byte {
	s.t.Helper()
	data, err := os.ReadFile(s.dev(name))
	if err != nil {
		s.t.Fatal(err)
	}
	return data
}

func TestWriteCopiesTheImageOntoTheChosenDrive(t *testing.T) {
	s := newSystem(t)
	s.drives()
	iso, data := s.image("fedora.iso", 5<<20+300)

	out, err := s.run("write", iso, "-to", "/dev/sdb", "-y")
	if err != nil {
		t.Fatalf("write: %v\n%s", err, out)
	}
	if !bytes.Equal(s.contents("sdb"), data) {
		t.Error("the drive does not hold the image")
	}
	if len(s.contents("sdb1")) != 0 {
		t.Error("writing the drive should not go through the partition's node")
	}
	for _, want := range []string{"fedora.iso", "SanDisk Ultra", "Everything on " + s.dev("sdb") + " will be erased"} {
		if !strings.Contains(out, want) {
			t.Errorf("the plan should mention %q:\n%s", want, out)
		}
	}
}

func TestWriteIntoASinglePartition(t *testing.T) {
	s := newSystem(t)
	s.drives()
	iso, data := s.image("backup.img", 70000)

	out, err := s.run("write", iso, "-to", "sdb1", "-y")
	if err != nil {
		t.Fatalf("write: %v\n%s", err, out)
	}
	if !bytes.Equal(s.contents("sdb1"), data) {
		t.Error("the partition does not hold the image")
	}
	if len(s.contents("sdb")) != 0 {
		t.Error("writing a partition must not touch the rest of the drive")
	}
	// A bootable image in a partition will not boot: say so.
	if !strings.Contains(out, "will not boot from here") {
		t.Errorf("expected a warning about booting from a partition:\n%s", out)
	}
}

func TestWriteRefusesWhatWouldDestroySomething(t *testing.T) {
	s := newSystem(t)
	s.drives()
	iso, _ := s.image("fedora.iso", 70000)

	cases := map[string]string{
		"nvme0n1":   "running system",
		"nvme0n1p2": "running system (/)",
		"sdz":       "no drive or partition",
	}
	for target, want := range cases {
		_, err := s.run("write", iso, "-to", target, "-y")
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("writing to %s: got %v, want an error mentioning %q", target, err, want)
		}
	}

	// A partition too small for the image.
	s.part("sdb", "sdb2", 2, 10)
	if _, err := s.run("write", iso, "-to", "sdb2", "-y"); err == nil || !strings.Contains(err.Error(), "image needs") {
		t.Errorf("expected a too-small partition to be refused, got %v", err)
	}
	if len(s.contents("sdb2")) != 0 {
		t.Error("a refused write must leave the target untouched")
	}
}

func TestWriteRefusesTheDriveTheImageIsStoredOn(t *testing.T) {
	s := newSystem(t)
	s.drives()
	media := s.mkdir("media", "stick")
	if real, err := filepath.EvalSymlinks(media); err == nil {
		media = real
	}
	s.mounted(s.dev("nvme0n1p2")+" /", s.dev("sdb1")+" "+media)
	iso := filepath.Join(media, "linux.iso")
	if err := os.WriteFile(iso, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{"sdb", "sdb1"} {
		_, err := s.run("write", iso, "-to", target, "-y")
		if err == nil || !strings.Contains(err.Error(), "image file is stored on it") {
			t.Errorf("writing to %s: got %v, want a refusal", target, err)
		}
	}
}

func TestWriteUnmountsTheTargetFirst(t *testing.T) {
	s := newSystem(t)
	s.drives()
	s.mounted(s.dev("nvme0n1p2")+" /", s.dev("sdb1")+" /run/media/user/STICK")
	iso, data := s.image("fedora.iso", 70000)

	var unmounted []string
	orig := unmountAsUser
	t.Cleanup(func() { unmountAsUser = orig })
	unmountAsUser = func(m device.Mount) error {
		unmounted = append(unmounted, m.Target)
		s.mounted(s.dev("nvme0n1p2") + " /")
		return nil
	}

	out, err := s.run("write", iso, "-to", "sdb", "-y")
	if err != nil {
		t.Fatalf("write: %v\n%s", err, out)
	}
	if len(unmounted) != 1 || unmounted[0] != "/run/media/user/STICK" {
		t.Errorf("unmounted %v, want the stick's filesystem", unmounted)
	}
	if !strings.Contains(out, "These are unmounted before") {
		t.Errorf("the plan should say the filesystem will be unmounted:\n%s", out)
	}
	if !bytes.Equal(s.contents("sdb"), data) {
		t.Error("the drive does not hold the image")
	}
}

func TestWriteListShowsTheChoicesAndWhyOthersAreNotOffered(t *testing.T) {
	s := newSystem(t)
	s.drives()
	iso, _ := s.image("fedora.iso", 70000)

	out, err := s.run("write", "-list", iso)
	if err != nil {
		t.Fatalf("write -list: %v", err)
	}
	for _, want := range []string{s.dev("sdb"), s.dev("sdb1"), "removable drive", "not offered, it holds the running system"} {
		if !strings.Contains(out, want) {
			t.Errorf("listing missing %q:\n%s", want, out)
		}
	}
	// The stick comes before the internal disk.
	if strings.Index(out, s.dev("sdb")) > strings.Index(out, s.dev("nvme0n1")) {
		t.Errorf("plugged-in drives should be listed first:\n%s", out)
	}
}

func TestAskTargetReadsTheNumberTyped(t *testing.T) {
	s := newSystem(t)
	s.drives()
	iso, _ := s.image("fedora.iso", 70000)
	img, err := flash.Inspect(iso)
	if err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	a := &app{ui: &ui.UI{Out: &out, Err: &out, In: strings.NewReader("7\nsdb1\n"), Interactive: true}}
	target, err := a.askTarget(img, newScanner())
	if err != nil {
		t.Fatal(err)
	}
	if target.Path != s.dev("sdb1") {
		t.Errorf("chose %q, want sdb1 (after rejecting an out-of-range number)", target.Path)
	}
	if !strings.Contains(out.String(), "Type one of the numbers shown") {
		t.Errorf("an invalid answer should be explained:\n%s", out.String())
	}

	a.ui.In = strings.NewReader("q\n")
	if target, err := a.askTarget(img, newScanner()); err != nil || target.Path != "" {
		t.Errorf("q should cancel, got %+v, %v", target, err)
	}
}

func TestDroppedImageIsWrittenNotRun(t *testing.T) {
	s := newSystem(t)
	s.drives()
	iso, _ := s.image("fedora.iso", 70000)

	// Without a terminal or a dialog program there is nobody to ask which
	// drive to use, so the error must say how to name one.
	_, err := s.run(iso)
	if err == nil || !strings.Contains(err.Error(), "-to") {
		t.Errorf("expected an image on the command line to go to the writer, got %v", err)
	}
}

func TestWriteRefusesACompressedImageAndSaysHowToUnpackIt(t *testing.T) {
	s := newSystem(t)
	s.drives()
	p := s.write("\x1f\x8b\x08\x00 rest of a gzip stream", "home", "user", "Downloads", "pi.img.gz")
	_, err := s.run("write", p, "-to", "sdb", "-y")
	if err == nil || !strings.Contains(err.Error(), "gunzip -k") {
		t.Errorf("expected advice to unpack the image, got %v", err)
	}
}

func TestWriteHelperWritesFromItsInputAndReportsProgress(t *testing.T) {
	s := newSystem(t)
	s.drives()
	_, data := s.image("fedora.iso", 5<<20)

	var out bytes.Buffer
	if err := writeHelper(bytes.NewReader(data), &out, "/dev/sdb", int64(len(data)), true, false); err != nil {
		t.Fatalf("writeHelper: %v\n%s", err, out.String())
	}
	if !bytes.Equal(s.contents("sdb"), data) {
		t.Error("the drive does not hold the image")
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if lines[len(lines)-1] != "done" {
		t.Errorf("the helper should end by confirming: %q", lines)
	}
	var phases []flash.Phase
	for _, l := range lines[:len(lines)-1] {
		m, err := flash.ParseMessage(l)
		if err != nil || m.Progress == nil {
			t.Fatalf("unexpected line %q", l)
		}
		phases = append(phases, m.Progress.Phase)
	}
	if phases[0] != flash.Writing || phases[len(phases)-1] != flash.Verifying {
		t.Errorf("phases = %v", phases)
	}
}

func TestWriteHelperChecksTheTargetItself(t *testing.T) {
	s := newSystem(t)
	s.drives()

	// Running as root, the helper must not trust its caller.
	var out bytes.Buffer
	err := writeHelper(strings.NewReader("x"), &out, "/dev/nvme0n1", 1, true, false)
	var ee *exitError
	if !errors.As(err, &ee) {
		t.Fatalf("expected a quiet exit status, got %v", err)
	}
	m, perr := flash.ParseMessage(out.String())
	if perr != nil || !strings.Contains(m.Error, "running system") {
		t.Errorf("expected the refusal to be reported over the protocol, got %q", out.String())
	}
	if len(s.contents("nvme0n1")) != 0 {
		t.Error("the system disk was written to")
	}
}

// fakeHelper writes a shell script standing in for the privileged helper.
func fakeHelper(t *testing.T, name, body string) *exec.Cmd {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body), 0o755); err != nil {
		t.Fatal(err)
	}
	return exec.Command(p)
}

func TestRunHelperRelaysProgressAndOutcome(t *testing.T) {
	var reports []flash.Progress
	report := func(p flash.Progress) { reports = append(reports, p) }

	// The image must reach the helper's input; it echoes the byte count.
	cmd := fakeHelper(t, "helper", `n=$(wc -c | tr -d ' ')
echo "progress write $n $n"
echo "progress verify $n $n"
echo done
`)
	if err := runHelper(context.Background(), cmd, strings.NewReader("12345"), report); err != nil {
		t.Fatalf("runHelper: %v", err)
	}
	if len(reports) != 2 || reports[0].Done != 5 || reports[1].Phase != flash.Verifying {
		t.Errorf("reports = %+v", reports)
	}

	cmd = fakeHelper(t, "helper", `cat >/dev/null; echo 'error "the drive was unplugged"'; exit 1`)
	if err := runHelper(context.Background(), cmd, strings.NewReader("x"), report); err == nil || err.Error() != "the drive was unplugged" {
		t.Errorf("the helper's own explanation should be passed on, got %v", err)
	}

	cmd = fakeHelper(t, "helper", `exit 0`)
	if err := runHelper(context.Background(), cmd, strings.NewReader(""), report); err == nil {
		t.Error("a helper that never confirms must not count as success")
	}

	// pkexec's status when the password dialog is dismissed.
	cmd = fakeHelper(t, "pkexec", `echo "Not authorized" >&2; exit 126`)
	if err := runHelper(context.Background(), cmd, strings.NewReader(""), report); err == nil || !strings.Contains(err.Error(), "permission was not given") {
		t.Errorf("a dismissed password prompt should be explained, got %v", err)
	}
}

func TestRunHelperStopsWhenCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// A helper that reports and then waits for more input than it will get.
	cmd := fakeHelper(t, "helper", `echo "progress write 1 10"; cat >/dev/null; exit 1`)
	err := runHelper(ctx, cmd, &endlessReader{}, func(flash.Progress) { cancel() })
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected cancellation, got %v", err)
	}
}

type endlessReader struct{}

func (endlessReader) Read(p []byte) (int, error) { return len(p), nil }

func TestDesktopInstallAddsTheDiskImageWriter(t *testing.T) {
	s := newSystem(t)
	if _, err := s.run("desktop", "install"); err != nil {
		t.Fatalf("desktop install: %v", err)
	}
	data, err := os.ReadFile(writerFilePath())
	if err != nil {
		t.Fatalf("writer entry not written: %v", err)
	}
	entry := string(data)
	for _, want := range []string{"Name=Disk Image Writer", "write --gui %f", "application/x-cd-image", "application/x-raw-disk-image"} {
		if !strings.Contains(entry, want) {
			t.Errorf("writer entry missing %q:\n%s", want, entry)
		}
	}
	if b, broken := inspectEntry(writerFilePath()); broken {
		t.Errorf("a fresh writer entry should launch: %+v", b)
	}

	if _, err := s.run("desktop", "uninstall"); err != nil {
		t.Fatalf("desktop uninstall: %v", err)
	}
	if _, err := os.Stat(writerFilePath()); !os.IsNotExist(err) {
		t.Error("writer entry survived uninstall")
	}
}

func TestLooksLikeDiskImage(t *testing.T) {
	for _, name := range []string{"fedora.iso", "Win11.ISO", "pi.img", "disk.raw"} {
		if !looksLikeDiskImage(name) {
			t.Errorf("%s should be a disk image", name)
		}
	}
	for _, name := range []string{"setup.exe", "write", "pi.img.xz", "notes.txt"} {
		if looksLikeDiskImage(name) {
			t.Errorf("%s should not be a disk image", name)
		}
	}
}
