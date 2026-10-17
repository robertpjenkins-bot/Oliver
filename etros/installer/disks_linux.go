//go:build linux

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/godbus/dbus/v5"
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
	// For tests: a file standing in for a removable drive.
	if p := os.Getenv("ETROS_TEST_DISK"); p != "" {
		st, err := os.Stat(p)
		if err != nil {
			return nil, err
		}
		d := Disk{ID: p, Path: p, Name: "Test drive", Size: st.Size()}
		describe(&d)
		return []Disk{d}, nil
	}
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
	return true, ""
}

const udisks = "org.freedesktop.UDisks2"

// udisksPath is the UDisks object for a device such as /dev/sdb.
func udisksPath(dev string) dbus.ObjectPath {
	return dbus.ObjectPath("/org/freedesktop/UDisks2/block_devices/" + strings.TrimPrefix(dev, "/dev/"))
}

// udisksCall calls a UDisks method, letting the desktop ask for a password
// only if its rules require one for this device.
func udisksCall(conn *dbus.Conn, dev, method string, args ...any) *dbus.Call {
	return conn.Object(udisks, udisksPath(dev)).Call(method, dbus.FlagAllowInteractiveAuthorization, args...)
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
	var children []string
	devs, _ := lsblk()
	for _, d := range devs {
		if d.Path == id {
			for _, c := range d.Children {
				if len(mounts(c)) > 0 {
					children = append(children, c.Path)
				}
			}
		}
	}

	if os.Geteuid() == 0 || os.Getenv("ETROS_TEST_DISK") != "" {
		for _, c := range children {
			if out, err := exec.Command("umount", c).CombinedOutput(); err != nil {
				return nil, fmt.Errorf("could not unmount %s: %s", c, out)
			}
		}
		f, err := os.OpenFile(id, os.O_RDWR, 0)
		if err != nil {
			return nil, err
		}
		return &diskTarget{File: f, size: disk.Size}, nil
	}

	// As yourself: through UDisks, as GNOME Disks and USB writers do.
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, errNeedsAdmin{fmt.Errorf("UDisks isn't available: %w", err)}
	}
	for _, c := range children {
		if err := udisksCall(conn, c, udisks+".Filesystem.Unmount", map[string]dbus.Variant{}).Err; err != nil {
			return nil, fmt.Errorf("could not unmount %s: %w", c, err)
		}
	}
	var fd dbus.UnixFD
	opts := map[string]dbus.Variant{}
	if err := udisksCall(conn, id, udisks+".Block.OpenDevice", "rw", opts).Store(&fd); err != nil {
		logf("UDisks OpenDevice: %v", err)
		return nil, errNeedsAdmin{fmt.Errorf("this computer won't let you write to %s yourself: %w", id, err)}
	}
	return &diskTarget{File: os.NewFile(uintptr(fd), id), size: disk.Size, udisks: conn}, nil
}

func (d *diskTarget) Finish() error {
	err := d.Sync()
	if cerr := d.Close(); err == nil {
		err = cerr
	}
	if d.udisks != nil {
		// Have the system read the new partition table.
		udisksCall(d.udisks, d.Name(), udisks+".Block.Rescan", map[string]dbus.Variant{})
	} else {
		exec.Command("blockdev", "--rereadpt", d.Name()).Run()
	}
	return err
}
