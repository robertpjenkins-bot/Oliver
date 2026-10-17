//go:build linux

package main

import (
	"os"
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"
)

// terminals that can run a program, with how each is told what to run.
var terminals = [][]string{
	{"x-terminal-emulator", "-e"},
	{"gnome-terminal", "--"},
	{"kgx", "--"},
	{"ptyxis", "--"},
	{"konsole", "-e"},
	{"xfce4-terminal", "-x"},
	{"mate-terminal", "-x"},
	{"lxterminal", "-e"},
	{"qterminal", "-e"},
	{"tilix", "-e"},
	{"terminator", "-x"},
	{"kitty"},
	{"alacritty", "-e"},
	{"wezterm", "start", "--"},
	{"foot"},
	{"xterm", "-e"},
}

// openInTerminal handles being double-clicked in a file manager: there is
// no terminal to talk to, so open one and run the installer in it. It
// reports whether one was opened (and this copy should stop).
func openInTerminal() bool {
	if isTerminal(os.Stdin) || (os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "") {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	args := append([]string{exe}, os.Args[1:]...)
	for _, t := range terminals {
		path, err := exec.LookPath(t[0])
		if err != nil {
			continue
		}
		cmd := exec.Command(path, append(t[1:], args...)...)
		// The window closes when the installer ends; make it wait first.
		cmd.Env = append(os.Environ(), "ETROS_PAUSE=1")
		if err := cmd.Start(); err == nil {
			logf("opened a terminal: %s", strings.Join(cmd.Args, " "))
			return true
		}
	}
	logf("double-clicked, but no terminal program was found")
	msg := "Etros Installer needs a terminal. Open one and run:\n" + exe
	for _, c := range [][]string{{"zenity", "--error", "--text=" + msg}, {"kdialog", "--error", msg}, {"notify-send", "Etros Installer", msg}} {
		if p, err := exec.LookPath(c[0]); err == nil && exec.Command(p, c[1:]...).Run() == nil {
			break
		}
	}
	return true
}

func isTerminal(f *os.File) bool {
	_, err := unix.IoctlGetTermios(int(f.Fd()), unix.TCGETS)
	return err == nil
}
