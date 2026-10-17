package main

// Fixing an Etros card already made: Etros's own files on the BATOCERA
// drive are swapped for this version's. Games, saves and settings live on
// SHARE and in Batocera's own files, which are left alone. The BATOCERA
// drive is an ordinary FAT drive that every computer opens without a
// password, so this never needs one either.

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// etrosStamp names this version of Etros. Images are stamped with it, so an
// image left in Downloads by an older Etros Maker can be told apart.
func etrosStamp() string {
	h := sha256.New()
	h.Write(etrosLayer)
	h.Write(etrosHook)
	fs.WalkDir(ollieFiles, "assets/ollie", func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			b, _ := ollieFiles.ReadFile(p)
			h.Write([]byte(p))
			h.Write(b)
		}
		return nil
	})
	return "etros " + hex.EncodeToString(h.Sum(nil))[:12]
}

// imageIsCurrent tells whether an Etros image was made by this version.
func imageIsCurrent(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return false
	}
	return gz.Comment == etrosStamp()
}

// etrosCardDirs lists the BATOCERA drives of Etros cards that are open on
// this computer (folders with etros/etros.squashfs in them).
func etrosCardDirs() []string {
	var out []string
	for _, dir := range mountedDirs() {
		if _, err := os.Stat(filepath.Join(dir, "etros", "etros.squashfs")); err == nil {
			out = append(out, dir)
		}
	}
	return out
}

// fixCard brings the Etros files on the BATOCERA drive at dir up to date.
func fixCard(dir string, rep Reporter) error {
	write := func(rel string, data []byte) error {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if old, err := os.ReadFile(p); err == nil && bytes.Equal(old, data) {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		tmp := p + ".new"
		if err := os.WriteFile(tmp, data, 0o644); err != nil {
			os.Remove(tmp)
			return err
		}
		if err := os.Rename(tmp, p); err != nil {
			os.Remove(tmp)
			return err
		}
		rep.Log("updated " + rel)
		return nil
	}

	rep.Stage("Updating Etros on the card")
	if err := write("etros/etros.squashfs", etrosLayer); err != nil {
		return err
	}
	for _, rel := range []string{"etros/pi5/initrd.lz4", "uInitrd"} {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		in, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		out, changed, err := UpdateInitrdHook(in)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		if changed {
			if err := write(rel, out); err != nil {
				return err
			}
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "etros", "ollie")); err == nil {
		err := fs.WalkDir(ollieFiles, "assets/ollie", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := ollieFiles.ReadFile(p)
			if err != nil {
				return err
			}
			return write("etros/ollie/"+strings.TrimPrefix(p, "assets/ollie/"), b)
		})
		if err != nil {
			return err
		}
	}
	// An Ollie OS start that went wrong before the fix is old news.
	os.Remove(filepath.Join(dir, "etros", "last-ollie-boot.txt"))
	syncDir(filepath.Join(dir, "etros"))
	syncDir(dir)
	return nil
}

func syncDir(dir string) {
	if d, err := os.Open(dir); err == nil {
		d.Sync()
		d.Close()
	}
}
