//go:build linux

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type lsblkDev struct {
	Name        string     `json:"name"`
	Path        string     `json:"path"`
	Size        int64      `json:"size"`
	Rm          bool       `json:"rm"`
	Hotplug     bool       `json:"hotplug"`
	Tran        string     `json:"tran"`
	Type        string     `json:"type"`
	Model       string     `json:"model"`
	Mountpoints []string   `json:"mountpoints"`
	Children    []lsblkDev `json:"children"`
}

func lsblk() ([]lsblkDev, error) {
	out, err := exec.Command("lsblk", "-J", "-b", "-o", "NAME,PATH,SIZE,RM,HOTPLUG,TRAN,TYPE,MODEL,MOUNTPOINTS").Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk: %w", err)
	}
	var v struct {
		Blockdevices []lsblkDev `json:"blockdevices"`
	}
	err = json.Unmarshal(out, &v)
	return v.Blockdevices, err
}

func mounts(d lsblkDev) []string {
	var out []string
	for _, m := range d.Mountpoints {
		if m != "" {
			out = append(out, m)
		}
	}
	for _, c := range d.Children {
		out = append(out, mounts(c)...)
	}
	return out
}

func listDisks() ([]Disk, error) {
	devs, err := lsblk()
	if err != nil {
		return nil, err
	}
	var out []Disk
	for _, d := range devs {
		if d.Type != "disk" || d.Size == 0 {
			continue
		}
		if !(d.Rm || d.Hotplug || d.Tran == "usb" || d.Tran == "mmc" || strings.HasPrefix(d.Name, "mmcblk")) {
			continue
		}
		system := false
		for _, m := range mounts(d) {
			if m == "/" || m == "/boot" || m == "/home" || m == "/usr" || m == "[SWAP]" {
				system = true
			}
		}
		if system {
			continue
		}
		name := strings.TrimSpace(d.Model)
		if name == "" {
			name = "Removable drive"
		}
		disk := Disk{ID: d.Path, Path: d.Path, Name: name, Size: d.Size}
		describe(&disk)
		out = append(out, disk)
	}
	return out, nil
}

func canWriteDisks() (bool, string) {
	if os.Geteuid() != 0 {
		return false, "Run the installer with sudo to write to a card. You can still save an image file."
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
		return nil, fmt.Errorf("%s is not a removable drive", id)
	}
	devs, _ := lsblk()
	for _, d := range devs {
		if d.Path != id {
			continue
		}
		for _, c := range d.Children {
			if len(mounts(c)) > 0 {
				if out, err := exec.Command("umount", c.Path).CombinedOutput(); err != nil {
					return nil, fmt.Errorf("could not unmount %s: %s", c.Path, out)
				}
			}
		}
	}
	f, err := os.OpenFile(id, os.O_RDWR|os.O_EXCL, 0)
	if err != nil {
		return nil, err
	}
	return &diskTarget{File: f, size: disk.Size}, nil
}

func (d *diskTarget) Finish() error {
	if err := d.Sync(); err != nil {
		return err
	}
	exec.Command("blockdev", "--rereadpt", d.Name()).Run()
	return d.Close()
}
