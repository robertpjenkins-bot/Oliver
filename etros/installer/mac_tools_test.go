//go:build linux

package main

// Etros Mac Installer's real work (macos_work.go) run from start to end,
// with stand-ins for macOS's tools (testdata/fakemac: diskutil, osascript,
// networksetup...) keeping a pretend 12-inch MacBook's drive, and stand-ins
// for Batocera for PCs and Arch Linux in Downloads.

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

type fakeMacPart struct {
	Num     int    `json:"num"`
	Content string `json:"content"`
	Name    string `json:"name"`
	Start   int64  `json:"start"`
	Size    int64  `json:"size"`
	Mount   string `json:"mount,omitempty"`
}

type fakeMacDisk struct {
	ID       string        `json:"id"`
	Name     string        `json:"name"`
	Size     int64         `json:"size"`
	Internal bool          `json:"internal"`
	Parts    []fakeMacPart `json:"parts"`
}

type fakeMacState struct {
	Disks          []fakeMacDisk       `json:"disks"`
	BootStore      string              `json:"boot_store"`
	APFSFree       int64               `json:"apfs_free"`
	NeedAdmin      bool                `json:"need_admin"`
	CancelPassword bool                `json:"cancel_password"`
	Volumes        string              `json:"volumes"`
	Wifi           string              `json:"wifi"`
	Busy           []string            `json:"busy"`
	Stuck          map[string]int      `json:"stuck,omitempty"`
	Procs          []string            `json:"procs,omitempty"`
	Open           map[string][]string `json:"open,omitempty"`
}

// macBook is the 12-inch MacBook: EFI, macOS (APFS) and an ExFAT partition
// made for Etros in Disk Utility; and a USB stick.
func macBook(volumes string) fakeMacState {
	const g = int64(1) << 30
	return fakeMacState{
		Disks: []fakeMacDisk{
			{ID: "disk0", Name: "APPLE SSD SM0256G", Size: 251 * g, Internal: true, Parts: []fakeMacPart{
				{Num: 1, Content: "EFI", Name: "EFI", Start: 20480, Size: 200 << 20},
				{Num: 2, Content: "Apple_APFS", Start: 210 << 20, Size: 180 * g},
				{Num: 3, Content: "Microsoft Basic Data", Name: "ETROS", Start: 210<<20 + 180*g, Size: 70 * g},
			}},
			{ID: "disk2", Name: "SanDisk Ultra", Size: 64 * g, Parts: []fakeMacPart{
				{Num: 1, Content: "Microsoft Basic Data", Name: "STICK", Start: 1 << 20, Size: 64*g - 2<<20},
			}},
		},
		BootStore: "disk0s2", APFSFree: 90 * g, NeedAdmin: true, Volumes: volumes, Wifi: "Jenkins Wi-Fi",
	}
}

// fakeMacEnv sets up the stand-ins, Downloads with Batocera for PCs and
// Arch Linux in it, and the state; it returns the state file.
func fakeMacEnv(t *testing.T, st fakeMacState) string {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("needs python3")
	}
	if os.Geteuid() == 0 {
		t.Skip("run as an ordinary user, as on a Mac (root never gets macOS's password window)")
	}
	tools, _ := filepath.Abs("testdata/fakemac")
	t.Setenv("PATH", tools+string(os.PathListSeparator)+os.Getenv("PATH"))
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	if st.Volumes == "" {
		st.Volumes = filepath.Join(home, "Volumes")
	}
	os.MkdirAll(filepath.Join(home, "Downloads"), 0o755)
	writeFakePCBatocera(t, filepath.Join(home, "Downloads", "batocera-x86_64-43-20261001-boot.tar.xz"))
	writeFakeBootstrap(t, filepath.Join(home, "Downloads", "archlinux-bootstrap-x86_64.tar.zst"))
	t.Setenv("ETROS_TEST_SMALL_FILES", "1")
	t.Setenv("ETROS_TEST_VOLUMES", filepath.Join(home, "NoVolumes"))
	state := filepath.Join(home, "fakemac.json")
	b, _ := json.Marshal(st)
	os.WriteFile(state, b, 0o644)
	t.Setenv("FAKEMAC_STATE", state)
	return state
}

