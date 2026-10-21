//go:build linux

package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const gb = int64(1) << 30

func macTestDisk(parts ...macPart) macDisk {
	return macDisk{Path: "/dev/sdb", Model: "APPLE SSD", Size: 251 * gb, Tran: "usb", PTType: "gpt", Children: parts}
}

func TestPlanMac(t *testing.T) {
	efi := macPart{Path: "/dev/sdb1", PartType: "c12a7328-f81f-11d2-ba4b-00a0c93ec93b", Start: 40, Size: 200 << 20}
	apfs := macPart{Path: "/dev/sdb2", PartType: gptAPFS, Start: 409640, Size: 180 * gb}
	bootcamp := macPart{Path: "/dev/sdb3", PartType: gptMSData, PartLabel: "BOOTCAMP", FSType: "ntfs", Start: 409640 + 180*gb/512, Size: 70 * gb}

	pl, err := planMac(macTestDisk(efi, apfs, bootcamp))
	if err != nil {
		t.Fatal(err)
	}
	if pl.bootN != 3 || pl.shareN != 4 || len(pl.bootCamp) != 1 || pl.start != bootcamp.Start*512 || !pl.roomForOllie() {
		t.Errorf("plan: %+v", pl)
	}

	// Older macOS: a Recovery HD partition between macOS and Boot Camp.
	hfs := apfs
	hfs.PartType = gptHFS
	rec := macPart{Path: "/dev/sdb3", PartType: gptAppleBoot, Start: apfs.Start + apfs.Size/512, Size: 650 << 20}
	bc4 := bootcamp
	bc4.Path, bc4.Start = "/dev/sdb4", rec.Start+rec.Size/512
	if pl, err := planMac(macTestDisk(efi, hfs, rec, bc4)); err != nil || pl.bootN != 4 || pl.shareN != 5 {
		t.Errorf("with Recovery HD: %+v %v", pl, err)
	}

	// Windows' recovery partition after Boot Camp goes too.
	winre := macPart{Path: "/dev/sdb4", PartType: gptWinRE, Start: bootcamp.Start + bootcamp.Size/512, Size: 500 << 20}
	if pl, err := planMac(macTestDisk(efi, apfs, bootcamp, winre)); err != nil || len(pl.bootCamp) != 2 {
		t.Errorf("with WinRE: %+v %v", pl, err)
	}

	// Not Boot Camp after macOS: hands off.
	linux := bootcamp
	linux.PartType = gptLinux
	if _, err := planMac(macTestDisk(efi, apfs, linux)); err == nil || !strings.Contains(err.Error(), "isn't Boot Camp") {
		t.Errorf("Linux partition: %v", err)
	}
	if _, err := planMac(macTestDisk(efi, apfs)); err == nil {
		t.Error("no Boot Camp: no error")
	}
	if _, err := planMac(macTestDisk(efi, bootcamp)); err == nil {
		t.Error("no macOS: no error")
	}
	// A gap in the numbers below Boot Camp.
	odd := apfs
	odd.Path = "/dev/sdb5"
	if _, err := planMac(macTestDisk(efi, odd, bootcamp)); err != nil {
		t.Errorf("free 2 and 3: %v", err)
	}
	sdb2 := macPart{Path: "/dev/sdb2", PartType: gptAppleBoot, Start: 409640, Size: 1 << 30}
	gap := apfs
	gap.Path = "/dev/sdb4"
	if _, err := planMac(macTestDisk(sdb2, gap, bootcamp)); err == nil || !strings.Contains(err.Error(), "numbered oddly") {
		t.Errorf("free 1 and 3: %v", err)
	}
	// Small Boot Camp: Etros, no room for Ollie OS.
	small := bootcamp
	small.Size = 20 * gb
	d := macTestDisk(efi, apfs, small)
	d.Size = (small.Start*512 + small.Size + 1<<20)
	if pl, err := planMac(d); err != nil || pl.roomForOllie() {
		t.Errorf("small: %+v %v", pl, err)
	}
	// Etros there already.
	b := bootcamp
	b.PartType, b.Label = gptMSData, "BATOCERA"
	s := winre
	s.PartType, s.Label = gptLinux, "SHARE"
	if pl, err := planMac(macTestDisk(efi, apfs, b, s)); err != nil || !pl.isEtros {
		t.Errorf("Etros: %+v %v", pl, err)
	}
}

