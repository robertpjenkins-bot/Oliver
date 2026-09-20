// Package device inspects the machine's block devices so a Windows program can
// be given access to one deliberately, rather than by running everything as
// root and hoping.
package device

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// Mount is a filesystem mounted from a device or one of its partitions.
type Mount struct {
	Source string `json:"source"`
	Target string `json:"target"`
	// ReadOnly reports whether this mount refuses writes, which is one of the
	// reasons a drive appears write-protected when the drive is nothing of
	// the kind.
	ReadOnly bool `json:"read_only,omitempty"`
}

// Partition is one slice of a device, such as /dev/sda1.
type Partition struct {
	Path   string  `json:"path"`
	Name   string  `json:"name"`
	Number int     `json:"number"`
	Size   int64   `json:"size_bytes"`
	Mounts []Mount `json:"mounts,omitempty"`
}

// Device is a whole block device, such as /dev/sda.
type Device struct {
	Path       string      `json:"path"`
	Name       string      `json:"name"`
	Model      string      `json:"model,omitempty"`
	Size       int64       `json:"size_bytes"`
	Removable  bool        `json:"removable"`
	Mounts     []Mount     `json:"mounts,omitempty"`
	Partitions []Partition `json:"partitions,omitempty"`

	// ReadOnly is the kernel's own read-only flag for this device.
	ReadOnly bool `json:"read_only,omitempty"`

	// LogicalBlockSize is the sector size the kernel addresses this device in,
	// almost always 512. A drive that reports anything else needs a filesystem
	// laid out for that size.
	LogicalBlockSize int `json:"logical_block_size,omitempty"`

	// Readable and Writable report the calling user's access to the raw
	// device. Without write access a program cannot repartition or image it,
	// whatever else is configured.
	Readable bool `json:"readable"`
	Writable bool `json:"writable"`
}

// Scanner reads block devices. Its roots are fields so tests can describe a
// machine without touching the real one.
type Scanner struct {
	SysBlock string // /sys/block
	Dev      string // /dev
	Mounts   string // /proc/mounts
	// Access reports the caller's permission on a device node.
	Access func(path string) (readable, writable bool)
}

// NewScanner returns a scanner for the running system.
func NewScanner() *Scanner {
	return &Scanner{
		SysBlock: "/sys/block",
		Dev:      "/dev",
		Mounts:   "/proc/mounts",
		Access:   accessOf,
	}
}

func accessOf(path string) (bool, bool) {
	return syscall.Access(path, 4) == nil, syscall.Access(path, 2) == nil
}

// criticalMounts are the mount points that, if served by a device, mean
// handing that device to a disk utility risks destroying the running system.
var criticalMounts = map[string]bool{
	"/": true, "/boot": true, "/boot/efi": true, "/efi": true,
	"/home": true, "/var": true, "/usr": true, "/etc": true,
	"/nix": true, "/sysroot": true,
}

// SystemCritical reports whether this device hosts the running system.
func (d *Device) SystemCritical() bool {
	for _, m := range d.Mounts {
		if criticalMounts[m.Target] {
			return true
		}
	}
	return false
}

// CriticalMounts lists the system mount points this device serves.
func (d *Device) CriticalMounts() []string {
	var out []string
	for _, m := range d.Mounts {
		if criticalMounts[m.Target] {
			out = append(out, m.Target)
		}
	}
	sort.Strings(out)
	return out
}

// Describe renders the device for a confirmation prompt.
func (d *Device) Describe() string {
	parts := []string{d.Path, humanSize(d.Size)}
	if d.Model != "" {
		parts = append(parts, d.Model)
	}
	if d.Removable {
		parts = append(parts, "removable")
	} else {
		parts = append(parts, "fixed")
	}
	return strings.Join(parts, "  ")
}

func humanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for n/div >= unit && exp < 4 {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTP"[exp])
}

// List returns every whole block device, excluding loop, ram and zram nodes
// that are never what a user means by "a drive".
func (s *Scanner) List() ([]*Device, error) {
	entries, err := os.ReadDir(s.SysBlock)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", s.SysBlock, err)
	}
	mounts := s.readMounts()

	var out []*Device
	for _, e := range entries {
		name := e.Name()
		if skipDevice(name) {
			continue
		}
		d := s.describe(name, mounts)
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func skipDevice(name string) bool {
	for _, prefix := range []string{"loop", "ram", "zram", "dm-", "md", "sr"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func (s *Scanner) describe(name string, mounts []Mount) *Device {
	d := &Device{
		Path: filepath.Join(s.Dev, name),
		Name: name,
	}
	base := filepath.Join(s.SysBlock, name)

	// /sys reports size in 512-byte sectors regardless of the real block size.
	if raw, err := os.ReadFile(filepath.Join(base, "size")); err == nil {
		if sectors, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64); err == nil {
			d.Size = sectors * 512
		}
	}
	if raw, err := os.ReadFile(filepath.Join(base, "removable")); err == nil {
		d.Removable = strings.TrimSpace(string(raw)) == "1"
	}
	if raw, err := os.ReadFile(filepath.Join(base, "device", "model")); err == nil {
		d.Model = strings.TrimSpace(string(raw))
	}
	d.ReadOnly = s.readOnlyFlag(name)
	if raw, err := os.ReadFile(filepath.Join(base, "queue", "logical_block_size")); err == nil {
		if size, err := strconv.Atoi(strings.TrimSpace(string(raw))); err == nil {
			d.LogicalBlockSize = size
		}
	}

	// A mount of any partition counts against the whole device: writing to
	// /dev/sda destroys /dev/sda1 along with it.
	for _, m := range mounts {
		if m.Source == d.Path || strings.HasPrefix(m.Source, d.Path) {
			d.Mounts = append(d.Mounts, m)
		}
	}
	d.Partitions = s.partitions(base, d, mounts)

	if s.Access != nil {
		d.Readable, d.Writable = s.Access(d.Path)
	}
	return d
}

// partitions lists the device's partitions, as /sys records them: a
// subdirectory of the device holding a "partition" file with its number.
func (s *Scanner) partitions(base string, d *Device, mounts []Mount) []Partition {
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil
	}
	var out []Partition
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), d.Name) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(base, e.Name(), "partition"))
		if err != nil {
			continue
		}
		number, err := strconv.Atoi(strings.TrimSpace(string(raw)))
		if err != nil {
			continue
		}
		p := Partition{
			Path:   filepath.Join(s.Dev, e.Name()),
			Name:   e.Name(),
			Number: number,
		}
		if raw, err := os.ReadFile(filepath.Join(base, e.Name(), "size")); err == nil {
			if sectors, err := strconv.ParseInt(strings.TrimSpace(string(raw)), 10, 64); err == nil {
				p.Size = sectors * 512
			}
		}
		for _, m := range mounts {
			if m.Source == p.Path {
				p.Mounts = append(p.Mounts, m)
			}
		}
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}