func readFakeMac(t *testing.T, state string) fakeMacState {
	var st fakeMacState
	b, _ := os.ReadFile(state)
	if err := json.Unmarshal(b, &st); err != nil {
		t.Fatal(err)
	}
	return st
}

func writeTar(t *testing.T, w *tar.Writer, files map[string][]byte) {
	for name, data := range files {
		if err := w.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		w.Write(data)
	}
	w.Close()
}

func writeFakePCBatocera(t *testing.T, path string) {
	var cpio bytes.Buffer
	gz := gzip.NewWriter(&cpio)
	var init cpioEntry
	init.name, init.data = "init", []byte(batoceraInit(t))
	init.hdr[1], init.hdr[4] = 0o100755, 1
	gz.Write(writeCPIO([]cpioEntry{init}))
	gz.Close()
	f, _ := os.Create(path)
	defer f.Close()
	x, _ := xz.NewWriter(f)
	writeTar(t, tar.NewWriter(x), map[string][]byte{
		"./boot/initrd.gz": cpio.Bytes(), "./boot/linux": []byte("kernel"), "./boot/batocera.update": bytes.Repeat([]byte("B"), 1<<20),
		"./EFI/BOOT/BOOTX64.EFI": []byte("shim"), "./batocera-boot.conf": []byte("sharedevice=INTERNAL\nautoresize=true\n"),
	})
	x.Close()
}

func writeFakeBootstrap(t *testing.T, path string) {
	f, _ := os.Create(path)
	defer f.Close()
	z, _ := zstd.NewWriter(f)
	writeTar(t, tar.NewWriter(z), map[string][]byte{"root.x86_64/usr/bin/pacman": []byte("#!/bin/sh\n"), "root.x86_64/etc/pacman.conf": []byte("[options]\n")})
	z.Close()
}

type logRep struct{ lines []string }

func (r *logRep) Stage(s string)           { r.lines = append(r.lines, "STAGE "+s) }
func (r *logRep) Log(s string)             { r.lines = append(r.lines, s) }
func (r *logRep) Progress(done, all int64) {}

func pickTarget(t *testing.T, id string) macTarget {
	ts, err := darwinMac{}.targets()
	if err != nil {
		t.Fatal(err)
	}
	for _, x := range ts {
		if x.part.id == id || (x.kind == "whole" && x.disk.id == id) {
			if x.why != "" {
				t.Fatalf("%s can't be used: %s", id, x.why)
			}
			return x
		}
	}
	t.Fatalf("no %s among %+v", id, ts)
	return macTarget{}
}

