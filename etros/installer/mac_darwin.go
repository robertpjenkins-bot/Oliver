//go:build darwin

package main

// Etros on an Intel Mac, from macOS itself (for Macs without Target Disk
// Mode, like the 12-inch MacBook): every drive and partition is listed, and
// the one chosen becomes BATOCERA (FAT32, Batocera for PCs with the Etros
// layer) and SHARE right after it (ExFAT here, as macOS can't make ext4;
// Etros makes it ext4 the first time it starts, see S11aetros-share). It can
// be a partition (Boot Camp, or one made for Etros), free room in macOS, or
// a whole USB drive, which then starts Etros on any Intel Mac. macOS's own
// diskutil does the partitions; if it wants the Mac's password for that,
// it's macOS asking, in its own password window. It all shows in Safari
// (macweb.go): no Terminal.
//
// Ollie OS can't be made in macOS: its Arch Linux goes on BATOCERA, with
// the Wi-Fi this Mac uses, and Etros makes Ollie OS the first time it
// starts (etros-setup-ollie-pc).

import (
	"os/exec"
	"runtime"
)

func macInstallSupported() bool { return runtime.GOARCH == "amd64" }

func runMacInstaller(o Options) error {
	return serveMacWeb(darwinMac{}, func(url string) error {
		if exec.Command("open", "-a", "Safari", url).Run() == nil {
			return nil
		}
		return exec.Command("open", url).Run()
	})
}

// macLoop (Etros Maker's menu on a Mac) opens the installer in Safari.
func (t *tui) macLoop() bool {
	if runtime.GOARCH != "amd64" {
		note(cRed + "This is for Intel Macs; Macs with Apple chips can't start Etros." + cReset)
		return false
	}
	note("Etros Mac Installer opens in Safari.")
	if err := runMacInstaller(t.opts); err != nil {
		note(cRed + err.Error() + cReset)
		return false
	}
	return true
}
