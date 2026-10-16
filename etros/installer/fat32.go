package main

// A FAT32 file system writer that lays every file out in one pass and then
// streams the whole volume out front to back. That lets the installer write
// the boot partition straight onto the SD card without building it in a
// temporary file first, and copy Batocera's multi-gigabyte system images in
// the order they come out of their archives.

import (
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf16"
)

const (
	sectorSize      = 512
	reservedSectors = 32
	numFATs         = 2
	dirEntrySize    = 32
)

// fatNode is a file or directory in the volume being built.
type fatNode struct {
	name     string
	dir      bool
	children []*fatNode
	parent   *fatNode

	size int64
	// open returns the file's contents when the volume is written. Files
	// whose open is nil must be fed through the streams given to WriteTo.
	open func() (io.ReadCloser, error)
	// streamKey marks a file whose contents arrive in a stream (see
	// FATVolume.WriteTo) instead of through open.
	streamKey string

	firstCluster uint32
	clusters     uint32
	dirData      []byte
}

// FATVolume collects files, then lays them out and writes the volume.
type FATVolume struct {
	Label   string
	root    *fatNode
	order   []*fatNode // allocation order of files with contents
	cluster uint32     // bytes per cluster

	totalSectors uint32
	fatSectors   uint32
	totalCluster uint32
	nextFree     uint32
	hiddenSecs   uint32
	volumeID     uint32
}

func NewFATVolume(label string) *FATVolume {
	return &FATVolume{Label: label, root: &fatNode{dir: true}, volumeID: 0x45545253}
}

// Dir returns the directory at path, creating it and its parents.
func (v *FATVolume) Dir(path string) *fatNode {
	n := v.root
	for _, part := range strings.Split(strings.Trim(path, "/"), "/") {
		if part == "" {
			continue
		}
		var next *fatNode
		for _, c := range n.children {
			if strings.EqualFold(c.name, part) {
				next = c
				break
			}
		}
		if next == nil {
			next = &fatNode{name: part, dir: true, parent: n}
			n.children = append(n.children, next)
		} else if !next.dir {
			panic("fat: " + path + " is a file")
		}
		n = next
	}
	return n
}

// Lookup finds a file or directory, ignoring case as FAT does.
func (v *FATVolume) Lookup(path string) *fatNode {
	n := v.root
	for _, part := range strings.Split(strings.Trim(path, "/"), "/") {
		if part == "" {
			continue
		}
		var next *fatNode
		for _, c := range n.children {
			if strings.EqualFold(c.name, part) {
				next = c
				break
			}
		}
		if next == nil {
			return nil
		}
		n = next
	}
	return n
}

func (v *FATVolume) addFile(path string, size int64) *fatNode {
	if v.Lookup(path) != nil {
		panic("fat: " + path + " added twice")
	}
	i := strings.LastIndex(path, "/")
	dir := v.root
	name := path
	if i >= 0 {
		dir = v.Dir(path[:i])
		name = path[i+1:]
	}
	n := &fatNode{name: name, size: size, parent: dir}
	dir.children = append(dir.children, n)
	v.order = append(v.order, n)
	return n
}

// AddBytes adds a file held in memory.
func (v *FATVolume) AddBytes(path string, data []byte) {
	n := v.addFile(path, int64(len(data)))
	n.open = func() (io.ReadCloser, error) {
		return io.NopCloser(strings.NewReader(string(data))), nil
	}
}

// AddOpener adds a file read from elsewhere when the volume is written.
func (v *FATVolume) AddOpener(path string, size int64, open func() (io.ReadCloser, error)) {
	n := v.addFile(path, size)
	n.open = open
}

// AddStreamed adds a file whose contents will come from a stream passed to
// WriteTo, under key. Streamed files are written in the order they are
// added, so add them in the order the stream delivers them.
func (v *FATVolume) AddStreamed(path string, size int64, key string) {
	n := v.addFile(path, size)
	n.streamKey = key
}

