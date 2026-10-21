//go:build linux

package main

// Etros on an Intel Mac, in place of Boot Camp's Windows, put there from
// this computer with the Mac in Target Disk Mode (its drive shows up here as
// a USB drive). macOS and its files stay as they are; the Boot Camp
// partitions become Etros's boot partition (BATOCERA, Batocera for PCs with
// the Etros layer) and its storage (SHARE, right after it, as Batocera
// expects). Ollie OS is Arch Linux for PCs with its own kernel (the Mac's
// keyboard and trackpad need drivers Batocera's kernel lacks): its file
// system is etros/ollie-rootfs.img on SHARE and its kernel is on the boot
// partition, started from Etros by etros-boot-ollie. The Mac starts Etros
// when Option is held at start-up ("EFI Boot").
//
// All of it through UDisks, like the cards: no password.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	gptAPFS        = "7c3457ef-0000-11aa-aa11-00306543ecac"
	gptHFS         = "48465300-0000-11aa-aa11-00306543ecac"
	gptCoreStorage = "53746f72-6167-11aa-aa11-00306543ecac"
	gptAppleBoot   = "426f6f74-0000-11aa-aa11-00306543ecac" // Recovery HD
	gptMSData      = "ebd0a0a2-b9e5-4433-87c0-68b6b72699c7"
	gptWinRE       = "de94bba4-06d1-4d40-a16a-bfd50179d6ac"
	gptLinux       = "0fc63daf-8483-4772-8e79-3d69d8477de4"
)

type macPart struct {
	Path      string `json:"path"`
	PartType  string `json:"parttype"`
	PartLabel string `json:"partlabel"`
	Label     string `json:"label"`
	FSType    string `json:"fstype"`
	Start     int64  `json:"start"` // in 512-byte sectors
	Size      int64  `json:"size"`
}

type macDisk struct {
	Path     string    `json:"path"`
	Model    string    `json:"model"`
	Size     int64     `json:"size"`
	Tran     string    `json:"tran"`
	Hotplug  bool      `json:"hotplug"`
	PTType   string    `json:"pttype"`
	Children []macPart `json:"children"`
}

func isMacOSPart(p macPart) bool {
	switch strings.ToLower(p.PartType) {
	case gptAPFS, gptHFS, gptCoreStorage, gptAppleBoot:
		return true
	}
	return false
}

// macPlan is what happens to a Mac's drive: the Boot Camp partitions at its
// end (Windows, its recovery, or an Etros put there before) are replaced.
type macPlan struct {
	disk          macDisk
	macOS         []macPart // stays
	bootCamp      []macPart // goes
	start         int64     // bytes: where Etros begins (to the end of the drive)
	size          int64     // bytes Etros gets
	bootN, shareN int       // the partition numbers BATOCERA and SHARE will get
	isEtros       bool      // it's Etros already
}

// planMac works out the plan for a drive, or why there can't be one.
func planMac(d macDisk) (*macPlan, error) {
	if d.PTType != "gpt" {
		return nil, errors.New("not a Mac drive (no GPT partition table)")
	}
	last := -1
	for i, p := range d.Children {
		if isMacOSPart(p) {
			last = i
		}
	}
	if last < 0 {
		return nil, errors.New("no macOS on it")
	}
	pl := &macPlan{disk: d, macOS: d.Children[:last+1]}
	for _, p := range d.Children[last+1:] {
		t := strings.ToLower(p.PartType)
		switch {
		case t == gptMSData || t == gptWinRE:
		case p.Label == "BATOCERA" || p.Label == "SHARE":
			pl.isEtros = true
		default:
			return nil, fmt.Errorf("%s (%s) comes after macOS and isn't Boot Camp; Etros won't touch it", p.Path, p.PartLabel)
		}
		pl.bootCamp = append(pl.bootCamp, p)
	}
	if len(pl.bootCamp) == 0 {
		return nil, errors.New("there's no Boot Camp (Windows) partition on it")
	}
	pl.start = pl.bootCamp[0].Start * 512
	pl.size = d.Size - pl.start - 1<<20 // the backup partition table is at the very end
	// Batocera finds its storage as the partition after its boot partition,
	// so they need numbers one after the other: the first two free ones.
	used := map[int]bool{}
	for _, p := range pl.macOS {
		used[partNum(p.Path)] = true
	}
	free := []int{}
	for n := 1; len(free) < 2; n++ {
		if !used[n] {
			free = append(free, n)
		}
	}
	pl.bootN, pl.shareN = free[0], free[1]
	if pl.shareN != pl.bootN+1 {
		return nil, fmt.Errorf("the partitions are numbered oddly (%d and %d free), and Batocera wouldn't find its storage", free[0], free[1])
	}
	if pl.size < macBootSize+macMinShare {
		return nil, fmt.Errorf("Boot Camp is too small for Etros (%s; it needs %s)", gbs(pl.size), gbs(macBootSize+macMinShare))
	}
	return pl, nil
}

