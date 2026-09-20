package device

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// "Target is busy" is the other message that stops people, and it never says
// what is busy. The kernel knows: every process holding the drive open has a
// link to it in /proc. Reading that needs nothing installed and no privilege
// beyond seeing one's own processes.

// Holder is a process with the drive open.
type Holder struct {
	PID     int    `json:"pid"`
	Command string `json:"command"`
	// Path is which of the drive's nodes it has open.
	Path string `json:"path"`
}

// Holders lists the processes holding this drive or any of its partitions
// open. Processes belonging to other users are included when they can be seen,
// and silently missed when they cannot, so an empty result is not a promise.
func Holders(d *Device) []Holder {
	paths := map[string]bool{d.Path: true}
	for _, p := range d.Partitions {
		paths[p.Path] = true
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}
	var out []Holder
	seen := map[string]bool{}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // not a process
		}
		fds, err := os.ReadDir(filepath.Join("/proc", e.Name(), "fd"))
		if err != nil {
			continue // gone, or another user's
		}
		for _, fd := range fds {
			target, err := os.Readlink(filepath.Join("/proc", e.Name(), "fd", fd.Name()))
			if err != nil || !paths[target] {
				continue
			}
			key := fmt.Sprintf("%d:%s", pid, target)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, Holder{PID: pid, Command: commandOf(pid), Path: target})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PID < out[j].PID })
	return out
}

func commandOf(pid int) string {
	raw, err := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "comm"))
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(raw))
}

// Describe renders a holder for a report.
func (h Holder) Describe() string {
	return fmt.Sprintf("%s (pid %d) has %s open", h.Command, h.PID, h.Path)
}

// ForceUnmount releases a mount that an ordinary unmount will not.
//
// The lazy unmount is the one that works when something still has the
// filesystem open: it detaches it now and cleans up when the last user lets
// go, which is what lets a drive be written to while a stale process holds a
// handle it will never use again.
func ForceUnmount(source string) error {
	attempts := [][]string{
		{"udisksctl", "unmount", "-f", "-b", source},
		{"umount", "-l", source},
		{"umount", "-f", source},
	}
	var last error
	for _, attempt := range attempts {
		if !haveTool(attempt[0]) {
			continue
		}
		out, err := runCommand(attempt[0], attempt[1:]...)
		if err == nil {
			return nil
		}
		last = fmt.Errorf("%s: %s", attempt[0], strings.TrimSpace(string(out)))
	}
	if last == nil {
		return fmt.Errorf("no unmount command is available")
	}
	return last
}
