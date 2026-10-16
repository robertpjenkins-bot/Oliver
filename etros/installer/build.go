package main

// Putting an Etros card together and writing it out.

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"embed"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ulikunitz/xz"
)

//go:embed assets/etros.squashfs
var etrosLayer []byte

//go:embed assets/share.ext4.gz
var shareImage []byte

//go:embed assets/my-roms-README.txt
var myRomsReadme []byte

//go:embed all:assets/ollie
var ollieFiles embed.FS

const (
	mib       = 1 << 20
	shareSize = 512 * mib // Batocera grows and formats a share of exactly this size
	// Room on the boot partition for games copied in later and for
	// Batocera's saved settings.
	fatFreeSpace = 2 << 30
	// Room for Ollie OS to grow into.
	ollieFreeSpace = 2 << 30
)

// Options for one install.
type Options struct {
	Sources  Sources
	Ollie    bool   // include Ollie OS
	GamesDir string // optional folder of games for my-roms
	CacheDir string
}

// Reporter receives what the install is doing.
type Reporter interface {
	Stage(name string)
	Log(msg string)
	Progress(done, total int64)
}

// smallFile is how big a file may be to be kept in memory while reading an
// archive; anything bigger is copied straight from the archive later.
// ETROS_SMALL_FILE lowers it, so tests can exercise the copying.
var smallFile = func() int64 {
	if n, err := strconv.ParseInt(os.Getenv("ETROS_SMALL_FILE"), 10, 64); err == nil {
		return n
	}
	return 64 * mib
}()

// mustRead are files Etros changes, so they are always read into memory.
var mustRead = map[string]bool{
	"config.txt": true, "batocera-boot.conf": true, "boot/initrd.lz4": true, "uInitrd": true,
}

type bootFile struct {
	path string // in the archive
	size int64
	data []byte // set for small files
}

// readBoot lists a Batocera boot.tar.xz, keeping small files in memory.
func readBoot(path string) ([]bootFile, error) {
	var files []bootFile
	err := eachTarFile(path, func(h *tar.Header, r io.Reader) error {
		f := bootFile{path: h.Name, size: h.Size}
		if h.Size <= smallFile || mustRead[h.Name] {
			b, err := io.ReadAll(r)
			if err != nil {
				return err
			}
			f.data = b
		}
		files = append(files, f)
		return nil
	})
	return files, err
}

func eachTarFile(path string, fn func(h *tar.Header, r io.Reader) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	x, err := xz.NewReader(bufio.NewReaderSize(f, 4*mib))
	if err != nil {
		return fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	t := tar.NewReader(x)
	for {
		h, err := t.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), err)
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		h.Name = strings.TrimPrefix(h.Name, "./")
		if err := fn(h, t); err != nil {
			return err
		}
	}
}

// cardPath is where a file from a board's boot partition goes on the
// shared card: the board's system directory moves to etros/<board>.
func cardPath(board, p string) string {
	if strings.HasPrefix(p, "boot/") {
		return "etros/" + board + "/" + strings.TrimPrefix(p, "boot/")
	}
	return p
}

// patchPi5Config points the Pi firmware at the kernel and initramfs in
// their new place.
func patchPi5Config(b []byte) ([]byte, error) {
	s := string(b)
	for _, r := range [][2]string{
		{"kernel=boot/linux", "kernel=etros/pi5/linux"},
		{"initramfs boot/initrd.lz4", "initramfs etros/pi5/initrd.lz4"},
	} {
		if !strings.Contains(s, r[0]) {
			return nil, fmt.Errorf("Pi 5 config.txt has no %q; Batocera's layout has changed", r[0])
		}
		s = strings.Replace(s, r[0], r[1], 1)
	}
	return []byte(s), nil
}

// mergeBootConf keeps the Pi 5's batocera-boot.conf and adds any setting
// only the R36S's has.
func mergeBootConf(pi, r36s []byte) []byte {
	keys := map[string]bool{}
	for _, line := range strings.Split(string(pi), "\n") {
		if k, _, ok := strings.Cut(strings.TrimSpace(line), "="); ok && !strings.HasPrefix(k, "#") {
			keys[strings.TrimSpace(k)] = true
		}
	}
	var extra []string
	for _, line := range strings.Split(string(r36s), "\n") {
		t := strings.TrimSpace(line)
		if k, _, ok := strings.Cut(t, "="); ok && !strings.HasPrefix(k, "#") && !keys[strings.TrimSpace(k)] {
			extra = append(extra, t)
		}
	}
	out := string(pi)
	if len(extra) > 0 {
		out = strings.TrimRight(out, "\n") + "\n\n# from the R36S\n" + strings.Join(extra, "\n") + "\n"
	}
	return []byte(out)
}

