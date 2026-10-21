package main

// Etros on an Intel Mac: what's the same whichever computer puts it there
// (mac_linux.go from a Steam Machine, mac_darwin.go on the Mac itself).

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/klauspost/compress/zstd"
)

const (
	macBootSize  = 8 << 30  // BATOCERA: Batocera for PCs is about 3.5 GB
	macMinShare  = 8 << 30  // SHARE without Ollie OS
	macOllieRoom = 24 << 30 // what Ollie OS takes of SHARE, at most

	archBootstrapURL = "https://geo.mirror.pkgbuild.com/iso/latest/archlinux-bootstrap-x86_64.tar.zst"
)

// partNum is the number at the end of a partition's name (sdb3, disk0s3: 3).
func partNum(id string) int {
	i := len(id)
	for i > 0 && id[i-1] >= '0' && id[i-1] <= '9' {
		i--
	}
	n, _ := strconv.Atoi(id[i:])
	return n
}

func pcBatoceraURL() string { return batoceraBase + "/x86_64/stable/last/boot.tar.xz" }

// getPCBatocera finds Batocera for PCs (its boot.tar.xz) in Downloads, or
// downloads it.
func (t *tui) getPCBatocera() (string, error) {
	for _, pat := range []string{"*x86_64*boot*.tar.xz", "*x86_64*.boot.tar.xz"} {
		if m, _ := filepath.Glob(filepath.Join(downloadsDir(), pat)); len(m) > 0 {
			note("Using " + filepath.Base(m[0]) + " from Downloads.")
			return m[0], nil
		}
	}
	rep := newTermReporter()
	defer rep.end()
	rep.Stage("Downloading Batocera for PCs (about 3 GB)")
	return fetch(pcBatoceraURL(), defaultCache(), "batocera-x86_64-boot", rep.Progress, rep.Log)
}

// getArchBootstrap finds Arch Linux's bootstrap tarball in Downloads, or
// asks for it, or downloads it.
func (t *tui) getArchBootstrap() (string, error) {
	if m, _ := filepath.Glob(filepath.Join(downloadsDir(), "archlinux-bootstrap*x86_64*.tar.zst")); len(m) > 0 {
		note("Using " + filepath.Base(m[0]) + " from Downloads.")
		return m[0], nil
	}
	for {
		p := t.askForFile("ARCH LINUX FOR OLLIE OS",
			"Drag archlinux-bootstrap-x86_64.tar.zst (from archlinux.org) onto this window, then press Enter.",
			"Or just press Enter to let Etros download it (about 150 MB).")
		if p == "" {
			rep := newTermReporter()
			rep.Stage("Getting Arch Linux")
			got, err := fetch(archBootstrapURL, defaultCache(), "archlinux-bootstrap", rep.Progress, rep.Log)
			rep.end()
			return got, err
		}
		n := strings.ToLower(filepath.Base(p))
		if strings.Contains(n, "bootstrap") && strings.Contains(n, "x86_64") {
			return p, nil
		}
		note(cRed + filepath.Base(p) + " isn't Arch Linux's bootstrap (archlinux-bootstrap-x86_64.tar.zst)." + cReset)
	}
}