func (pl *macPlan) roomForOllie() bool { return pl.size >= macBootSize+macMinShare+macOllieRoom }

// findMacs lists the Macs (in Target Disk Mode) plugged into this computer.
func findMacs() []*macPlan {
	out, err := exec.Command("lsblk", "-J", "-b", "-l", "-o",
		"NAME,PKNAME,TYPE,PATH,MODEL,SIZE,TRAN,HOTPLUG,PTTYPE,PARTTYPE,PARTLABEL,LABEL,FSTYPE,START").Output()
	if err != nil {
		return nil
	}
	return macsFromLsblk(out)
}

// macsFromLsblk makes the plans from lsblk's flat list (-l), its partitions
// put with their drives (by PKNAME).
func macsFromLsblk(out []byte) []*macPlan {
	var v struct {
		Blockdevices []struct {
			Name      string `json:"name"`
			PKName    string `json:"pkname"`
			Type      string `json:"type"`
			Path      string `json:"path"`
			Model     string `json:"model"`
			Size      int64  `json:"size"`
			Tran      string `json:"tran"`
			Hotplug   bool   `json:"hotplug"`
			PTType    string `json:"pttype"`
			PartType  string `json:"parttype"`
			PartLabel string `json:"partlabel"`
			Label     string `json:"label"`
			FSType    string `json:"fstype"`
			Start     int64  `json:"start"`
		} `json:"blockdevices"`
	}
	if json.Unmarshal(out, &v) != nil {
		return nil
	}
	disks := map[string]*macDisk{}
	var order []string
	for _, b := range v.Blockdevices {
		if b.Type == "disk" || b.Type == "loop" {
			disks[b.Name] = &macDisk{Path: b.Path, Model: strings.TrimSpace(b.Model), Size: b.Size, Tran: b.Tran, Hotplug: b.Hotplug, PTType: b.PTType}
			order = append(order, b.Name)
		}
	}
	for _, b := range v.Blockdevices {
		if d := disks[b.PKName]; b.Type == "part" && d != nil {
			d.Children = append(d.Children, macPart{Path: b.Path, PartType: b.PartType, PartLabel: b.PartLabel,
				Label: b.Label, FSType: b.FSType, Start: b.Start, Size: b.Size})
		}
	}
	var macs []*macPlan
	for _, name := range order {
		d := *disks[name]
		if d.Tran != "usb" && !d.Hotplug && d.Path != os.Getenv("ETROS_TEST_MAC_DISK") { // for tests
			continue // never this computer's own drive
		}
		sort.Slice(d.Children, func(i, j int) bool { return d.Children[i].Start < d.Children[j].Start })
		if pl, err := planMac(d); err == nil {
			macs = append(macs, pl)
		} else {
			for _, p := range d.Children {
				if isMacOSPart(p) {
					logf("Mac drive %s: %v", d.Path, err)
					macs = append(macs, &macPlan{disk: d, size: -1})
					break
				}
			}
		}
	}
	return macs
}

