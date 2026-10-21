//go:build darwin

package main

// Etros on an Intel Mac, from macOS itself (for Macs without Target Disk
// Mode, like the 12-inch MacBook): every drive and partition is listed, and
// the one chosen becomes BATOCERA (FAT32, Batocera for PCs with the Etros
// layer) and SHARE right after it (ExFAT here, as macOS can't make ext4;
// Etros makes it ext4 the first time it starts, see S11aetros-share). It can
// be a partition (Boot Camp, or one made for Etros), free room in macOS, or
// a whole USB drive, which then starts Etros on any Intel Mac. macOS's own
// diskutil does the partitions; if it wants the Mac's password for that,
// it's macOS asking, in its own password window. It all shows in Safari
// (macweb.go): no Terminal.
//
// Ollie OS can't be made in macOS: its Arch Linux goes on BATOCERA, with
// the Wi-Fi this Mac uses, and Etros makes Ollie OS the first time it
// starts (etros-setup-ollie-pc).

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func macInstallSupported() bool { return runtime.GOARCH == "amd64" }

func runMacInstaller(o Options) error {
	return serveMacWeb(darwinMac{}, func(url string) error {
		if exec.Command("open", "-a", "Safari", url).Run() == nil {
			return nil
		}
		return exec.Command("open", url).Run()
	})
}

// darwinMac is the Mac installer's work, for its page.
type darwinMac struct{}

func (darwinMac) targets() ([]macTarget, error) {
	disks, bootStore, err := allDrives()
	if err != nil {
		return nil, err
	}
	return macTargets(disks, bootStore), nil
}

func (darwinMac) roomFree(part string) int64 { return apfsFree(part) }

func (darwinMac) wifiName() string {
	port := "en0"
	if out, err := exec.Command("networksetup", "-listallhardwareports").Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for i, l := range lines {
			if strings.Contains(l, "Wi-Fi") || strings.Contains(l, "AirPort") {
				if i+1 < len(lines) && strings.HasPrefix(lines[i+1], "Device: ") {
					port = strings.TrimSpace(strings.TrimPrefix(lines[i+1], "Device: "))
				}
			}
		}
	}
	out, err := exec.Command("networksetup", "-getairportnetwork", port).Output()
	if err != nil || !strings.Contains(string(out), "Current") {
		return ""
	}
	_, ssid, _ := strings.Cut(strings.TrimSpace(string(out)), ": ")
	return ssid
}

func (darwinMac) install(x macTarget, etros int64, ssid, key string, rep Reporter) error {
	withOllie := etros >= macBootSize+macMinShare+macOllieRoom

	// Everything that can fail before the drive is changed.
	batocera, err := fetchPCBatocera(rep)
	if err != nil {
		return fmt.Errorf("getting Batocera for PCs: %w", err)
	}
	bootstrap := ""
	if withOllie {
		if bootstrap, err = fetchArchBootstrap(rep); err != nil {
			return fmt.Errorf("getting Arch Linux: %w", err)
		}
	}
	rep.Stage("Checking Batocera for PCs")
	initrd, err := pcInitrd(batocera)
	if err != nil {
		return err
	}

	rep.Stage("Making room for Etros")
	if err := makeEtrosPartitions(x, etros, rep); err != nil {
		return err
	}
	after, _, err := allDrives()
	if err != nil {
		return err
	}
	var boot, share dPart
	for _, d := range after {
		if d.id != x.disk.id {
			continue
		}
		for i, p := range d.parts {
			if p.name == "BATOCERA" && i+1 < len(d.parts) && d.parts[i+1].name == "SHARE" {
				boot, share = p, d.parts[i+1]
			}
		}
	}
	if boot.id == "" {
		return errors.New("macOS didn't make the BATOCERA and SHARE partitions")
	}
	if share.num() != boot.num()+1 {
		return fmt.Errorf("the new partitions are %s and %s, but Batocera needs its storage right after its boot partition", boot.id, share.id)
	}
	dir := mountPoint(boot.id)
	if dir == "" {
		return errors.New("couldn't open the BATOCERA partition")
	}

	rep.Stage("Copying Batocera and Etros")
	if err := copyPCBatocera(batocera, initrd, dir, rep); err != nil {
		return fmt.Errorf("copying onto BATOCERA: %w", err)
	}
	// SHARE is ExFAT for now: Etros makes it ext4 the first time it starts.
	if err := os.WriteFile(filepath.Join(dir, "etros", "format-share"), []byte(share.id+"\n"), 0o644); err != nil {
		return err
	}
	if withOllie {
		rep.Stage("Putting Arch Linux on for Ollie OS")
		if err := gzBootstrap(bootstrap, filepath.Join(dir, "etros", "archlinux-bootstrap-x86_64.tar.gz"), rep); err != nil {
			return err
		}
		if ssid != "" {
			if err := os.WriteFile(filepath.Join(dir, "etros", "wifi.txt"), []byte(ssid+"\n"+key+"\n"), 0o600); err != nil {
				return err
			}
		}
	}
	syncDir(filepath.Join(dir, "etros"))
	syncDir(dir)
	rep.Stage("Done")
	return nil
}

