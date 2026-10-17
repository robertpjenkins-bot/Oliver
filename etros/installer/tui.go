package main

// The installer's terminal interface.

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Colours, after Super Metroid. Empty when the terminal can't show them.
var (
	cOrange, cYellow, cRed, cPink, cCyan, cBlue, cDim, cBold, cReset string
)

func setColours(on bool) {
	if !on {
		return
	}
	cOrange, cYellow, cRed = "\x1b[38;5;208m", "\x1b[38;5;220m", "\x1b[38;5;196m"
	cPink, cCyan, cBlue = "\x1b[38;5;205m", "\x1b[38;5;81m", "\x1b[38;5;27m"
	cDim, cBold, cReset = "\x1b[2m", "\x1b[1m", "\x1b[0m"
}

var banner = []string{
	"███████╗████████╗██████╗  ██████╗ ███████╗",
	"██╔════╝╚══██╔══╝██╔══██╗██╔═══██╗██╔════╝",
	"█████╗     ██║   ██████╔╝██║   ██║███████╗",
	"██╔══╝     ██║   ██╔══██╗██║   ██║╚════██║",
	"███████╗   ██║   ██║  ██║╚██████╔╝███████║",
	"╚══════╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚══════╝",
}

func showBanner() {
	fmt.Println()
	fmt.Printf("  %sENERGY 99%s  %s■ ■ ■ ■ ■ ■%s      %sversion %s%s\n\n", cBold, cReset, cPink, cReset, cDim, version, cReset)
	fmt.Printf("              %sWELCOME TO%s\n", cCyan, cReset)
	shades := []string{cYellow, cYellow, cOrange, cOrange, cRed, cBlue}
	for i, line := range banner {
		fmt.Printf("  %s%s%s\n", shades[i], line, cReset)
	}
	fmt.Printf("\n  %sTHE LAST CARTRIDGE IS IN CAPTIVITY.\n  THE GALAXY IS AT PEACE...%s\n\n", cDim, cReset)
}

type tui struct {
	in    *bufio.Reader
	found Found
	opts  Options
}

func (t *tui) ask(prompt string) string {
	fmt.Printf("%s> %s%s ", cOrange, prompt, cReset)
	line, err := t.in.ReadString('\n')
	logf("asked %q, answered %q", prompt, strings.TrimSpace(line))
	if err != nil && line == "" {
		fmt.Println()
		os.Exit(0)
	}
	return strings.TrimSpace(line)
}

func (t *tui) yes(prompt string, def bool) bool {
	hint := "Y/n"
	if !def {
		hint = "y/N"
	}
	for {
		a := strings.ToLower(t.ask(fmt.Sprintf("%s [%s]", prompt, hint)))
		switch a {
		case "":
			return def
		case "y", "yes":
			return true
		case "n", "no":
			return false
		}
	}
}

func heading(s string) { fmt.Printf("\n%s== %s ==%s\n", cOrange, s, cReset) }
func note(s string)    { fmt.Printf("  %s\n", s) }
func gbs(n int64) string {
	return fmt.Sprintf("%.1f GB", float64(n)/1e9)
}

func runTUI(o Options) error {
	t := &tui{in: bufio.NewReader(os.Stdin), opts: o}
	showBanner()
	note("Etros puts Batocera for the " + cBold + "Raspberry Pi 5" + cReset + " and the " + cBold + "R36S" + cReset + " on one microSD card.")

	for {
		heading("LOOKING IN DOWNLOADS")
		t.found = scan()
		for _, d := range scanDirs() {
			note(cDim + d + cReset)
		}
		t.report()

		heading("WHAT NOW?")
		var choices []func() error
		item := func(label string, fn func() error) {
			choices = append(choices, fn)
			fmt.Printf("  %s%d%s  %s\n", cYellow, len(choices), cReset, label)
		}
		if len(t.found.Etros) > 0 {
			img := t.found.Etros[0]
			item("Write "+cBold+baseName(img.Path)+cReset+" to an SD card", func() error { return t.writeToCard(img.Path) })
			item("Make a new Etros image (replaces nothing; saves a new one to Downloads)", func() error {
				_, err := t.build()
				return err
			})
		} else {
			item("Make Etros and write it to an SD card", func() error {
				// Choose the card first, so the long build needs no one there.
				disk, ok, err := t.pickCard()
				if !ok || err != nil {
					return err
				}
				path, err := t.build()
				if err != nil {
					return err
				}
				return t.writeImageTo(path, disk)
			})
			item("Make Etros and save it to Downloads (to write later)", func() error {
				_, err := t.build()
				return err
			})
		}
		item("Look again", func() error { return nil })
		fmt.Printf("  %sq%s  Quit\n", cYellow, cReset)

		a := strings.ToLower(t.ask("Choose"))
		if a == "q" || a == "quit" {
			return nil
		}
		var n int
		if _, err := fmt.Sscan(a, &n); err != nil || n < 1 || n > len(choices) {
			continue
		}
		if err := choices[n-1](); err != nil {
			logf("ERROR: %v", err)
			fmt.Printf("\n%sSOMETHING WENT WRONG:%s %v\n", cRed, cReset, err)
			note("This is also saved in etros-installer-log.txt in your Downloads.")
			note("Downloads are kept, so trying again is quicker.")
			t.ask("Press Enter to go back")
		}
	}
}

