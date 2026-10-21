//go:build darwin

package main

// Etros on this Mac, from macOS itself (for Macs without Target Disk Mode,
// like the 12-inch MacBook): Boot Camp's partition, or room taken from
// macOS when there's no Boot Camp, becomes BATOCERA (FAT32, Batocera for
// PCs with the Etros layer) and SHARE right after it (ExFAT here, as macOS
// can't make ext4; Etros makes it ext4 the first time it starts, see
// S11aetros-share). macOS's own diskutil does the partitions; if it wants
// the Mac's password for that, it's macOS asking, in this window.
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

// internalPlan reads this Mac's own drive.
func internalPlan() (*dPlan, error) {
	out, err := exec.Command("diskutil", "list", "-plist", "internal", "physical").Output()
	if err != nil {
		return nil, fmt.Errorf("diskutil list: %w", err)
	}
	v, err := parsePlist(out)
	if err != nil {
		return nil, err
	}
	pl, err := planThisMac(v)
	if err == nil && pl.store != "" {
		pl.free = apfsFree(pl.store)
	}
	return pl, err
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
	pl, err := internalPlan()
	if err != nil {
		return err
	}
	heading("THIS MAC'S DRIVE")
	for _, p := range pl.macOS {
		fmt.Printf("  %s✔%s %-32s %8s  %sstays%s\n", cCyan, cReset, p.content+" "+p.name, gbs(p.size), cCyan, cReset)
	}
	for _, p := range pl.bootCamp {
		fmt.Printf("  %s✘%s %-32s %8s  %sis deleted%s\n", cRed, cReset, "Boot Camp "+p.name, gbs(p.size), cRed, cReset)
	}
	var etros int64
	if len(pl.bootCamp) > 0 {
		etros = pl.bootCamp[0].size
		if pl.isEtros {
			note(cYellow + "This Mac has Etros already: putting it on again deletes its games and saves." + cReset)
			note("(To just update it, use Etros Updater instead.)")
		}
	} else {
		note("There's no Boot Camp on this Mac, so Etros takes room from macOS instead.")
		note("macOS has " + gbs(pl.free) + " free.")
		max := pl.free - 20<<30 // leave macOS some room
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
	fmt.Printf("  %s+%s %-32s %8s\n", cYellow, cReset, "Etros (BATOCERA and SHARE)", gbs(etros))
	withOllie := etros >= macBootSize+macMinShare+macOllieRoom
	if withOllie {
		note("Ollie OS (Arch Linux with KDE Plasma) is made the first time Etros starts, over")
		note("Wi-Fi (30 to 60 minutes; Etros works meanwhile).")
	} else {
		note("There isn't room for Ollie OS too (it needs " + gbs(macBootSize+macMinShare+macOllieRoom) + "), so it's Etros on its own.")
	}
	fmt.Println()
	if len(pl.bootCamp) > 0 {
		note(cBold + "Everything in Boot Camp (Windows and its files) will be deleted." + cReset)
	}
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
	if len(pl.bootCamp) > 0 {
		for _, p := range pl.bootCamp[1:] {
			if err := diskutil("eraseVolume", "free", "none", p.id); err != nil {
				return err
			}
		}
		err = diskutil("splitPartition", pl.bootCamp[0].id, "MS-DOS FAT32", "BATOCERA", "8G", "ExFAT", "SHARE", "R")
	} else {
		err = diskutil("apfs", "resizeContainer", pl.store, strconv.FormatInt(apfsSize(pl.store)-etros, 10),
			"MS-DOS FAT32", "BATOCERA", "8G", "ExFAT", "SHARE", "0")
	}
	if err != nil {
		return err
	}
	after, err := internalPlan()
	if err != nil {
		return err
	}
	var boot, share dPart
	for _, p := range after.parts {
		switch p.name {
		case "BATOCERA":
			boot = p
		case "SHARE":
			share = p
		}
	}
	if boot.id == "" || share.id == "" {
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
	rep.Stage("Copying Batocera and Etros onto this Mac")
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

	fmt.Printf("\n%s  ETROS IS ON THIS MAC%s\n\n", cYellow, cReset)
	note("1. Restart the Mac, holding " + cBold + "Option (⌥)" + cReset + " as it starts, and choose " + cBold + "EFI Boot" + cReset + ".")
	note("   (Started normally, it's macOS as before.)")
	note("2. The first start takes a little longer: Etros sets up its storage.")
	if withOllie {
		note("3. Then Etros makes Ollie OS over Wi-Fi in the background and says when it's ready")
		note("   (30 to 60 minutes). Then: ETROS → Boot into Ollie OS.")
	}
	note("In Etros, use a game controller: the Mac's own keyboard and trackpad don't work in")
	note("Batocera. They do in Ollie OS.")
	t.ask("Press Enter to finish")
	return nil
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