func macInstallSupported() bool { return filesInstallSupported() }

// runMacInstaller is Etros Mac Installer on its own.
func runMacInstaller(o Options) error {
	t := &tui{in: bufio.NewReader(os.Stdin), opts: o}
	showTitle("ETROS MAC INSTALLER")
	note("Puts Etros on an Intel Mac in place of Boot Camp's Windows. macOS and your Mac")
	note("files stay. " + cBold + "No password needed." + cReset)
	for {
		heading("WHICH MAC?")
		fmt.Printf("  %s1%s  A Mac that can do Target Disk Mode %s(plugged in here with a cable)%s\n", cYellow, cReset, cDim, cReset)
		fmt.Printf("  %s2%s  Any other Intel Mac, like the 12-inch MacBook %s(get its downloads onto a USB drive)%s\n", cYellow, cReset, cDim, cReset)
		fmt.Printf("  %sq%s  Quit\n", cYellow, cReset)
		switch strings.ToLower(t.ask("Type 1, 2 or q:")) {
		case "1":
			t.macLoop()
			return nil
		case "2":
			if t.macUSB() {
				return nil
			}
		case "q", "b":
			return nil
		}
	}
}

// usbDrives are the drives open here that a Mac can read (FAT, ExFAT, NTFS).
func usbDrives() []string {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 3 {
			continue
		}
		dir := unescapeMount(fields[1])
		switch fields[2] {
		case "vfat", "exfat", "ntfs", "ntfs3", "fuseblk":
			if strings.HasPrefix(dir, "/run/media/") || strings.HasPrefix(dir, "/media/") {
				out = append(out, dir)
			}
		}
	}
	return out
}

// macUSB puts Batocera for PCs and Arch Linux on a USB drive, for a Mac
// that can't do Target Disk Mode: files copied from a USB drive aren't
// marked as downloaded, so macOS doesn't stop to check them, and Etros Mac
// Installer on the Mac finds them there and downloads nothing.
func (t *tui) macUSB() bool {
	heading("GET THE MAC'S DOWNLOADS ONTO A USB DRIVE")
	note("Plug in a USB drive formatted FAT32 or ExFAT (about 4 GB free), then press Enter.")
	note("(Ext4 drives, the Linux kind, can't be read by a Mac.)")
	for {
		drives := usbDrives()
		for i, d := range drives {
			fmt.Printf("  %s%d%s  %s %s(%s free)%s\n", cYellow, i+1, cReset, d, cDim, gbs(freeSpace(d)), cReset)
		}
		if len(drives) == 0 {
			note("No FAT32 or ExFAT drive open yet.")
		}
		fmt.Printf("  %sr%s  Look again    %sb%s  Back\n", cYellow, cReset, cYellow, cReset)
		a := strings.ToLower(t.ask("Which drive?"))
		if a == "b" || a == "q" {
			return false
		}
		var n int
		if _, err := fmt.Sscan(a, &n); err != nil || n < 1 || n > len(drives) {
			continue
		}
		dest := filepath.Join(drives[n-1], "Etros for Mac")
		if err := macUSBCopy(dest); err != nil {
			logf("ERROR: USB for the Mac: %v", err)
			fmt.Printf("\n%s  THAT DIDN'T WORK%s\n\n", cRed, cReset)
			note(err.Error())
			t.ask("Press Enter to go back")
			continue
		}
		closed := closeCard(drives[n-1])
		fmt.Printf("\n%s  THE USB DRIVE IS READY%s\n\n", cYellow, cReset)
		if !closed {
			note("0. Eject it first (in Dolphin, right-click it, Safely Remove).")
		}
		note("1. Also copy Etros Mac Installer (app).zip onto it, if it isn't there.")
		note("2. Plug it into the Mac. Copy Etros Mac Installer (app).zip from it to the Mac's")
		note("   Desktop and double-click it, then double-click Etros Installer.")
		note("3. Etros Installer finds Batocera and Arch Linux on the USB drive: no downloads,")
		note("   and no waiting for macOS's check.")
		t.ask("Press Enter to finish")
		return true
	}
}

