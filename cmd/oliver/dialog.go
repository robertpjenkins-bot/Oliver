package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
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

// pick asks the user to choose one row of a table. The first cell of each row
// is its ID: it is what is returned, and it is not shown.
func (d dialogKit) pick(title, prompt string, columns []string, rows [][]string) (string, bool) {
	switch d.kind {
	case "zenity":
		args := []string{"--list", "--title", title, "--text", escapeMarkup(prompt), "--column", "id"}
		for _, c := range columns {
			args = append(args, "--column", c)
		}
		args = append(args, "--hide-column", "1", "--print-column", "1", "--width", "760", "--height", "420")
		for _, r := range rows {
			args = append(args, r...)
		}
		return d.output(args...)
	default: // kdialog
		args := []string{"--title", title, "--menu", prompt}
		for _, r := range rows {
			args = append(args, r[0], strings.TrimSpace(strings.Join(r[1:], "   ")))
		}
		return d.output(args...)
	}
}

// escapeMarkup protects text from zenity, which reads Pango markup: a file
// called "Tom & Jerry.iso" would otherwise blank the label.
func escapeMarkup(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
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
		return exec.Command(d.bin, "--question", "--no-markup", "--title", title, "--text", prompt).Run() == nil
	default:
		return exec.Command(d.bin, "--title", title, "--yesno", prompt).Run() == nil
	}
}

// info shows a message.
func (d dialogKit) info(title, body string) {
	switch d.kind {
	case "zenity":
		_ = exec.Command(d.bin, "--info", "--no-markup", "--title", title, "--text", body, "--width", "480").Run()
	default:
		_ = exec.Command(d.bin, "--title", title, "--msgbox", body).Run()
	}
}

// errorBox shows a failure.
func (d dialogKit) errorBox(title, body string) {
	switch d.kind {
	case "zenity":
		_ = exec.Command(d.bin, "--error", "--no-markup", "--title", title, "--text", body, "--width", "560").Run()
	default:
		_ = exec.Command(d.bin, "--title", title, "--error", body).Run()
	}
}

// progressBar is a progress dialog the user can cancel.
type progressBar interface {
	// Update moves the bar to a percentage and replaces its text.
	Update(percent int, text string)
	// Cancelled is closed when the user presses Cancel.
	Cancelled() <-chan struct{}
	Close()
}

// progress opens a progress dialog. When the desktop cannot show one, the
// returned bar does nothing, and the work goes ahead without it.
func (d dialogKit) progress(title, text string) progressBar {
	if d.kind == "zenity" {
		if p := startZenityProgress(d.bin, title, text); p != nil {
			return p
		}
	} else if p := startKdialogProgress(d.bin, title, text); p != nil {
		return p
	}
	return noProgress{}
}

type noProgress struct{}

func (noProgress) Update(int, string)         {}
func (noProgress) Cancelled() <-chan struct{} { return nil }
func (noProgress) Close()                     {}

// zenityProgress drives `zenity --progress`, which reads a percentage, or a
// "# text" line for its label, from standard input.
type zenityProgress struct {
	in        io.WriteCloser
	cmd       *exec.Cmd
	exited    chan struct{}
	cancelled chan struct{}
	mu        sync.Mutex
	closing   bool
}

func startZenityProgress(bin, title, text string) *zenityProgress {
	cmd := exec.Command(bin, "--progress", "--title", title, "--text", escapeMarkup(text),
		"--percentage", "0", "--width", "520")
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil
	}
	if err := cmd.Start(); err != nil {
		return nil
	}
	p := &zenityProgress{in: in, cmd: cmd, exited: make(chan struct{}), cancelled: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		p.mu.Lock()
		// Exiting on its own, rather than being closed, means Cancel.
		if !p.closing {
			close(p.cancelled)
		}
		p.mu.Unlock()
		close(p.exited)
	}()
	return p
}