// Layout assigns clusters to everything. freeBytes is extra room to leave
// on the volume. It must be called before Size or WriteTo.
func (v *FATVolume) Layout(freeBytes int64) {
	v.cluster = 32 * 1024

	// Directory sizes depend only on their entries.
	var dirs []*fatNode
	var walk func(n *fatNode)
	walk = func(n *fatNode) {
		if n.dir {
			dirs = append(dirs, n)
			sort.SliceStable(n.children, func(i, j int) bool { return n.children[i].name < n.children[j].name })
			for _, c := range n.children {
				walk(c)
			}
		}
	}
	walk(v.root)

	// Files from each stream must sit together, in the order added, so that
	// each stream is read once from start to end.
	rank := map[string]int{"": 0}
	for _, f := range v.order {
		if _, ok := rank[f.streamKey]; !ok {
			rank[f.streamKey] = len(rank)
		}
	}
	sort.SliceStable(v.order, func(i, j int) bool { return rank[v.order[i].streamKey] < rank[v.order[j].streamKey] })

	next := uint32(2)
	alloc := func(n *fatNode, bytes int64) {
		c := uint32((bytes + int64(v.cluster) - 1) / int64(v.cluster))
		if n.dir && c == 0 {
			c = 1
		}
		if c == 0 {
			return
		}
		n.firstCluster = next
		n.clusters = c
		next += c
	}
	for _, d := range dirs {
		alloc(d, int64(dirEntriesLen(d))*dirEntrySize)
	}
	for _, f := range v.order {
		alloc(f, f.size)
	}
	used := next - 2
	free := uint32(freeBytes / int64(v.cluster))
	v.totalCluster = used + free
	if v.totalCluster < 65525 { // the smallest a FAT32 volume may be
		v.totalCluster = 65525
	}
	v.nextFree = next
	v.fatSectors = (v.totalCluster+2)*4/sectorSize + 1
	v.totalSectors = reservedSectors + numFATs*v.fatSectors + v.totalCluster*(v.cluster/sectorSize)

	for _, d := range dirs {
		d.dirData = buildDir(d, v.cluster*d.clusters)
	}
}

// Size is the volume's size in bytes.
func (v *FATVolume) Size() int64 { return int64(v.totalSectors) * sectorSize }

// SetHiddenSectors records where the partition starts, as the boot sector
// expects.
func (v *FATVolume) SetHiddenSectors(lba uint32) { v.hiddenSecs = lba }

func dirEntriesLen(d *fatNode) int {
	n := 1 // volume label in root, "." and ".." elsewhere
	if d.parent != nil {
		n = 2
	}
	for _, c := range d.children {
		n += 1 + lfnCount(c.name)
	}
	return n
}

func lfnCount(name string) int {
	return (len(utf16.Encode([]rune(name))) + 12) / 13
}

// shortName makes a unique 8.3 name. Every entry also carries its full long
// name, so this only has to be valid and unique within the directory.
func shortName(i int, name string) [11]byte {
	var s [11]byte
	copy(s[:], fmt.Sprintf("ET%06X", i))
	ext := ""
	if j := strings.LastIndex(name, "."); j > 0 {
		ext = strings.ToUpper(name[j+1:])
	}
	k := 8
	for _, r := range ext {
		if k == 11 {
			break
		}
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			s[k] = byte(r)
			k++
		}
	}
	for ; k < 11; k++ {
		s[k] = ' '
	}
	return s
}

func lfnChecksum(s [11]byte) byte {
	var sum byte
	for _, b := range s {
		sum = (sum>>1 | sum<<7) + b
	}
	return sum
}

// 2026-01-01 00:00, the timestamp on every entry.
const fatDate = (2026-1980)<<9 | 1<<5 | 1

func putEntry(b []byte, name [11]byte, attr byte, cluster uint32, size uint32) {
	copy(b[0:11], name[:])
	b[11] = attr
	binary.LittleEndian.PutUint16(b[14:], 0)
	binary.LittleEndian.PutUint16(b[16:], fatDate)
	binary.LittleEndian.PutUint16(b[18:], fatDate)
	binary.LittleEndian.PutUint16(b[20:], uint16(cluster>>16))
	binary.LittleEndian.PutUint16(b[22:], 0)
	binary.LittleEndian.PutUint16(b[24:], fatDate)
	binary.LittleEndian.PutUint16(b[26:], uint16(cluster))
	binary.LittleEndian.PutUint32(b[28:], size)
}

