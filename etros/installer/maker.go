package main

// Etros Maker: the installer without the card writing, so it never needs a
// password. It turns the Batocera for Raspberry Pi 5 in Downloads into an
// Etros image, also in Downloads, for Raspberry Pi Imager to put on a card.

import (
	"bufio"
	"bytes"
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
		archOK := filesInstallSupported()
		if archOK {
			note("Ollie OS (the desktop for downloading ROMs) is Arch Linux with KDE Plasma.")
			note("This computer makes it from Arch Linux ARM after writing the card (30 to 60 minutes).")
		} else if len(f.Ollie) == 0 {
			note("Ollie OS (the desktop for downloading ROMs) is made from Raspberry Pi OS.")
			note("It isn't in Downloads yet, so Etros Maker downloads it (1.3 GB, no password).")
		}
		o.Ollie = t.yes("Include Ollie OS?", true)
		o.OllieArch = false
		t.archTarball = ""
		if o.Ollie && archOK {
			p, err := t.getArchTarball()
			if err != nil {
				note(cRed + "Couldn't get Arch Linux ARM: " + err.Error() + cReset)
				continue
			}
			o.OllieArch, t.archTarball = true, p
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
			showOllieReport(string(b))
		}

		if cardHasR36S(dir) {
			t.reportR36S(dir)
		}

		// A card set to start Windows 11 by an earlier Etros: set it back.
		if b, err := os.ReadFile(filepath.Join(dir, "autoboot.txt")); err == nil && strings.Contains(string(b), "boot_partition=4") {
			if os.Remove(filepath.Join(dir, "autoboot.txt")) == nil {
				note("This card was set to start Windows 11; it starts Etros again now.")
			}
		}
		// Windows 11 partitions an earlier Etros made room for: give the
		// space back.
		if addR36SSupported() && cardHasWindows(dir) {
			heading("WINDOWS 11 PARTITIONS")
			note("This card still has the room an earlier Etros made for Windows 11 (about 49 GB).")
			note("Etros Updater can give it back to your games: the card is made again, and your")
			note("games, saves and settings are copied off and put back.")
			if t.yes("Remove the Windows 11 partitions?", true) {
				if err := t.remakeWithoutWindows(dir); err != nil {
					logf("ERROR: removing Windows: %v", err)
					fmt.Printf("\n%s  UPDATE FAILED%s\n\n", cRed, cReset)
					note("Removing the Windows 11 partitions didn't work: " + err.Error())
					t.ask("Press Enter to go back")
					return false
				}
				fmt.Printf("\n%s  CARD UPDATED, WITHOUT WINDOWS 11%s\n\n", cYellow, cReset)
				note("The space is back on the storage partition, with your games, saves and settings.")
				t.ask("Press Enter to finish")
				return true
			}
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
		changed := 0
		for attempt := 1; attempt <= 2; attempt++ {
			if attempt == 2 {
				note(cYellow + "That didn't work: " + err.Error() + cReset)
				note("Trying again...")
			}
			rep := newTermReporter()
			var n int
			n, err = fixCard(dir, rep)
			changed += n
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
		if changed == 0 {
			note(cCyan + "Already up to date: this card has the newest Etros, nothing needed changing." + cReset)
		}
		showCardContents(dir)
		if _, err := os.Stat(filepath.Join(dir, "etros", "pi5")); err == nil {
			t.offerProton(dir)
			if err := t.offerArch(dir); err != nil {
				logf("ERROR: Arch Linux: %v", err)
				note(cRed + "Couldn't make Ollie OS Arch Linux: " + err.Error() + cReset)
			}
		}
		if cardHasR36S(dir) {
			t.chooseR36SScreen(dir)
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

// chooseR36SScreen lets the person pick their R36S's screen; a black R36S
// screen usually means the card is set for a different one.
func (t *tui) chooseR36SScreen(dir string) {
	cur := r36sScreen(dir)
	heading("R36S SCREEN")
	note("R36S units have different screens. If your R36S starts but its screen stays")
	note("black, the card is set for a different screen: pick the next one here, and")
	note("try again. (The R36S's light comes on even when the screen stays black.)")
	for i, s := range r36sScreens {
		mark := ""
		if i == cur {
			mark = cCyan + "  <- set now" + cReset
		}
		fmt.Printf("  %s%d%s  %s%s\n", cYellow, i, cReset, s.name, mark)
	}
	next := (cur + 1) % len(r36sScreens)
	for {
		a := strings.TrimSpace(t.ask(fmt.Sprintf("Which screen? (a number, Enter keeps %d, n tries the next: %d)", cur, next)))
		i := cur
		switch {
		case a == "":
			return
		case strings.EqualFold(a, "n"):
			i = next
		default:
			if _, err := fmt.Sscan(a, &i); err != nil || i < 0 || i >= len(r36sScreens) {
				continue
			}
		}
		if err := setR36SScreen(dir, i); err != nil {
			note(cRed + "Couldn't set the screen: " + err.Error() + cReset)
			continue
		}
		note("Set for: " + r36sScreens[i].name)
		return
	}
}

// reportR36S says how far the R36S got the last time it started from this
// card, from the notes Etros leaves there (etros/last-start-*.txt).
func (t *tui) reportR36S(dir string) {
	read := func(board string) []string {
		b, err := os.ReadFile(filepath.Join(dir, "etros", "last-start-"+board+".txt"))
		if err != nil {
			return nil
		}
		return strings.Split(strings.TrimSpace(string(b)), "\n")
	}
	pi, r36s := read("pi5"), read("r36s")
	heading("R36S: WHAT HAPPENED LAST TIME")
	code := ""
	for _, l := range pi {
		if strings.HasPrefix(l, "R36S start-up code: ") {
			code = strings.TrimPrefix(l, "R36S start-up code: ")
		}
	}
	switch {
	case pi == nil:
		note("The Pi 5 hasn't started this card since Etros began keeping notes.")
		note(cYellow + "Start it in the Pi 5 first" + cReset + " (until Etros shows): that puts the R36S's")
		note("start-up code in place. Without it the R36S stays black.")
	case code != "":
		note("Pi 5, last start: R36S start-up code " + code + ".")
	}
	switch {
	case r36s == nil:
		note("R36S: it never got as far as Etros's start-up from this card.")
		if code == "in place" || code == "put in place just now" {
			note("So the R36S's own start-up code ran but didn't start Batocera, or didn't run at")
			note("all. Check the card is in the R36S's " + cBold + "TF1" + cReset + " slot (the one marked OS / TF1).")
		}
	case len(r36s) == 1:
		note("R36S: Etros's start-up ran, but Batocera didn't finish starting.")
	default:
		note("R36S: Batocera started (" + strings.TrimPrefix(r36s[1], "Batocera running, ") + ").")
		if len(r36s) > 2 {
			note("It was set for: " + strings.TrimPrefix(r36s[2], "screen file: "))
		}
		note(cYellow + "So if its screen was black, it's the screen setting:" + cReset + " pick the next one below.")
	}
}

// showCardContents lists what the Etros card at dir has, so it can be seen
// that everything is there.
func showCardContents(dir string) {
	has := func(rel string) bool {
		_, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel)))
		return err == nil
	}
	wine, _ := filepath.Glob(filepath.Join(dir, "etros", "wine", "wine-*.tar.xz"))
	layer, _ := os.ReadFile(filepath.Join(dir, "etros", "etros.squashfs"))
	heading("ON THIS CARD")
	line := func(ok bool, what, missing string) {
		if ok {
			fmt.Printf("  %s✔%s %s\n", cCyan, cReset, what)
		} else {
			fmt.Printf("  %s✘%s %s %s(%s)%s\n", cRed, cReset, what, cDim, missing, cReset)
		}
	}
	line(bytes.Equal(layer, etrosLayer), "Etros (theme, My ROMs, menu music, Windows and OS folders)", "not the newest")
	line(has("etros/pi5"), "Raspberry Pi 5", "not on this card")
	line(cardHasR36S(dir), "R36S", "not on this card: Etros Updater can add it")
	line(has("etros/ollie"), "Ollie OS (Boot into Ollie OS, Internet Archive)", "made without Ollie OS")
	line(len(wine) > 0, "Wine, for Windows programs on the Pi 5", "couldn't be added: put wine-10.0-amd64-wow64.tar.xz in Downloads")
}

// showOllieReport shows the parts of Ollie OS's last report
// (etros/ollie-log.txt, written by ollie-check) that say why it didn't
// reach its desktop.
func showOllieReport(report string) {
	lines := strings.Split(report, "\n")
	if len(lines) < 2 {
		return
	}
	status := strings.TrimSpace(lines[1])
	note("Ollie OS's last report: " + status)
	if strings.HasPrefix(status, "OK") {
		note("(the whole report is etros/ollie-log.txt on the card's BATOCERA drive)")
		return
	}
	section := ""
	shown := map[string]int{}
	limit := map[string]int{"failed services": 8, "X (the login screen)": 6, "the desktop (KDE Plasma)": 6}
	for _, l := range lines[2:] {
		if strings.HasPrefix(l, "== ") && strings.HasSuffix(l, " ==") {
			section = strings.TrimSuffix(strings.TrimPrefix(l, "== "), " ==")
			continue
		}
		l = strings.TrimSpace(l)
		if l == "" || shown[section] >= limit[section] {
			continue
		}
		if shown[section] == 0 {
			note(cYellow + section + ":" + cReset)
		}
		note("  " + l)
		shown[section]++
	}
	note("(the whole report is etros/ollie-log.txt on the card's BATOCERA drive)")
}
