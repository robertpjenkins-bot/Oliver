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

// Where Ollie OS for ARM is published (.github/workflows/ollie-os-arm.yml).
// The repository is private, so it's downloaded in a browser, logged in.
const ollieArmRelease = "https://github.com/robertpjenkins-bot/Oliver/releases/tag/ollie-os-arm-newest"

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
		if len(f.Wine) > 0 {
			o.Sources.Wine = f.Wine[0].Path
		}

		heading("OPTIONS")
		if len(f.PCOllie) > 0 {
			note(cYellow + "Found " + baseName(f.PCOllie[0].Path) + ": that's Ollie OS for PCs." + cReset)
			note("The Raspberry Pi 5 and R36S have ARM processors and can't run PC systems, so")
			note("Etros needs Ollie OS for ARM (ollie-os-arm64.img.xz) instead.")
		}
		if len(f.Ollie) == 0 {
			note("Ollie OS (the desktop for downloading ROMs) is made from Raspberry Pi OS.")
			note("It isn't in Downloads yet, so Etros Maker downloads it (1.3 GB, no password).")
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
		fmt.Printf("  %s2%s  Etros Updater %s(update my Etros card, keeps your games and saves)%s\n", cYellow, cReset, cDim, cReset)
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

// fixLoop is the Etros Updater: pick an Etros card on this computer and
// bring it up to date, trying once more if that fails. It returns false to
// go back to the menu.
func (t *tui) fixLoop() bool {
	for {
		heading("ETROS UPDATER: SELECT A DRIVE")
		cards := etrosCardDirs()
		if len(cards) == 0 {
			note("No Etros card found. Put the card in (a card reader is fine), wait a few")
			note("seconds, then press Enter.")
		}
		for i, c := range cards {
			fmt.Printf("  %s%d%s  Etros card at %s\n", cYellow, i+1, cReset, c)
		}
		fmt.Printf("  %sr%s  Look again    %sb%s  Back\n", cYellow, cReset, cYellow, cReset)
		a := strings.ToLower(t.ask("Which drive? (its number, r or b)"))
		if a == "b" || a == "q" {
			return false
		}
		var n int
		if _, err := fmt.Sscan(a, &n); err != nil || n < 1 || n > len(cards) {
			continue
		}
		dir := cards[n-1]
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

		// A card made without the R36S: offer to add it.
		if !cardHasR36S(dir) {
			heading("THIS CARD HAS NO R36S PART")
			note("It was made for the Raspberry Pi 5 only, so the R36S can't start from it")
			note("(that's the black screen). Etros Updater can add the R36S: it copies your games,")
			note("saves and settings to this computer, makes the card again with both devices,")
			note("and puts everything back. It downloads Batocera for the R36S if it isn't in")
			note("Downloads. No password needed.")
			if !addR36SSupported() {
				note(cYellow + "On this computer that isn't possible: make a new card with Etros Maker (option 1)." + cReset)
			} else if t.yes("Add the R36S to this card?", true) {
				if err := t.remakeWithR36S(dir); err != nil {
					logf("ERROR: adding the R36S: %v", err)
					fmt.Printf("\n%s  UPDATE FAILED%s\n\n", cRed, cReset)
					note("Adding the R36S didn't work: " + err.Error())
					t.ask("Press Enter to go back")
					return false
				}
				fmt.Printf("\n%s  CARD UPDATED, WITH THE R36S%s\n\n", cYellow, cReset)
				note("1. Put the card in your Raspberry Pi 5 and switch it on once, until Etros")
				note("   shows. That puts the R36S's start-up code in place.")
				note("2. Then the same card starts in your R36S too.")
				note("3. Your games, saves and settings are back on the card.")
				t.ask("Press Enter to finish")
				return true
			}
		}

		var err error
		for attempt := 1; attempt <= 2; attempt++ {
			if attempt == 2 {
				note(cYellow + "That didn't work: " + err.Error() + cReset)
				note("Trying again...")
			}
			rep := newTermReporter()
			err = fixCard(dir, rep)
			rep.end()
			if err == nil {
				break
			}
			logf("update attempt %d: %v", attempt, err)
		}
		if err != nil {
			logf("ERROR: update failed: %v", err)
			fmt.Printf("\n%s  UPDATE FAILED%s\n\n", cRed, cReset)
			note("Etros Updater tried twice and couldn't update the card:")
			note("  " + err.Error())
			note("Nothing on the card was lost: what couldn't be updated is still the old version.")
			note("Check the card isn't locked (the little switch on its side), or make a new card")
			note("with Etros Maker. This is also saved in etros-installer-log.txt in Downloads.")
			t.ask("Press Enter to go back")
			return false
		}
		closed := closeCard(dir)
		fmt.Printf("\n%s  CARD UPDATED%s\n\n", cYellow, cReset)
		if closed {
			note("1. Take the card out and put it back in your Raspberry Pi 5 or R36S.")
		} else {
			note("1. Eject the card (in the file manager), then put it back in your Raspberry Pi 5 or R36S.")
		}
		note("2. Your games, saves and settings are all still there.")
		t.ask("Press Enter to finish")
		return true
	}
}

// runUpdater is the Etros Updater on its own: straight to choosing a card.
func runUpdater(o Options) error {
	t := &tui{in: bufio.NewReader(os.Stdin), opts: o}
	showTitle("ETROS UPDATER")
	note("Brings an Etros card up to date: the newest Etros, Ollie OS's settings and")
	note("My ROMs. Your games, saves and settings stay. " + cBold + "No password needed." + cReset)
	t.fixLoop()
	return nil
}
