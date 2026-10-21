//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFindLocal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	vols := filepath.Join(home, "Volumes")
	t.Setenv("ETROS_TEST_VOLUMES", vols)
	write := func(rel string, size int64, age time.Duration) string {
		p := filepath.Join(home, rel)
		os.MkdirAll(filepath.Dir(p), 0o755)
		f, _ := os.Create(p)
		f.Truncate(size)
		f.Close()
		when := time.Now().Add(-age)
		os.Chtimes(p, when, when)
		return p
	}
	if p := findLocal(localBatocera); p != "" {
		t.Fatalf("found %s in an empty home", p)
	}
	// On a USB drive from the Steam Machine, two folders down.
	usb := write("Volumes/STICK/Etros/batocera-x86_64-43-20261001-boot.tar.xz", 600<<20, time.Hour)
	if p := findLocal(localBatocera); p != usb {
		t.Errorf("USB drive: got %q", p)
	}
	// A newer one on the Desktop wins; a half-downloaded, a too-small and a
	// Pi 5 one don't count.
	desk := write("Desktop/boot-x86_64.tar.xz", 600<<20, time.Minute)
	write("Downloads/batocera-x86_64-boot.tar.xz.part", 600<<20, 0)
	write("Downloads/x86_64-boot-tiny.tar.xz", 1<<20, 0)
	write("Downloads/batocera-bcm2712-boot.tar.xz", 600<<20, 0)
	write("Library/Caches/batocera-x86_64-boot.tar.xz", 600<<20, 0)
	if p := findLocal(localBatocera); p != desk {
		t.Errorf("newest: got %q, want %q", p, desk)
	}
	// As batocera.org names it (boot.tar.xz): used when there's nothing surer.
	os.Remove(usb)
	os.Remove(desk)
	plain := write("Volumes/STICK/boot.tar.xz", 600<<20, 0)
	if p := findLocal(localBatocera); p != plain {
		t.Errorf("plain boot.tar.xz: got %q, want %q", p, plain)
	}
	sure := write("Documents/batocera-x86_64-boot.tar.xz", 600<<20, time.Hour)
	if p := findLocal(localBatocera); p != sure {
		t.Errorf("an x86_64 one goes before a plain one: got %q", p)
	}
	// What Etros downloaded before, in its own cache.
	arch := write(".cache/etros-installer/archlinux-bootstrap-1a2b3c4d.zst", 160<<20, 0)
	if p := findLocal(localArch); p != arch {
		t.Errorf("Arch Linux from the cache: got %q", p)
	}
}
