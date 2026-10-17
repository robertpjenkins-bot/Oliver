package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Set ETROS_TEST_CARD_DIR to the files of an older Etros card's BATOCERA
// drive; the fixed initramfs is left there for the boot test.
func TestFixCard(t *testing.T) {
	dir := os.Getenv("ETROS_TEST_CARD_DIR")
	if dir == "" {
		t.Skip("set ETROS_TEST_CARD_DIR")
	}
	os.WriteFile(filepath.Join(dir, "etros", "last-ollie-boot.txt"), []byte("x"), 0o644)
	if err := fixCard(dir, nopReporter{}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "etros", "etros.squashfs")); !bytes.Equal(b, etrosLayer) {
		t.Error("etros.squashfs not updated")
	}
	in, _ := os.ReadFile(filepath.Join(dir, "etros", "pi5", "initrd.lz4"))
	if _, changed, err := UpdateInitrdHook(in); err != nil || changed {
		t.Errorf("initrd hook not current: changed=%v err=%v", changed, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "etros", "last-ollie-boot.txt")); err == nil {
		t.Error("old Ollie OS report left behind")
	}
	// Running it again changes nothing.
	if err := fixCard(dir, nopReporter{}); err != nil {
		t.Fatal(err)
	}
}
