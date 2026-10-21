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
// it's macOS asking, in this window.
//
// Ollie OS can't be made in macOS: its Arch Linux goes on BATOCERA, with
// the Wi-Fi this Mac uses, and Etros makes Ollie OS the first time it
// starts (etros-setup-ollie-pc).

import (
	"bufio"
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
	t := &tui{in: bufio.NewReader(os.Stdin), opts: o}
	showTitle("ETROS MAC INSTALLER")
	note("Puts Etros and Ollie OS on this Mac in place of Boot Camp's Windows. macOS and")
	note("your Mac files stay.")
	t.macLoop()
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

// diskutil runs macOS's diskutil, and again with sudo (the Mac's
// password, asked by macOS in this window) if it needs an administrator.
func diskutil(args ...string) error {
	logf("diskutil %s", strings.Join(args, " "))
	cmd := exec.Command("diskutil", args...)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if cmd.Run() == nil {
		return nil
	}
	if os.Geteuid() == 0 {
		return fmt.Errorf("diskutil %s didn't work", args[0])
	}
	fmt.Println()
	note(cYellow + "macOS wants your Mac password for that (it's macOS asking, not Etros):" + cReset)
	cmd = exec.Command("sudo", append([]string{"diskutil"}, args...)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("diskutil %s didn't work: %w", args[0], err)
	}
	return nil
}

func (t *tui) macLoop() bool {
	if runtime.GOARCH != "amd64" {
		note(cRed + "This is for Intel Macs; Macs with Apple chips can't start Etros." + cReset)
		return false
	}
	if err := t.putEtrosOnThisMac(); err != nil {
		logf("ERROR: Mac: %v", err)
		fmt.Printf("\n%s  ETROS DIDN'T GO ON THIS MAC%s\n\n", cRed, cReset)
		note(err.Error())
		note("This is also in etros-installer-log.txt in Downloads.")
		t.ask("Press Enter to finish")
		return false
	}
	return true
}

func (t *tui) putEtrosOnThisMac() error {
	disks, bootStore, err := allDrives()
	if err != nil {
		return err
	}
	targets := macTargets(disks, bootStore)
	var usable []macTarget
	heading("DRIVES AND PARTITIONS")
	shown := ""
	for _, x := range targets {
		if x.disk.id != shown {
			shown = x.disk.id
			where := "USB / external"
			if x.disk.internal {
				where = "this Mac's own drive"
			}
			fmt.Printf("\n  %s%s%s (%s, %s, %s)\n", cBold, orDefault(x.disk.name, "Drive"), cReset, x.disk.id, where, gbs(x.disk.size))
		}
		id := x.part.id
		if x.kind == "whole" {
			id = x.disk.id
		}
		size := x.part.size
		if x.kind == "whole" {
			size = x.disk.size
		}
		if x.why != "" {
			fmt.Printf("     %s·  %-9s %-26s %8s  %s%s\n", cDim, id, x.what, gbs(size), x.why, cReset)
			continue
		}
		usable = append(usable, x)
		fmt.Printf("  %s%3d%s  %-9s %-26s %8s  %s\n", cYellow, len(usable), cReset, id, x.what, gbs(size), x.action)
	}
	fmt.Println()
	if len(usable) == 0 {
		return errors.New("there's nowhere Etros can go: make a partition for it in Disk Utility, or plug in a USB drive")
	}
	var x macTarget
	for {
		a := strings.ToLower(t.ask(fmt.Sprintf("Where should Etros go? (1 to %d, q to stop)", len(usable))))
		if a == "q" || a == "b" {
			return errors.New("stopped: nothing was changed")
		}
		var n int
		if _, err := fmt.Sscan(a, &n); err == nil && n >= 1 && n <= len(usable) {
			x = usable[n-1]
			break
		}
	}

	var etros int64
	switch x.kind {
	case "whole":
		etros = x.disk.size
	case "part":
		etros = x.part.size
		for _, e := range x.extra {
			etros += e.size
		}
	case "room":
		free := apfsFree(x.part.id)
		note("macOS has " + gbs(free) + " free.")
		max := free - 20<<30 // leave macOS some room
		if max < macBootSize+macMinShare {
			return fmt.Errorf("macOS hasn't enough free room for Etros (it needs %s, and macOS keeps 20 GB)", gbs(macBootSize+macMinShare))
		}
		def := min(max, 64<<30) / (1 << 30)
		for etros == 0 {
			a := t.ask(fmt.Sprintf("How many GB for Etros (games go here too; %d to %d)? [%d]", (macBootSize+macMinShare)>>30, max>>30, def))
			n := def
			if a != "" {
				if _, err := fmt.Sscan(a, &n); err != nil {
					continue
				}
			}
			if n<<30 >= macBootSize+macMinShare && n<<30 <= max {
				etros = n << 30
			}
		}
	}
	withOllie := etros >= macBootSize+macMinShare+macOllieRoom
	heading("ETROS GOES ON " + strings.ToUpper(orDefault(x.disk.name, x.disk.id)))
	switch x.kind {
	case "whole":
		note(cBold + "Everything on " + orDefault(x.disk.name, x.disk.id) + " (" + x.disk.id + ") will be deleted." + cReset)
		note("Then it starts Etros on any Intel Mac: plug it in and hold Option at start-up.")
	case "part":
		note(cBold + "Everything on " + x.part.id + " (" + x.what + ") will be deleted." + cReset)
	case "room":
		note(gbs(etros) + " of macOS's free room becomes Etros. macOS and its files stay.")
	}
	if withOllie {
		note("Ollie OS (Arch Linux with KDE Plasma) is made the first time Etros starts, over")
		note("Wi-Fi (30 to 60 minutes; Etros works meanwhile).")
	} else {
		note("There isn't room for Ollie OS too (it needs " + gbs(macBootSize+macMinShare+macOllieRoom) + "), so it's Etros on its own.")
	}
	fmt.Println()
	if strings.ToLower(t.ask("Type ERASE to go ahead (anything else to stop):")) != "erase" {
		return errors.New("stopped: nothing was changed")
	}

	// Everything that can fail before the drive is changed.
	batocera, err := t.getPCBatocera()
	if err != nil {
		return fmt.Errorf("getting Batocera for PCs: %w", err)
	}
	bootstrap := ""
	if withOllie {
		if bootstrap, err = t.getArchBootstrap(); err != nil {
			return fmt.Errorf("getting Arch Linux: %w", err)
		}
	}
	rep := newTermReporter()
	rep.Stage("Checking Batocera for PCs")
	initrd, err := pcInitrd(batocera)
	rep.end()
	if err != nil {
		return err
	}
	ssid, key := "", ""
	if withOllie {
		ssid, key = t.askWifi()
	}

	heading("MAKING ROOM FOR ETROS")
	if err := makeEtrosPartitions(x, etros); err != nil {
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

	rep = newTermReporter()
	defer rep.end()
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
	rep.end()

	fmt.Printf("\n%s  ETROS IS ON %s%s\n\n", cYellow, strings.ToUpper(orDefault(x.disk.name, x.disk.id)), cReset)
	step := 1
	if x.kind == "whole" {
		note("1. Eject the drive in Finder and plug it into the Intel Mac (with an adapter if needed).")
		step++
	}
	note(fmt.Sprintf("%d. Restart the Mac holding %sOption (⌥)%s as it starts, and choose %sEFI Boot%s.", step, cBold, cReset, cBold, cReset))
	note("   (Started normally, it's macOS as before.)")
	note(fmt.Sprintf("%d. The first start takes a little longer: Etros sets up its storage.", step+1))
	if withOllie {
		note(fmt.Sprintf("%d. Then Etros makes Ollie OS over Wi-Fi in the background and says when it's", step+2))
		note("   ready (30 to 60 minutes). Then: ETROS → Boot into Ollie OS.")
	}
	note("In Etros, use a game controller: the 12-inch MacBook's own keyboard and trackpad")
	note("don't work in Batocera. They do in Ollie OS.")
	t.ask("Press Enter to finish")
	return nil
}

// makeEtrosPartitions makes BATOCERA (8 GB) and SHARE (etros less that, in
// all) at the target, one after the other.
func makeEtrosPartitions(x macTarget, etros int64) error {
	switch x.kind {
	case "whole":
		return diskutil("partitionDisk", x.disk.id, "GPT", "MS-DOS FAT32", "BATOCERA", "8G", "ExFAT", "SHARE", "R")
	case "room":
		return diskutil("apfs", "resizeContainer", x.part.id, strconv.FormatInt(apfsSize(x.part.id)-etros, 10)+"B",
			"MS-DOS FAT32", "BATOCERA", "8G", "ExFAT", "SHARE", "0")
	}
	for _, p := range append([]dPart{x.part}, x.extra...) {
		if err := diskutil("eraseVolume", "free", "none", p.id); err != nil {
			return err
		}
	}
	if err := diskutil("addPartition", x.after, "MS-DOS FAT32", "BATOCERA", "8G"); err != nil {
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
				return diskutil("addPartition", p.id, "ExFAT", "SHARE", strconv.FormatInt(share, 10)+"B")
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

// askWifi asks for the password of the Wi-Fi this Mac is on, for Etros.
func (t *tui) askWifi() (ssid, key string) {
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
	if out, err := exec.Command("networksetup", "-getairportnetwork", port).Output(); err == nil {
		if _, s, ok := strings.Cut(strings.TrimSpace(string(out)), ": "); ok && strings.Contains(string(out), "Current") {
			ssid = s
		}
	}
	heading("WI-FI FOR ETROS")
	note("Etros needs the internet the first time it starts, to make Ollie OS.")
	if ssid == "" {
		ssid = t.ask("Your Wi-Fi's name (Enter to skip):")
		if ssid == "" {
			return "", ""
		}
	} else {
		note("This Mac is on " + cBold + ssid + cReset + ".")
	}
	key = t.ask("Its password (Enter if it has none):")
	return ssid, key
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
