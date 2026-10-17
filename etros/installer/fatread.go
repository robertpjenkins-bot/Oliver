package main

// Reading the boot partition out of a compressed Batocera disk image
// (.img.gz or .img.xz, as batocera.org offers them) without unpacking it to
// disk. The image can only be read front to back, so each read is a pass
// over the whole thing that picks up what it needs as it goes by.

import (
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf16"
)

// bootSource is one device's Batocera boot partition.
type bootSource interface {
	// Files lists every file, small ones with their contents, in the order
	// Stream delivers the big ones.
	Files() ([]bootFile, error)
	// Stream calls fn with the contents of each file named in want, in
	// Files order.
	Stream(want map[string]bool, fn func(name string, r io.Reader) error) error
	// Loader is the start of the disk image, ahead of the boot partition,
	// or nil if this source is not a disk image.
	Loader() []byte
}

// tarSource is a Batocera boot.tar.xz.
type tarSource struct{ path string }

func (t tarSource) Files() ([]bootFile, error) { return readBoot(t.path) }
func (t tarSource) Loader() []byte             { return nil }
func (t tarSource) Stream(want map[string]bool, fn func(string, io.Reader) error) error {
	return eachTarFile(t.path, func(h *tarHeader, r io.Reader) error {
		if !want[h.Name] {
			return nil
		}
		return fn(h.Name, r)
	})
}

type fatEntry struct {
	path    string
	size    int64
	first   uint32
	isDir   bool
	chain   []uint32
	data    []byte // collected contents (directories, small files)
	collect bool
}

// imageSource is a Batocera disk image.
type imageSource struct {
	path   string
	loader []byte

	partStart   int64
	clusterSize int64
	dataStart   int64 // within the partition
	rootCluster uint32
	fat         []uint32
	files       []*fatEntry // in cluster order
}

func (s *imageSource) Loader() []byte { return s.loader }

func (s *imageSource) chainOf(first uint32) ([]uint32, error) {
	var chain []uint32
	for c := first; c >= 2 && c < 0x0FFFFFF8; c = s.fat[c] & 0x0FFFFFFF {
		if int(c) >= len(s.fat) || len(chain) > len(s.fat) {
			return nil, fmt.Errorf("damaged file system (bad cluster chain)")
		}
		chain = append(chain, c)
	}
	return chain, nil
}

// header reads the partition table, the start of the disk (for the R36S
// bootloader) and the file system's allocation table.
func (s *imageSource) header() (io.Reader, func(), error) {
	r, closer, err := openImage(s.path)
	if err != nil {
		return nil, nil, err
	}
	fail := func(e error) (io.Reader, func(), error) { closer(); return nil, nil, e }
	mbr := make([]byte, 512)
	if _, err := io.ReadFull(r, mbr); err != nil {
		return fail(err)
	}
	parts := readMBR(mbr)
	if len(parts) == 0 || parts[0].kind != 0x0C && parts[0].kind != 0x0B {
		return fail(fmt.Errorf("%s doesn't start with a FAT boot partition; is it a Batocera image?", s.path))
	}
	s.partStart = int64(parts[0].start) * 512
	rest := make([]byte, s.partStart-512)
	if _, err := io.ReadFull(r, rest); err != nil {
		return fail(err)
	}
	s.loader = append(mbr, rest...)

	boot := make([]byte, 512)
	if _, err := io.ReadFull(r, boot); err != nil {
		return fail(err)
	}
	bps := int64(binary.LittleEndian.Uint16(boot[11:]))
	spc := int64(boot[13])
	reserved := int64(binary.LittleEndian.Uint16(boot[14:]))
	nfats := int64(boot[16])
	fatSecs := int64(binary.LittleEndian.Uint32(boot[36:]))
	if bps != 512 || spc == 0 || fatSecs == 0 || string(boot[82:87]) != "FAT32" {
		return fail(fmt.Errorf("%s: the boot partition isn't FAT32", s.path))
	}
	s.clusterSize = bps * spc
	s.rootCluster = binary.LittleEndian.Uint32(boot[44:])
	s.dataStart = (reserved + nfats*fatSecs) * bps
	if _, err := io.CopyN(io.Discard, r, reserved*bps-512); err != nil {
		return fail(err)
	}
	raw := make([]byte, fatSecs*bps)
	if _, err := io.ReadFull(r, raw); err != nil {
		return fail(err)
	}
	s.fat = make([]uint32, len(raw)/4)
	for i := range s.fat {
		s.fat[i] = binary.LittleEndian.Uint32(raw[i*4:])
	}
	if _, err := io.CopyN(io.Discard, r, (nfats-1)*fatSecs*bps); err != nil {
		return fail(err)
	}
	return r, closer, nil
}

