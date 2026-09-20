package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// A drive that will not accept writes is refusing for one of a handful of
// reasons, and they are not equally fixable. Two are a flag somebody can
// clear. One is a switch on the side of the stick. One is the flash controller
// having decided the drive is worn out and put itself beyond writing for good.
//
// Saying which of those it is matters more than trying to force it: the first
// two take a second to put right, and no amount of software fixes the last.

// ProtectionKind says why a drive will not take writes.
type ProtectionKind string

const (
	// KernelFlag is the read-only flag the kernel holds for a block device.
	// It is what `blockdev --setro` sets, what some udev rules set, and what
	// gets left behind after a device has been through an error. It clears.
	KernelFlag ProtectionKind = "kernel-flag"
	// ReadOnlyMount is a filesystem from the drive mounted read-only. The
	// drive itself is writable; what is in the way is the mount.
	ReadOnlyMount ProtectionKind = "read-only-mount"
	// ReadOnlyFilesystem is a filesystem that cannot be written to by its
	// nature — what an installation image leaves behind.
	ReadOnlyFilesystem ProtectionKind = "read-only-filesystem"
	// Hardware is the drive itself refusing: a physical switch, or a
	// controller that has locked the flash. Nothing here can change it.
	Hardware ProtectionKind = "hardware"
	// GPTAttribute is the read-only bit a GPT entry can carry, which every
	// operating system obeys and almost nothing reports.
	GPTAttribute ProtectionKind = "gpt-attribute"
	// DirtyFilesystem is a filesystem that marked itself unsafe to write to
	// after an unclean unmount.
	DirtyFilesystem ProtectionKind = "dirty-filesystem"
	// FilesystemErrors is a filesystem the kernel stopped writing to because
	// it found something wrong with it.
	FilesystemErrors ProtectionKind = "filesystem-errors"
)

// Protection is one reason a drive will not take writes.
type Protection struct {
	Kind      ProtectionKind `json:"kind"`
	Summary   string         `json:"summary"`
	Detail    string         `json:"detail"`
	Clearable bool           `json:"clearable"`
	// Where is the device or mount point the reason applies to.
	Where string `json:"where,omitempty"`
}

// WriteProtection reports every reason this drive will not take writes, most
// easily fixed first. An empty result means nothing is standing in the way.
//
// It looks in three places, because the causes live in three places and a tool
// that checks only one leaves people trying things at random: the kernel's own
// flags, the mounts in front of the drive, and the drive itself.
func (s *Scanner) WriteProtection(d *Device) []Protection {
	var out []Protection

	if s.readOnlyFlag(d.Name) {
		out = append(out, Protection{
			Kind:    KernelFlag,
			Summary: "the kernel is holding this device read-only",
			Detail: "A flag on the device itself is refusing writes before they reach the drive. " +
				"It is set by `blockdev --setro`, by some udev rules, and by the kernel after a " +
				"device has thrown errors. Clearing it takes a moment and changes nothing on the drive.",
			Clearable: true,
			Where:     d.Path,
		})
	}
	for _, p := range d.Partitions {
		if s.readOnlyFlag(filepath.Join(d.Name, p.Name)) {
			out = append(out, Protection{
				Kind:      KernelFlag,
				Summary:   fmt.Sprintf("the kernel is holding %s read-only", p.Name),
				Detail:    "The same flag, on one partition rather than the whole drive.",
				Clearable: true,
				Where:     p.Path,
			})
		}
	}

	for _, m := range d.Mounts {
		if !m.ReadOnly {
			continue
		}
		out = append(out, Protection{
			Kind:    ReadOnlyMount,
			Summary: fmt.Sprintf("%s is mounted read-only at %s", m.Source, m.Target),
			Detail: "The drive will take writes; this mount will not pass them on. Unmounting it " +
				"is enough, and is what has to happen before writing to the whole drive anyway.",
			Clearable: true,
			Where:     m.Source,
		})
	}

	return append(out, s.OnDiskProtection(d)...)
}

// Clearable reports how many of these Oliver can do something about.
func Clearable(protections []Protection) int {
	n := 0
	for _, p := range protections {
		if p.Clearable {
			n++
		}
	}
	return n
}

// readOnlyFlag reads the ro flag /sys keeps for a device or partition.
func (s *Scanner) readOnlyFlag(name string) bool {
	raw, err := os.ReadFile(filepath.Join(s.SysBlock, name, "ro"))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(raw)) == "1"
}

// HardwareProtection is what is left when every flag has been cleared and the
// drive still will not take a write.
func HardwareProtection(d *Device) Protection {
	return Protection{
		Kind:    Hardware,
		Summary: "the drive itself is refusing to be written to",
		Detail: "Every flag this program can reach is clear, so the refusal is coming from the " +
			"drive. Either it has a physical lock switch — check the side of the stick, and the " +
			"adapter if it is an SD card — or its controller has put the flash beyond writing, " +
			"which is what worn-out flash does to protect what is still readable. If it is the " +
			"second, the drive is finished: copy anything you need off it while it still reads.",
		Clearable: false,
		Where:     d.Path,
	}
}

