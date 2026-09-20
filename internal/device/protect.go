package device

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
			Where:     m.Target,
		})
	}
	return out
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

// ClearReadOnly clears the kernel's read-only flag on a device. It is a
// variable so that tests can describe a machine that has no real block device
// on it, since the ioctl behind it only answers for one.
var ClearReadOnly = clearReadOnly

// Unprotect clears the flags that can be cleared, and reports what it did.
// It does not touch the contents of the drive.
func Unprotect(d *Device, protections []Protection) ([]string, error) {
	var done []string
	var failed []string

	for _, p := range protections {
		switch p.Kind {
		case KernelFlag:
			if err := ClearReadOnly(p.Where); err != nil {
				failed = append(failed, fmt.Sprintf("%s: %v", p.Where, err))
				continue
			}
			done = append(done, "cleared the kernel's read-only flag on "+p.Where)
		case ReadOnlyMount:
			// Unmounting is handled with the rest of the mounts, since a write
			// needs them gone whether they are read-only or not.
		}
	}
	if len(failed) > 0 {
		return done, fmt.Errorf("could not clear write protection: %s", strings.Join(failed, "; "))
	}
	return done, nil
}