// pass reads the data area once, front to back, calling visit for each
// cluster that belongs to a known entry.
func (s *imageSource) pass(owner map[uint32]*fatEntry, visit func(e *fatEntry, c uint32, data []byte) error) error {
	r, closer, err := s.header()
	if err != nil {
		return err
	}
	defer closer()
	// visit may claim more clusters, so work out where to stop as we go.
	var last uint32
	known := -1
	buf := make([]byte, s.clusterSize)
	for c := uint32(2); ; c++ {
		if len(owner) != known {
			known = len(owner)
			last = 0
			for k := range owner {
				last = max(last, k)
			}
		}
		if c > last {
			return nil
		}
		e, ok := owner[c]
		if !ok {
			if _, err := io.CopyN(io.Discard, r, s.clusterSize); err != nil {
				return err
			}
			continue
		}
		if _, err := io.ReadFull(r, buf); err != nil {
			return err
		}
		if err := visit(e, c, buf); err != nil {
			return err
		}
	}
}

func (s *imageSource) Files() ([]bootFile, error) {
	if s.files == nil {
		if err := s.scan(); err != nil {
			return nil, err
		}
	}
	var out []bootFile
	for _, e := range s.files {
		if e.isDir {
			continue
		}
		f := bootFile{path: e.path, size: e.size}
		if e.collect {
			f.data = e.data[:e.size]
		}
		out = append(out, f)
	}
	return out, nil
}

