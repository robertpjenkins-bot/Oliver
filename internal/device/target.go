package device

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Partition is one partition of a drive, such as /dev/sdb1.
type Partition struct {
	Path     string   `json:"path"`
	Name     string   `json:"name"`
	Number   int      `json:"number"`
	Size     int64    `json:"size_bytes"`
	Label    string   `json:"label,omitempty"`
	FSType   string   `json:"fs_type,omitempty"`
	Mounts   []Mount  `json:"mounts,omitempty"`
	Holders  []string `json:"holders,omitempty"`
	Swap     bool     `json:"swap,omitempty"`
	ReadOnly bool     `json:"read_only,omitempty"`
	Writable bool     `json:"writable"`
}

// partitions lists a drive's partitions. The kernel presents each one as a
// subdirectory of the drive's /sys entry that holds a "partition" file.
func (s *Scanner) partitions(d *Device, base string, mounts []Mount, swaps map[string]bool) []*Partition {
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var out []*Partition
	for _, e := range entries {
		dir := filepath.Join(base, e.Name())
		number, err := strconv.Atoi(readTrimmed(filepath.Join(dir, "partition")))
		if err != nil {
			continue
		}
		p := &Partition{
			Path:   filepath.Join(s.Dev, e.Name()),
			Name:   e.Name(),
			Number: number,
		}
		if sectors, err := strconv.ParseInt(readTrimmed(filepath.Join(dir, "size")), 10, 64); err == nil {
			p.Size = sectors * 512
		}
		for _, m := range mounts {
			if m.Source == p.Path {
				p.Mounts = append(p.Mounts, m)
			}
		}
		p.Holders = listDir(filepath.Join(dir, "holders"))
		p.Swap = swaps[p.Path]
		p.ReadOnly = readTrimmed(filepath.Join(dir, "ro")) == "1"
		udev := s.udevProperties(dir)
		p.Label, p.FSType = udev["ID_FS_LABEL"], udev["ID_FS_TYPE"]
		if s.Access != nil {
			_, p.Writable = s.Access(p.Path)
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}

// udevProperties reads what udev recorded about a device: the filesystem on
// it and the bus it hangs off. Reading the record rather than the device means
// it works without permission to open the device.
func (s *Scanner) udevProperties(sysDir string) map[string]string {
	props := map[string]string{}
	if s.Udev == "" {
		return props
	}
	id := readTrimmed(filepath.Join(sysDir, "dev")) // "8:17"
	if id == "" {
		return props
	}
	f, err := os.Open(filepath.Join(s.Udev, "b"+id))
	if err != nil {
		return props
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line, ok := strings.CutPrefix(scan.Text(), "E:")
		if !ok {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			props[k] = v
		}
	}
	return props
}

// readSwaps returns the devices in use as swap space.
func (s *Scanner) readSwaps() map[string]bool {
	out := map[string]bool{}
	if s.Swaps == "" {
		return out
	}
	data, err := os.ReadFile(s.Swaps)
	if err != nil {
		return out
	}
	for i, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if i == 0 || len(fields) == 0 { // the first line is a header
			continue
		}
		out[unescapeMount(fields[0])] = true
	}
	return out
}

func readTrimmed(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func listDir(path string) []string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		out = append(out, e.Name())
	}
	return out
}

// Target is somewhere an image can be written: a whole drive, or one of its
// partitions.
type Target struct {
	Path string `json:"path"`
	Size int64  `json:"size_bytes"`
	// Disk is the drive itself, or the drive the partition belongs to.
	Disk *Device `json:"-"`
	// Partition is nil when the target is the whole drive.
	Partition *Partition `json:"-"`
	Writable  bool       `json:"writable"`
}

// Whole reports whether the target is an entire drive.
func (t Target) Whole() bool { return t.Partition == nil }

// External reports whether the target is on a drive the user plugged in,
// rather than one built into the machine.
func (t Target) External() bool { return t.Disk.Removable || t.Disk.USB }

// Mounts lists the filesystems that have to be unmounted before writing.
// Writing a whole drive destroys every partition on it, so all of them count.
func (t Target) Mounts() []Mount {
	if t.Whole() {
		return t.Disk.Mounts
	}
	return t.Partition.Mounts
}

