package main

// Etros Maker: the installer without the card writing, so it never needs a
// password. It turns the Batocera for Raspberry Pi 5 in Downloads into an
// Etros image, also in Downloads, for Raspberry Pi Imager to put on a card.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// mode is "maker" in Etros Maker builds (set by build-dist.sh).
var mode = "installer"

const batoceraDownload = "https://batocera.org/download"

func runMaker(o Options) error {
	t := &tui{in: bufio.NewReader(os.Stdin), opts: o}
	showBanner()
	note("Etros Maker turns Batocera for the " + cBold + "Raspberry Pi 5" + cReset + " into Etros:")
	note("the Super Metroid theme, the WELCOME TO ETROS screen, Ollie OS and your games.")
	note("It makes an image file in Downloads. " + cBold + "No password needed." + cReset)

	if done, err := t.startMenu(); done || err != nil {
		return err
	}

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
		if len(f.PCOllie) > 0 {
			note(cYellow + "Found " + baseName(f.PCOllie[0].Path) + ": that's Ollie OS for PCs." + cReset)
			note("The Raspberry Pi 5 and R36S have ARM processors and can't run PC systems, so")
			note("Etros's Ollie OS is a Raspberry Pi desktop instead (Bobby wallpaper, browser, Downloads).")
		}
		if len(f.Ollie) == 0 {
			note("Ollie OS (the desktop for downloading ROMs, with Bobby on the wallpaper)")
			note("needs a Raspberry Pi system image. Press Enter to download Raspberry Pi OS")
			note("(1.3 GB, no password), or paste a download link to a Raspberry Pi (ARM) image.")
		}
		o.Ollie = t.yes("Include Ollie OS?", true)
		if o.Ollie && len(f.Ollie) == 0 {
			link := strings.Trim(t.ask("Download link (Enter for Raspberry Pi OS):"), `"' `)
			switch {
			case link == "":
				o.Sources.RPiOS = ""
			case isURL(link) || fileExists(link):
				o.Sources.RPiOS = link
			default:
				note(cRed + "That isn't a web address (it should start with https://); using Raspberry Pi OS." + cReset)
				o.Sources.RPiOS = ""
			}
		}
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

		fmt.Printf("\n  %sYour Etros image is ready:%s %s\n", cCyan, cReset, path)
		return t.offerWrite(path)
	}
}

// offerWrite asks whether to put the image on a drive now.
func (t *tui) offerWrite(path string) error {
	heading("PUT IT ON A DRIVE?")
	fmt.Printf("  %s1%s  Yes, choose the SD card or USB drive now\n", cYellow, cReset)
	fmt.Printf("  %s2%s  No, I'll use Raspberry Pi Imager or balenaEtcher later\n", cYellow, cReset)
	for {
		a := strings.ToLower(t.ask("Type 1 or 2:"))
		switch {
		case a == "1" || a == "y" || a == "yes" || a == "":
			return t.writeLoop(path)
		case a == "2" || a == "n" || a == "no" || a == "q":
			t.laterInstructions(path)
			return nil
		}
		note("Please type 1 (put it on a drive now) or 2 (do it later).")
	}
}

// writeLoop writes the image to a drive, offering another go if that fails
// or is stopped, but never just going round in circles.
func (t *tui) writeLoop(path string) error {
	for {
		disk, ok, err := t.pickCard()
		if err == nil && ok {
			if err = t.writeImageTo(path, disk); err == nil {
				return nil
			}
		}
		if err != nil {
			logf("ERROR: %v", err)
			fmt.Printf("\n%sSOMETHING WENT WRONG:%s %v\n", cRed, cReset, err)
			note("This is also saved in etros-installer-log.txt in your Downloads.")
		}
		if !t.yes("Try putting it on a drive again?", true) {
			t.laterInstructions(path)
			return nil
		}
	}
}