// openHint turns the kernel's refusal to open a device into the reason for it.
func openHint(path string, err error) error {
	if os.IsPermission(err) {
		return fmt.Errorf("permission denied: writing to %s needs root", path)
	}
	return err
}

// ClearReadOnly clears the kernel's read-only flag on a device. It is a
// variable so that tests can describe a machine that has no real block device
// on it, since the ioctl behind it only answers for one.
var ClearReadOnly = clearReadOnly

// ClearGPTReadOnly and ClearDirtyFlag are variables for the same reason: the
// drives a test describes are files, and these write to real ones.
var (
	ClearGPTReadOnly = clearGPTReadOnly
	ClearDirtyFlag   = clearDirtyFlag
)

// Unprotect clears everything standing between the drive and a write that can
// be cleared, and reports what it did. It changes no file on the drive: every
// step here rewrites a flag, not data.
//
// Clearing a dirty filesystem's mark is the one step with a cost worth
// knowing: it tells the system the volume is clean without checking that it
// is. Callers gate it behind their own consent, which is what force is for.
func Unprotect(d *Device, protections []Protection, force bool) ([]string, error) {
	var done []string
	var failed []string

	fail := func(where string, err error) {
		failed = append(failed, fmt.Sprintf("%s: %v", where, err))
	}

	for _, p := range protections {
		switch p.Kind {
		case KernelFlag:
			if err := ClearReadOnly(p.Where); err != nil {
				fail(p.Where, err)
				continue
			}
			done = append(done, "cleared the kernel's read-only flag on "+p.Where)

		case ReadOnlyMount:
			if err := unmountOne(p.Where); err != nil {
				fail(p.Where, err)
				continue
			}
			done = append(done, "unmounted "+p.Where)

		case GPTAttribute:
			if err := ClearGPTReadOnly(p.Where); err != nil {
				fail(p.Where, err)
				continue
			}
			done = append(done, "cleared the read-only attribute in the partition table of "+p.Where)

		case DirtyFilesystem:
			if !p.Clearable {
				continue // NTFS: the flag is not ours to clear
			}
			if !force {
				continue // marking a volume clean without checking it is the caller's call
			}
			if err := ClearDirtyFlag(p.Where); err != nil {
				fail(p.Where, err)
				continue
			}
			done = append(done, "marked the filesystem on "+p.Where+" clean")
		}
	}
	if len(failed) > 0 {
		return done, fmt.Errorf("could not clear write protection: %s", strings.Join(failed, "; "))
	}
	return done, nil
}

// MakeWritable does whatever it takes to get a drive ready to be written to,
// and reports each thing it did.
//
// It exists because knowing why a drive is read-only and then failing on it is
// no use to anybody. By the time this is called the caller has already agreed
// to destroy what is on the drive, so clearing a flag that merely says "do not
// write" is the smaller act by a long way — and it is the difference between
// a tool that explains the problem and one that solves it.
//
// A dirty filesystem is not touched: it stops a volume being mounted for
// writing, not the drive being written to, and it is about to be overwritten
// anyway.
func MakeWritable(s *Scanner, d *Device) ([]string, error) {
	var steps []string

	blockers := []Protection{}
	for _, p := range s.WriteProtection(d) {
		if p.Kind == KernelFlag || p.Kind == GPTAttribute {
			blockers = append(blockers, p)
		}
	}
	if len(blockers) > 0 {
		done, err := Unprotect(d, blockers, false)
		steps = append(steps, done...)
		if err != nil {
			return steps, err
		}
	}

	if err := TestWrite(d.Path); err == nil {
		return steps, nil
	}

	// Still refusing. A stick whose controller latched read-only after an
	// error stays that way until it is power-cycled, and the kernel can do
	// that without anybody touching the drive.
	if _, isUSB := USBPath(d); !isUSB {
		return steps, TestWrite(d.Path)
	}
	if err := ResetUSB(d); err != nil {
		return steps, fmt.Errorf("the drive is still refusing writes, and resetting it failed: %w", err)
	}
	steps = append(steps, "reset the USB device, which had latched itself read-only")

	// It disappears and comes back, possibly under another name.
	fresh, err := waitForDevice(s, d.Path)
	if err != nil {
		return steps, err
	}
	if blockers := s.WriteProtection(fresh); len(blockers) > 0 {
		done, _ := Unprotect(fresh, blockers, false)
		steps = append(steps, done...)
	}
	return steps, TestWrite(fresh.Path)
}

// waitForDevice waits for a drive to come back after a reset.
func waitForDevice(s *Scanner, path string) (*Device, error) {
	var last error
	for attempt := 0; attempt < 20; attempt++ {
		time.Sleep(250 * time.Millisecond)
		d, err := s.Find(path)
		if err != nil {
			last = err
			continue
		}
		return d, nil
	}
	return nil, fmt.Errorf("%s did not come back after being reset: %w", path, last)
}
