//go:build linux

package main

import (
	"bufio"
	"bytes"
	"os"
	"strconv"
	"strings"

	"github.com/godbus/dbus/v5"
)

// mountedDirs lists where drives are open. Any BATOCERA drive that isn't
// open yet is opened first, through UDisks (no password).
func mountedDirs() []string {
	openBatoceraDrives()
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) >= 3 && fields[2] == "vfat" {
			out = append(out, unescapeMount(fields[1]))
		}
	}
	return out
}

// /proc/mounts writes spaces and the like as \040.
func unescapeMount(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			if n, err := strconv.ParseUint(s[i+1:i+4], 8, 8); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func openBatoceraDrives() {
	conn, err := dbus.SystemBus()
	if err != nil {
		return
	}
	var objs map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	err = conn.Object(udisks, "/org/freedesktop/UDisks2").Call(
		"org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&objs)
	if err != nil {
		return
	}
	for path, ifaces := range objs {
		block, ok := ifaces[udisks+".Block"]
		fsys, isFS := ifaces[udisks+".Filesystem"]
		if !ok || !isFS {
			continue
		}
		label, _ := block["IdLabel"].Value().(string)
		if !strings.EqualFold(label, "BATOCERA") {
			continue
		}
		if mp, _ := fsys["MountPoints"].Value().([][]byte); len(mp) > 0 && len(bytes.TrimRight(mp[0], "\x00")) > 0 {
			continue
		}
		var dir string
		if err := conn.Object(udisks, path).Call(udisks+".Filesystem.Mount", 0, noOpts()).Store(&dir); err == nil {
			logf("opened %s at %s", path, dir)
		} else {
			logf("could not open %s: %v", path, err)
		}
	}
}

// closeCard closes the BATOCERA drive at dir so the card can come out.
func closeCard(dir string) bool {
	conn, err := dbus.SystemBus()
	if err != nil {
		return false
	}
	var objs map[dbus.ObjectPath]map[string]map[string]dbus.Variant
	if conn.Object(udisks, "/org/freedesktop/UDisks2").Call(
		"org.freedesktop.DBus.ObjectManager.GetManagedObjects", 0).Store(&objs) != nil {
		return false
	}
	for path, ifaces := range objs {
		fsys, ok := ifaces[udisks+".Filesystem"]
		if !ok {
			continue
		}
		mps, _ := fsys["MountPoints"].Value().([][]byte)
		for _, mp := range mps {
			if string(bytes.TrimRight(mp, "\x00")) == dir {
				return conn.Object(udisks, path).Call(udisks+".Filesystem.Unmount", 0, noOpts()).Err == nil
			}
		}
	}
	return false
}