func (t *tui) laterInstructions(path string) {
	fmt.Printf("\n%s  MISSION COMPLETE%s\n\n", cYellow, cReset)
	fmt.Printf("  Your Etros image: %s%s%s\n\n", cBold, path, cReset)
	note("Put it on a microSD card (32 GB or bigger) with Raspberry Pi Imager:")
	note("  1. Open Raspberry Pi Imager (raspberrypi.com/software).")
	note("  2. Device: Raspberry Pi 5.  Operating system: Use custom, then pick the file above.")
	note("  3. Storage: your SD card.  Then Next, and No to the settings question.")
	note("balenaEtcher works too: Flash from file.")
	t.ask("Press Enter to finish")
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

// startMenu asks what to do. It returns done=true when there's nothing left
// to do (fixed, written or quit); otherwise a new card is made.
func (t *tui) startMenu() (bool, error) {
	for {
		var current string
		heading("WHAT DO YOU WANT TO DO?")
		fmt.Printf("  %s1%s  Make a new Etros card %s(recommended)%s\n", cYellow, cReset, cCyan, cReset)
		fmt.Printf("  %s2%s  Fix my Etros card %s(quick: updates Etros, keeps your games and saves)%s\n", cYellow, cReset, cDim, cReset)
		for _, e := range scan().Etros {
			if imageIsCurrent(e.Path) {
				current = e.Path
				fmt.Printf("  %s3%s  Put %s on a drive %s(made earlier, up to date)%s\n", cYellow, cReset, baseName(e.Path), cDim, cReset)
				break
			}
			logf("%s was made by an older Etros Maker", e.Path)
		}
		fmt.Printf("  %sq%s  Quit\n", cYellow, cReset)
		switch a := strings.ToLower(t.ask("Type a number or q:")); {
		case a == "1" || a == "":
			return false, nil
		case a == "2":
			if t.fixLoop() {
				return true, nil
			}
		case a == "3" && current != "":
			return true, t.writeLoop(current)
		case a == "q":
			return true, nil
		}
	}
}

// fixLoop finds an Etros card on this computer and updates it. It returns
// false to go back to the menu.
func (t *tui) fixLoop() bool {
	heading("FIX MY ETROS CARD")
	for {
		cards := etrosCardDirs()
		if len(cards) == 0 {
			note("No Etros card found. Put the card in (a card reader is fine), wait a few")
			note("seconds, then press Enter. b goes back.")
			if strings.ToLower(t.ask("Enter to look again, or b:")) == "b" {
				return false
			}
			continue
		}
		dir := cards[0]
		if len(cards) > 1 {
			for i, c := range cards {
				fmt.Printf("  %s%d%s  %s\n", cYellow, i+1, cReset, c)
			}
			var n int
			if _, err := fmt.Sscan(t.ask("Which card?"), &n); err != nil || n < 1 || n > len(cards) {
				continue
			}
			dir = cards[n-1]
		}
		note("Found an Etros card: " + dir)
		if why, err := os.ReadFile(filepath.Join(dir, "etros", "last-ollie-boot.txt")); err == nil {
			note("Last time: " + strings.TrimSpace(string(why)))
		}
		if b, err := os.ReadFile(filepath.Join(dir, "etros", "ollie-log.txt")); err == nil {
			lines := strings.SplitN(string(b), "\n", 3)
			if len(lines) >= 2 {
				note("Ollie OS's last report: " + strings.TrimSpace(lines[1]))
			}
			note("(the whole report is etros/ollie-log.txt on the card's BATOCERA drive)")
		}
		rep := newTermReporter()
		err := fixCard(dir, rep)
		rep.end()
		if err != nil {
			logf("ERROR: %v", err)
			fmt.Printf("\n%sSOMETHING WENT WRONG:%s %v\n", cRed, cReset, err)
			note("Choose 1 in the menu to make a new card instead.")
			return false
		}
		closed := closeCard(dir)
		fmt.Printf("\n%s  CARD FIXED%s\n\n", cYellow, cReset)
		if closed {
			note("1. Take the card out and put it back in your Raspberry Pi 5.")
		} else {
			note("1. Eject the card (in the file manager), then put it back in your Raspberry Pi 5.")
		}
		note("2. ETROS -> Boot into Ollie OS now restarts into Ollie OS.")
		note("3. If Ollie OS still doesn't start, Etros comes back and saves why in")
		note("   etros/last-ollie-boot.txt on the card; run this again and it shows it.")
		note("   If it says the copy is incomplete, choose 1 to make a new card.")
		t.ask("Press Enter to finish")
		return true
	}
}
