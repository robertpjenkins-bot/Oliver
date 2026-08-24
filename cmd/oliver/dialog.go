package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// This file provides the small set of desktop dialogs Oliver needs when it is
// opened without a terminal. Both zenity (GNOME) and kdialog (KDE) are
// supported, since a user has whichever their desktop shipped.

type dialogKit struct {
	bin  string
	kind string // "zenity" or "kdialog"
}

// findDialogKit returns the dialog program available on this desktop.
func findDialogKit() (dialogKit, bool) {
	for _, k := range []dialogKit{{kind: "kdialog"}, {kind: "zenity"}} {
		if p, err := exec.LookPath(k.kind); err == nil {
			k.bin = p
			return k, true
		}
	}
	return dialogKit{}, false
}

// output runs a dialog and returns what the user chose, and whether they
// confirmed rather than cancelled.
func (d dialogKit) output(args ...string) (string, bool) {
	cmd := exec.Command(d.bin, args...)
	cmd.Stderr = nil
	out, err := cmd.Output()
	if err != nil {
		return "", false // a non-zero status means cancelled
	}
	return strings.TrimSpace(string(out)), true
}

// menuChoice is one entry in an action menu.
type menuChoice struct {
	ID    string
	Label string
	Note  string
}

// menu asks the user to pick an action, returning the chosen ID.
func (d dialogKit) menu(title, prompt string, choices []menuChoice) (string, bool) {
	switch d.kind {
	case "zenity":
		args := []string{"--list", "--title", title, "--text", prompt,
			"--column", "id", "--column", "Action", "--column", "Details",
			"--hide-column", "1", "--print-column", "1",
			"--width", "620", "--height", "380"}
		for _, c := range choices {
			args = append(args, c.ID, c.Label, c.Note)
		}
		return d.output(args...)
	default: // kdialog
		args := []string{"--title", title, "--menu", prompt}
		for _, c := range choices {
			args = append(args, c.ID, c.Label)
		}
		return d.output(args...)
	}
}

// openFile asks for a file. filter is a human label plus patterns.
func (d dialogKit) openFile(title, label string, patterns []string) (string, bool) {
	switch d.kind {
	case "zenity":
		return d.output("--file-selection", "--title", title,
			"--file-filter", label+" | "+strings.Join(patterns, " "),
			"--file-filter", "All files | *")
	default:
		start, _ := os.UserHomeDir()
		return d.output("--title", title, "--getopenfilename", start, strings.Join(patterns, " "))
	}
}

// entry asks for a line of text.
func (d dialogKit) entry(title, prompt, initial string) (string, bool) {
	switch d.kind {
	case "zenity":
		return d.output("--entry", "--title", title, "--text", prompt, "--entry-text", initial)
	default:
		return d.output("--title", title, "--inputbox", prompt, initial)
	}
}

// showText displays a scrollable block of text.
func (d dialogKit) showText(title, body string) {
	f, err := os.CreateTemp("", "oliver-*.txt")
	if err != nil {
		return
	}
	defer os.Remove(f.Name())
	_, _ = f.WriteString(body)
	_ = f.Close()

	switch d.kind {
	case "zenity":
		_ = exec.Command(d.bin, "--text-info", "--title", title,
			"--filename", f.Name(), "--width", "760", "--height", "520").Run()
	default:
		_ = exec.Command(d.bin, "--title", title, "--textbox", f.Name(), "760", "520").Run()
	}
}

// confirm asks a yes/no question.
func (d dialogKit) confirm(title, prompt string) bool {
	switch d.kind {
	case "zenity":
		return exec.Command(d.bin, "--question", "--title", title, "--text", prompt).Run() == nil
	default:
		return exec.Command(d.bin, "--title", title, "--yesno", prompt).Run() == nil
	}
}

// info shows a message.
func (d dialogKit) info(title, body string) {
	switch d.kind {
	case "zenity":
		_ = exec.Command(d.bin, "--info", "--title", title, "--text", body, "--width", "480").Run()
	default:
		_ = exec.Command(d.bin, "--title", title, "--msgbox", body).Run()
	}
}

// errorBox shows a failure.
func (d dialogKit) errorBox(title, body string) {
	switch d.kind {
	case "zenity":
		_ = exec.Command(d.bin, "--error", "--title", title, "--text", body, "--width", "560").Run()
	default:
		_ = exec.Command(d.bin, "--title", title, "--error", body).Run()
	}
}

// windowsProgramPatterns are the file types Oliver can launch.
var windowsProgramPatterns = []string{"*.exe", "*.msi", "*.bat", "*.cmd", "*.EXE", "*.MSI"}

func fmtErr(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