// checkEtrosOn checks the BATOCERA partition's files after an install.
func checkEtrosOn(t *testing.T, st fakeMacState, disk string, wantBoot, wantShare int, ollie bool) {
	t.Helper()
	var boot, share *fakeMacPart
	for i := range st.Disks {
		if st.Disks[i].ID != disk {
			continue
		}
		for j := range st.Disks[i].Parts {
			p := &st.Disks[i].Parts[j]
			switch p.Name {
			case "BATOCERA":
				boot = p
			case "SHARE":
				share = p
			}
		}
	}
	if boot == nil || share == nil {
		t.Fatalf("no BATOCERA and SHARE on %s: %+v", disk, st.Disks)
	}
	if boot.Num != wantBoot || share.Num != wantShare || boot.Size != 8<<30 || share.Start < boot.Start+boot.Size {
		t.Errorf("partitions: BATOCERA %+v SHARE %+v", boot, share)
	}
	dir := boot.Mount
	for _, f := range []string{"boot/linux", "boot/batocera.update", "EFI/BOOT/BOOTX64.EFI", "etros/etros.squashfs", "etros/intel-mac.txt", "etros/format-share"} {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Error(err)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "boot/initrd.gz")); !bytes.Contains(gunzipT(t, b), []byte("etros-hook.sh")) {
		t.Error("initrd.gz hasn't the Etros hook")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "batocera-boot.conf")); strings.Contains(string(b), "\nautoresize=true") {
		t.Error("autoresize still on")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "etros/format-share")); strings.TrimSpace(string(b)) != disk+"s"+strconv.Itoa(wantShare) {
		t.Errorf("format-share says %q", b)
	}
	_, errB := os.Stat(filepath.Join(dir, "etros/archlinux-bootstrap-x86_64.tar.gz"))
	wifi, errW := os.ReadFile(filepath.Join(dir, "etros/wifi.txt"))
	if ollie && (errB != nil || errW != nil || string(wifi) != "Jenkins Wi-Fi\nhunter2\n") {
		t.Errorf("Ollie OS's Arch Linux and Wi-Fi: %v %v %q", errB, errW, wifi)
	}
	if !ollie && (errB == nil || errW == nil) {
		t.Error("Arch Linux or Wi-Fi put on without room for Ollie OS")
	}
}

func gunzipT(t *testing.T, b []byte) []byte {
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	out.ReadFrom(zr)
	return out.Bytes()
}

// The ExFAT partition made in Disk Utility, with macOS wanting the Mac's
// password for diskutil (the password window, osascript, says yes).
func TestMacInstallOnPartition(t *testing.T) {
	state := fakeMacEnv(t, macBook(""))
	x := pickTarget(t, "disk0s3")
	rep := &logRep{}
	if err := (darwinMac{}).install(x, 70<<30, "Jenkins Wi-Fi", "hunter2", rep); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(rep.lines, "\n"))
	}
	st := readFakeMac(t, state)
	checkEtrosOn(t, st, "disk0", 3, 4, true)
	calls, _ := os.ReadFile(state + ".log")
	if n := strings.Count(string(calls), "osascript"); n != 1 {
		t.Errorf("asked for the password %d times", n)
	}
	for _, want := range []string{"diskutil eraseVolume free none disk0s3", "osascript -e do shell script", "diskutil addPartition disk0s2 MS-DOS FAT32 BATOCERA 8G", "diskutil addPartition disk0s3 ExFAT SHARE"} {
		if !strings.Contains(string(calls), want) {
			t.Errorf("no %q in:\n%s", want, calls)
		}
	}
	// macOS's partitions are as they were.
	if p := st.Disks[0].Parts; p[0].Content != "EFI" || p[1].Content != "Apple_APFS" || p[1].Size != 180<<30 {
		t.Errorf("macOS touched: %+v", p[:2])
	}
}

func TestMacInstallCancelledPassword(t *testing.T) {
	st := macBook("")
	st.CancelPassword = true
	fakeMacEnv(t, st)
	err := (darwinMac{}).install(pickTarget(t, "disk0s3"), 70<<30, "", "", &logRep{})
	if err == nil || !strings.Contains(err.Error(), "Cancel") {
		t.Fatalf("got %v", err)
	}
}

func TestMacInstallInMacOSRoom(t *testing.T) {
	st := macBook("")
	st.Disks[0].Parts = st.Disks[0].Parts[:2] // no partition for Etros: room from macOS
	st.Disks[0].Parts[1].Size = 250 << 30
	state := fakeMacEnv(t, st)
	x := pickTarget(t, "disk0s2")
	if x.kind != "room" {
		t.Fatalf("macOS offered as %q", x.kind)
	}
	if err := (darwinMac{}).install(x, 40<<30, "Jenkins Wi-Fi", "hunter2", &logRep{}); err != nil {
		t.Fatal(err)
	}
	after := readFakeMac(t, state)
	checkEtrosOn(t, after, "disk0", 3, 4, true)
	if after.Disks[0].Parts[1].Size != 210<<30 {
		t.Errorf("macOS is %d bytes, not 40 GB less", after.Disks[0].Parts[1].Size)
	}
}