func buildDir(d *fatNode, bytes uint32) []byte {
	out := make([]byte, bytes)
	p := 0
	if d.parent == nil {
		var label [11]byte
		copy(label[:], fmt.Sprintf("%-11s", "BATOCERA"))
		putEntry(out[p:], label, 0x08, 0, 0)
		p += dirEntrySize
	} else {
		var dot, dotdot [11]byte
		copy(dot[:], ".          ")
		copy(dotdot[:], "..         ")
		putEntry(out[p:], dot, 0x10, d.firstCluster, 0)
		p += dirEntrySize
		parent := d.parent.firstCluster
		if d.parent.parent == nil {
			parent = 0 // ".." pointing at the root is written as cluster 0
		}
		putEntry(out[p:], dotdot, 0x10, parent, 0)
		p += dirEntrySize
	}
	for i, c := range d.children {
		short := shortName(i, c.name)
		sum := lfnChecksum(short)
		units := utf16.Encode([]rune(c.name))
		n := lfnCount(c.name)
		for seq := n; seq >= 1; seq-- {
			e := out[p : p+dirEntrySize]
			e[0] = byte(seq)
			if seq == n {
				e[0] |= 0x40
			}
			e[11] = 0x0F
			e[13] = sum
			offsets := []int{1, 3, 5, 7, 9, 14, 16, 18, 20, 22, 24, 28, 30}
			for k, off := range offsets {
				idx := (seq-1)*13 + k
				var u uint16 = 0xFFFF
				if idx < len(units) {
					u = units[idx]
				} else if idx == len(units) {
					u = 0
				}
				binary.LittleEndian.PutUint16(e[off:], u)
			}
			p += dirEntrySize
		}
		attr := byte(0x20)
		size := uint32(c.size)
		if c.dir {
			attr, size = 0x10, 0
		}
		putEntry(out[p:], short, attr, c.firstCluster, size)
		p += dirEntrySize
	}
	return out
}

func (v *FATVolume) bootSector() []byte {
	b := make([]byte, sectorSize)
	copy(b, []byte{0xEB, 0x58, 0x90})
	copy(b[3:], "ETROS   ")
	binary.LittleEndian.PutUint16(b[11:], sectorSize)
	b[13] = byte(v.cluster / sectorSize)
	binary.LittleEndian.PutUint16(b[14:], reservedSectors)
	b[16] = numFATs
	b[21] = 0xF8
	binary.LittleEndian.PutUint16(b[24:], 63)
	binary.LittleEndian.PutUint16(b[26:], 255)
	binary.LittleEndian.PutUint32(b[28:], v.hiddenSecs)
	binary.LittleEndian.PutUint32(b[32:], v.totalSectors)
	binary.LittleEndian.PutUint32(b[36:], v.fatSectors)
	binary.LittleEndian.PutUint32(b[44:], 2) // root directory cluster
	binary.LittleEndian.PutUint16(b[48:], 1) // FSInfo sector
	binary.LittleEndian.PutUint16(b[50:], 6) // backup boot sector
	b[64] = 0x80
	b[66] = 0x29
	binary.LittleEndian.PutUint32(b[67:], v.volumeID)
	copy(b[71:], fmt.Sprintf("%-11s", v.Label))
	copy(b[82:], "FAT32   ")
	b[510], b[511] = 0x55, 0xAA
	return b
}

func (v *FATVolume) fsInfo() []byte {
	b := make([]byte, sectorSize)
	binary.LittleEndian.PutUint32(b[0:], 0x41615252)
	binary.LittleEndian.PutUint32(b[484:], 0x61417272)
	binary.LittleEndian.PutUint32(b[488:], v.totalCluster+2-v.nextFree)
	binary.LittleEndian.PutUint32(b[492:], v.nextFree)
	binary.LittleEndian.PutUint32(b[508:], 0xAA550000)
	return b
}