func (s *Scanner) readMounts() []Mount {
	f, err := os.Open(s.Mounts)
	if err != nil {
		return nil
	}
	defer f.Close()

	var out []Mount
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		fields := strings.Fields(scan.Text())
		// Only real block devices matter; the table is mostly virtual
		// filesystems. Compare against the configured device root rather than
		// a hardcoded /dev so the scanner stays relocatable.
		if len(fields) < 2 || !strings.HasPrefix(fields[0], s.Dev+"/") {
			continue
		}
		m := Mount{Source: fields[0], Target: unescapeMount(fields[1])}
		// Field four is the mount options; "ro" among them is why a drive
		// that is perfectly writable will not take a write.
		if len(fields) > 3 {
			for _, opt := range strings.Split(fields[3], ",") {
				if opt == "ro" {
					m.ReadOnly = true
					break
				}
			}
		}
		out = append(out, m)
	}
	return out
}

// unescapeMount decodes the octal escapes /proc/mounts uses for spaces and
// other awkward characters in mount points.
func unescapeMount(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+3 < len(s) {
			if n, err := strconv.ParseInt(s[i+1:i+4], 8, 16); err == nil {
				b.WriteByte(byte(n))
				i += 3
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// Find returns the device for a path such as /dev/sda or a bare name.
// A partition resolves to its parent device, since Windows disk utilities
// operate on whole drives.
func (s *Scanner) Find(query string) (*Device, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, fmt.Errorf("no device given")
	}
	q = strings.TrimPrefix(q, s.Dev+"/")
	q = strings.TrimPrefix(q, "/dev/")

	devices, err := s.List()
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		if d.Name == q {
			return d, nil
		}
	}
	// Maybe a partition: sda1 belongs to sda.
	for _, d := range devices {
		if strings.HasPrefix(q, d.Name) && len(q) > len(d.Name) {
			return d, fmt.Errorf("%s is a partition; use the whole device %s", query, d.Path)
		}
	}
	return nil, fmt.Errorf("no such block device %q", query)
}

// AccessAdvice explains how to obtain read/write access to a device without
// running the whole program as root.
func AccessAdvice(d *Device) string {
	if d.Writable {
		return ""
	}
	group := "disk"
	return fmt.Sprintf(
		"You do not have permission to write to %s. Writing to a drive needs it, and nothing "+
			"Oliver does can grant it. Either run this one command with sudo, or add yourself to "+
			"the %q group (`sudo usermod -aG %s $USER`, then log out and back in) to stop having to.",
		d.Path, group, group)
}

// Busy reports whether anything on this device is mounted. Writing to a drive
// the system is reading from produces a corrupt drive and, often, a corrupt
// cache of the old filesystem in the bargain.
func (d *Device) Busy() bool { return len(d.Mounts) > 0 }

// unmountCommands are tried in order. udisksctl is first because on a desktop
// it unmounts a user's drive without asking for a password, which plain umount
// will not do.
var unmountCommands = [][]string{
	{"udisksctl", "unmount", "-b"},
	{"umount"},
}

// runCommand is a variable so tests can watch what would have been run.
var runCommand = func(name string, args ...string) ([]byte, error) {
	return exec.Command(name, args...).CombinedOutput()
}

// Unmount releases every filesystem this device has mounted, so the drive can
// be written to safely. It reports what it could not release.
func Unmount(d *Device) error {
	var failed []string
	for _, m := range d.Mounts {
		if err := unmountOne(m.Source); err != nil {
			failed = append(failed, fmt.Sprintf("%s (%s): %v", m.Source, m.Target, err))
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("could not unmount %s", strings.Join(failed, "; "))
	}
	return nil
}

func unmountOne(source string) error {
	var last error
	for _, c := range unmountCommands {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		out, err := runCommand(c[0], append(c[1:], source)...)
		if err == nil {
			return nil
		}
		last = fmt.Errorf("%s: %s", c[0], strings.TrimSpace(string(out)))
	}
	if last == nil {
		return fmt.Errorf("no unmount command is available")
	}
	return last
}