func TestMacInstallOnUSBDrive(t *testing.T) {
	state := fakeMacEnv(t, macBook(""))
	x := pickTarget(t, "disk2")
	if err := (darwinMac{}).install(x, 64<<30, "Jenkins Wi-Fi", "hunter2", &logRep{}); err != nil {
		t.Fatal(err)
	}
	checkEtrosOn(t, readFakeMac(t, state), "disk2", 2, 3, true)
}

// A small partition: Etros without Ollie OS (nothing for it put on).
func TestMacInstallNoRoomForOllie(t *testing.T) {
	st := macBook("")
	st.Disks[0].Parts[2].Size = 20 << 30
	state := fakeMacEnv(t, st)
	if err := (darwinMac{}).install(pickTarget(t, "disk0s3"), 20<<30, "Jenkins Wi-Fi", "hunter2", &logRep{}); err != nil {
		t.Fatal(err)
	}
	checkEtrosOn(t, readFakeMac(t, state), "disk0", 3, 4, false)
}

// Partition numbers with a gap: said in words, before anything's copied.
func TestMacInstallNumberGap(t *testing.T) {
	st := macBook("")
	st.Disks[0].Parts[2].Size = 40 << 30
	st.Disks[0].Parts = append(st.Disks[0].Parts, fakeMacPart{Num: 4, Content: "Microsoft Basic Data", Name: "OTHER", Start: 210<<20 + 220<<30, Size: 30 << 30})
	fakeMacEnv(t, st)
	err := (darwinMac{}).install(pickTarget(t, "disk0s3"), 40<<30, "", "", &logRep{})
	if err == nil || !strings.Contains(err.Error(), "very next number") {
		t.Fatalf("got %v", err)
	}
}

// Nothing in Downloads and no internet: said in words, and the drive is
// untouched (the downloads come before any change).
func TestMacInstallNoInternet(t *testing.T) {
	state := fakeMacEnv(t, macBook(""))
	os.RemoveAll(filepath.Join(os.Getenv("HOME"), "Downloads"))
	old := batoceraBase
	batoceraBase = "http://127.0.0.1:1"
	defer func() { batoceraBase = old }()
	err := (darwinMac{}).install(pickTarget(t, "disk0s3"), 70<<30, "", "", &logRep{})
	if err == nil || !strings.Contains(err.Error(), "couldn't download Batocera for PCs") || !strings.Contains(err.Error(), "internet") {
		t.Fatalf("got %v", err)
	}
	if calls, _ := os.ReadFile(state + ".log"); strings.Contains(string(calls), "eraseVolume") {
		t.Error("the drive was changed before the downloads worked")
	}
}

func TestMacChecks(t *testing.T) {
	fakeMacEnv(t, macBook(""))
	cs := darwinMac{}.checks()
	got := map[string]macCheck{}
	for _, c := range cs {
		got[strings.SplitN(c.Name, " ", 3)[0]+" "+strings.SplitN(c.Name, " ", 3)[1]] = c
	}
	if c := got["An Intel"]; !c.OK || !strings.Contains(c.Name, "M-5Y31") {
		t.Errorf("Intel: %+v", c)
	}
	if c := got["macOS 11.7.11"]; !c.OK {
		t.Errorf("version: %+v", cs)
	}
	if c := got["Batocera for"]; !c.OK || !strings.Contains(c.Name, "already on this Mac") || !strings.Contains(c.Why, "Downloads") {
		t.Errorf("Batocera: %+v", c)
	}
	if (darwinMac{}).wifiName() != "Jenkins Wi-Fi" {
		t.Error("Wi-Fi name")
	}
}

