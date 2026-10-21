//go:build linux

package main

// Adding the R36S to an Etros card made without it. Its boot partition was
// sized for the Pi 5 alone, and on Linux partitions can't be grown without
// the administrator password, so the card is made again with both devices:
// everything on its storage partition (games, saves, settings) is copied to
// this computer first and put back after.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

func addR36SSupported() bool { return filesInstallSupported() }

// shareFillsCard tells whether the SHARE partition of the card whose
// BATOCERA drive is at dir reaches (within 16 MiB of) the end of the card.
func shareFillsCard(dir string) bool {
	disk, err := cardDisk(dir)
	if err != nil {
		return false
	}
	out, err := exec.Command("lsblk", "-J", "-b", "-o", "PATH,SIZE,START", disk).Output()
	if err != nil {
		return false
	}
	var v struct {
		Blockdevices []struct {
			Size     int64 `json:"size"`
			Children []struct {
				Size  int64 `json:"size"`
				Start int64 `json:"start"`
			} `json:"children"`
		} `json:"blockdevices"`
	}
	if json.Unmarshal(out, &v) != nil || len(v.Blockdevices) == 0 || len(v.Blockdevices[0].Children) < 2 {
		return false
	}
	d := v.Blockdevices[0]
	share := d.Children[len(d.Children)-1]
	return share.Start*512+share.Size >= d.Size-16<<20
}

// cardDisk is the whole drive the BATOCERA drive at dir is on (/dev/sdb).
func cardDisk(dir string) (string, error) {
	devs, err := lsblk()
	if err != nil {
		return "", err
	}
	for _, d := range devs {
		for _, c := range d.Children {
			for _, m := range c.Mountpoints {
				if m == dir {
					return d.Path, nil
				}
			}
		}
	}
	return "", fmt.Errorf("can't tell which drive %s is on", dir)
}

// openShare mounts the card's SHARE partition (partition 2) through UDisks
// and returns where, and how to close it again.
func openShare(disk string) (string, func(), error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return "", nil, err
	}
	part := disk + "2"
	if last := disk[len(disk)-1]; last >= '0' && last <= '9' { // mmcblk0, nvme0n1, loop0
		part = disk + "p2"
	}
	obj := conn.Object(udisks, udisksPath(part))
	var mp [][]byte
	if v, err := obj.GetProperty(udisks + ".Filesystem.MountPoints"); err == nil {
		v.Store(&mp)
	}
	if len(mp) > 0 {
		return string(bytes.TrimRight(mp[0], "\x00")), func() {}, nil
	}
	var dir string
	if err := obj.Call(udisks+".Filesystem.Mount", 0, noOpts()).Store(&dir); err != nil {
		return "", nil, explainUDisks("could not open the card's storage (SHARE)", err)
	}
	return dir, func() { obj.Call(udisks+".Filesystem.Unmount", 0, noOpts()) }, nil
}

// copyTree copies the files under src to dest, leaving out what skip says.
// Files that can't be read are counted, not fatal.
func copyTree(src, dest string, skip func(rel string) bool, rep Reporter) (copied, unreadable int, err error) {
	var total, done int64
	filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		rel, _ := filepath.Rel(src, p)
		if rel != "." && skip(filepath.ToSlash(rel)) {
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if err == nil && info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	err = filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		rel, _ := filepath.Rel(src, p)
		if rel == "." {
			return nil
		}
		if skip(filepath.ToSlash(rel)) {
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if err != nil {
			unreadable++
			logf("backup: can't read %s: %v", rel, err)
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dest, rel)
		switch {
		case info.IsDir():
			return os.MkdirAll(target, 0o755)
		case info.Mode()&os.ModeSymlink != 0:
			if l, err := os.Readlink(p); err == nil {
				os.Remove(target)
				os.Symlink(l, target)
			}
			return nil
		case !info.Mode().IsRegular():
			return nil
		}
		in, err := os.Open(p)
		if err != nil {
			unreadable++
			logf("backup: can't read %s: %v", rel, err)
			return nil
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		n, err := io.Copy(out, &countingReader{r: in, p: func(d, _ int64) { rep.Progress(done+d, total) }})
		done += n
		if cerr := out.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		os.Chtimes(target, info.ModTime(), info.ModTime())
		copied++
		return nil
	})
	return copied, unreadable, err
}

// remakeWithR36S makes the card at dir again with the R36S added.
func (t *tui) remakeWithR36S(dir string) error {
	f := scan()
	o := t.opts
	o.NoR36S = false
	o.Ollie = true
	if len(f.Pi5) > 0 {
		o.Sources.Pi5Boot = f.Pi5[0].Path
	}
	if len(f.R36S) > 0 {
		o.Sources.R36SBoot = f.R36S[0].Path
		note("Using " + baseName(f.R36S[0].Path) + " for the R36S.")
	} else if p := t.askForFile("BATOCERA FOR THE R36S",
		"Drag Batocera for the R36S (the .img.gz from batocera.org) onto this window, then press Enter.",
		"Or just press Enter to let Etros Updater download it."); p != "" {
		o.Sources.R36SBoot = p
	}
	if len(f.Ollie) > 0 {
		o.Sources.RPiOS = f.Ollie[0].Path
	}
	if len(f.Wine) > 0 {
		o.Sources.Wine = f.Wine[0].Path
	}

	disk, err := cardDisk(dir)
	if err != nil {
		return err
	}

	// 1. Everything on the card's storage, to this computer.
	share, closeShare, err := openShare(disk)
	if err != nil {
		return err
	}
	backup := filepath.Join(defaultCache(), "card-backup-"+time.Now().Format("2006-01-02-150405"))
	heading("COPYING YOUR GAMES, SAVES AND SETTINGS TO THIS COMPUTER")
	note("Kept in " + backup + " until they're back on the card.")
	rep := newTermReporter()
	rep.Stage("Copying the card's storage")
	skip := func(rel string) bool {
		return rel == "lost+found" || rel == ollieFileOnShare
	}
	n, bad, err := copyTree(share, backup, skip, rep)
	rep.end()
	closeShare()
	if err != nil {
		return fmt.Errorf("copying the card's storage to this computer: %w (nothing on the card was changed)", err)
	}
	logf("backup: %d files to %s, %d unreadable", n, backup, bad)
	if bad > 0 {
		note(fmt.Sprintf("%d files couldn't be read and stay behind (they're listed in etros-installer-log.txt).", bad))
	}

	// 2. A new Etros image with both devices.
	image, err := t.make(o)
	if err != nil {
		return fmt.Errorf("making Etros with the R36S: %w (nothing on the card was changed)", err)
	}

	// 3. Onto the card, then everything back. Each tried twice.
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		if attempt == 2 {
			note(cYellow + "That didn't work: " + lastErr.Error() + cReset)
			note("Trying again...")
		}
		heading("PUTTING ETROS WITH THE R36S ON THE CARD")
		rep := newTermReporter()
		lastErr = filesInstall(image, disk, rep)
		rep.end()
		if lastErr != nil {
			continue
		}
		share, closeShare, err := openShare(disk)
		if err != nil {
			lastErr = err
			continue
		}
		rep = newTermReporter()
		rep.Stage("Putting your games, saves and settings back")
		_, _, lastErr = copyTree(backup, share, func(string) bool { return false }, rep)
		rep.end()
		syncDir(share)
		closeShare()
		if lastErr == nil {
			os.RemoveAll(backup)
			return nil
		}
	}
	return fmt.Errorf("%w. Your games, saves and settings are safe in %s", lastErr, backup)
}