// Card is a planned Etros card.
type Card struct {
	vol       *FATVolume
	loader    []byte
	boots     map[string]string          // board -> boot.tar.xz path
	streamed  map[string]map[string]bool // board -> archive paths written from the stream
	rpiOS     string                     // Raspberry Pi OS image, if Ollie OS is included
	rootStart int64                      // the Raspberry Pi OS root partition, within its image
	rootSize  int64

	fatStart, ollieStart, ollieSize, shareStart, Size int64
}

// Plan downloads everything and lays the card out.
func Plan(o Options, rep Reporter) (*Card, error) {
	s := o.Sources
	s.defaults()
	c := &Card{boots: map[string]string{}, streamed: map[string]map[string]bool{}}
	logf := func(f string, a ...any) { rep.Log(fmt.Sprintf(f, a...)) }

	rep.Stage("Downloading Batocera for the Raspberry Pi 5")
	var err error
	if c.boots["pi5"], err = fetch(s.Pi5Boot, o.CacheDir, "batocera-pi5", rep.Progress, rep.Log); err != nil {
		return nil, err
	}
	rep.Stage("Downloading Batocera for the R36S")
	if c.boots["r36s"], err = fetch(s.R36SBoot, o.CacheDir, "batocera-r36s", rep.Progress, rep.Log); err != nil {
		return nil, err
	}
	rep.Stage("Getting the R36S bootloader")
	img := s.R36SImage
	if img == "" {
		if img, err = r36sImageURL(s.R36SBoot); err != nil {
			return nil, err
		}
	}
	if c.loader, err = r36sLoader(img); err != nil {
		return nil, err
	}
	if o.Ollie {
		rep.Stage("Downloading Raspberry Pi OS for Ollie OS")
		if c.rpiOS, err = fetch(s.RPiOS, o.CacheDir, "raspios", rep.Progress, rep.Log); err != nil {
			return nil, err
		}
		if c.rootStart, c.rootSize, err = rpiRootPartition(c.rpiOS); err != nil {
			return nil, err
		}
	}

	rep.Stage("Reading Batocera")
	files := map[string][]bootFile{}
	for _, board := range []string{"pi5", "r36s"} {
		logf("reading the %s files", board)
		if files[board], err = readBoot(c.boots[board]); err != nil {
			return nil, err
		}
	}

	rep.Stage("Preparing the boot partition")
	vol := NewFATVolume("BATOCERA")
	c.vol = vol
	type placed struct {
		board string
		f     bootFile
	}
	byPath := map[string]placed{}
	var piConf, r36sConf []byte
	for _, board := range []string{"pi5", "r36s"} {
		c.streamed[board] = map[string]bool{}
		for _, f := range files[board] {
			dest := cardPath(board, f.path)
			data := f.data
			switch {
			case board == "pi5" && f.path == "config.txt":
				if data, err = patchPi5Config(data); err != nil {
					return nil, err
				}
			case board == "pi5" && f.path == "boot/initrd.lz4", board == "r36s" && f.path == "uInitrd":
				logf("adding the Etros start-up to the %s initramfs", board)
				if data, err = PatchInitrd(data); err != nil {
					return nil, fmt.Errorf("%s initramfs: %w", board, err)
				}
			case f.path == "batocera-boot.conf":
				if board == "pi5" {
					piConf = data
				} else {
					r36sConf = data
				}
				continue
			}
			key := strings.ToLower(dest)
			if prev, ok := byPath[key]; ok {
				if prev.f.data != nil && data != nil && bytes.Equal(prev.f.data, data) {
					continue
				}
				logf("note: both systems have %s; keeping the Pi 5's", dest)
				continue
			}
			byPath[key] = placed{board, f}
			if data != nil {
				vol.AddBytes(dest, data)
			} else {
				vol.AddStreamed(dest, f.size, board)
				c.streamed[board][f.path] = true
			}
		}
	}
	if piConf == nil || r36sConf == nil {
		return nil, fmt.Errorf("a Batocera download has no batocera-boot.conf; is it really a Batocera boot.tar.xz?")
	}
	vol.AddBytes("batocera-boot.conf", mergeBootConf(piConf, r36sConf))
	for _, need := range []string{"etros/pi5/linux", "etros/pi5/initrd.lz4", "linux", "uInitrd", "boot.ini"} {
		if vol.Lookup(need) == nil {
			return nil, fmt.Errorf("the Batocera downloads are missing %s", need)
		}
	}

	vol.Dir("boot") // each board's etros/<board> is bound here at start-up
	vol.AddBytes("etros/etros.squashfs", etrosLayer)
	vol.AddBytes("etros/next-boot", []byte("batocera\n"))
	if o.Ollie {
		err := fs.WalkDir(ollieFiles, "assets/ollie", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			b, err := ollieFiles.ReadFile(p)
			if err != nil {
				return err
			}
			vol.AddBytes("etros/ollie/"+strings.TrimPrefix(p, "assets/ollie/"), b)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	vol.AddBytes("my-roms/README.txt", myRomsReadme)
	for _, sys := range []string{"snes", "gba", "gb", "gbc", "nes", "n64", "nds", "megadrive", "psx"} {
		vol.Dir("my-roms/" + sys)
	}
	if o.GamesDir != "" {
		if err := addGames(vol, o.GamesDir, rep); err != nil {
			return nil, err
		}
	}

	vol.Layout(fatFreeSpace)
	c.fatStart = int64(len(c.loader))
	vol.SetHiddenSectors(uint32(c.fatStart / sectorSize))
	end := c.fatStart + roundMiB(vol.Size())
	if o.Ollie {
		c.ollieStart = end
		c.ollieSize = roundMiB(c.rootSize + ollieFreeSpace)
		end += c.ollieSize
	}
	c.shareStart = end
	c.Size = end + shareSize
	logf("the card needs %.1f GB", float64(c.Size)/1e9)
	return c, nil
}

func roundMiB(n int64) int64 { return (n + mib - 1) / mib * mib }

// addGames copies a folder of games into my-roms; Etros adds them, and
// applies any ROM-hack patches, when it next starts.
func addGames(vol *FATVolume, dir string, rep Reporter) error {
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			return nil
		}
		if !strings.Contains(rel, "/") {
			rep.Log("skipping " + rel + ": put games in a folder named after their console (snes, gba, ...)")
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if vol.Lookup("my-roms/"+rel) != nil {
			return nil
		}
		rep.Log("adding " + rel)
		vol.AddOpener("my-roms/"+rel, info.Size(), func() (io.ReadCloser, error) { return os.Open(p) })
		return nil
	})
}