func TestRoomForDownloadsSaysWhy(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("ETROS_TEST_NO_ROOM_CHECK", "")
	if why := roomForDownloads(1 << 60); !strings.Contains(why, "free for the downloads") || !strings.Contains(why, "Trash") {
		t.Errorf("got %q", why)
	}
}

// The page with the real work and the stand-ins, for a browser test:
// ETROS_WEB_REAL=FILE (the address is written there), and FAKEMAC_CANCEL=1
// for macOS's password window to be cancelled.
func TestMacWebReal(t *testing.T) {
	out := os.Getenv("ETROS_WEB_REAL")
	if out == "" {
		t.Skip("set ETROS_WEB_REAL")
	}
	st := macBook("")
	st.CancelPassword = os.Getenv("FAKEMAC_CANCEL") == "1"
	if os.Getenv("FAKEMAC_READY") == "1" { // BATOCERA and SHARE made in Disk Utility
		const g = int64(1) << 30
		st.Disks[0].Parts = append(st.Disks[0].Parts[:2],
			fakeMacPart{Num: 3, Content: "Microsoft Basic Data", Name: "BATOCERA", Start: 210<<20 + 180*g, Size: 8 * g},
			fakeMacPart{Num: 4, Content: "Microsoft Basic Data", Name: "SHARE", Start: 210<<20 + 188*g, Size: 62 * g})
		st.Stuck = map[string]int{"disk0s3": 99}
	}
	state := fakeMacEnv(t, st)
	if os.Getenv("FAKEMAC_NO_INTERNET") == "1" {
		os.RemoveAll(filepath.Join(os.Getenv("HOME"), "Downloads"))
		old := batoceraBase
		batoceraBase = "http://127.0.0.1:1"
		defer func() { batoceraBase = old }()
	}
	serveMacWeb(darwinMac{}, func(url string) error {
		return os.WriteFile(out, []byte(url+"\n"+state), 0o644)
	})
}

// Arch Linux can't be had: Etros goes on anyway, without Ollie OS.
func TestMacInstallWithoutArch(t *testing.T) {
	state := fakeMacEnv(t, macBook(""))
	os.Remove(filepath.Join(os.Getenv("HOME"), "Downloads", "archlinux-bootstrap-x86_64.tar.zst"))
	old := archBootstrapURLs
	archBootstrapURLs = []string{"http://127.0.0.1:1/archlinux-bootstrap-x86_64.tar.zst"}
	defer func() { archBootstrapURLs = old }()
	rep := &logRep{}
	if err := (darwinMac{}).install(pickTarget(t, "disk0s3"), 70<<30, "Jenkins Wi-Fi", "hunter2", rep); err != nil {
		t.Fatal(err)
	}
	checkEtrosOn(t, readFakeMac(t, state), "disk0", 3, 4, false)
	if !strings.Contains(strings.Join(rep.lines, "\n"), "Ollie OS left out") {
		t.Error("didn't say Ollie OS was left out")
	}
}

// Batocera and Arch Linux only on a USB stick (put there on the Steam
// Machine), and no internet: found there, nothing downloaded.
func TestMacInstallFromUSBStick(t *testing.T) {
	state := fakeMacEnv(t, macBook(""))
	home := os.Getenv("HOME")
	vols := filepath.Join(home, "Volumes")
	t.Setenv("ETROS_TEST_VOLUMES", vols)
	os.MkdirAll(filepath.Join(vols, "STICK", "Etros"), 0o755)
	for _, f := range []string{"batocera-x86_64-43-20261001-boot.tar.xz", "archlinux-bootstrap-x86_64.tar.zst"} {
		if err := os.Rename(filepath.Join(home, "Downloads", f), filepath.Join(vols, "STICK", "Etros", f)); err != nil {
			t.Fatal(err)
		}
	}
	oldB, oldA := batoceraBase, archBootstrapURLs
	batoceraBase, archBootstrapURLs = "http://127.0.0.1:1", []string{"http://127.0.0.1:1/a.zst"}
	defer func() { batoceraBase, archBootstrapURLs = oldB, oldA }()
	rep := &logRep{}
	if err := (darwinMac{}).install(pickTarget(t, "disk0s3"), 70<<30, "Jenkins Wi-Fi", "hunter2", rep); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(rep.lines, "\n"))
	}
	checkEtrosOn(t, readFakeMac(t, state), "disk0", 3, 4, true)
	log := strings.Join(rep.lines, "\n")
	if !strings.Contains(log, filepath.Join("STICK", "Etros", "batocera-x86_64")) || strings.Contains(log, "Downloading") {
		t.Errorf("didn't use the stick's copies:\n%s", log)
	}
}

