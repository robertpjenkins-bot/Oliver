package main

// Etros image files: saving a card as one, and writing one onto a card.

import (
	"archive/zip"
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/ulikunitz/xz"
)

// gzTarget saves a card as a .img.gz, which balenaEtcher and Raspberry Pi
// Imager write directly. Writes must move forwards; gaps become zeros.
type gzTarget struct {
	f    *os.File
	bw   *bufio.Writer
	gz   *gzip.Writer
	pos  int64
	size int64
}

func newGzTarget(path string, size int64) (*gzTarget, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	bw := bufio.NewWriterSize(f, 4*mib)
	gz, _ := gzip.NewWriterLevel(bw, gzip.BestSpeed)
	return &gzTarget{f: f, bw: bw, gz: gz, size: size}, nil
}

func (g *gzTarget) WriteAt(p []byte, off int64) (int, error) {
	if off < g.pos {
		return 0, fmt.Errorf("internal error: image writes went backwards")
	}
	if off > g.pos {
		if _, err := io.CopyN(g.gz, zeroReader{}, off-g.pos); err != nil {
			return 0, err
		}
		g.pos = off
	}
	n, err := g.gz.Write(p)
	g.pos += int64(n)
	return n, err
}

func (g *gzTarget) Finish() error {
	if g.pos < g.size {
		if _, err := io.CopyN(g.gz, zeroReader{}, g.size-g.pos); err != nil {
			return err
		}
	}
	if err := g.gz.Close(); err != nil {
		return err
	}
	if err := g.bw.Flush(); err != nil {
		return err
	}
	chownToUser(g.f.Name())
	return g.f.Close()
}

// openEtrosImage opens an Etros image for reading, whatever it is packed
// in: .img, .img.gz, .img.xz, or a .zip holding one of those. counted
// reports how much of the file on disk has been read, for progress.
func openEtrosImage(path string) (r io.Reader, counted *countingReader, total int64, closer func(), err error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, nil, 0, nil, err
	}
	lower := strings.ToLower(path)
	if strings.HasSuffix(lower, ".zip") {
		z, err := zip.OpenReader(path)
		if err != nil {
			return nil, nil, 0, nil, err
		}
		for _, f := range z.File {
			name := strings.ToLower(f.Name)
			if !hasSuffix(name, imageExts...) {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				z.Close()
				return nil, nil, 0, nil, err
			}
			c := &countingReader{r: rc, total: int64(f.UncompressedSize64)}
			inner, err := decompressByName(name, c)
			if err != nil {
				rc.Close()
				z.Close()
				return nil, nil, 0, nil, err
			}
			return inner, c, c.total, func() { rc.Close(); z.Close() }, nil
		}
		z.Close()
		return nil, nil, 0, nil, fmt.Errorf("%s has no Etros image inside", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, 0, nil, err
	}
	c := &countingReader{r: bufio.NewReaderSize(f, 4*mib), total: st.Size()}
	inner, err := decompressByName(lower, c)
	if err != nil {
		f.Close()
		return nil, nil, 0, nil, err
	}
	return inner, c, st.Size(), func() { f.Close() }, nil
}

func decompressByName(name string, r io.Reader) (io.Reader, error) {
	switch {
	case strings.HasSuffix(name, ".gz"):
		return gzip.NewReader(r)
	case strings.HasSuffix(name, ".xz"):
		return xz.NewReader(r)
	}
	return r, nil
}

// imageSize is how much of a card an image needs: the end of its last
// partition, from its partition table.
func imageSize(mbr []byte) (int64, error) {
	parts := readMBR(mbr)
	if len(parts) == 0 {
		return 0, fmt.Errorf("this doesn't look like a disk image (no partition table)")
	}
	var end int64
	for _, p := range parts {
		end = max(end, int64(p.start+p.count)*512)
	}
	return end, nil
}

// writeImage copies an Etros image onto a card. Its partition table goes on
// last, as in Card.Write.
func writeImage(path string, out writeTarget, outSize int64, rep Reporter) error {
	rep.Stage("Writing " + baseName(path) + " to the card")
	r, counted, total, closer, err := openEtrosImage(path)
	if err != nil {
		return err
	}
	defer closer()
	mbr := make([]byte, 512)
	if _, err := io.ReadFull(r, mbr); err != nil {
		return fmt.Errorf("reading %s: %w", baseName(path), err)
	}
	need, err := imageSize(mbr)
	if err != nil {
		return err
	}
	if outSize > 0 && need > outSize {
		return fmt.Errorf("the card is too small: this image needs %.1f GB, the card has %.1f GB", float64(need)/1e9, float64(outSize)/1e9)
	}
	counted.p = func(done, _ int64) { rep.Progress(done, total) }
	sw := &seqWriter{w: out, off: 512}
	if _, err := io.CopyN(sw, r, need-512); err != nil && err != io.EOF {
		return fmt.Errorf("writing the card: %w", err)
	}
	if err := sw.Flush(); err != nil {
		return err
	}
	if _, err := out.WriteAt(mbr, 0); err != nil {
		return err
	}
	rep.Progress(total, total)
	return nil
}

func baseName(p string) string {
	if i := strings.LastIndexAny(p, `/\`); i >= 0 {
		return p[i+1:]
	}
	return p
}

// chownToUser gives a file made under sudo back to the person who ran it.
func chownToUser(path string) {
	uid, err1 := strconv.Atoi(os.Getenv("SUDO_UID"))
	gid, err2 := strconv.Atoi(os.Getenv("SUDO_GID"))
	if err1 == nil && err2 == nil && os.Geteuid() == 0 {
		os.Chown(path, uid, gid)
	}
}
