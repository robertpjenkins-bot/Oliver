// Package ui handles Oliver's output: aligned tables for a terminal, and
// desktop dialogs when it was launched from a file manager with no terminal
// attached.
package ui

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// UI writes program output.
type UI struct {
	Out io.Writer
	Err io.Writer
	// In is where answers to prompts are read from.
	In    io.Reader
	Color bool
	// Interactive reports whether a terminal is attached. When it is not,
	// errors are shown as desktop notifications instead of vanishing.
	Interactive bool
	Quiet       bool

	in *bufio.Reader
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
		In:          os.Stdin,
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

// OnStderr returns a view of this UI whose tables and Printf lines go to
// standard error instead of standard output.
//
// Output that is addressed to a person — a list to choose from, a summary of
// what is about to be destroyed — must not land in the same stream as output a
// program asked for with --json. Prompts and progress already go to stderr;
// this keeps their surrounding context with them.
func (u *UI) OnStderr() *UI {
	// The buffered prompt reader is created before the copy so that every view
	// shares it. Two readers over one input would each buffer what the other
	// is waiting for, and the second question asked would never be answered.
	u.reader()
	clone := *u
	clone.Out = u.Err
	return &clone
}

// reader returns the prompt reader, buffered once so that an answer typed
// ahead of the next question is not lost between them.
func (u *UI) reader() *bufio.Reader {
	if u.in == nil {
		src := u.In
		if src == nil {
			src = os.Stdin
		}
		u.in = bufio.NewReader(src)
	}
	return u.in
}

// Ask puts a question and returns the answer with its whitespace removed. The
// second result is false when there is nobody to answer: no terminal, or the
// input ended.
func (u *UI) Ask(question string) (string, bool) {
	if !u.Interactive {
		return "", false
	}
	fmt.Fprintf(u.Err, "%s ", question)
	line, err := u.reader().ReadString('\n')
	if err != nil && line == "" {
		fmt.Fprintln(u.Err)
		return "", false
	}
	return strings.TrimSpace(line), true
}

// Confirm asks a yes/no question. Without a terminal it returns the default.
func (u *UI) Confirm(question string, def bool) bool {
	hint := "[y/N]"
	if def {
		hint = "[Y/n]"
	}
	answer, ok := u.Ask(question + " " + hint)
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

// ConfirmWord asks the user to type a specific word before something
// irreversible happens. A yes/no prompt is answered by reflex; typing the name
// of the drive about to be erased is not.
func (u *UI) ConfirmWord(question, word string) bool {
	answer, ok := u.Ask(question)
	if !ok {
		return false
	}
	return strings.EqualFold(answer, word)
}

// Bar reports the progress of something slow enough to need reporting.
//
// On a terminal it redraws one line in place. Piped into a file or a log it
// writes a line each time another tenth is done, because a log full of
// carriage returns is unreadable and a log with no progress at all leaves
// somebody wondering whether a four gigabyte write has stalled.
type Bar struct {
	ui      *UI
	label   string
	started time.Time
	drawn   bool
	step    int64
}

// NewBar starts a progress display labelled with what is happening.
func (u *UI) NewBar(label string) *Bar {
	return &Bar{ui: u, label: label, started: time.Now(), step: -1}
}

// Update redraws the bar. detail is shown after the percentage: a transfer
// rate, a remaining time, whatever the caller has.
func (b *Bar) Update(done, total int64, detail string) {
	if b.ui.Quiet || total <= 0 {
		return
	}
	pct := float64(done) / float64(total)
	if pct > 1 {
		pct = 1
	}

	if !b.ui.Interactive {
		// One line per tenth, and never the same tenth twice.
		if step := int64(pct * 10); step > b.step {
			b.step = step
			fmt.Fprintf(b.ui.Err, "%s %3.0f%%  %s\n", b.label, pct*100, detail)
		}
		return
	}

	const width = 28
	filled := int(pct * width)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Fprintf(b.ui.Err, "\r\x1b[K%s %s %3.0f%%  %s", b.label, b.ui.Cyan(bar), pct*100, b.ui.Dim(detail))
	b.drawn = true
}

// Done ends the bar, leaving one summary line where it was.
func (b *Bar) Done(summary string) {
	if b.ui.Quiet {
		return
	}
	if b.drawn {
		fmt.Fprint(b.ui.Err, "\r\x1b[K")
	}
	fmt.Fprintf(b.ui.Err, "%s %s\n", b.label, summary)
	b.drawn = false
}