// The Steam Machine side: Batocera and Arch Linux onto a USB drive, named
// so the Mac's installer finds them there.
func TestMacUSBCopy(t *testing.T) {
	fakeMacEnv(t, macBook(""))
	usb := t.TempDir()
	dest := filepath.Join(usb, "Etros for Mac")
	if err := macUSBCopy(dest); err != nil {
		t.Fatal(err)
	}
	// On the Mac: only the USB drive has them.
	os.RemoveAll(filepath.Join(os.Getenv("HOME"), "Downloads"))
	t.Setenv("ETROS_TEST_VOLUMES", filepath.Dir(usb))
	if p := findLocal(localBatocera); p != filepath.Join(dest, "batocera-x86_64-boot.tar.xz") {
		t.Errorf("Batocera on the USB drive: found %q", p)
	}
	if p := findLocal(localArch); p != filepath.Join(dest, "archlinux-bootstrap-x86_64.tar.zst") {
		t.Errorf("Arch Linux on the USB drive: found %q", p)
	}
}

// Unpacking happens once (a second time uses it again), and a boot.tar.xz
// that isn't the PC one is said in words.
func TestUnpackPCBatocera(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache"))
	pc := filepath.Join(home, "boot.tar.xz")
	writeFakePCBatocera(t, pc)
	rep := &logRep{}
	dir, err := unpackPCBatocera(pc, rep)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "boot", "batocera.update")); err != nil {
		t.Error(err)
	}
	again := &logRep{}
	if d2, err := unpackPCBatocera(pc, again); err != nil || d2 != dir || !strings.Contains(strings.Join(again.lines, "\n"), "unpacked earlier") {
		t.Errorf("second time: %v %q %v", err, d2, again.lines)
	}
	pi := filepath.Join(home, "boot (1).tar.xz")
	f, _ := os.Create(pi)
	x, _ := xz.NewWriter(f)
	writeTar(t, tar.NewWriter(x), map[string][]byte{"boot/linux": []byte("k"), "boot/initrd.lz4": []byte("i")})
	x.Close()
	f.Close()
	if _, err := unpackPCBatocera(pi, &logRep{}); err == nil || !strings.Contains(err.Error(), "isn't Batocera for PCs") {
		t.Errorf("Pi 5 one: %v", err)
	}
	// Without the computer's tar, it's unpacked here instead.
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, "cache2"))
	t.Setenv("PATH", t.TempDir())
	if dir, err := unpackPCBatocera(pc, &logRep{}); err != nil {
		t.Errorf("without tar: %v", err)
	} else if _, err := os.Stat(filepath.Join(dir, "EFI", "BOOT", "BOOTX64.EFI")); err != nil {
		t.Errorf("without tar: %v", err)
	}
}