// rpiRootPartition finds the root partition in a Raspberry Pi OS image.
func rpiRootPartition(path string) (start, size int64, err error) {
	r, closer, err := openImage(path)
	if err != nil {
		return 0, 0, err
	}
	defer closer()
	head := make([]byte, 512)
	if _, err := io.ReadFull(r, head); err != nil {
		return 0, 0, fmt.Errorf("Raspberry Pi OS image: %w", err)
	}
	parts := readMBR(head)
	if len(parts) < 2 || parts[1].kind != 0x83 {
		return 0, 0, fmt.Errorf("Raspberry Pi OS image: no Linux root partition where expected")
	}
	return int64(parts[1].start) * 512, int64(parts[1].count) * 512, nil
}

// openImage opens a disk image, decompressing .xz and .gz on the fly.
func openImage(path string) (io.Reader, func(), error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	br := bufio.NewReaderSize(f, 4*mib)
	switch {
	case strings.HasSuffix(path, ".xz"):
		x, err := xz.NewReader(br)
		if err != nil {
			f.Close()
			return nil, nil, err
		}
		return x, func() { f.Close() }, nil
	case strings.HasSuffix(path, ".gz"):
		g, err := gzip.NewReader(br)
		if err != nil {
			f.Close()
			return nil, nil, err
		}
		return g, func() { f.Close() }, nil
	}
	return br, func() { f.Close() }, nil
}

func (c *Card) mbr() []byte {
	b := make([]byte, 512)
	copy(b, c.loader[:440]) // keep whatever boot code the R36S image has
	sig := make([]byte, 4)
	rand.Read(sig)
	copy(b[440:], sig)
	entry := func(i int, boot bool, kind byte, start, size int64) {
		e := b[446+16*i:]
		if boot {
			e[0] = 0x80
		}
		copy(e[1:4], []byte{0xFE, 0xFF, 0xFF})
		e[4] = kind
		copy(e[5:8], []byte{0xFE, 0xFF, 0xFF})
		binary.LittleEndian.PutUint32(e[8:], uint32(start/512))
		binary.LittleEndian.PutUint32(e[12:], uint32(size/512))
	}
	// Partition 2 must be the share, and last on the card: Batocera finds it
	// as "the partition after the boot one" and grows it to the end.
	entry(0, true, 0x0C, c.fatStart, c.fatEnd()-c.fatStart)
	entry(1, false, 0x83, c.shareStart, shareSize)
	if c.ollieSize > 0 {
		entry(2, false, 0x83, c.ollieStart, c.ollieSize)
	}
	b[510], b[511] = 0x55, 0xAA
	return b
}