// Problem explains why writing to this target is refused, or returns "" when
// it is allowed. A mounted filesystem is not a problem in itself: it can be
// unmounted first. A filesystem the running system depends on cannot.
func (t Target) Problem() string {
	if t.Whole() {
		if c := t.Disk.CriticalMounts(); len(c) > 0 {
			return "it holds the running system (" + strings.Join(c, ", ") + ")"
		}
	} else {
		var c []string
		for _, m := range t.Partition.Mounts {
			if criticalMounts[m.Target] {
				c = append(c, m.Target)
			}
		}
		if len(c) > 0 {
			sort.Strings(c)
			return "it holds the running system (" + strings.Join(c, ", ") + ")"
		}
	}

	parts := t.Disk.Partitions
	if !t.Whole() {
		parts = []*Partition{t.Partition}
	}
	var holders []string
	swap := false
	if t.Whole() {
		holders = append(holders, t.Disk.Holders...)
		swap = t.Disk.Swap
	}
	for _, p := range parts {
		holders = append(holders, p.Holders...)
		swap = swap || p.Swap
	}
	switch {
	case len(holders) > 0:
		return "it is in use by " + strings.Join(holders, ", ") +
			" (an opened encrypted volume, LVM or RAID); close that first"
	case swap:
		return "it is in use as swap space"
	case (t.Disk.ReadOnly || (!t.Whole() && t.Partition.ReadOnly)) && t.External():
		return "it is write-protected (check the lock switch on an SD card)"
	case t.Disk.ReadOnly || (!t.Whole() && t.Partition.ReadOnly):
		return "it is read-only"
	case t.Size == 0:
		return "it has no media in it"
	}
	return ""
}

// Name is the short label for the target, such as "SanDisk Ultra" or
// "partition 1 of SanDisk Ultra".
func (t Target) Name() string {
	disk := t.Disk.Model
	if disk == "" {
		disk = t.Disk.Name
	}
	if t.Whole() {
		return disk
	}
	return fmt.Sprintf("partition %d of %s", t.Partition.Number, disk)
}

// Details describes what is there now, for a person choosing where to write.
func (t Target) Details() string {
	var parts []string
	switch {
	case t.Disk.USB:
		parts = append(parts, "USB drive")
	case t.Disk.Removable:
		parts = append(parts, "removable drive")
	default:
		parts = append(parts, "internal disk")
	}

	label, fstype := t.Disk.Label, t.Disk.FSType
	if !t.Whole() {
		label, fstype = t.Partition.Label, t.Partition.FSType
	}
	if fstype != "" {
		fs := fstype
		if label != "" {
			fs += " " + strconv.Quote(label)
		}
		parts = append(parts, fs)
	}
	if t.Whole() && len(t.Disk.Partitions) > 0 {
		n := len(t.Disk.Partitions)
		parts = append(parts, fmt.Sprintf("%d partition%s", n, plural(n)))
	}
	for _, m := range t.Mounts() {
		parts = append(parts, "mounted at "+m.Target)
	}
	return strings.Join(parts, ", ")
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// Targets lists every whole drive and partition. Drives the user plugged in
// come first, since they are nearly always what an image is meant for.
// Readers with no card in them are left out.
func (s *Scanner) Targets() ([]Target, error) {
	devices, err := s.List()
	if err != nil {
		return nil, err
	}
	var out []Target
	for _, d := range devices {
		if d.Size == 0 {
			continue
		}
		out = append(out, Target{Path: d.Path, Size: d.Size, Disk: d, Writable: d.Writable})
		for _, p := range d.Partitions {
			out = append(out, Target{Path: p.Path, Size: p.Size, Disk: d, Partition: p, Writable: p.Writable})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].External() && !out[j].External()
	})
	return out, nil
}

// FindTarget returns the drive or partition named by a path such as /dev/sdb1
// or a bare name such as sdb1.
func (s *Scanner) FindTarget(query string) (Target, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return Target{}, fmt.Errorf("no drive or partition given")
	}
	// Stable names such as /dev/disk/by-id/usb-... are links to the node.
	if filepath.IsAbs(q) {
		if real, err := filepath.EvalSymlinks(q); err == nil {
			q = real
		}
	}
	q = strings.TrimPrefix(q, s.Dev+"/")
	q = strings.TrimPrefix(q, "/dev/")

	targets, err := s.Targets()
	if err != nil {
		return Target{}, err
	}
	for _, t := range targets {
		if filepath.Base(t.Path) == q {
			return t, nil
		}
	}
	return Target{}, fmt.Errorf("no drive or partition called %q (is it plugged in?)", query)
}

// MountFor returns the mounted filesystem a file lives on.
func (s *Scanner) MountFor(path string) (Mount, bool) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Mount{}, false
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		abs = real
	}
	var best Mount
	found := false
	for _, m := range s.readMounts() {
		inside := m.Target == "/" || abs == m.Target || strings.HasPrefix(abs, m.Target+"/")
		if inside && (!found || len(m.Target) > len(best.Target)) {
			best, found = m, true
		}
	}
	return best, found
}

// Holds reports whether a device path is the target itself or something the
// target contains, so writing the target would destroy it.
func (t Target) Holds(source string) bool {
	if source == t.Path {
		return true
	}
	if t.Whole() {
		for _, p := range t.Disk.Partitions {
			if p.Path == source {
				return true
			}
		}
		for _, m := range t.Disk.Mounts {
			if m.Source == source {
				return true
			}
		}
	}
	return false
}