func macUSBCopy(dest string) error {
	rep := newTermReporter()
	defer rep.end()
	batocera, err := fetchPCBatocera(rep)
	if err != nil {
		return err
	}
	arch, err := fetchArchBootstrap(rep)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	need := int64(0)
	for _, f := range []string{batocera, arch} {
		if st, err := os.Stat(f); err == nil {
			need += st.Size()
		}
	}
	if free := freeSpace(dest); free >= 0 && free < need {
		return fmt.Errorf("the USB drive needs %s free and has %s", gbs(need), gbs(free))
	}
	for src, name := range map[string]string{batocera: "batocera-x86_64-boot.tar.xz", arch: "archlinux-bootstrap-x86_64.tar.zst"} {
		rep.Stage("Copying " + name + " onto the USB drive")
		if err := copyFile(src, filepath.Join(dest, name), rep); err != nil {
			if strings.Contains(err.Error(), "file too large") {
				return fmt.Errorf("%s is too big for a FAT32 drive: format the USB drive as ExFAT and try again", name)
			}
			return err
		}
	}
	syncDir(dest)
	return nil
}

// macLoop picks a Mac and puts Etros on it. It returns false to go back.
func (t *tui) macLoop() bool {
	if !macInstallSupported() {
		note(cRed + "This needs Linux with UDisks (like a Steam Machine or Steam Deck)." + cReset)
		return false
	}
	heading("PLUG IN THE MAC")
	note("1. Turn the Mac off.")
	note("2. Connect it to this computer with a USB-C cable.")
	note("3. Turn the Mac on and hold " + cBold + "T" + cReset + " until a USB symbol shows on its screen")
	note("   (Target Disk Mode: its drive shows up here like a USB drive).")
	for {
		macs := findMacs()
		fmt.Println()
		if len(macs) == 0 {
			note("No Mac found yet. Do the steps above, wait a few seconds, then press Enter.")
		}
		for i, m := range macs {
			what := "Boot Camp → Etros"
			if m.size < 0 {
				what = cRed + "no Boot Camp partition to use" + cReset
			} else if m.isEtros {
				what = "has Etros already"
			}
			fmt.Printf("  %s%d%s  %s %s (%s): %s\n", cYellow, i+1, cReset, m.disk.Model, m.disk.Path, gbs(m.disk.Size), what)
		}
		fmt.Printf("  %sr%s  Look again    %sb%s  Back\n", cYellow, cReset, cYellow, cReset)
		a := strings.ToLower(t.ask("Which Mac? (its number, r or b)"))
		if a == "b" || a == "q" {
			return false
		}
		var n int
		if _, err := fmt.Sscan(a, &n); err != nil || n < 1 || n > len(macs) {
			continue
		}
		m := macs[n-1]
		if m.size < 0 {
			_, err := planMac(m.disk)
			note(cRed + "Etros can't go on this Mac: " + err.Error() + cReset)
			note("It goes where Boot Camp's Windows is: set up Boot Camp on the Mac first (Boot Camp")
			note("Assistant, in macOS), even without finishing Windows, then try again.")
			continue
		}
		if err := t.putEtrosOnMac(m); err != nil {
			logf("ERROR: Mac: %v", err)
			fmt.Printf("\n%s  ETROS DIDN'T GO ON THE MAC%s\n\n", cRed, cReset)
			note(err.Error())
			note("macOS wasn't touched. This is also in etros-installer-log.txt in Downloads.")
			t.ask("Press Enter to go back")
			continue
		}
		return true
	}
}