// Proton for the Windows folder: GE-Proton, the community Proton most
// people use outside Steam. About 500 MB, so it goes on SHARE
// (system/etros/downloads), where etros-launch-windows unpacks it.
const protonFallback = "GE-Proton10-34"

func latestProtonURL() string {
	tag := protonFallback
	req, err := http.NewRequest("GET", "https://api.github.com/repos/GloriousEggroll/proton-ge-custom/releases/latest", nil)
	if err == nil {
		req.Header.Set("User-Agent", userAgent)
		if resp, err := client.Do(req); err == nil {
			var v struct {
				Tag string `json:"tag_name"`
			}
			if resp.StatusCode == http.StatusOK && json.NewDecoder(resp.Body).Decode(&v) == nil && strings.HasPrefix(v.Tag, "GE-Proton") {
				tag = v.Tag
			}
			resp.Body.Close()
		}
	}
	return fmt.Sprintf("https://github.com/GloriousEggroll/proton-ge-custom/releases/download/%s/%s.tar.gz", tag, tag)
}

// cardHasProton tells whether the card's SHARE has Proton (unpacked, or
// waiting to be).
func cardHasProton(share string) bool {
	if _, err := os.Stat(filepath.Join(share, "system", "etros", "proton", "current", "proton")); err == nil {
		return true
	}
	have, _ := filepath.Glob(filepath.Join(share, "system", "etros", "downloads", "GE-Proton*.tar.gz"))
	return len(have) > 0
}

// offerProton adds Proton to the card at dir if it hasn't got it, when asked.
func (t *tui) offerProton(dir string) {
	disk, err := cardDisk(dir)
	if err != nil {
		return
	}
	share, closeShare, err := openShare(disk)
	if err != nil {
		logf("proton: %v", err)
		return
	}
	defer closeShare()
	if cardHasProton(share) {
		fmt.Printf("  %s✔%s Proton, for Windows programs on the Pi 5\n", cCyan, cReset)
		return
	}
	heading("PROTON FOR WINDOWS PROGRAMS")
	note("Windows programs try Wine, Oliver, Steam, then Proton, then Box64. Proton isn't")
	note("on this card yet. Etros Updater can download it (about 500 MB) and put it on.")
	if !t.yes("Add Proton?", true) {
		return
	}
	rep := newTermReporter()
	defer rep.end()
	rep.Stage("Getting Proton")
	url := latestProtonURL()
	p, err := fetch(url, defaultCache(), "proton", rep.Progress, rep.Log)
	if err != nil {
		rep.end()
		note(cYellow + "Couldn't download Proton: " + err.Error() + cReset)
		return
	}
	dest := filepath.Join(share, "system", "etros", "downloads", path.Base(url))
	rep.Stage("Putting Proton on the card")
	if err := copyFile(p, dest, rep); err != nil {
		rep.end()
		note(cYellow + "Couldn't put Proton on the card: " + err.Error() + cReset)
		return
	}
	syncDir(filepath.Dir(dest))
	rep.Log("added " + path.Base(url))
}
