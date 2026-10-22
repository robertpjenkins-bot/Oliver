//go:build linux

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godbus/dbus/v5"
)

// The Etros USB drive's partitions, made through UDisks on a drive (a loop
// device) given in ETROS_TEST_UDISKS_DEV.
func TestMacUSBPartitions(t *testing.T) {
	dev := os.Getenv("ETROS_TEST_UDISKS_DEV")
	if dev == "" {
		t.Skip("set ETROS_TEST_UDISKS_DEV to a drive UDisks may erase")
	}
	conn, err := dbus.SystemBus()
	if err != nil {
		t.Fatal(err)
	}
	u := &udisksDrive{conn: conn, dev: dev}
	if err := u.call(udisksPath(dev), "Block.Format", "gpt", noOpts()).Err; err != nil {
		t.Fatal(err)
	}
	boot, share, uuid, closeAll, err := udisksEtrosPartitions(u, dev, mib, macUSBLabel, nopReporter{})
	if err != nil {
		t.Fatal(err)
	}
	defer closeAll()
	if uuid == "" {
		t.Error("no PARTUUID for the boot partition")
	}
	os.MkdirAll(filepath.Join(boot, "EFI", "BOOT"), 0o755)
	os.MkdirAll(filepath.Join(boot, "etros"), 0o755)
	cfg := filepath.Join(boot, "EFI", "BOOT", "grub.cfg")
	os.WriteFile(cfg, []byte("linux /boot/linux label=BATOCERA console=tty3\n"), 0o644)
	if err := usbBootFiles(boot); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(cfg)
	if !strings.Contains(string(b), "label=ETROSUSB ") {
		t.Errorf("grub.cfg: %s", b)
	}
	if _, err := os.Stat(filepath.Join(boot, "etros", "usb.txt")); err != nil {
		t.Error(err)
	}
	if err := os.WriteFile(filepath.Join(share, "x"), []byte("x"), 0o644); err != nil {
		t.Error(err)
	}
}
