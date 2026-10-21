//go:build linux

package main

import "testing"

func TestIsArchTarball(t *testing.T) {
	for name, want := range map[string]bool{
		"/home/deck/Downloads/ArchLinuxARM-aarch64-latest.tar.gz": true,
		"ArchLinuxARM-rpi-aarch64-latest.tar.gz":                  true,
		"ArchLinuxARM-armv7-latest.tar.gz":                        false,
		"archlinux-2026.10.01-x86_64.iso":                         false,
		"2026-10-01-raspios-trixie-arm64.img.xz":                  false,
	} {
		if got := isArchTarball(name); got != want {
			t.Errorf("%s: got %v", name, got)
		}
	}
}