// The ExFAT partition is open (Finder, Spotlight) and the downloads are on
// it, as when they were copied there: it's closed with force, and nothing
// is needed from it once it's erased.
func TestMacInstallBusyPartitionWithDownloadsOnIt(t *testing.T) {
	st := macBook("")
	state := fakeMacEnv(t, st)
	home := os.Getenv("HOME")
	vols := filepath.Join(home, "Volumes")
	etrosVol := filepath.Join(vols, "ETROS")
	os.MkdirAll(etrosVol, 0o755)
	for _, f := range []string{"batocera-x86_64-43-20261001-boot.tar.xz", "archlinux-bootstrap-x86_64.tar.zst"} {
		if err := os.Rename(filepath.Join(home, "Downloads", f), filepath.Join(etrosVol, f)); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("ETROS_TEST_VOLUMES", vols)
	cur := readFakeMac(t, state)
	cur.Volumes = vols
	cur.Busy = []string{"disk0s3"}
	cur.Disks[0].Parts[2].Mount = etrosVol
	b, _ := json.Marshal(cur)
	os.WriteFile(state, b, 0o644)

	rep := &logRep{}
	if err := (darwinMac{}).install(pickTarget(t, "disk0s3"), 70<<30, "Jenkins Wi-Fi", "hunter2", rep); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(rep.lines, "\n"))
	}
	checkEtrosOn(t, readFakeMac(t, state), "disk0", 3, 4, true)
	if _, err := os.Stat(filepath.Join(etrosVol, "archlinux-bootstrap-x86_64.tar.zst")); err == nil {
		t.Error("the erased partition's files are still there: the stand-in didn't erase")
	}
	calls, _ := os.ReadFile(state + ".log")
	if !strings.Contains(string(calls), "diskutil unmount force disk0s3") {
		t.Errorf("not closed with force:\n%s", calls)
	}
}

// Still busy after all: said in words.
func TestMacInstallBusyMessage(t *testing.T) {
	out := []byte("0:182: execution error: Error: -69877: The target disk is being used")
	err := diskutilError([]string{"eraseVolume", "free", "none", "disk0s3"}, out)
	if !strings.Contains(err.Error(), "erase disk0s3") || !strings.Contains(err.Error(), "still using it") {
		t.Errorf("got %v", err)
	}
}

func quickRetries(t *testing.T) {
	oldT, oldW := eraseTries, eraseWait
	eraseTries, eraseWait = 4, 10*time.Millisecond
	t.Cleanup(func() { eraseTries, eraseWait = oldT, oldW })
}

// macOS is still checking the ExFAT partition for a while: Etros waits and
// tries again, and gets there, without asking for the password each time.
func TestMacInstallWaitsForPartition(t *testing.T) {
	quickRetries(t)
	st := macBook("")
	st.Stuck = map[string]int{"disk0s3": 2}
	state := fakeMacEnv(t, st)
	rep := &logRep{}
	if err := (darwinMac{}).install(pickTarget(t, "disk0s3"), 70<<30, "Jenkins Wi-Fi", "hunter2", rep); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(rep.lines, "\n"))
	}
	checkEtrosOn(t, readFakeMac(t, state), "disk0", 3, 4, true)
	calls, _ := os.ReadFile(state + ".log")
	if !strings.Contains(string(calls), "for i in $(seq 4)") || strings.Count(string(calls), "diskutil eraseVolume free none disk0s3") < 3 {
		t.Errorf("didn't try again:\n%s", calls)
	}
	// However many tries and steps: the password once.
	if n := strings.Count(string(calls), "osascript"); n != 1 {
		t.Errorf("asked for the password %d times:\n%s", n, calls)
	}
}

// It never lets go: the error says what has it.
func TestMacInstallSaysWhoUsesIt(t *testing.T) {
	quickRetries(t)
	st := macBook("")
	st.Stuck = map[string]int{"disk0s3": 99}
	st.Procs = []string{"/System/Library/Filesystems/exfat.fs/Contents/Resources/fsck_exfat"}
	st.Open = map[string][]string{"disk0s3": {"Finder"}}
	fakeMacEnv(t, st)
	err := (darwinMac{}).install(pickTarget(t, "disk0s3"), 70<<30, "", "", &logRep{})
	if err == nil || !strings.Contains(err.Error(), "fsck_exfat") || !strings.Contains(err.Error(), "Finder has it open") {
		t.Fatalf("got %v", err)
	}
}

