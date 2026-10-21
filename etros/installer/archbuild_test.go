//go:build linux

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Making Ollie OS from a stand-in Arch Linux ARM tarball (ETROS_ARCH_TARBALL:
// aarch64 busybox with pacman and the rest as stubs), as a person without a
// password would.
func TestBuildArchHere(t *testing.T) {
	tarball := os.Getenv("ETROS_ARCH_TARBALL")
	if tarball == "" {
		t.Skip("set ETROS_ARCH_TARBALL to a stand-in Arch Linux ARM tarball")
	}
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("ETROS_TEST_NO_ROOM_CHECK", "1")
	share := t.TempDir()
	if err := buildArchHere(tarball, share, nopReporter{}); err != nil {
		t.Fatal(err)
	}
	img := filepath.Join(share, "etros", "ollie-rootfs.img")
	out, err := exec.Command("debugfs", "-R", "cat /etc/ollie-os-arch", img).Output()
	if err != nil || len(out) == 0 {
		t.Fatalf("no /etc/ollie-os-arch in Ollie OS: %v", err)
	}
	if _, err := os.Stat(img + ".new"); err == nil {
		t.Error("left " + img + ".new behind")
	}
}