func (t *tui) report() {
	f := t.found
	line := func(what string, list []foundFile, missing string) {
		if len(list) > 0 {
			fmt.Printf("  %s✔%s %-22s %s %s(%s)%s\n", cCyan, cReset, what, baseName(list[0].Path), cDim, gbs(list[0].Size), cReset)
		} else {
			fmt.Printf("  %s·%s %-22s %s%s%s\n", cDim, cReset, what, cDim, missing, cReset)
		}
	}
	logf("scan: etros=%v pi5=%v r36s=%v ollie=%v games=%v", f.Etros, f.Pi5, f.R36S, f.Ollie, f.Games)
	line("Etros image", f.Etros, "none yet")
	if len(f.Etros) == 0 {
		line("Batocera, Pi 5", f.Pi5, "not found, will download")
		line("Batocera, R36S", f.R36S, "not found, will download")
		line("Ollie OS", f.Ollie, "not found, will download Raspberry Pi OS")
		if len(f.Games) > 0 {
			fmt.Printf("  %s✔%s %-22s %s\n", cCyan, cReset, "Your games", f.Games[0])
		}
	}
}

// build makes an Etros image in Downloads and returns its path.
func (t *tui) build() (string, error) {
	o := t.opts
	f := t.found
	if len(f.Pi5) > 0 && o.Sources.Pi5Boot == "" {
		o.Sources.Pi5Boot = f.Pi5[0].Path
	}
	if len(f.R36S) > 0 && o.Sources.R36SBoot == "" {
		o.Sources.R36SBoot = f.R36S[0].Path
	}
	if len(f.Ollie) > 0 && o.Sources.RPiOS == "" {
		o.Sources.RPiOS = f.Ollie[0].Path
	}

	heading("OPTIONS")
	if len(f.Ollie) > 0 {
		note("Ollie OS: using " + baseName(f.Ollie[0].Path))
	} else {
		note("Ollie OS is the desktop for downloading ROMs, with Bobby on the wallpaper.")
		note("It needs Raspberry Pi OS: a 1.3 GB download and about 8 GB on the card.")
	}
	o.Ollie = t.yes("Include Ollie OS?", true)

	if o.GamesDir == "" && len(f.Games) > 0 {
		if t.yes("Add your games from "+f.Games[0]+"?", true) {
			o.GamesDir = f.Games[0]
		}
	} else if o.GamesDir == "" {
		note("Your games: a folder with one folder per console inside (snes, gba, ...).")
		note("ROM-hack patches (.bps, .ips) go next to the game they patch.")
		for {
			dir := strings.Trim(t.ask("Games folder (Enter to skip):"), `"'`)
			if dir == "" {
				break
			}
			if st, err := os.Stat(dir); err == nil && st.IsDir() {
				o.GamesDir = dir
				break
			}
			note(cRed + "That folder doesn't exist." + cReset)
		}
	}

	heading("MAKING ETROS")
	rep := newTermReporter()
	card, err := Plan(o, rep)
	if err != nil {
		rep.end()
		return "", err
	}
	name := "etros-" + time.Now().Format("2006-01-02") + ".img.gz"
	path := filepath.Join(downloadsDir(), name)
	out, err := newGzTarget(path+".part", card.Size)
	if err != nil {
		rep.end()
		return "", err
	}
	if err := card.Write(out, rep, false); err != nil {
		out.Finish()
		os.Remove(path + ".part")
		rep.end()
		return "", err
	}
	if err := out.Finish(); err != nil {
		rep.end()
		return "", err
	}
	rep.end()
	if err := os.Rename(path+".part", path); err != nil {
		return "", err
	}
	fmt.Printf("\n  %sSaved:%s %s\n", cCyan, cReset, path)
	note("Next time the installer finds it, and you can write it to more cards straight away.")
	return path, nil
}

// pickCard asks which card to use and for ERASE; ok is false if the person
// backed out.
func (t *tui) pickCard() (disk Disk, ok bool, err error) {
	heading("CHOOSE THE DRIVE")
	for {
		disks, err := listDisks()
		if err != nil {
			return disk, false, err
		}
		if len(disks) == 0 {
			note("No SD cards or USB drives found. Put the card in (a card reader or USB drive), then press r.")
		}
		for i, d := range disks {
			warn := ""
			if d.Size < 30e9 {
				warn = cRed + "  (may be too small)" + cReset
			}
			fmt.Printf("  %s%d%s  %s %s(%s, %s)%s%s\n", cYellow, i+1, cReset, d.Name, cDim, gbs(d.Size), d.Path, cReset, warn)
		}
		fmt.Printf("  %sr%s  Look again    %sb%s  Back\n", cYellow, cReset, cYellow, cReset)
		a := strings.ToLower(t.ask("Which drive? (its number, r or b)"))
		if a == "b" {
			return disk, false, nil
		}
		var n int
		if _, err := fmt.Sscan(a, &n); err == nil && n >= 1 && n <= len(disks) {
			disk = disks[n-1]
			break
		}
		if a != "r" && a != "" {
			if len(disks) == 0 {
				note("There are no drives to pick yet. Put the card in, then press Enter to look again.")
			} else {
				note(fmt.Sprintf("Type a number from 1 to %d, r to look again, or b to go back.", len(disks)))
			}
		}
		heading("CHOOSE THE DRIVE")
	}
	fmt.Printf("\n  %sEVERYTHING ON %s (%s) WILL BE ERASED.%s\n", cRed, strings.ToUpper(disk.Name), gbs(disk.Size), cReset)
	a := strings.ToLower(t.ask("Type ERASE to go on (or anything else to stop):"))
	if a != "erase" {
		note("Stopped: you typed " + fmt.Sprintf("%q", a) + ", not ERASE. Nothing was written.")
		return disk, false, nil
	}
	return disk, true, nil
}

