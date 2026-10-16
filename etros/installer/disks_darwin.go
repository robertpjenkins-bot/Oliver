//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var diskLine = regexp.MustCompile(`(?m)^(/dev/disk[0-9]+) \(external, physical\):`)

func plistValue(plist, key string) string {
	re := regexp.MustCompile(`<key>` + key + `</key>\s*<(?:string|integer)>([^<]*)<`)
	if m := re.FindStringSubmatch(plist); m != nil {
		return m[1]
	}
	return ""
}

func listDisks() ([]Disk, error) {
	out, err := exec.Command("diskutil", "list", "external", "physical").Output()
	if err != nil {
		return nil, fmt.Errorf("diskutil: %w", err)
	}
	var disks []Disk
	for _, m := range diskLine.FindAllStringSubmatch(string(out), -1) {
		info, err := exec.Command("diskutil", "info", "-plist", m[1]).Output()
		if err != nil {
			continue
		}
		size, _ := strconv.ParseInt(plistValue(string(info), "TotalSize"), 10, 64)
		name := strings.TrimSpace(plistValue(string(info), "MediaName"))
		if name == "" {
			name = "External drive"
		}
		d := Disk{ID: m[1], Path: m[1], Name: name, Size: size}
		describe(&d)
		disks = append(disks, d)
	}
	return disks, nil
}

func canWriteDisks() (bool, string) {
	if os.Geteuid() != 0 {
		return false, "Writing to a card needs administrator rights."
	}
	return true, ""
}

func openDisk(id string) (*diskTarget, error) {
	disks, err := listDisks()
	if err != nil {
		return nil, err
	}
	var disk *Disk
	for i := range disks {
		if disks[i].ID == id {
			disk = &disks[i]
		}
	}
	if disk == nil {
		return nil, fmt.Errorf("%s is not an external drive", id)
	}
	if out, err := exec.Command("diskutil", "unmountDisk", id).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("could not unmount %s: %s", id, out)
	}
	// The raw device is much faster than the buffered one.
	f, err := os.OpenFile(strings.Replace(id, "/dev/disk", "/dev/rdisk", 1), os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	return &diskTarget{File: f, size: disk.Size}, nil
}

func (d *diskTarget) Finish() error {
	d.Sync()
	err := d.Close()
	exec.Command("diskutil", "eject", strings.Replace(d.Name(), "/dev/rdisk", "/dev/disk", 1)).Run()
	return err
}