// Etros Installer run from the very partition chosen: said, and nothing
// erased.
func TestMacInstallFromTheChosenPartition(t *testing.T) {
	st := macBook("")
	state := fakeMacEnv(t, st)
	exe, _ := os.Executable()
	cur := readFakeMac(t, state)
	cur.Disks[0].Parts[2].Mount = filepath.Dir(filepath.Dir(exe))
	b, _ := json.Marshal(cur)
	os.WriteFile(state, b, 0o644)
	err := (darwinMac{}).install(pickTarget(t, "disk0s3"), 70<<30, "", "", &logRep{})
	if err == nil || !strings.Contains(err.Error(), "move it to the Desktop") {
		t.Fatalf("got %v", err)
	}
	if calls, _ := os.ReadFile(state + ".log"); strings.Contains(string(calls), "eraseVolume") {
		t.Error("erased anyway")
	}
}

// BATOCERA (FAT) and SHARE (ExFAT) made in Disk Utility, with macOS never
// letting go of the drive: Etros's files go on BATOCERA and nothing is
// erased, no password asked.
func TestMacInstallOnPartitionsMadeInDiskUtility(t *testing.T) {
	const g = int64(1) << 30
	st := macBook("")
	st.Disks[0].Parts = append(st.Disks[0].Parts[:2],
		fakeMacPart{Num: 3, Content: "Microsoft Basic Data", Name: "BATOCERA", Start: 210<<20 + 180*g, Size: 8 * g},
		fakeMacPart{Num: 4, Content: "Microsoft Basic Data", Name: "SHARE", Start: 210<<20 + 188*g, Size: 62 * g})
	st.Stuck = map[string]int{"disk0s3": 99, "disk0s4": 99}
	state := fakeMacEnv(t, st)
	x := pickTarget(t, "disk0s3")
	if x.kind != "ready" || len(x.extra) != 1 {
		t.Fatalf("offered as %q %+v", x.kind, x.extra)
	}
	rep := &logRep{}
	if err := (darwinMac{}).install(x, 70*g, "Jenkins Wi-Fi", "hunter2", rep); err != nil {
		t.Fatalf("%v\n%s", err, strings.Join(rep.lines, "\n"))
	}
	checkEtrosOn(t, readFakeMac(t, state), "disk0", 3, 4, true)
	calls, _ := os.ReadFile(state + ".log")
	for _, no := range []string{"eraseVolume", "addPartition", "osascript", "unmount"} {
		if strings.Contains(string(calls), no) {
			t.Errorf("did %s:\n%s", no, calls)
		}
	}
}

func TestMacTargetsReady(t *testing.T) {
	const g = int64(1) << 30
	d := dDisk{id: "disk0", internal: true, size: 251 * g, parts: []dPart{
		{"disk0s1", "EFI", "EFI", 200 << 20}, {"disk0s2", "Apple_APFS", "", 180 * g},
		{"disk0s3", "Microsoft Basic Data", "BATOCERA", 8 * g}, {"disk0s5", "Microsoft Basic Data", "SHARE", 60 * g},
	}}
	for _, x := range macTargets([]dDisk{d}, "disk0s2") {
		if x.part.id == "disk0s3" && (x.kind != "ready" || !strings.Contains(x.why, "numbered")) {
			t.Errorf("gap in the numbers: %+v", x)
		}
	}
	d.parts[3].id = "disk0s4"
	d.parts[2].content = "Apple_HFS"
	for _, x := range macTargets([]dDisk{d}, "disk0s2") {
		if x.part.id == "disk0s3" && !strings.Contains(x.why, "MS-DOS") {
			t.Errorf("not FAT: %+v", x)
		}
	}
}