func (t *tui) putEtrosOnMac(m *macPlan) error {
	heading("THIS MAC'S DRIVE")
	for _, p := range m.macOS {
		fmt.Printf("  %s✔%s %-40s %s  %sstays%s\n", cCyan, cReset, describePart(p), gbs(p.Size), cCyan, cReset)
	}
	for _, p := range m.bootCamp {
		fmt.Printf("  %s✘%s %-40s %s  %sis deleted%s\n", cRed, cReset, describePart(p), gbs(p.Size), cRed, cReset)
	}
	fmt.Printf("  %s+%s %-40s %s\n", cYellow, cReset, "Etros (BATOCERA) and its storage (SHARE)", gbs(m.size))
	if m.isEtros {
		note(cYellow + "This Mac has Etros already: putting it on again deletes its games and saves too." + cReset)
		note("(To just update it, use Etros Updater instead.)")
	}
	withOllie := m.roomForOllie()
	if withOllie {
		withOllie = t.yes("Include Ollie OS (Arch Linux with KDE Plasma, made now: 30 to 60 minutes)?", true)
	} else {
		note("There isn't room for Ollie OS too (it needs " + gbs(macBootSize+macMinShare+macOllieRoom) + "), so it's Etros on its own.")
	}
	fmt.Println()
	note(cBold + "Everything in Boot Camp (Windows and its files) will be deleted." + cReset)
	if strings.ToLower(t.ask("Type ERASE to go ahead (anything else to go back):")) != "erase" {
		return errors.New("stopped: nothing was changed")
	}

	// Everything that can fail before the Mac's drive is changed.
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
	defer rep.end()
	rep.Stage("Checking Batocera for PCs")
	initrd, err := pcInitrd(batocera)
	if err != nil {
		return err
	}

	boot, share, closeAll, err := makeMacPartitions(m, rep)
	if err != nil {
		return err
	}
	defer closeAll()

	rep.Stage("Copying Batocera and Etros onto the Mac")
	if err := copyPCBatocera(batocera, initrd, boot, rep); err != nil {
		return fmt.Errorf("copying onto the boot partition: %w", err)
	}
	syncDir(boot)

	ollieErr := error(nil)
	if withOllie {
		ollieErr = buildArchPC(bootstrap, share, boot, rep)
		if ollieErr != nil {
			logf("Ollie OS on the Mac: %v", ollieErr)
		}
	}
	rep.end()
	closeAll()

	fmt.Printf("\n%s  ETROS IS ON THE MAC%s\n\n", cYellow, cReset)
	if ollieErr != nil {
		note(cRed + "Ollie OS couldn't be made: " + ollieErr.Error() + cReset)
		note("(everything it did is in " + archBuildLog() + "). Etros works without it.")
		fmt.Println()
	}
	note("1. Hold the Mac's power button until it turns off, then unplug the cable.")
	note("2. Turn it on holding " + cBold + "Option (⌥)" + cReset + ", and choose " + cBold + "EFI Boot" + cReset + ". That's Etros.")
	note("   (Turned on normally, it starts macOS as before.)")
	note("3. In Etros, use a game controller (USB or Bluetooth): the Mac's own keyboard and")
	note("   trackpad don't work in Batocera. They do in Ollie OS (ETROS → Boot into Ollie OS).")
	t.ask("Press Enter to finish")
	return nil
}

func describePart(p macPart) string {
	name := p.PartLabel
	switch strings.ToLower(p.PartType) {
	case gptAPFS, gptHFS, gptCoreStorage:
		name = "macOS (" + p.PartLabel + ")"
	case gptAppleBoot:
		name = "macOS Recovery"
	case gptMSData:
		name = "Boot Camp: " + p.PartLabel
	case gptWinRE:
		name = "Boot Camp: Windows recovery"
	}
	if strings.EqualFold(p.PartType, "c12a7328-f81f-11d2-ba4b-00a0c93ec93b") {
		name = "EFI (the Mac's start-up files)"
	}
	if p.Label == "BATOCERA" || p.Label == "SHARE" {
		name = "Etros: " + p.Label
	}
	return name
}