// fatEnd is where the boot partition ends.
func (c *Card) fatEnd() int64 {
	if c.ollieSize > 0 {
		return c.ollieStart
	}
	return c.shareStart
}

// Write puts the card onto w, which must be at least c.Size bytes.
func (c *Card) Write(w io.WriterAt, rep Reporter) error {
	sw := &seqWriter{w: w}
	total := c.Size
	progress := func() { rep.Progress(sw.done.Load(), total) }

	// The partition table goes on last. Until then the card has none, so
	// Windows can't spot the new partitions half way through, mount them,
	// and refuse further writes to them; and a card whose writing was
	// interrupted doesn't look like a working one.
	rep.Stage("Writing the R36S bootloader")
	sw.Write(make([]byte, 512))
	sw.Write(c.loader[512:])
	if sw.err != nil {
		return sw.err
	}

	rep.Stage("Writing the boot partition (Batocera for both devices)")
	stop := sw.tick(progress)
	streams := map[string]func(func(string, io.Reader) error) error{}
	for _, board := range []string{"pi5", "r36s"} {
		board := board
		streams[board] = func(emit func(string, io.Reader) error) error {
			return eachTarFile(c.boots[board], func(h *tar.Header, r io.Reader) error {
				if !c.streamed[board][h.Name] {
					return nil
				}
				return emit(h.Name, r)
			})
		}
	}
	err := c.vol.WriteTo(sw, streams)
	stop()
	if err == nil {
		err = sw.err
	}
	if err != nil {
		return err
	}

	if c.ollieSize > 0 {
		rep.Stage("Writing Ollie OS")
		sw.SkipTo(c.ollieStart)
		r, closer, err := openImage(c.rpiOS)
		if err != nil {
			return err
		}
		stop := sw.tick(progress)
		if _, err = io.CopyN(io.Discard, r, c.rootStart); err == nil {
			_, err = io.CopyN(sw, r, c.rootSize)
		}
		stop()
		closer()
		if err != nil {
			return fmt.Errorf("copying Raspberry Pi OS: %w", err)
		}
	}

	rep.Stage("Writing the storage partition")
	sw.SkipTo(c.shareStart)
	g, err := gzip.NewReader(bytes.NewReader(shareImage))
	if err != nil {
		return err
	}
	if _, err := io.Copy(sw, g); err != nil {
		return err
	}
	if err := sw.Flush(); err != nil {
		return err
	}
	rep.Stage("Writing the partition table")
	if _, err := w.WriteAt(c.mbr(), 0); err != nil {
		return err
	}
	progress()
	return nil
}

// seqWriter writes forwards through a disk in sector-aligned blocks of up to
// 4 MiB, as raw disks on Windows require.
type seqWriter struct {
	w    io.WriterAt
	off  int64 // where the buffer starts
	done atomic.Int64
	buf  []byte
	err  error
}

const block = 4 * mib

func (s *seqWriter) Write(p []byte) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	n := len(p)
	if s.buf == nil {
		s.buf = make([]byte, 0, block)
	}
	for len(p) > 0 {
		end := s.off + int64(len(s.buf))
		room := int(block - end%block) // stop at the next 4 MiB boundary
		k := min(room, len(p))
		s.buf = append(s.buf, p[:k]...)
		p = p[k:]
		if (s.off+int64(len(s.buf)))%block == 0 {
			if err := s.flushBuf(); err != nil {
				return 0, err
			}
		}
	}
	return n, nil
}

func (s *seqWriter) flushBuf() error {
	if len(s.buf) == 0 {
		return nil
	}
	if _, err := s.w.WriteAt(s.buf, s.off); err != nil {
		s.err = err
		return err
	}
	s.off += int64(len(s.buf))
	s.done.Store(s.off)
	s.buf = s.buf[:0]
	return nil
}

// Flush writes what is buffered, padded to a whole sector.
func (s *seqWriter) Flush() error {
	if s.err != nil {
		return s.err
	}
	if r := len(s.buf) % sectorSize; r != 0 {
		s.buf = append(s.buf, make([]byte, sectorSize-r)...)
	}
	return s.flushBuf()
}

// SkipTo moves forward to off, which must be sector-aligned.
func (s *seqWriter) SkipTo(off int64) {
	if s.Flush() != nil {
		return
	}
	if off < s.off {
		s.err = fmt.Errorf("internal error: seeking backwards from %d to %d", s.off, off)
		return
	}
	s.off = off
	s.done.Store(off)
}

// tick calls f four times a second until the returned stop is called.
func (s *seqWriter) tick(f func()) (stop func()) {
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(250 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				f()
			}
		}
	}()
	return func() { close(done) }
}