// pcInitrd is Batocera for PCs' start-up file with Etros's hook added.
func pcInitrd(bootTarball string) ([]byte, error) {
	var initrd []byte
	err := eachTarFile(bootTarball, func(h *tarHeader, r io.Reader) error {
		if h.Name == "boot/initrd.gz" {
			b, err := io.ReadAll(r)
			initrd = b
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if initrd == nil {
		return nil, fmt.Errorf("%s isn't Batocera for PCs (no boot/initrd.gz)", filepath.Base(bootTarball))
	}
	out, err := PatchInitrd(initrd)
	if err != nil {
		return nil, fmt.Errorf("adding Etros to Batocera's start-up: %w", err)
	}
	return out, nil
}

// copyPCBatocera copies Batocera for PCs' boot files to boot, with Etros.
func copyPCBatocera(bootTarball string, initrd []byte, boot string, rep Reporter) error {
	st, err := os.Stat(bootTarball)
	if err != nil {
		return err
	}
	var done int64
	err = eachTarFile(bootTarball, func(h *tarHeader, r io.Reader) error {
		dest := filepath.Join(boot, filepath.FromSlash(h.Name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if h.Name == "boot/initrd.gz" {
			return os.WriteFile(dest, initrd, 0o644)
		}
		f, err := os.Create(dest)
		if err != nil {
			return err
		}
		// Progress by the compressed size, roughly: the tarball is xz.
		n, err := io.Copy(f, &countingReader{r: r, p: func(d, _ int64) {
			rep.Progress(min(done+d/3, st.Size()), st.Size())
		}})
		done += n / 3
		if cerr := f.Close(); err == nil {
			err = cerr
		}
		return err
	})
	if err != nil {
		return err
	}
	rep.Progress(st.Size(), st.Size())
	if err := os.MkdirAll(filepath.Join(boot, "etros"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(boot, "etros", "etros.squashfs"), etrosLayer, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(boot, "etros", "intel-mac.txt"), []byte("Etros on an Intel Mac, in place of Boot Camp\n"), 0o644); err != nil {
		return err
	}
	// SHARE is made to fill the rest already.
	return noAutoresize(boot)
}

// gzBootstrap rewrites Arch Linux's bootstrap tarball (zstd, all in
// root.x86_64/) as a plain .tar.gz of the file system itself, which Etros
// on the Mac can unpack with the tools Batocera has.
func gzBootstrap(src, dest string, rep Reporter) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, _ := in.Stat()
	zr, err := zstd.NewReader(&countingReader{r: in, p: func(d, _ int64) { rep.Progress(d, st.Size()) }})
	if err != nil {
		return err
	}
	defer zr.Close()
	out, err := os.Create(dest + ".new")
	if err != nil {
		return err
	}
	gw, _ := gzip.NewWriterLevel(out, gzip.BestSpeed)
	tw := tar.NewWriter(gw)
	tr := tar.NewReader(zr)
	err = func() error {
		for {
			h, err := tr.Next()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}
			name := strings.TrimPrefix(strings.TrimPrefix(h.Name, "./"), "root.x86_64")
			name = strings.TrimPrefix(name, "/")
			if name == "" {
				continue
			}
			h.Name = name
			if h.Typeflag == tar.TypeLink {
				h.Linkname = strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(h.Linkname, "./"), "root.x86_64"), "/")
			}
			if err := tw.WriteHeader(h); err != nil {
				return err
			}
			if _, err := io.Copy(tw, tr); err != nil {
				return err
			}
		}
	}()
	for _, c := range []io.Closer{tw, gw, out} {
		if cerr := c.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		os.Remove(dest + ".new")
		return fmt.Errorf("%s: %w", filepath.Base(src), err)
	}
	return os.Rename(dest+".new", dest)
}

// This Mac's drive as diskutil sees it (mac_darwin.go), planned here so
// it can be checked anywhere.

type dPart struct {
	id, content, name string
	size              int64
}

func (p dPart) num() int { return partNum(p.id) }

type dPlan struct {
	disk     string
	size     int64
	parts    []dPart
	macOS    []dPart
	bootCamp []dPart
	store    string // APFS's partition, to make room in when there's no Boot Camp
	free     int64  // room in APFS
	isEtros  bool
}

func isMacOSContent(c string) bool {
	switch c {
	case "Apple_APFS", "Apple_HFS", "Apple_CoreStorage", "Apple_Boot", "Apple_APFS_Recovery":
		return true
	}
	return false
}

// planThisMac works out the plan from diskutil list -plist.
func planThisMac(v any) (*dPlan, error) {
	disks, _ := plistGet(v, "AllDisksAndPartitions").([]any)
	for i := range disks {
		pl := &dPlan{disk: plistString(disks[i], "DeviceIdentifier"), size: plistInt(disks[i], "Size")}
		parts, _ := plistGet(disks[i], "Partitions").([]any)
		for j := range parts {
			pl.parts = append(pl.parts, dPart{
				id:      plistString(parts[j], "DeviceIdentifier"),
				content: plistString(parts[j], "Content"),
				name:    plistString(parts[j], "VolumeName"),
				size:    plistInt(parts[j], "Size"),
			})
		}
		last := -1
		for j, p := range pl.parts {
			if isMacOSContent(p.content) {
				last = j
				if p.content == "Apple_APFS" {
					pl.store = p.id
				}
			}
		}
		if last < 0 {
			continue
		}
		pl.macOS = pl.parts[:last+1]
		for _, p := range pl.parts[last+1:] {
			switch {
			case p.name == "BATOCERA" || p.name == "SHARE" || (pl.isEtros && p.content == "Linux Filesystem"):
				pl.isEtros = true // (SHARE is ext4 once Etros has started)
			case p.content == "Microsoft Basic Data" || p.content == "Windows_NTFS" || p.content == "Windows Recovery" ||
				p.content == "Windows_FAT_32" || p.content == "DOS_FAT_32" || p.content == "Microsoft Reserved":
			default:
				return nil, fmt.Errorf("%s (%s) comes after macOS and isn't Boot Camp; Etros won't touch it", p.id, p.content)
			}
			pl.bootCamp = append(pl.bootCamp, p)
		}
		return pl, nil
	}
	return nil, errors.New("couldn't find this Mac's drive with macOS on it")
}
