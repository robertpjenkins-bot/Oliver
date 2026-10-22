package main

// Etros Mac Installer in the Terminal window, with no Safari page: the
// same checks, choices and work as the page (macweb.go), asked and shown
// as text. Start Etros Installer.command runs it this way.

import (
	"errors"
	"fmt"
	"strings"
)

// macText puts Etros on a Mac, asking in the terminal.
func (t *tui) macText(b macBackend) error {
	showTitle("ETROS MAC INSTALLER")
	note("Puts Etros and Ollie OS on this Mac. macOS and your files stay.")

	heading("CHECKS")
	note("Checking this Mac and the internet...")
	blocked := ""
	for _, c := range b.checks() {
		mark := cCyan + "✔" + cReset
		if !c.OK {
			mark = cYellow + "!" + cReset
			if c.Blocking {
				mark = cRed + "✘" + cReset
				if blocked == "" {
					blocked = c.Name + ": " + c.Why
				}
			}
		}
		fmt.Printf("  %s %s\n", mark, c.Name)
		if c.Why != "" {
			fmt.Printf("      %s%s%s\n", cDim, c.Why, cReset)
		}
	}
	if blocked != "" {
		return errors.New("can't go ahead yet: " + blocked)
	}

	for {
		ts, err := b.targets()
		if err != nil {
			return err
		}
		heading("WHERE SHOULD ETROS GO?")
		var usable []macTarget
		shown := ""
		for _, x := range ts {
			if x.disk.id != shown {
				shown = x.disk.id
				where := "USB / external"
				if x.disk.internal {
					where = "this Mac's own drive"
				}
				fmt.Printf("\n  %s%s%s (%s, %s, %s)\n", cBold, orDefault(x.disk.name, "Drive"), cReset, x.disk.id, where, gbs(x.disk.size))
			}
			id, size := x.part.id, x.part.size
			if x.kind == "whole" {
				id, size = x.disk.id, x.disk.size
			}
			for _, e := range x.extra {
				size += e.size
			}
			if x.why != "" {
				fmt.Printf("     %s·  %-9s %-26s %8s  %s%s\n", cDim, id, x.what, gbs(size), x.why, cReset)
				continue
			}
			usable = append(usable, x)
			fmt.Printf("  %s%3d%s  %-9s %-26s %8s  %s\n", cYellow, len(usable), cReset, id, x.what, gbs(size), x.action)
		}
		fmt.Println()
		if len(usable) == 0 {
			return errors.New("there's nowhere Etros can go: make BATOCERA and SHARE in Disk Utility (see the READ ME), or plug in a USB drive")
		}
		a := strings.ToLower(t.ask(fmt.Sprintf("Where should Etros go? (1 to %d, r to look again, q to stop)", len(usable))))
		if a == "q" {
			return errors.New("stopped: nothing was changed")
		}
		var n int
		if _, err := fmt.Sscan(a, &n); err != nil || n < 1 || n > len(usable) {
			continue
		}
		x := usable[n-1]

		var etros int64
		switch x.kind {
		case "whole":
			etros = x.disk.size
		case "part", "ready":
			etros = x.part.size
			for _, e := range x.extra {
				etros += e.size
			}
		case "room":
			free := b.roomFree(x.part.id)
			max := free - 20<<30 // leave macOS some room
			if max < macBootSize+macMinShare {
				note(cRed + "macOS hasn't enough free room for Etros." + cReset)
				continue
			}
			def := min(max, 64<<30) >> 30
			for etros == 0 {
				a := t.ask(fmt.Sprintf("How many GB for Etros (%d to %d)? [%d]", macMinGB, max>>30, def))
				n := def
				if a != "" {
					if _, err := fmt.Sscan(a, &n); err != nil {
						continue
					}
				}
				if n<<30 >= macBootSize+macMinShare && n<<30 <= max {
					etros = n << 30
				}
			}
		}

		heading("ETROS GOES ON " + strings.ToUpper(orDefault(x.disk.name, x.disk.id)))
		switch x.kind {
		case "whole":
			note(cBold + "Everything on " + x.disk.id + " will be erased." + cReset + " Then it starts Etros on any Intel Mac.")
		case "part":
			note(cBold + "Everything on " + x.part.id + " (" + x.what + ") will be erased." + cReset)
		case "room":
			note(gbs(etros) + " of macOS's free room becomes Etros. macOS and its files stay.")
		case "ready":
			note("Etros's files go on BATOCERA (" + x.part.id + "). Nothing is erased.")
		}
		ssid, key := "", ""
		if etros >= macBootSize+macMinShare+macOllieRoom {
			note("Ollie OS (Arch Linux with KDE Plasma) is made the first time Etros starts, over Wi-Fi.")
			ssid = b.wifiName()
			if ssid != "" {
				if a := t.ask("Wi-Fi name [" + ssid + "]:"); a != "" {
					ssid = a
				}
			} else {
				ssid = t.ask("Wi-Fi name (Enter to skip):")
			}
			if ssid != "" {
				key = t.ask("Wi-Fi password:")
			}
		} else {
			note(fmt.Sprintf("There isn't room for Ollie OS too (Etros needs %d GB in all for it), so it's Etros on its own.", macOllieGB))
		}
		if x.kind != "ready" {
			note("If macOS wants your Mac password to change the drive, it asks in its own window.")
		}
		if strings.ToLower(t.ask("Type ERASE to go ahead (anything else to choose again):")) != "erase" {
			continue
		}

		rep := newTermReporter()
		err = b.install(x, etros, ssid, key, rep)
		rep.end()
		if err != nil {
			return err
		}
		fmt.Printf("\n%s  ETROS IS ON IT%s\n\n", cYellow, cReset)
		step := 1
		if x.kind == "whole" {
			note("1. Eject the drive in Finder and plug it into the Intel Mac.")
			step++
		}
		note(fmt.Sprintf("%d. Restart the Mac holding Option (⌥) and choose EFI Boot. Started normally, it's macOS.", step))
		note(fmt.Sprintf("%d. The first start takes a little longer: Etros sets up its storage.", step+1))
		if ssid != "" {
			note(fmt.Sprintf("%d. Etros then makes Ollie OS over Wi-Fi (30 to 60 minutes) and says when it's ready:", step+2))
			note("   ETROS → Boot into Ollie OS.")
		}
		note("In Etros, use a game controller; the Mac's own keyboard and trackpad work in Ollie OS.")
		return nil
	}
}