// writeToCard asks which card and writes the image to it.
func (t *tui) writeToCard(image string) error {
	disk, ok, err := t.pickCard()
	if !ok || err != nil {
		return err
	}
	return t.writeImageTo(image, disk)
}

// writeImageTo writes the image to a card already chosen.
func (t *tui) writeImageTo(image string, disk Disk) error {
	// Linux desktops: partitions and files only, which needs no password.
	if filesInstallSupported() {
		rep := newTermReporter()
		err := filesInstall(image, disk.ID, rep)
		rep.end()
		if err != nil {
			return err
		}
		return t.done()
	}
	// Without the rights to write to a drive, a copy of the installer that
	// has them does the writing (Windows asks Yes/No, macOS for a password).
	if can, _ := canWriteDisks(); !can {
		if handled, err := elevatedWrite(image, disk.ID); handled {
			if err != nil {
				return fmt.Errorf("writing the drive: %w", err)
			}
			return t.done()
		}
	}
	rep := newTermReporter()
	out, err := openDisk(disk.ID)
	var admin errNeedsAdmin
	if errors.As(err, &admin) {
		// Not allowed as yourself: write with administrator rights instead.
		note(admin.Error())
		if handled, err := elevatedWrite(image, disk.ID); handled {
			if err != nil {
				return fmt.Errorf("writing with administrator rights: %w", err)
			}
			return t.done()
		}
	}
	if err != nil {
		return err
	}
	if err := writeImage(image, out, out.size, rep); err != nil {
		out.Finish()
		rep.end()
		return err
	}
	rep.Stage("Finishing")
	err = out.Finish()
	rep.end()
	if err != nil {
		return err
	}
	return t.done()
}

func (t *tui) done() error {
	fmt.Printf("\n%s  MISSION COMPLETE%s\n\n", cYellow, cReset)
	note("1. Put the card in your R36S or Raspberry Pi 5 and switch it on.")
	note("2. The first start takes a few minutes. Don't switch it off.")
	note("3. Ollie OS is under ETROS -> Boot into Ollie OS in the system list.")
	note("4. To add games later, copy them into my-roms on the BATOCERA drive.")
	t.ask("Press Enter to finish")
	return nil
}

// termReporter draws a progress bar on one line.
type termReporter struct {
	stage string
	last  time.Time
	start time.Time
	bar   bool
}

func newTermReporter() *termReporter { return &termReporter{} }

func (r *termReporter) end() {
	if r.bar {
		fmt.Println()
		r.bar = false
	}
}

func (r *termReporter) Stage(s string) {
	logf("STAGE %s", s)
	r.end()
	r.stage = s
	r.start = time.Now()
	fmt.Printf("%s▸ %s%s\n", cCyan, s, cReset)
}

func (r *termReporter) Log(s string) {
	logf("  %s", s)
	r.end()
	fmt.Printf("  %s%s%s\n", cDim, s, cReset)
}

func (r *termReporter) Progress(done, total int64) {
	if time.Since(r.last) < 200*time.Millisecond && done != total {
		return
	}
	r.last = time.Now()
	const width = 30
	if total <= 0 {
		fmt.Printf("\r  %s", gbs(done))
		r.bar = true
		return
	}
	frac := min(1, float64(done)/float64(total))
	filled := int(frac * width)
	eta := ""
	if el := time.Since(r.start); frac > 0.02 && frac < 1 {
		eta = " " + timeLeft(time.Duration(float64(el)*(1-frac)/frac))
	}
	fmt.Printf("\r\x1b[K  %s%s%s%s%s %3.0f%%  %s / %s%s", cOrange, strings.Repeat("█", filled), cDim,
		strings.Repeat("░", width-filled), cReset, frac*100, gbs(done), gbs(total), eta)
	r.bar = true
}

func timeLeft(d time.Duration) string {
	switch m := int(d.Round(time.Minute).Minutes()); {
	case m < 1:
		return "under a minute left"
	case m == 1:
		return "about 1 minute left"
	case m < 60:
		return fmt.Sprintf("about %d minutes left", m)
	default:
		return fmt.Sprintf("about %dh %02dm left", m/60, m%60)
	}
}
