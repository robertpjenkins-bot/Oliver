package device

import (
	"errors"
	"fmt"
	"strings"
)

// Reason says why a drive must not be written to. It is a value rather than a
// message so that a caller can phrase the way past a refusal in its own terms:
// the command line names a flag, a graphical front end offers a checkbox, and
// neither should have to match on the text of an error.
type Reason int

const (
	// SystemDisk is the drive the running system lives on. It is the one
	// refusal with no way past it.
	SystemDisk Reason = iota
	// FixedDisk is a drive that is not removable media. A caller may permit
	// it deliberately.
	FixedDisk
	// Mounted is a drive the system is still reading from.
	Mounted
	// NotWritable is a drive this user has no permission to write to.
	NotWritable
)

// Overridable reports whether a caller may proceed anyway, given permission.
func (r Reason) Overridable() bool { return r == FixedDisk || r == Mounted }

// Refusal explains why an operation on a drive was not allowed.
type Refusal struct {
	Reason Reason
	Device *Device
	Action string // "write to", "erase", "format"
	// Detail carries the specifics: the system mount points served, or the
	// filesystems currently mounted.
	Detail string
}

func (r *Refusal) Error() string {
	switch r.Reason {
	case SystemDisk:
		return fmt.Sprintf("refusing to %s %s: it holds the running system (%s)",
			r.Action, r.Device.Path, r.Detail)
	case FixedDisk:
		return fmt.Sprintf("%s is a fixed disk, not removable media (%s)",
			r.Device.Path, r.Device.Describe())
	case Mounted:
		return fmt.Sprintf("%s is in use (%s). Writing to a drive the system is reading from produces a corrupt drive",
			r.Device.Path, r.Detail)
	case NotWritable:
		return fmt.Sprintf("cannot write to %s. %s", r.Device.Path, AccessAdvice(r.Device))
	}
	return fmt.Sprintf("%s cannot be used", r.Device.Path)
}

// CheckWritable reports why a drive must not be written to, ignoring whether
// anything is currently mounted from it — that is CheckIdle's question, asked
// separately because a caller may want to unmount and continue.
//
// allowFixed permits a drive that is not removable media. Nothing permits the
// drive the running system lives on.
func CheckWritable(d *Device, action string, allowFixed bool) error {
	if d.SystemCritical() {
		return &Refusal{Reason: SystemDisk, Device: d, Action: action,
			Detail: strings.Join(d.CriticalMounts(), ", ")}
	}
	if !d.Removable && !allowFixed {
		return &Refusal{Reason: FixedDisk, Device: d, Action: action}
	}
	if !d.Writable {
		return &Refusal{Reason: NotWritable, Device: d, Action: action}
	}
	return nil
}

// CheckIdle reports whether the system is still using the drive.
func CheckIdle(d *Device, action string) error {
	if !d.Busy() {
		return nil
	}
	var where []string
	for _, m := range d.Mounts {
		where = append(where, m.Source+" at "+m.Target)
	}
	return &Refusal{Reason: Mounted, Device: d, Action: action,
		Detail: strings.Join(where, ", ")}
}

// RefusedFor returns the reason behind an error, and whether it was a refusal
// at all.
func RefusedFor(err error) (Reason, bool) {
	var r *Refusal
	if errors.As(err, &r) {
		return r.Reason, true
	}
	return 0, false
}