// allDrives reads every drive and partition, and which partition the
// running macOS is on.
func allDrives() ([]dDisk, string, error) {
	out, err := exec.Command("diskutil", "list", "-plist", "physical").Output()
	if err != nil {
		return nil, "", fmt.Errorf("diskutil list: %w", err)
	}
	v, err := parsePlist(out)
	if err != nil {
		return nil, "", err
	}
	info := func(id string) any {
		out, err := exec.Command("diskutil", "info", "-plist", id).Output()
		if err != nil {
			return nil
		}
		v, _ := parsePlist(out)
		return v
	}
	boot := ""
	if v := info("/"); v != nil {
		boot = plistString(v, "APFSPhysicalStores", 0, "APFSPhysicalStore")
	}
	return readDisks(v, info), boot, nil
}

// apfsFree is the free room in the APFS container on the partition store.
func apfsFree(store string) int64 {
	out, err := exec.Command("diskutil", "apfs", "list", "-plist").Output()
	if err != nil {
		return 0
	}
	v, err := parsePlist(out)
	if err != nil {
		return 0
	}
	cs, _ := plistGet(v, "Containers").([]any)
	for i := range cs {
		stores, _ := plistGet(cs[i], "PhysicalStores").([]any)
		for j := range stores {
			if plistString(stores[j], "DeviceIdentifier") == store {
				return plistInt(cs[i], "CapacityFree")
			}
		}
	}
	return 0
}

// diskutil runs macOS's diskutil, and again as administrator if it needs
// that (macOS asks for the Mac's password in its own window).
func diskutil(rep Reporter, args ...string) error {
	logf("diskutil %s", strings.Join(args, " "))
	rep.Log("diskutil " + strings.Join(args, " "))
	out, err := exec.Command("diskutil", args...).CombinedOutput()
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if l != "" {
			rep.Log(l)
		}
	}
	if err == nil {
		return nil
	}
	if os.Geteuid() == 0 {
		return fmt.Errorf("diskutil %s didn't work: %s", args[0], lastLine(out))
	}
	rep.Log("macOS wants the Mac's password for that (macOS is asking, not Etros)")
	quoted := []string{"/usr/sbin/diskutil"}
	for _, a := range args {
		quoted = append(quoted, "'"+strings.ReplaceAll(a, "'", `'\''`)+"'")
	}
	script := fmt.Sprintf("do shell script %q with administrator privileges with prompt %q",
		strings.Join(quoted, " "), "Etros Installer: macOS needs your password to change the drive.")
	out, err = exec.Command("osascript", "-e", script).CombinedOutput()
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\r") {
		if l != "" {
			rep.Log(l)
		}
	}
	if err != nil {
		return fmt.Errorf("diskutil %s didn't work: %s", args[0], lastLine(out))
	}
	return nil
}

func lastLine(b []byte) string {
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(b), "\r", "\n")), "\n")
	return lines[len(lines)-1]
}

// macLoop (Etros Maker's menu on a Mac) opens the installer in Safari.
func (t *tui) macLoop() bool {
	if runtime.GOARCH != "amd64" {
		note(cRed + "This is for Intel Macs; Macs with Apple chips can't start Etros." + cReset)
		return false
	}
	note("Etros Mac Installer opens in Safari.")
	if err := runMacInstaller(t.opts); err != nil {
		note(cRed + err.Error() + cReset)
		return false
	}
	return true
}

// makeEtrosPartitions makes BATOCERA (8 GB) and SHARE (etros less that, in
// all) at the target, one after the other.
func makeEtrosPartitions(x macTarget, etros int64, rep Reporter) error {
	switch x.kind {
	case "whole":
		return diskutil(rep, "partitionDisk", x.disk.id, "GPT", "MS-DOS FAT32", "BATOCERA", "8G", "ExFAT", "SHARE", "R")
	case "room":
		return diskutil(rep, "apfs", "resizeContainer", x.part.id, strconv.FormatInt(apfsSize(x.part.id)-etros, 10)+"B",
			"MS-DOS FAT32", "BATOCERA", "8G", "ExFAT", "SHARE", "0")
	}
	for _, p := range append([]dPart{x.part}, x.extra...) {
		if err := diskutil(rep, "eraseVolume", "free", "none", p.id); err != nil {
			return err
		}
	}
	if err := diskutil(rep, "addPartition", x.after, "MS-DOS FAT32", "BATOCERA", "8G"); err != nil {
		return err
	}
	disks, _, err := allDrives()
	if err != nil {
		return err
	}
	for _, d := range disks {
		for _, p := range d.parts {
			if d.id == x.disk.id && p.name == "BATOCERA" {
				share := (etros - macBootSize - 16<<20) / mib * mib
				return diskutil(rep, "addPartition", p.id, "ExFAT", "SHARE", strconv.FormatInt(share, 10)+"B")
			}
		}
	}
	return errors.New("macOS didn't make the BATOCERA partition")
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func apfsSize(store string) int64 {
	out, err := exec.Command("diskutil", "info", "-plist", store).Output()
	if err != nil {
		return 0
	}
	v, err := parsePlist(out)
	if err != nil {
		return 0
	}
	return plistInt(v, "Size")
}

// mountPoint is where a partition is open in Finder, opening it if needed.
func mountPoint(id string) string {
	get := func() string {
		out, err := exec.Command("diskutil", "info", "-plist", id).Output()
		if err != nil {
			return ""
		}
		v, err := parsePlist(out)
		if err != nil {
			return ""
		}
		return plistString(v, "MountPoint")
	}
	if mp := get(); mp != "" {
		return mp
	}
	exec.Command("diskutil", "mount", id).Run()
	return get()
}
