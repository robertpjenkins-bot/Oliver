package main

// The Mac installer's work on macOS (macweb.go shows it in Safari), done
// with macOS's own tools: diskutil, networksetup, osascript. It builds on
// every system, so it can be tested with stand-ins for those tools
// (mac_tools_test.go); only macOS runs it for real (mac_darwin.go).

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

func (darwinMac) checks() []macCheck {
	intel := macCheck{Name: "An Intel Mac", OK: runtime.GOARCH == "amd64", Blocking: true}
	if out, err := exec.Command("sysctl", "-n", "machdep.cpu.brand_string").Output(); err == nil {
		b := strings.TrimSpace(string(out))
		intel.Name = "An Intel Mac (" + b + ")"
		intel.OK = !strings.Contains(b, "Apple")
	}
	if !intel.OK {
		intel.Why = "Macs with Apple chips (M1 and newer) can't start Etros"
	}
	ver := macCheck{Name: "macOS", OK: true}
	if out, err := exec.Command("sw_vers", "-productVersion").Output(); err == nil {
		ver.Name = "macOS " + strings.TrimSpace(string(out))
	}
	return append([]macCheck{intel, ver}, downloadChecks()...)
}

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
			// Etros works without Ollie OS: carry on.
			logf("Arch Linux: %v", err)
			rep.Log("Ollie OS left out: " + err.Error())
			withOllie = false
		}
	}
	unpacked, err := unpackPCBatocera(batocera, rep)
	if err != nil {
		return err
	}
	rep.Stage("Checking Batocera for PCs")
	initrd, err := pcInitrd(unpacked)
	if err != nil {
		return err
	}

	// Arch Linux made ready here first too: after this, nothing is read from
	// where the downloads were, which may be the very partition erased next.
	archGz := ""
	if withOllie {
		rep.Stage("Getting Arch Linux ready for Ollie OS")
		archGz = filepath.Join(defaultCache(), "archlinux-bootstrap-x86_64.tar.gz")
		if err := gzBootstrap(bootstrap, archGz, rep); err != nil {
			logf("Arch Linux: %v", err)
			rep.Log("Ollie OS left out: " + err.Error())
			withOllie = false
		}
	}

	rep.Stage("Making room for Etros")
	boot, share, err := makeEtrosPartitions(x, etros, rep)
	if err != nil {
		return err
	}
	dir := mountPoint(boot.id)
	if dir == "" {
		return errors.New("couldn't open the BATOCERA partition")
	}

	rep.Stage("Copying Batocera and Etros")
	if err := copyPCBatocera(unpacked, initrd, dir, rep); err != nil {
		return fmt.Errorf("copying onto BATOCERA: %w", err)
	}
	// SHARE is ExFAT for now: Etros makes it ext4 the first time it starts.
	if err := os.WriteFile(filepath.Join(dir, "etros", "format-share"), []byte(share.id+"\n"), 0o644); err != nil {
		return err
	}
	if withOllie {
		rep.Stage("Putting Arch Linux on for Ollie OS")
		if err := copyFile(archGz, filepath.Join(dir, "etros", "archlinux-bootstrap-x86_64.tar.gz"), rep); err != nil {
			return err
		}
		os.Remove(archGz)
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
// that (macOS asks for the Mac's password in its own window). Its errors
// say why in words.
func diskutil(rep Reporter, args ...string) error {
	logf("diskutil %s", strings.Join(args, " "))
	rep.Log("diskutil " + strings.Join(args, " "))
	out, err := exec.Command("diskutil", args...).CombinedOutput()
	logLines(rep, out)
	if err == nil {
		return nil
	}
	if os.Geteuid() == 0 {
		return diskutilError(args, out)
	}
	rep.Log("macOS wants the Mac's password for that (macOS is asking, not Etros)")
	path, err := exec.LookPath("diskutil")
	if err != nil {
		path = "/usr/sbin/diskutil"
	}
	quoted := []string{path}
	for _, a := range args {
		quoted = append(quoted, "'"+strings.ReplaceAll(a, "'", `'\''`)+"'")
	}
	script := fmt.Sprintf("do shell script %q with administrator privileges with prompt %q",
		strings.Join(quoted, " "), "Etros Installer: macOS needs your password to change the drive.")
	out, err = exec.Command("osascript", "-e", script).CombinedOutput()
	logLines(rep, out)
	if err != nil {
		if strings.Contains(string(out), "-128") || strings.Contains(strings.ToLower(string(out)), "user canceled") {
			return errors.New("you clicked Cancel in macOS's password window, so macOS didn't change the drive. " +
				"Try again and type your Mac password there (it's the one you log in with)")
		}
		return diskutilError(args, out)
	}
	return nil
}

// forceUnmount closes a partition (or every partition of a drive), even
// with something using it. It needs no password; it's fine if it was closed.
func forceUnmount(rep Reporter, id string) {
	verb := "unmount"
	if !strings.Contains(strings.TrimPrefix(id, "disk"), "s") {
		verb = "unmountDisk"
	}
	out, err := exec.Command("diskutil", verb, "force", id).CombinedOutput()
	logf("diskutil %s force %s: %v %s", verb, id, err, strings.TrimSpace(string(out)))
	if err == nil {
		rep.Log("closed " + id)
	}
}

func logLines(rep Reporter, out []byte) {
	for _, l := range strings.Split(strings.ReplaceAll(strings.TrimSpace(string(out)), "\r", "\n"), "\n") {
		if l != "" {
			rep.Log(l)
		}
	}
}

// diskutilError says in words why diskutil didn't do something.
func diskutilError(args []string, out []byte) error {
	last := lastLine(out)
	why := ""
	switch {
	case strings.Contains(last, "-69879") || strings.Contains(last, "Couldn't open"):
		why = "macOS couldn't open the drive: something is using it. Close any Finder windows or programs using it, eject anything on it, and try again"
	case strings.Contains(strings.ToLower(last), "being used") || strings.Contains(strings.ToLower(last), "in use") || strings.Contains(strings.ToLower(last), "busy"):
		why = "something on the Mac is still using it. Close any Finder windows showing it and anything opened from it, then try again (restarting the Mac first always works)"
	case strings.Contains(last, "-69888") || strings.Contains(last, "unmount"):
		why = "macOS couldn't unmount the partition: something is using it. Close Finder windows and programs using it, and try again"
	case strings.Contains(last, "-69519") || strings.Contains(strings.ToLower(last), "not enough") || strings.Contains(last, "too small"):
		why = "there isn't enough room there"
	case strings.Contains(strings.ToLower(last), "password") || strings.Contains(last, "-60005"):
		why = "macOS didn't accept the password"
	case last == "":
		why = "macOS didn't say why"
	default:
		why = "macOS said: " + last
	}
	return fmt.Errorf("macOS's Disk Utility couldn't %s: %s", diskutilWhat(args), why)
}

func diskutilWhat(args []string) string {
	switch args[0] {
	case "eraseVolume":
		return "erase " + args[len(args)-1]
	case "addPartition":
		return "make the " + args[3] + " partition"
	case "partitionDisk":
		return "erase " + args[1]
	case "apfs":
		return "make room in macOS"
	}
	return strings.Join(args, " ")
}

func lastLine(b []byte) string {
	lines := strings.Split(strings.TrimSpace(strings.ReplaceAll(string(b), "\r", "\n")), "\n")
	return lines[len(lines)-1]
}

// makeEtrosPartitions makes BATOCERA (8 GB) and SHARE (etros less that, in
// all) at the target, one after the other, and says which they are.
func makeEtrosPartitions(x macTarget, etros int64, rep Reporter) (boot, share dPart, err error) {
	before := map[string]bool{}
	if disks, _, err := allDrives(); err == nil {
		for _, d := range disks {
			for _, p := range d.parts {
				before[p.id] = true
			}
		}
	}
	// The new ones, once made.
	made := func(name string) (dPart, bool) {
		disks, _, _ := allDrives()
		for _, d := range disks {
			for _, p := range d.parts {
				if d.id == x.disk.id && p.name == name && (x.kind == "whole" || !before[p.id] || p.id == x.part.id) {
					return p, true
				}
			}
		}
		return dPart{}, false
	}
	after := x.after
	switch x.kind {
	case "whole":
		forceUnmount(rep, x.disk.id)
		if err := diskutil(rep, "partitionDisk", x.disk.id, "GPT", "MS-DOS FAT32", "BATOCERA", "8G", "ExFAT", "SHARE", "R"); err != nil {
			return boot, share, err
		}
	case "room":
		// Make macOS's APFS smaller, then the two new partitions in the room.
		size := apfsSize(x.part.id)
		if size == 0 {
			return boot, share, errors.New("macOS didn't say how big its APFS partition is")
		}
		if err := diskutil(rep, "apfs", "resizeContainer", x.part.id, strconv.FormatInt(size-etros, 10)+"B"); err != nil {
			return boot, share, err
		}
		after = x.part.id
	case "part":
		for _, p := range append([]dPart{x.part}, x.extra...) {
			// Finder, Spotlight or anything else with it open would stop the
			// erase ("being used"): close it whatever has it open.
			forceUnmount(rep, p.id)
			if err := diskutil(rep, "eraseVolume", "free", "none", p.id); err != nil {
				return boot, share, err
			}
		}
	}
	if x.kind != "whole" {
		if err := diskutil(rep, "addPartition", after, "MS-DOS FAT32", "BATOCERA", "8G"); err != nil {
			return boot, share, err
		}
		b, ok := made("BATOCERA")
		if !ok {
			return boot, share, errors.New("macOS said it made the BATOCERA partition, but it isn't there")
		}
		// The rest of the room, less a little: partitions are lined up on
		// whole megabytes, so it's never quite all of it.
		left := (etros - macBootSize - 64<<20) / mib * mib
		if err := diskutil(rep, "addPartition", b.id, "ExFAT", "SHARE", strconv.FormatInt(left, 10)+"B"); err != nil {
			return boot, share, err
		}
	}
	var ok1, ok2 bool
	boot, ok1 = made("BATOCERA")
	share, ok2 = made("SHARE")
	if !ok1 || !ok2 {
		return boot, share, errors.New("macOS didn't make the BATOCERA and SHARE partitions")
	}
	if share.num() != boot.num()+1 {
		return boot, share, fmt.Errorf("macOS numbered the new partitions %s and %s, but Batocera needs its storage to be the very next number after its boot partition; there's a gap in this drive's numbers (use a whole USB drive instead, or tell the person who made Etros)", boot.id, share.id)
	}
	return boot, share, nil
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