func (p *zenityProgress) Update(percent int, text string) {
	// zenity expands C escapes in a "#" line, which is the only way to give
	// the label a second line.
	text = strings.NewReplacer(`\`, `\\`, "\n", `\n`).Replace(text)
	fmt.Fprintf(p.in, "%d\n# %s\n", percent, text)
}

func (p *zenityProgress) Cancelled() <-chan struct{} { return p.cancelled }

func (p *zenityProgress) Close() {
	p.mu.Lock()
	p.closing = true
	p.mu.Unlock()
	p.in.Close()
	_ = p.cmd.Process.Kill()
	<-p.exited
}

// kdialogProgress drives `kdialog --progressbar`, which detaches and is then
// controlled over D-Bus.
type kdialogProgress struct {
	bus       dbusTool
	service   string
	object    string
	cancelled chan struct{}
	stop      chan struct{}
	once      sync.Once
}

func startKdialogProgress(bin, title, text string) *kdialogProgress {
	// Without a way to send it D-Bus calls the dialog could never be closed,
	// so do not open it at all.
	bus, ok := findDBusTool()
	if !ok {
		return nil
	}
	out, err := exec.Command(bin, "--title", title, "--progressbar", text, "100").Output()
	ref := strings.Fields(string(out)) // "org.kde.kdialog-1234 /ProgressDialog"
	if err != nil || len(ref) != 2 {
		return nil
	}
	p := &kdialogProgress{bus: bus, service: ref[0], object: ref[1],
		cancelled: make(chan struct{}), stop: make(chan struct{})}
	p.call("showCancelButton", "bool", "true")
	go func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-p.stop:
				return
			case <-tick.C:
				if strings.Contains(p.call("wasCancelled", "", ""), "true") {
					close(p.cancelled)
					return
				}
			}
		}
	}()
	return p
}

func (p *kdialogProgress) Update(percent int, text string) {
	p.call("value", "int", fmt.Sprint(percent))
	p.call("setLabelText", "string", text)
}

func (p *kdialogProgress) Cancelled() <-chan struct{} { return p.cancelled }

func (p *kdialogProgress) Close() {
	p.once.Do(func() {
		close(p.stop)
		p.call("close", "", "")
	})
}

// call invokes a method of kdialog's progress dialog, or sets its value
// property, and returns the reply.
func (p *kdialogProgress) call(method, kind, arg string) string {
	const iface = "org.kde.kdialog.ProgressDialog"
	var args []string
	if p.bus.qdbus {
		args = []string{p.service, p.object}
		switch {
		case method == "value":
			args = append(args, "Set", "", "value", arg)
		case arg != "":
			args = append(args, method, arg)
		default:
			args = append(args, method)
		}
	} else {
		args = []string{"--session", "--print-reply", "--dest=" + p.service, p.object}
		switch {
		case method == "value":
			args = append(args, "org.freedesktop.DBus.Properties.Set",
				"string:"+iface, "string:value", "variant:int32:"+arg)
		case kind == "bool":
			args = append(args, iface+"."+method, "boolean:"+arg)
		case kind != "":
			args = append(args, iface+"."+method, kind+":"+arg)
		default:
			args = append(args, iface+"."+method)
		}
	}
	out, _ := exec.Command(p.bus.bin, args...).Output()
	return string(out)
}

// dbusTool is a command-line D-Bus client.
type dbusTool struct {
	bin   string
	qdbus bool // qdbus syntax rather than dbus-send's
}

func findDBusTool() (dbusTool, bool) {
	for _, name := range []string{"qdbus6", "qdbus", "qdbus-qt6", "qdbus-qt5"} {
		if p, err := exec.LookPath(name); err == nil {
			return dbusTool{bin: p, qdbus: true}, true
		}
	}
	if p, err := exec.LookPath("dbus-send"); err == nil {
		return dbusTool{bin: p}, true
	}
	return dbusTool{}, false
}

// windowsProgramPatterns are the file types Oliver can launch.
var windowsProgramPatterns = []string{"*.exe", "*.msi", "*.bat", "*.cmd", "*.EXE", "*.MSI"}

func fmtErr(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%v", err)
}