// Putting Batocera for PCs and Ollie OS on a Mac, past the partitions (which
// need UDisks): from stand-ins in ETROS_PC_BATOCERA (a boot.tar.xz) and
// ETROS_PC_BOOTSTRAP (an archlinux-bootstrap tarball with pacman and the
// rest as stubs), as a person without a password would.
func TestMacFiles(t *testing.T) {
	batocera, bootstrap := os.Getenv("ETROS_PC_BATOCERA"), os.Getenv("ETROS_PC_BOOTSTRAP")
	if batocera == "" || bootstrap == "" {
		t.Skip("set ETROS_PC_BATOCERA and ETROS_PC_BOOTSTRAP")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("ETROS_TEST_NO_ROOM_CHECK", "1")
	boot, share := t.TempDir(), t.TempDir()
	initrd, err := pcInitrd(batocera)
	if err != nil {
		t.Fatal(err)
	}
	if err := copyPCBatocera(batocera, initrd, boot, nopReporter{}); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"boot/linux", "boot/batocera.update", "EFI/BOOT/BOOTX64.EFI", "etros/etros.squashfs", "etros/intel-mac.txt"} {
		if _, err := os.Stat(filepath.Join(boot, f)); err != nil {
			t.Error(err)
		}
	}
	if b, _ := os.ReadFile(filepath.Join(boot, "boot", "initrd.gz")); !bytes.Equal(b, initrd) {
		t.Error("initrd.gz isn't Etros's")
	}
	if b, _ := os.ReadFile(filepath.Join(boot, "batocera-boot.conf")); strings.Contains(string(b), "\nautoresize=true") {
		t.Error("autoresize still on")
	}
	if !isMacCard(boot) {
		t.Error("not seen as a Mac")
	}
	if err := buildArchPC(bootstrap, share, boot, nopReporter{}); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"vmlinuz-linux", "initramfs-linux.img"} {
		if _, err := os.Stat(filepath.Join(boot, f)); err != nil {
			t.Error(err)
		}
	}
	img := filepath.Join(share, "etros", "ollie-rootfs.img")
	cat := func(p string) string {
		out, _ := exec.Command("debugfs", "-R", "cat "+p, img).Output()
		return string(out)
	}
	if !strings.Contains(cat("/etc/fstab"), "LABEL=BATOCERA  /boot") {
		t.Error("fstab: " + cat("/etc/fstab"))
	}
	calls := cat("/root-calls.txt")
	for _, want := range []string{"pacman-key --populate archlinux", "linux linux-firmware", "vulkan-intel", "mkinitcpio -k /boot/vmlinuz-linux", "HOOKS=(base udev modconf kms keyboard block filesystems etros)"} {
		if !strings.Contains(calls, want) {
			t.Errorf("no %q in:\n%s", want, calls)
		}
	}
	if strings.Contains(calls, "vulkan-broadcom") || strings.Contains(calls, "-Rdd") {
		t.Error("Pi things done on a PC:\n" + calls)
	}
	if !strings.Contains(cat("/etc/initcpio/hooks/etros"), "ollie-rootfs.img") || !strings.Contains(cat("/usr/local/bin/ollie-boot-batocera"), "kexec") {
		t.Error("start-up hook or Ollie files missing")
	}
	if out, _ := exec.Command("debugfs", "-R", "stat /usr/local/sbin/ollie-setup", img).Output(); !strings.Contains(string(out), "Mode:  0755") || !strings.Contains(string(out), "User:     0") {
		t.Errorf("ollie-setup: %s", out)
	}
	if out, _ := exec.Command("debugfs", "-R", "ls /boot", img).Output(); strings.Contains(string(out), "vmlinuz") {
		t.Error("kernel left in Ollie OS's /boot (the boot partition goes there)")
	}
	if !strings.Contains(cat("/etc/pacman.d/mirrorlist"), "geo.mirror.pkgbuild.com/$repo/os/$arch") {
		t.Error("mirrorlist: " + cat("/etc/pacman.d/mirrorlist"))
	}
}

func TestMacsFromLsblk(t *testing.T) {
	out := []byte(`{"blockdevices":[
	{"name":"nvme0n1","pkname":null,"type":"disk","path":"/dev/nvme0n1","model":"Steam SSD","size":512110190592,"tran":"nvme","hotplug":false,"pttype":"gpt"},
	{"name":"nvme0n1p1","pkname":"nvme0n1","type":"part","path":"/dev/nvme0n1p1","size":1,"parttype":"ebd0a0a2-b9e5-4433-87c0-68b6b72699c7","start":2048},
	{"name":"sdb","pkname":null,"type":"disk","path":"/dev/sdb","model":"APPLE SSD SM0256G ","size":251000193024,"tran":"usb","hotplug":true,"pttype":"gpt"},
	{"name":"sdb3","pkname":"sdb","type":"part","path":"/dev/sdb3","size":75161927680,"parttype":"ebd0a0a2-b9e5-4433-87c0-68b6b72699c7","partlabel":"BOOTCAMP","label":"BOOTCAMP","fstype":"ntfs","start":343613440},
	{"name":"sdb1","pkname":"sdb","type":"part","path":"/dev/sdb1","size":209715200,"parttype":"c12a7328-f81f-11d2-ba4b-00a0c93ec93b","start":40},
	{"name":"sdb2","pkname":"sdb","type":"part","path":"/dev/sdb2","size":175690018816,"parttype":"7c3457ef-0000-11aa-aa11-00306543ecac","start":409640}
	]}`)
	macs := macsFromLsblk(out)
	if len(macs) != 1 || macs[0].disk.Path != "/dev/sdb" || macs[0].disk.Model != "APPLE SSD SM0256G" ||
		macs[0].bootN != 3 || len(macs[0].bootCamp) != 1 || macs[0].bootCamp[0].Path != "/dev/sdb3" {
		t.Errorf("%+v", macs)
	}
}
