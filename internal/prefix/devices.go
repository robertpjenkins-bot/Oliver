package prefix

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Mapping is a drive letter in a prefix pointing at a real block device.
type Mapping struct {
	Letter string `json:"letter"`
	// Device is the raw block device the letter's "::" link resolves to.
	Device string `json:"device"`
	// Mount is the directory the letter itself points at, if any.
	Mount string `json:"mount,omitempty"`
}

// dosDevices is where Wine keeps the drive-letter links for a prefix.
func (p *Prefix) dosDevices() string {
	return filepath.Join(p.WinePrefix, "dosdevices")
}

// Devices lists the block devices currently exposed to this prefix.
//
// Wine's convention is that "e:" points at a directory and "e::" points at the
// device node behind it. A program that opens a drive for raw access follows
// the second link, so both are needed for a device to be fully usable.
func (p *Prefix) Devices() ([]Mapping, error) {
	entries, err := os.ReadDir(p.dosDevices())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	byLetter := map[string]*Mapping{}
	get := func(letter string) *Mapping {
		if m, ok := byLetter[letter]; ok {
			return m
		}
		m := &Mapping{Letter: letter}
		byLetter[letter] = m
		return m
	}

	for _, e := range entries {
		name := e.Name()
		target, err := os.Readlink(filepath.Join(p.dosDevices(), name))
		if err != nil {
			continue
		}
		switch {
		case strings.HasSuffix(name, "::"):
			get(strings.TrimSuffix(name, "::")).Device = target
		case strings.HasSuffix(name, ":"):
			get(strings.TrimSuffix(name, ":")).Mount = target
		}
	}

	var out []Mapping
	for _, m := range byLetter {
		// Only report letters actually backed by a device node; every prefix
		// has c: and z: pointing at ordinary directories.
		if m.Device != "" {
			out = append(out, *m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Letter < out[j].Letter })
	return out, nil
}

// FreeLetter returns an unused drive letter, skipping the ones Wine reserves.
func (p *Prefix) FreeLetter() (string, error) {
	used := map[string]bool{}
	if entries, err := os.ReadDir(p.dosDevices()); err == nil {
		for _, e := range entries {
			name := strings.TrimSuffix(strings.TrimSuffix(e.Name(), "::"), ":")
			used[strings.ToLower(name)] = true
		}
	}
	// c: is the Windows drive and z: is the filesystem root; a and b are the
	// historical floppy letters and confuse some installers.
	for c := 'e'; c <= 'y'; c++ {
		if !used[string(c)] {
			return string(c), nil
		}
	}
	return "", fmt.Errorf("no free drive letters in this prefix")
}

// MapDevice exposes a block device to the prefix under a drive letter.
//
// mountPoint may be empty; when given, the letter itself points at it so the
// device is also browsable as a normal drive.
func (p *Prefix) MapDevice(letter, devicePath, mountPoint string) error {
	letter = strings.ToLower(strings.TrimSuffix(letter, ":"))
	if len(letter) != 1 || letter[0] < 'a' || letter[0] > 'z' {
		return fmt.Errorf("invalid drive letter %q", letter)
	}
	if letter == "c" {
		return fmt.Errorf("refusing to remap c:, which is the prefix's own Windows drive")
	}
	if err := os.MkdirAll(p.dosDevices(), 0o755); err != nil {
		return err
	}

	// The "::" link is the one that matters for raw device access.
	rawLink := filepath.Join(p.dosDevices(), letter+"::")
	_ = os.Remove(rawLink)
	if err := os.Symlink(devicePath, rawLink); err != nil {
		return fmt.Errorf("linking %s: %w", rawLink, err)
	}

	target := mountPoint
	if target == "" {
		// With nothing mounted, point the letter at an empty directory inside
		// the prefix so the drive exists without exposing anything else.
		target = filepath.Join(p.Dir, "drives", letter)
		if err := os.MkdirAll(target, 0o755); err != nil {
			return err
		}
	}
	dirLink := filepath.Join(p.dosDevices(), letter+":")
	_ = os.Remove(dirLink)
	if err := os.Symlink(target, dirLink); err != nil {
		return fmt.Errorf("linking %s: %w", dirLink, err)
	}
	return nil
}

// UnmapDevice removes a drive letter's device links.
func (p *Prefix) UnmapDevice(letter string) error {
	letter = strings.ToLower(strings.TrimSuffix(letter, ":"))
	if len(letter) != 1 {
		return fmt.Errorf("invalid drive letter %q", letter)
	}
	if letter == "c" || letter == "z" {
		return fmt.Errorf("refusing to remove %s:, which Wine requires", letter)
	}
	var removed int
	for _, suffix := range []string{"::", ":"} {
		path := filepath.Join(p.dosDevices(), letter+suffix)
		if err := os.Remove(path); err == nil {
			removed++
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	if removed == 0 {
		return fmt.Errorf("%s: is not mapped in this prefix", letter)
	}
	return nil
}
