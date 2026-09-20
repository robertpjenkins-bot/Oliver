package device

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Nothing in this program can write to a drive without the privilege to do it,
// and telling somebody that is not the same as helping them. Most of the
// reports that Oliver "cannot fix" a drive are this: an ordinary account, a
// device node it may not open, and a message that reads like a failure of the
// program rather than a missing sudo.

// Elevated reports whether this process can open device nodes for writing. It
// is a variable so a test can describe an ordinary account while running as
// whatever the machine running the tests happens to be.
var Elevated = func() bool { return os.Geteuid() == 0 }

// Self returns the path to run to start this program again.
//
// Inside an AppImage the running binary is an extracted copy in a temporary
// mount, which is not what a person or a password prompt should be pointed at.
// The AppImage runtime puts the real path in the environment for exactly this.
func Self() (string, error) {
	if appimage := os.Getenv("APPIMAGE"); appimage != "" {
		return appimage, nil
	}
	return os.Executable()
}

// Elevator is a way of running this program as root.
type Elevator struct {
	// Command is the program that asks for the privilege.
	Command string
	// Graphical reports whether it asks in a window rather than a terminal,
	// which is what matters when there is no terminal to ask in.
	Graphical bool
}

// FindElevator returns how this machine can grant privilege, preferring the
// kind that suits where the request is coming from.
func FindElevator(interactive bool) (Elevator, bool) {
	candidates := []Elevator{
		{Command: "sudo"},
		{Command: "doas"},
		{Command: "pkexec", Graphical: true},
	}
	if !interactive {
		// With no terminal, a password prompt that needs one is no use.
		candidates = []Elevator{{Command: "pkexec", Graphical: true}}
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c.Command); err == nil {
			return c, true
		}
	}
	return Elevator{}, false
}

// ElevateCommand returns the command line that would run this program as root
// with the given arguments, for showing to somebody.
func ElevateCommand(e Elevator, args []string) string {
	self, err := Self()
	if err != nil {
		self = "oliver"
	}
	return strings.Join(append([]string{e.Command, self}, args...), " ")
}

// Elevate replaces this process with the same command running as root.
//
// It does not return on success: the new process takes over, keeping the
// terminal, so a password prompt and everything after it happen where the user
// is already looking.
func Elevate(e Elevator, args []string) error {
	self, err := Self()
	if err != nil {
		return fmt.Errorf("could not work out how to run this program again: %w", err)
	}
	path, err := exec.LookPath(e.Command)
	if err != nil {
		return err
	}

	argv := append([]string{e.Command}, self)
	argv = append(argv, args...)
	// An AppImage that had to unpack itself must keep doing so as root, or the
	// second run fails differently from the first.
	env := os.Environ()
	if os.Getenv("APPIMAGE_EXTRACT_AND_RUN") != "" && e.Command != "pkexec" {
		argv = append([]string{e.Command, "APPIMAGE_EXTRACT_AND_RUN=1", self}, args...)
	}
	return execProcess(path, argv, env)
}
