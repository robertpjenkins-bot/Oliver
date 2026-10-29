// Package ui handles Oliver's output: aligned tables for a terminal, and
// desktop dialogs when it was launched from a file manager with no terminal
// attached.
package ui

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// UI writes program output.
type UI struct {
	Out io.Writer
	Err io.Writer
	// In is where answers to prompts are read from; nil means standard input.
	In    io.Reader
	Color bool
	// Interactive reports whether a terminal is attached. When it is not,
	// errors are shown as desktop notifications instead of vanishing.
	Interactive bool
	Quiet       bool
}

// New builds a UI for the current process.
//
// Colour follows standard output, since that is where the coloured tables go:
// piping `oliver backends` into another program must not hand it escape codes.
// Interactivity follows standard error, which is where prompts and progress
// appear.
func New() *UI {
	colorOK := isTerminal(os.Stdout) && os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"
	return &UI{
		Out:         os.Stdout,
		Err:         os.Stderr,
		Color:       colorOK,
		Interactive: isTerminal(os.Stderr),
	}
}

func isTerminal(f *os.File) bool {
	st, err := f.Stat()
	if err != nil {
		return false
	}
	return st.Mode()&os.ModeCharDevice != 0
}

// ANSI colour helpers. They return the text unchanged when colour is off.
func (u *UI) paint(code, s string) string {
	if !u.Color {
		return s
	}
	return "\x1b[" + code + "m" + s + "\x1b[0m"
}

func (u *UI) Bold(s string) string   { return u.paint("1", s) }
func (u *UI) Dim(s string) string    { return u.paint("2", s) }
func (u *UI) Green(s string) string  { return u.paint("32", s) }
func (u *UI) Yellow(s string) string { return u.paint("33", s) }
func (u *UI) Red(s string) string    { return u.paint("31", s) }
func (u *UI) Cyan(s string) string   { return u.paint("36", s) }

// Printf writes to standard output.
func (u *UI) Printf(format string, args ...any) {
	fmt.Fprintf(u.Out, format+"\n", args...)
}

// Info writes a progress line to stderr, so it never contaminates piped output.
func (u *UI) Info(format string, args ...any) {
	if u.Quiet {
		return
	}
	fmt.Fprintf(u.Err, format+"\n", args...)
}

// Warn writes a highlighted warning.
func (u *UI) Warn(format string, args ...any) {
	fmt.Fprintf(u.Err, "%s %s\n", u.Yellow("warning:"), fmt.Sprintf(format, args...))
}

// Error reports a failure. Without a terminal it also raises a desktop dialog,
// because a double-clicked program that fails silently is impossible to debug.
func (u *UI) Error(err error) {
	fmt.Fprintf(u.Err, "%s %v\n", u.Red("error:"), err)
	if !u.Interactive {
		notify("Oliver could not run this program", err.Error())
	}
}

// notify shows a desktop dialog using whatever the desktop provides.
func notify(title, body string) {
	candidates := [][]string{
		{"zenity", "--error", "--no-wrap", "--title", title, "--text", body},
		{"kdialog", "--error", body, "--title", title},
		{"notify-send", "--urgency=critical", title, body},
		{"xmessage", "-center", title + "\n\n" + body},
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c[0]); err != nil {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		if err := cmd.Start(); err == nil {
			_ = cmd.Wait()
			return
		}
	}
}

// Table renders aligned columns.
type Table struct {
	headers []string
	rows    [][]string
	ui      *UI
}

// NewTable starts a table with the given column headers.
func (u *UI) NewTable(headers ...string) *Table {
	return &Table{headers: headers, ui: u}
}

// Row appends a row. Missing cells are padded, extra cells are kept.
func (t *Table) Row(cells ...string) { t.rows = append(t.rows, cells) }

// Len returns the number of data rows.
func (t *Table) Len() int { return len(t.rows) }

// Render writes the table to the UI's output.
func (t *Table) Render() {
	cols := len(t.headers)
	for _, r := range t.rows {
		if len(r) > cols {
			cols = len(r)
		}
	}
	widths := make([]int, cols)
	cell := func(row []string, i int) string {
		if i < len(row) {
			return row[i]
		}
		return ""
	}
	for i := 0; i < cols; i++ {
		if i < len(t.headers) {
			widths[i] = displayWidth(t.headers[i])
		}
		for _, r := range t.rows {
			if w := displayWidth(cell(r, i)); w > widths[i] {
				widths[i] = w
			}
		}
	}

	var b strings.Builder
	for i := 0; i < cols; i++ {
		h := ""
		if i < len(t.headers) {
			h = t.headers[i]
		}
		b.WriteString(t.ui.Bold(pad(h, widths[i], i == cols-1)))
		if i < cols-1 {
			b.WriteString("  ")
		}
	}
	fmt.Fprintln(t.ui.Out, strings.TrimRight(b.String(), " "))

	for _, r := range t.rows {
		var line strings.Builder
		for i := 0; i < cols; i++ {
			line.WriteString(pad(cell(r, i), widths[i], i == cols-1))
			if i < cols-1 {
				line.WriteString("  ")
			}
		}
		fmt.Fprintln(t.ui.Out, strings.TrimRight(line.String(), " "))
	}
}

// displayWidth measures a string ignoring ANSI escapes, so coloured cells
// still line up.
func displayWidth(s string) int {
	w, inEscape := 0, false
	for _, r := range s {
		switch {
		case inEscape:
			if r == 'm' {
				inEscape = false
			}
		case r == '\x1b':
			inEscape = true
		default:
			w++
		}
	}
	return w
}

func pad(s string, width int, last bool) string {
	if last {
		return s
	}
	if d := width - displayWidth(s); d > 0 {
		return s + strings.Repeat(" ", d)
	}
	return s
}

// Confirm asks a yes/no question. Without a terminal it returns the default.
func (u *UI) Confirm(question string, def bool) bool {
	if !u.Interactive {
		return def
	}
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}
	fmt.Fprintf(u.Err, "%s %s ", question, hint)
	answer, ok := u.readLine()
	if !ok {
		return def
	}
	switch strings.ToLower(answer) {
	case "y", "yes":
		return true
	case "n", "no":
		return false
	}
	return def
}

// Ask prints a prompt and returns the line typed in reply. It reports false
// when there is no terminal to ask, or the input ended.
func (u *UI) Ask(prompt string) (string, bool) {
	if !u.Interactive {
		return "", false
	}
	fmt.Fprintf(u.Err, "%s ", prompt)
	return u.readLine()
}

// readLine reads one line a byte at a time. Buffering would swallow input
// meant for the next prompt, or for a program started afterwards.
func (u *UI) readLine() (string, bool) {
	in := u.In
	if in == nil {
		in = os.Stdin
	}
	var line []byte
	b := make([]byte, 1)
	for {
		n, err := in.Read(b)
		if n == 1 {
			if b[0] == '\n' {
				break
			}
			line = append(line, b[0])
		}
		if err != nil {
			if len(line) == 0 {
				return "", false
			}
			break
		}
	}
	return strings.TrimSpace(string(line)), true
}
