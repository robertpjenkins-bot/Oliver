// Etros Installer puts Etros on a microSD card: Batocera for the Raspberry
// Pi 5 and the R36S on one card, with the Etros look and Ollie OS.
//
// Run it with no arguments for the step-by-step installer. It looks in
// Downloads (and the folder it sits in) for an Etros image, Batocera
// downloads, an Ollie OS image and your games, and downloads what's missing.
//
// For scripts:
//
//	etros-installer -list
//	etros-installer -target file:etros.img.gz      make an image file
//	etros-installer -target disk:ID                 make Etros and write it to a card
//	etros-installer -write etros.img.gz -target disk:ID
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var version = "dev"

func main() {
	setupTerminal()
	var (
		list     = flag.Bool("list", false, "list the cards that can be written to, and exit")
		target   = flag.String("target", "", "disk:ID (from -list) or file:PATH (.img or .img.gz)")
		write    = flag.String("write", "", "write this Etros image to -target disk:ID instead of making one")
		yes      = flag.Bool("yes", false, "don't ask before erasing the card")
		pause    = flag.Bool("pause", false, "wait for Enter before closing")
		noOllie  = flag.Bool("no-ollie", false, "leave out Ollie OS")
		games    = flag.String("games", "", "folder of your games (one folder per console) to put on the card")
		elevated = flag.Bool("elevated", false, "internal: already asked for administrator rights")
		s        Sources
	)
	flag.StringVar(&s.Pi5Boot, "pi5-boot", "", "Batocera for the Pi 5: .img.gz or boot.tar.xz (file or URL)")
	flag.StringVar(&s.R36SBoot, "r36s-boot", "", "Batocera for the R36S: .img.gz or boot.tar.xz (file or URL)")
	flag.StringVar(&s.R36SImage, "r36s-image", "", "Batocera .img.gz for the R36S, for its bootloader, when -r36s-boot is a boot.tar.xz")
	flag.StringVar(&s.RPiOS, "rpios", "", "Ollie OS: an Ollie OS or Raspberry Pi OS 64-bit desktop image (file or URL)")
	flag.StringVar(&s.Channel, "channel", "", "Batocera release channel (default stable)")
	cache := flag.String("cache", defaultCache(), "where to keep downloads")
	patchOnly := flag.String("patch-initrd", "", "for testing: add the Etros hook to this initramfs, writing it to -target file:PATH")
	flag.Parse()

	if *patchOnly != "" {
		in, err := os.ReadFile(*patchOnly)
		if err == nil {
			var out []byte
			if out, err = PatchInitrd(in); err == nil {
				err = os.WriteFile(strings.TrimPrefix(*target, "file:"), out, 0o644)
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	opts := Options{Sources: s, Ollie: !*noOllie, GamesDir: *games, CacheDir: *cache}
	logPath := openLog()
	defer crashGuard(logPath)

	var err error
	switch {
	case *list:
		var disks []Disk
		if disks, err = listDisks(); err == nil {
			for _, d := range disks {
				fmt.Printf("disk:%s\t%s\n", d.ID, d.Label)
			}
			if ok, why := canWriteDisks(); !ok {
				fmt.Fprintln(os.Stderr, why)
			}
		}
	case *target != "" || *write != "":
		err = runCLI(opts, *target, *write, *yes)
		if *pause && err == nil {
			fmt.Printf("\n%sDone.%s The drive is ready.\n", cYellow, cReset)
			waitForEnter()
		}
	case mode == "maker":
		// Etros Maker only makes an image file, so it never asks for
		// administrator rights.
		if openInTerminal() {
			return
		}
		err = runMaker(opts)
	default:
		if openInTerminal() {
			return
		}
		if !*elevated && relaunchElevated() {
			return
		}
		err = runTUI(opts)
	}
	if err != nil {
		logf("ERROR: %v", err)
		fmt.Fprintf(os.Stderr, "\n%setros-installer:%s %v\n", cRed, cReset, err)
		if logPath != "" {
			fmt.Fprintln(os.Stderr, "The log is in "+logPath)
		}
		if pauseOnExit() || *pause {
			waitForEnter()
		}
		os.Exit(1)
	}
}

func defaultCache() string {
	if os.Getenv("SUDO_USER") != "" {
		// Under sudo, keep downloads with the person who ran it.
		return filepath.Join(homeDir(), ".cache", "etros-installer")
	}
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "etros-installer")
	}
	return filepath.Join(os.TempDir(), "etros-installer")
}

// writeTarget is something a card can be written to.
type writeTarget interface {
	WriteAt(p []byte, off int64) (int, error)
	Finish() error
}

type fileTarget struct{ *os.File }

func (f fileTarget) Finish() error {
	if err := f.Sync(); err != nil {
		return err
	}
	chownToUser(f.Name())
	return f.Close()
}

func runCLI(o Options, target, image string, yes bool) error {
	kind, where, _ := strings.Cut(target, ":")
	if kind != "disk" && kind != "file" || where == "" {
		return fmt.Errorf("-target must be disk:ID (see -list) or file:PATH")
	}
	if image != "" && kind != "disk" {
		return fmt.Errorf("-write needs -target disk:ID")
	}
	if kind == "disk" {
		if ok, why := canWriteDisks(); !ok {
			return errors.New(why)
		}
		if !yes {
			fmt.Printf("Everything on %s will be erased. Type ERASE to go on: ", target)
			var answer string
			fmt.Scanln(&answer)
			if answer != "ERASE" {
				return errors.New("stopped; nothing was written")
			}
		}
	}
	if o.GamesDir != "" {
		if st, err := os.Stat(o.GamesDir); err != nil || !st.IsDir() {
			return fmt.Errorf("games folder %q not found", o.GamesDir)
		}
	}
	rep := newTermReporter()
	defer rep.end()

	if image != "" && filesInstallSupported() {
		return filesInstall(image, where, rep)
	}
	if image != "" {
		d, err := openDisk(where)
		if err != nil {
			return err
		}
		if err := writeImage(image, d, d.size, rep); err != nil {
			d.Finish()
			return err
		}
		return d.Finish()
	}

	card, err := Plan(o, rep)
	if err != nil {
		return err
	}
	var out writeTarget
	switch {
	case kind == "disk":
		d, err := openDisk(where)
		if err != nil {
			return err
		}
		if d.size < card.Size {
			d.Close()
			return fmt.Errorf("the card is too small: Etros needs %.1f GB, the card has %.1f GB", float64(card.Size)/1e9, float64(d.size)/1e9)
		}
		out = d
	case strings.HasSuffix(strings.ToLower(where), ".gz"):
		if out, err = newGzTarget(where, card.Size); err != nil {
			return err
		}
	default:
		f, err := os.Create(where)
		if err != nil {
			return err
		}
		if err := f.Truncate(card.Size); err != nil {
			f.Close()
			return err
		}
		out = fileTarget{f}
	}
	if err := card.Write(out, rep, kind == "disk"); err != nil {
		out.Finish()
		return err
	}
	rep.Stage("Finishing")
	return out.Finish()
}