// fatTable is one copy of the allocation table.
func (v *FATVolume) fatTable() []byte {
	t := make([]byte, int(v.fatSectors)*sectorSize)
	binary.LittleEndian.PutUint32(t[0:], 0x0FFFFFF8)
	binary.LittleEndian.PutUint32(t[4:], 0x0FFFFFFF)
	var mark func(n *fatNode)
	mark = func(n *fatNode) {
		for i := uint32(0); i < n.clusters; i++ {
			c := n.firstCluster + i
			val := c + 1
			if i == n.clusters-1 {
				val = 0x0FFFFFFF
			}
			binary.LittleEndian.PutUint32(t[c*4:], val)
		}
		for _, ch := range n.children {
			mark(ch)
		}
	}
	mark(v.root)
	return t
}

// WriteTo writes the volume. streams supplies the contents of streamed
// files: for each key, a function that is called once and must write the
// files added under that key, in order, by calling emit for each.
func (v *FATVolume) WriteTo(w io.Writer, streams map[string]func(emit func(path string, r io.Reader) error) error) error {
	boot := v.bootSector()
	info := v.fsInfo()
	reserved := make([]byte, reservedSectors*sectorSize)
	copy(reserved[0:], boot)
	copy(reserved[sectorSize:], info)
	copy(reserved[6*sectorSize:], boot)
	copy(reserved[7*sectorSize:], info)
	if _, err := w.Write(reserved); err != nil {
		return err
	}
	fat := v.fatTable()
	for i := 0; i < numFATs; i++ {
		if _, err := w.Write(fat); err != nil {
			return err
		}
	}

	// The data region, in cluster order: directories, then files.
	var written int64
	pad := func(n *fatNode, wrote int64) error {
		want := int64(n.clusters) * int64(v.cluster)
		if wrote > n.size {
			return fmt.Errorf("%s: got more data than its %d bytes", n.name, n.size)
		}
		if wrote < n.size {
			return fmt.Errorf("%s: got %d of its %d bytes", n.name, wrote, n.size)
		}
		_, err := w.Write(make([]byte, want-wrote))
		written += want
		return err
	}
	var dirs []*fatNode
	var walk func(n *fatNode)
	walk = func(n *fatNode) {
		if n.dir {
			dirs = append(dirs, n)
			for _, c := range n.children {
				walk(c)
			}
		}
	}
	walk(v.root)
	sort.Slice(dirs, func(i, j int) bool { return dirs[i].firstCluster < dirs[j].firstCluster })
	for _, d := range dirs {
		if _, err := w.Write(d.dirData); err != nil {
			return err
		}
		written += int64(len(d.dirData))
	}

	i := 0
	for i < len(v.order) {
		n := v.order[i]
		if n.streamKey == "" {
			if n.size > 0 {
				r, err := n.open()
				if err != nil {
					return err
				}
				k, err := io.Copy(w, r)
				r.Close()
				if err != nil {
					return err
				}
				if err := pad(n, k); err != nil {
					return err
				}
			}
			i++
			continue
		}
		// A run of files from one stream.
		key := n.streamKey
		j := i
		for j < len(v.order) && v.order[j].streamKey == key {
			j++
		}
		run := v.order[i:j]
		stream, ok := streams[key]
		if !ok {
			return fmt.Errorf("no stream for %s", key)
		}
		k := 0
		err := stream(func(path string, r io.Reader) error {
			if k >= len(run) {
				return fmt.Errorf("stream %s: unexpected file %s", key, path)
			}
			n := run[k]
			k++
			got, err := io.Copy(w, io.LimitReader(r, n.size+1))
			if err != nil {
				return err
			}
			return pad(n, got)
		})
		if err != nil {
			return err
		}
		for ; k < len(run); k++ { // empty files the stream did not mention
			if run[k].size != 0 {
				return fmt.Errorf("stream %s ended before %s", key, run[k].name)
			}
		}
		i = j
	}

	// Free space after the last file.
	rest := int64(v.totalSectors-reservedSectors-numFATs*v.fatSectors)*sectorSize - written
	_, err := io.CopyN(w, zeroReader{}, rest)
	return err
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	clear(p)
	return len(p), nil
}