// makeMacPartitions replaces the Boot Camp partitions with BATOCERA and
// SHARE and opens both.
func makeMacPartitions(m *macPlan, rep Reporter) (boot, share string, closeAll func(), err error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return "", "", nil, fmt.Errorf("can't reach UDisks: %w", err)
	}
	u := &udisksDrive{conn: conn, dev: m.disk.Path}
	rep.Stage("Taking Boot Camp off the Mac")
	for _, p := range m.bootCamp {
		path := udisksPath(p.Path)
		var mp [][]byte
		if v, err := conn.Object(udisks, path).GetProperty(udisks + ".Filesystem.MountPoints"); err == nil && v.Store(&mp) == nil && len(mp) > 0 {
			if err := u.call(path, "Filesystem.Unmount", noOpts()).Err; err != nil {
				return "", "", nil, explainUDisks("could not unmount "+p.Path, err)
			}
		}
		if err := u.call(path, "Partition.Delete", noOpts()).Err; err != nil {
			return "", "", nil, explainUDisks("could not delete "+p.Path, err)
		}
		rep.Log("deleted " + describePart(p))
	}

	disk := udisksPath(m.disk.Path)
	var bootPart, sharePart dbus.ObjectPath
	start := (m.start + mib - 1) / mib * mib
	err = u.call(disk, "PartitionTable.CreatePartitionAndFormat",
		uint64(start), uint64(macBootSize), gptMSData, "BATOCERA", noOpts(),
		"vfat", map[string]dbus.Variant{"label": dbus.MakeVariant("BATOCERA")}).Store(&bootPart)
	if err != nil {
		return "", "", nil, explainUDisks("could not make Etros's boot partition", err)
	}
	rep.Log("made Etros's boot partition (BATOCERA)")
	err = u.call(disk, "PartitionTable.CreatePartitionAndFormat",
		uint64(start+macBootSize), uint64(0), gptLinux, "SHARE", noOpts(),
		"ext4", map[string]dbus.Variant{
			"label":          dbus.MakeVariant("SHARE"),
			"take-ownership": dbus.MakeVariant(true),
		}).Store(&sharePart)
	if err != nil {
		return "", "", nil, explainUDisks("could not make Etros's storage partition", err)
	}
	rep.Log("made Etros's storage partition (SHARE)")
	num := func(p dbus.ObjectPath) uint32 {
		var n uint32
		if v, err := conn.Object(udisks, p).GetProperty(udisks + ".Partition.Number"); err == nil {
			v.Store(&n)
		}
		return n
	}
	if b, s := num(bootPart), num(sharePart); s != b+1 {
		return "", "", nil, fmt.Errorf("the new partitions got numbers %d and %d, and Batocera needs its storage right after its boot partition", b, s)
	}

	mount := func(p dbus.ObjectPath) (string, error) {
		var dir string
		if err := u.call(p, "Filesystem.Mount", noOpts()).Store(&dir); err != nil {
			var mp [][]byte
			if v, gerr := conn.Object(udisks, p).GetProperty(udisks + ".Filesystem.MountPoints"); gerr == nil {
				if v.Store(&mp) == nil && len(mp) > 0 {
					return strings.TrimRight(string(mp[0]), "\x00"), nil
				}
			}
			return "", explainUDisks("could not open a partition", err)
		}
		return dir, nil
	}
	unmount := func(p dbus.ObjectPath) {
		for i := 0; i < 10; i++ {
			if u.call(p, "Filesystem.Unmount", noOpts()).Err == nil {
				return
			}
			time.Sleep(time.Second)
		}
	}
	if boot, err = mount(bootPart); err != nil {
		return "", "", nil, err
	}
	if share, err = mount(sharePart); err != nil {
		unmount(bootPart)
		return "", "", nil, err
	}
	closed := false
	return boot, share, func() {
		if !closed {
			closed = true
			syncDir(boot)
			syncDir(share)
			unmount(sharePart)
			unmount(bootPart)
		}
	}, nil
}

func showCrash(msg string) {}
