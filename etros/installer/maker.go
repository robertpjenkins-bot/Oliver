package main

// Etros Maker: the installer without the card writing, so it never needs a
// password. It turns the Batocera for Raspberry Pi 5 in Downloads into an
// Etros image, also in Downloads, for Raspberry Pi Imager to put on a card.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// mode is "maker" in Etros Maker builds (set by build-dist.sh).
var mode = "installer"

const batoceraDownload = "https://batocera.org/download"

func runMaker(o Options) error {
	t := &tui{in: bufio.NewReader(os.Stdin), opts: o}
	showBanner()
	note("Etros Maker turns Batocera for the " + cBold + "Raspberry Pi 5" + cReset + " into Etros:")
	note("the Super Metroid theme, the WELCOME TO ETROS screen, Ollie OS and your games.")
	note("It makes an image file in Downloads. " + cBold + "No password needed." + cReset)

	for {
		heading("LOOKING IN DOWNLOADS")
		f := scan()
		logf("scan: pi5=%v r36s=%v ollie=%v games=%v", f.Pi5, f.R36S, f.Ollie, f.Games)
		line := func(what string, list []foundFile, missing string) {
			if len(list) > 0 {
				fmt.Printf("  %s✔%s %-22s %s %s(%s)%s\n", cCyan, cReset, what, baseName(list[0].Path), cDim, gbs(list[0].Size), cReset)
			} else {
				fmt.Printf("  %s·%s %-22s %s%s%s\n", cDim, cReset, what, cDim, missing, cReset)
			}
		}
		line("Batocera, Pi 5", f.Pi5, "not found")
		line("Ollie OS", f.Ollie, "not found, can download Raspberry Pi OS")
		if len(f.Games) > 0 {
			fmt.Printf("  %s✔%s %-22s %s\n", cCyan, cReset, "Your games", f.Games[0])
		}
		if len(f.R36S) > 0 {
			line("Batocera, R36S", f.R36S, "")
		}

		if len(f.Pi5) == 0 {
			heading("BATOCERA FOR THE RASPBERRY PI 5 ISN'T IN DOWNLOADS")
			note("Get it from " + cCyan + batoceraDownload + cReset + " (choose Raspberry Pi 5),")
			note("or let Etros Maker download it now (about 3 GB).")
			fmt.Printf("  %s1%s  Download it now\n  %s2%s  I've put it in Downloads, look again\n  %sq%s  Quit\n", cYellow, cReset, cYellow, cReset, cYellow, cReset)
			switch t.ask("Choose") {
			case "1":
			case "q", "Q":
				return nil
			default:
				continue
			}
		} else {
			o.Sources.Pi5Boot = f.Pi5[0].Path
		}
		if len(f.Ollie) > 0 {
			o.Sources.RPiOS = f.Ollie[0].Path
		}

		heading("OPTIONS")
		if len(f.Ollie) == 0 {
			note("Ollie OS (the desktop for downloading ROMs, with Bobby on the wallpaper)")
			note("needs Raspberry Pi OS: a 1.3 GB download, no password.")
		}
		o.Ollie = t.yes("Include Ollie OS?", true)
		o.GamesDir = ""
		if len(f.Games) > 0 && t.yes("Add your games from "+f.Games[0]+"?", true) {
			o.GamesDir = f.Games[0]
		}
		o.NoR36S = true
		if len(f.R36S) > 0 && t.yes("Batocera for the R36S is here too. Make the card work in the R36S as well?", true) {
			o.NoR36S = false
			o.Sources.R36SBoot = f.R36S[0].Path
		}

		path, err := t.make(o)
		if err != nil {
			logf("ERROR: %v", err)
			fmt.Printf("\n%sSOMETHING WENT WRONG:%s %v\n", cRed, cReset, err)
			note("This is also saved in etros-installer-log.txt in your Downloads.")
			if t.ask("Press Enter to try again, or q to quit") == "q" {
				return nil
			}
			continue
		}

		fmt.Printf("\n%s  MISSION COMPLETE%s\n\n", cYellow, cReset)
		fmt.Printf("  Your Etros image: %s%s%s\n\n", cBold, path, cReset)
		note("Put it on a microSD card (32 GB or bigger) with Raspberry Pi Imager:")
		note("  1. Open Raspberry Pi Imager (raspberrypi.com/software).")
		note("  2. Device: Raspberry Pi 5.  Operating system: Use custom, then pick the file above.")
		note("  3. Storage: your SD card.  Then Next, and No to the settings question.")
		note("balenaEtcher works too: Flash from file.")
		t.ask("Press Enter to finish")
		return nil
	}
}

// make builds the image into Downloads.
func (t *tui) make(o Options) (string, error) {
	heading("MAKING ETROS")
	rep := newTermReporter()
	defer rep.end()
	card, err := Plan(o, rep)
	if err != nil {
		return "", err
	}
	name := "etros-pi5-" + time.Now().Format("2006-01-02") + ".img.gz"
	if !o.NoR36S {
		name = "etros-pi5-r36s-" + time.Now().Format("2006-01-02") + ".img.gz"
	}
	path := filepath.Join(downloadsDir(), name)
	out, err := newGzTarget(path+".part", card.Size)
	if err != nil {
		return "", err
	}
	if err := card.Write(out, rep, false); err != nil {
		out.Finish()
		os.Remove(path + ".part")
		return "", err
	}
	if err := out.Finish(); err != nil {
		return "", err
	}
	return path, os.Rename(path+".part", path)
}