// scan reads the directory tree and the small files. Directories normally
// come before their contents, so one pass usually finds everything; if one
// turns up after a file it lists has gone by, another pass collects it.
func (s *imageSource) scan() error {
	if _, closer, err := s.header(); err != nil {
		return err
	} else {
		closer()
	}
	root := &fatEntry{path: "", isDir: true, first: s.rootCluster, collect: true}
	var err error
	if root.chain, err = s.chainOf(s.rootCluster); err != nil {
		return err
	}
	all := []*fatEntry{root}
	for passes := 0; ; passes++ {
		if passes > 8 {
			return fmt.Errorf("%s: couldn't read the boot partition's directories", s.path)
		}
		owner := map[uint32]*fatEntry{}
		pending := 0
		for _, e := range all {
			if e.collect && int64(len(e.data)) < int64(len(e.chain))*s.clusterSize {
				e.data = e.data[:0]
				for _, c := range e.chain {
					owner[c] = e
				}
				pending++
			}
		}
		if pending == 0 {
			break
		}
		err := s.pass(owner, func(e *fatEntry, c uint32, data []byte) error {
			e.data = append(e.data, data...)
			if !e.isDir || int64(len(e.data)) < int64(len(e.chain))*s.clusterSize {
				return nil
			}
			kids, err := parseDir(e.data, e.path)
			if err != nil {
				return err
			}
			for _, k := range kids {
				if k.chain, err = s.chainOf(k.first); err != nil {
					return err
				}
				k.collect = k.isDir || k.size <= smallFile || mustRead[k.path]
				all = append(all, k)
				// Claim the clusters still ahead of us in this pass.
				ahead := true
				for _, kc := range k.chain {
					if kc <= c {
						ahead = false
					}
				}
				if k.collect && ahead {
					for _, kc := range k.chain {
						owner[kc] = k
					}
				}
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("%s: %w", s.path, err)
		}
	}
	for _, e := range all[1:] {
		if !e.collect {
			// Big files are copied as they go by, so they must be stored in
			// one piece, as freshly made images are.
			for i := 1; i < len(e.chain); i++ {
				if e.chain[i] != e.chain[i-1]+1 {
					return fmt.Errorf("%s: %s is fragmented; use a freshly downloaded image", s.path, e.path)
				}
			}
		}
		if int64(len(e.chain))*s.clusterSize < e.size {
			return fmt.Errorf("%s: %s is damaged", s.path, e.path)
		}
	}
	s.files = all[1:]
	sort.SliceStable(s.files, func(i, j int) bool { return s.files[i].first < s.files[j].first })
	return nil
}

func (s *imageSource) Stream(want map[string]bool, fn func(string, io.Reader) error) error {
	owner := map[uint32]*fatEntry{}
	var order []*fatEntry
	for _, e := range s.files {
		if want[e.path] && e.size > 0 {
			order = append(order, e)
			for _, c := range e.chain {
				owner[c] = e
			}
		}
	}
	// Each file's clusters are consecutive and files don't overlap, so a
	// file is a contiguous stretch of the pass. Hand each to fn through a
	// pipe as its clusters go by.
	var pw *io.PipeWriter
	var cur *fatEntry
	var written int64
	errc := make(chan error, 1)
	finish := func() error {
		if pw == nil {
			return nil
		}
		pw.Close()
		err := <-errc
		pw, cur = nil, nil
		return err
	}
	err := s.pass(owner, func(e *fatEntry, c uint32, data []byte) error {
		if e != cur {
			if err := finish(); err != nil {
				return err
			}
			var pr *io.PipeReader
			pr, pw = io.Pipe()
			cur, written = e, 0
			go func(name string) {
				err := fn(name, pr)
				pr.CloseWithError(err)
				errc <- err
			}(e.path)
		}
		n := min(int64(len(data)), e.size-written)
		written += n
		if _, err := pw.Write(data[:n]); err != nil {
			return err
		}
		return nil
	})
	if ferr := finish(); err == nil {
		err = ferr
	}
	return err
}

// parseDir reads the entries of a FAT directory.
func parseDir(b []byte, dir string) ([]*fatEntry, error) {
	var out []*fatEntry
	var lfn []uint16
	var lfnParts [][]uint16
	for p := 0; p+32 <= len(b); p += 32 {
		e := b[p : p+32]
		if e[0] == 0 {
			break
		}
		if e[0] == 0xE5 {
			lfnParts = nil
			continue
		}
		if e[11] == 0x0F {
			part := make([]uint16, 0, 13)
			for _, off := range []int{1, 3, 5, 7, 9, 14, 16, 18, 20, 22, 24, 28, 30} {
				part = append(part, binary.LittleEndian.Uint16(e[off:]))
			}
			if e[0]&0x40 != 0 {
				lfnParts = nil
			}
			lfnParts = append([][]uint16{part}, lfnParts...)
			continue
		}
		if e[11]&0x08 != 0 { // volume label
			lfnParts = nil
			continue
		}
		name := ""
		if lfnParts != nil {
			lfn = lfn[:0]
			for _, part := range lfnParts {
				lfn = append(lfn, part...)
			}
			for i, u := range lfn {
				if u == 0 {
					lfn = lfn[:i]
					break
				}
			}
			name = string(utf16.Decode(lfn))
		} else {
			base := strings.TrimRight(string(e[0:8]), " ")
			ext := strings.TrimRight(string(e[8:11]), " ")
			if e[12]&0x08 != 0 {
				base = strings.ToLower(base)
			}
			if e[12]&0x10 != 0 {
				ext = strings.ToLower(ext)
			}
			name = base
			if ext != "" {
				name += "." + ext
			}
		}
		lfnParts = nil
		if name == "." || name == ".." {
			continue
		}
		first := uint32(binary.LittleEndian.Uint16(e[20:]))<<16 | uint32(binary.LittleEndian.Uint16(e[26:]))
		path := name
		if dir != "" {
			path = dir + "/" + name
		}
		out = append(out, &fatEntry{
			path:  path,
			size:  int64(binary.LittleEndian.Uint32(e[28:])),
			first: first,
			isDir: e[11]&0x10 != 0,
		})
	}
	return out, nil
}
