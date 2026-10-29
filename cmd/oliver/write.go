package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/device"
	"github.com/robertpjenkins-bot/oliver/internal/flash"
	"github.com/robertpjenkins-bot/oliver/internal/ui"
)

const writeSynopsis = `oliver write <image> [-to <drive or partition>]

Writes an ISO or disk image to a whole drive or to a single partition.

  oliver write fedora.iso                 choose where it goes, then write it
  oliver write fedora.iso -to /dev/sdb    write it to that drive
  oliver write backup.img -to /dev/sdb2   write it into one partition
  oliver write -list                      show where an image could go

Dropping an image onto Oliver does the same from the desktop: you are asked
which drive or partition to use, and nothing is written until you confirm.
Everything already on the target is replaced, and the result is read back
afterwards to check the drive really holds it.

The disk the running system lives on is never offered. Filesystems mounted
from the target are unmounted first. If your user cannot write to the drive,
you are asked for your password through pkexec or sudo, and only the writing
itself runs with those rights.`

// newScanner reads the machine's drives. It is a variable so tests can
// describe a machine without touching the real one.
var newScanner = device.NewScanner

// unmountAsUser unmounts a filesystem without root, the way a file manager's
// eject button does. It is a variable so tests do not touch real mounts.
var unmountAsUser = func(m device.Mount) error {
	if _, err := exec.LookPath("udisksctl"); err == nil {
		if exec.Command("udisksctl", "unmount", "-b", m.Source).Run() == nil {
			return nil
		}
	}
	return exec.Command("umount", m.Target).Run()
}

// diskImagePatterns are the files offered when choosing an image.
var diskImagePatterns = []string{"*.iso", "*.img", "*.raw", "*.ISO", "*.IMG", "*.RAW"}

// looksLikeDiskImage reports whether an argument names a disk image, which is
// written to a drive rather than run.
func looksLikeDiskImage(arg string) bool {
	switch strings.ToLower(filepath.Ext(arg)) {
	case ".iso", ".img", ".raw":
		return true
	}
	return false
}

func (a *app) cmdWrite(args []string) error {
	fs := a.newFlagSet("write", writeSynopsis)
	to := fs.String("to", "", "the drive or partition to write to, such as /dev/sdb or /dev/sdb1")
	yes := fs.Bool("y", false, "write without asking for confirmation")
	noVerify := fs.Bool("no-verify", false, "do not read the image back to check it")
	list := fs.Bool("list", false, "list the drives and partitions an image could be written to")
	gui := fs.Bool("gui", false, "ask with desktop dialogs, even from a terminal")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("one image at a time: got %d", fs.NArg())
	}
	path := fs.Arg(0)

	if *list {
		return a.writeList(path)
	}

	// Started from a file manager — an image dropped onto Oliver, or opened
	// with it — there is no terminal to ask in, so ask with dialogs.
	if *gui || (!a.ui.Interactive && *to == "") {
		if kit, ok := findDialogKit(); ok {
			return a.guiWrite(kit, path, !*noVerify)
		}
		if *gui {
			return fmt.Errorf("no dialog program found: install zenity or kdialog, or run `oliver write` in a terminal")
		}
	}
	if path == "" {
		return fmt.Errorf("usage: oliver write <image> [-to <drive or partition>]")
	}
	return a.cliWrite(path, *to, *yes, !*noVerify)
}

// inspectImage reads an image and refuses one that cannot be written as it is.
func inspectImage(path string) (*flash.Image, error) {
	img, err := flash.Inspect(path)
	if err != nil {
		return nil, err
	}
	if img.Compression != "" {
		return nil, fmt.Errorf("%s is compressed with %s, and written as it is the drive would hold the compressed data. "+
			"Unpack it first:\n  %s\nthen write the file that produces", img.Name(), img.Compression, img.UnpackCommand())
	}
	return img, nil
}

// writeProblem explains why an image cannot go to a target, or returns "".
func writeProblem(img *flash.Image, t device.Target, scanner *device.Scanner) string {
	if p := t.Problem(); p != "" {
		return p
	}
	if img == nil {
		return ""
	}
	if img.Size > t.Size {
		return fmt.Sprintf("it holds %s and the image needs %s", humanBytes(t.Size), humanBytes(img.Size))
	}
	// Writing over the drive the image is read from destroys the image
	// halfway through.
	if m, ok := scanner.MountFor(img.Path); ok && t.Holds(m.Source) {
		return "the image file is stored on it"
	}
	return ""
}

// writeWarnings lists what is allowed but probably not what was meant.
func writeWarnings(img *flash.Image, t device.Target) []string {
	var out []string
	if img.Format == flash.ISO {
		out = append(out, "This ISO has no partition table, so written byte for byte it will not boot from a USB drive. "+
			"Windows installation ISOs are the usual example; for those, use Ventoy or WoeUSB, which prepare the drive properly.")
	}
	if !t.Whole() && img.Bootable() {
		out = append(out, "This is a partition, not a whole drive. A PC starts from a drive, so the image will not boot from here; "+
			"choose the whole drive to make a bootable USB stick.")
	}
	if !t.External() {
		where := "This is an internal disk"
		if !t.Whole() {
			where = "This partition is on an internal disk"
		}
		out = append(out, where+", not a drive you plugged in. Make sure it is the one you mean.")
	}
	if mounts := t.Mounts(); len(mounts) > 0 {
		var at []string
		for _, m := range mounts {
			at = append(at, m.Target)
		}
		out = append(out, "Mounted from it now: "+strings.Join(at, ", ")+". These are unmounted before writing.")
	}
	return out
}

// cliWrite is the terminal flow.
func (a *app) cliWrite(path, to string, yes, verify bool) error {
	u := a.ui
	img, err := inspectImage(path)
	if err != nil {
		return err
	}
	scanner := newScanner()

	var target device.Target
	if to != "" {
		target, err = scanner.FindTarget(to)
	} else {
		target, err = a.askTarget(img, scanner)
	}
	if err != nil {
		return err
	}
	if target.Path == "" {
		u.Info("cancelled; nothing was written")
		return nil
	}
	if p := writeProblem(img, target, scanner); p != "" {
		return fmt.Errorf("cannot write to %s: %s", target.Path, p)
	}

	u.Printf("%s", u.Bold("Write"))
	u.Printf("  %s  %s", img.Name(), u.Dim(humanBytes(img.Size)+", "+img.Describe()))
	u.Printf("%s", u.Bold("to"))
	u.Printf("  %s  %s", target.Path, u.Dim(humanBytes(target.Size)+", "+target.Name()+", "+target.Details()))
	for _, w := range writeWarnings(img, target) {
		u.Printf("")
		for _, line := range wrap(w, 76) {
			u.Printf("%s", u.Yellow(line))
		}
	}
	u.Printf("")
	u.Printf("%s", u.Red("Everything on "+target.Path+" will be erased."))
	if !yes && !u.Confirm("Write the image?", false) {
		u.Info("cancelled; nothing was written")
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	bar := &cliProgress{ui: u}
	started := false
	err = a.performWrite(ctx, img, target, verify, false, func(p flash.Progress) {
		started = true
		bar.report(p)
	})
	bar.end()
	if err != nil {
		switch {
		case ctx.Err() != nil && !started:
			// Stopped at the password prompt, say, before a byte went out.
			u.Info("cancelled; nothing was written")
			return nil
		case ctx.Err() != nil:
			return fmt.Errorf("stopped. %s now holds part of the image; write it again, or reformat it, before using it", target.Path)
		}
		return err
	}

	checked := ""
	if verify {
		checked = " and checked it"
	}
	u.Info("%s wrote %s to %s%s", u.Green("✓"), img.Name(), target.Path, checked)
	if target.External() {
		u.Info("It is safe to remove the drive.")
	}
	return nil
}

// askTarget lists the choices in a terminal and reads one. A zero Target
// means the user backed out.
func (a *app) askTarget(img *flash.Image, scanner *device.Scanner) (device.Target, error) {
	u := a.ui
	if !u.Interactive {
		return device.Target{}, fmt.Errorf("say where to write it with -to; `oliver write -list %s` shows the choices", img.Path)
	}
	targets, err := scanner.Targets()
	if err != nil {
		return device.Target{}, err
	}
	u.Printf("Where should %s go?", u.Bold(img.Name()))
	u.Printf("")
	choices := a.printTargets(img, targets, scanner)
	if len(choices) == 0 {
		return device.Target{}, fmt.Errorf("no drive or partition can take this image; plug in a USB drive and try again")
	}
	u.Printf("")
	for {
		answer, ok := u.Ask(fmt.Sprintf("Choose 1-%d, or q to cancel:", len(choices)))
		if !ok || answer == "" || strings.EqualFold(answer, "q") {
			return device.Target{}, nil
		}
		if n, err := strconv.Atoi(answer); err == nil && n >= 1 && n <= len(choices) {
			return choices[n-1], nil
		}
		for _, c := range choices {
			if answer == c.Path || answer == filepath.Base(c.Path) {
				return c, nil
			}
		}
		u.Printf("%s", u.Yellow("Type one of the numbers shown."))
	}
}

// printTargets shows every drive and partition, numbering the ones the image
// can go to and saying why the others cannot. It returns the numbered ones.
func (a *app) printTargets(img *flash.Image, targets []device.Target, scanner *device.Scanner) []device.Target {
	u := a.ui
	var choices []device.Target
	t := u.NewTable("#", "TARGET", "SIZE", "WHAT IS THERE")
	for _, tg := range targets {
		name := tg.Path
		if !tg.Whole() {
			name = "  " + name
		}
		what := tg.Name() + ", " + tg.Details()
		if p := writeProblem(img, tg, scanner); p != "" {
			t.Row("", u.Dim(name), u.Dim(humanBytes(tg.Size)), u.Dim(tg.Name()+": ")+u.Red("not offered, "+p))
			continue
		}
		choices = append(choices, tg)
		t.Row(strconv.Itoa(len(choices)), name, humanBytes(tg.Size), what)
	}
	if t.Len() == 0 {
		u.Printf("No drives found.")
		return nil
	}
	t.Render()
	return choices
}

// writeList is `oliver write -list`.
func (a *app) writeList(path string) error {
	var img *flash.Image
	if path != "" {
		var err error
		if img, err = inspectImage(path); err != nil {
			return err
		}
		a.ui.Printf("%s  %s", a.ui.Bold(img.Name()), a.ui.Dim(humanBytes(img.Size)+", "+img.Describe()))
		a.ui.Printf("")
	}
	scanner := newScanner()
	targets, err := scanner.Targets()
	if err != nil {
		return err
	}
	a.printTargets(img, targets, scanner)
	return nil
}

// performWrite unmounts what has to be unmounted, then writes: directly when
// this user may write to the device, and through the privileged helper when
// not.
func (a *app) performWrite(ctx context.Context, img *flash.Image, target device.Target, verify, gui bool, report func(flash.Progress)) error {
	for _, m := range target.Mounts() {
		_ = unmountAsUser(m)
	}
	// Look again: an unmount may have failed, or the drive been pulled out.
	current, err := newScanner().FindTarget(target.Path)
	if err != nil {
		return err
	}
	if p := current.Problem(); p != "" {
		return fmt.Errorf("cannot write to %s: %s", current.Path, p)
	}
	stillMounted := len(current.Mounts()) > 0

	switch {
	case os.Geteuid() == 0:
		if err := unmountAsRoot(current); err != nil {
			return err
		}
		return writeDirect(ctx, img, current.Path, verify, report)
	case current.Writable && !stillMounted:
		return writeDirect(ctx, img, current.Path, verify, report)
	}
	return a.writeElevated(ctx, img, current, verify, stillMounted, gui, report)
}

func writeDirect(ctx context.Context, img *flash.Image, path string, verify bool, report func(flash.Progress)) error {
	src, err := os.Open(img.Path)
	if err != nil {
		return err
	}
	defer src.Close()
	return flash.Run(ctx, flash.Job{Target: path, Source: src, Size: img.Size, Verify: verify}, report)
}

// unmountAsRoot unmounts every filesystem mounted from a target.
func unmountAsRoot(t device.Target) error {
	for _, m := range t.Mounts() {
		if err := syscall.Unmount(m.Target, 0); err != nil {
			if errors.Is(err, syscall.EBUSY) {
				return fmt.Errorf("%s is busy: close any window or program using files on it, then try again", m.Target)
			}
			return fmt.Errorf("unmounting %s: %w", m.Target, err)
		}
	}
	return nil
}

// writeElevated runs the write as root through a helper copy of Oliver.
func (a *app) writeElevated(ctx context.Context, img *flash.Image, target device.Target, verify, unmount, gui bool, report func(flash.Progress)) error {
	terminal := !gui && a.ui.Interactive
	elevate, err := elevationCommand(terminal)
	if err != nil {
		return err
	}
	bin, cleanup, err := helperBinary()
	if err != nil {
		return err
	}
	defer cleanup()

	args := append(append([]string{}, elevate[1:]...), bin, "write-helper",
		"-device", target.Path, "-size", strconv.FormatInt(img.Size, 10))
	if !verify {
		args = append(args, "-no-verify")
	}
	if unmount {
		args = append(args, "-unmount")
	}

	src, err := os.Open(img.Path)
	if err != nil {
		return err
	}
	defer src.Close()

	cmd := exec.Command(elevate[0], args...)
	if terminal {
		// sudo's own messages, such as a mistyped password, belong on screen.
		cmd.Stderr = a.ui.Err
		a.ui.Info("Writing to %s needs administrator rights.", target.Path)
	}
	return runHelper(ctx, cmd, src, report)
}

// elevationCommand returns the program that runs the helper as root: sudo in
// a terminal, where it can ask for the password, and pkexec on the desktop,
// where it asks with a dialog.
func elevationCommand(terminal bool) ([]string, error) {
	if terminal {
		if _, err := exec.LookPath("sudo"); err == nil {
			return []string{"sudo", "--"}, nil
		}
	}
	if _, err := exec.LookPath("pkexec"); err == nil {
		return []string{"pkexec"}, nil
	}
	return nil, errors.New("writing to this drive needs administrator rights, and neither pkexec nor sudo is " +
		"available to ask for them. Run Oliver as root for this, or add yourself to the disk group " +
		"(`sudo usermod -aG disk $USER`, then log out and back in)")
}

// helperBinary returns a copy of this program that root can execute.
//
// Inside an AppImage the running binary sits in a FUSE mount that only the
// user who mounted it can enter — root included — so pkexec and sudo would
// fail with "permission denied" before the helper even started. A copy
// outside the mount does not have that problem.
func helperBinary() (string, func(), error) {
	exe, err := os.Executable()
	if err != nil {
		return "", nil, err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	if os.Getenv("APPIMAGE") == "" {
		return exe, func() {}, nil
	}

	// The runtime directory, unlike /tmp, is not mounted noexec on hardened
	// systems.
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	tmp, err := os.MkdirTemp(dir, "oliver-writer-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tmp) }
	dst := filepath.Join(tmp, "oliver")
	if err := copyFile(exe, dst, 0o755); err != nil {
		cleanup()
		return "", nil, err
	}
	return dst, cleanup, nil
}

// runHelper starts the privileged helper, feeds it the image and relays its
// progress.
//
// The helper runs as root, so it cannot be signalled. Cancelling ctx closes
// both pipes instead: the helper sees the image end early, or loses the
// reader of its progress, and stops either way.
func runHelper(ctx context.Context, cmd *exec.Cmd, src io.Reader, report func(flash.Progress)) error {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	if cmd.Stderr == nil {
		cmd.Stderr = &stderr
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", filepath.Base(cmd.Path), err)
	}

	finished := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			stdin.Close()
			stdout.Close()
		case <-finished:
		}
	}()
	go func() {
		_, _ = io.Copy(stdin, src)
		stdin.Close()
	}()

	var helperErr string
	done := false
	scan := bufio.NewScanner(stdout)
	for scan.Scan() {
		m, err := flash.ParseMessage(scan.Text())
		if err != nil {
			continue
		}
		switch {
		case m.Progress != nil:
			report(*m.Progress)
		case m.Error != "":
			helperErr = m.Error
		case m.Done:
			done = true
		}
	}
	waitErr := cmd.Wait()
	close(finished)

	switch {
	case done && waitErr == nil:
		return nil
	case helperErr != "":
		return errors.New(helperErr)
	case ctx.Err() != nil:
		return ctx.Err()
	}

	detail := strings.TrimSpace(stderr.String())
	if detail != "" {
		detail = ": " + detail
	}
	// pkexec's own statuses for a dismissed or refused password prompt.
	var exit *exec.ExitError
	if errors.As(waitErr, &exit) && filepath.Base(cmd.Path) == "pkexec" && (exit.ExitCode() == 126 || exit.ExitCode() == 127) {
		return fmt.Errorf("administrator permission was not given, so nothing was written%s", detail)
	}
	if waitErr == nil {
		return fmt.Errorf("the writer finished without confirming the write%s", detail)
	}
	return fmt.Errorf("the writer stopped unexpectedly (%v)%s", waitErr, detail)
}

// cmdWriteHelper is the privileged side of `oliver write`. It is not meant to
// be run by hand: it reads the image from standard input and reports progress
// on standard output, one line at a time.
func (a *app) cmdWriteHelper(args []string) error {
	fs := flag.NewFlagSet("write-helper", flag.ContinueOnError)
	dev := fs.String("device", "", "device to write")
	size := fs.Int64("size", 0, "bytes of image to read from standard input")
	noVerify := fs.Bool("no-verify", false, "skip reading the image back")
	unmount := fs.Bool("unmount", false, "unmount the device's filesystems first")
	if err := fs.Parse(args); err != nil {
		return err
	}
	return writeHelper(os.Stdin, os.Stdout, *dev, *size, !*noVerify, *unmount)
}

func writeHelper(in io.Reader, out io.Writer, dev string, size int64, verify, unmount bool) error {
	emit := func(m flash.Message) { fmt.Fprintln(out, m) }
	err := func() error {
		// Check again, now as root. The helper must not rely on its caller
		// having refused the system disk.
		target, err := newScanner().FindTarget(dev)
		if err != nil {
			return err
		}
		if p := target.Problem(); p != "" {
			return fmt.Errorf("cannot write to %s: %s", target.Path, p)
		}
		if size <= 0 || size > target.Size {
			return fmt.Errorf("an image of %d bytes does not fit on %s", size, target.Path)
		}
		if unmount {
			if err := unmountAsRoot(target); err != nil {
				return err
			}
		}
		job := flash.Job{Target: target.Path, Source: in, Size: size, Verify: verify}
		return flash.Run(context.Background(), job, func(p flash.Progress) {
			emit(flash.Message{Progress: &p})
		})
	}()
	if err != nil {
		emit(flash.Message{Error: err.Error()})
		// An exitError keeps main from raising a desktop dialog as root.
		return &exitError{code: 1, err: err}
	}
	emit(flash.Message{Done: true})
	return nil
}

// cliProgress draws a progress line in a terminal.
type cliProgress struct {
	ui    *ui.UI
	phase flash.Phase
	start time.Time
	last  time.Time
	open  bool // a progress line is on screen and needs ending
}

func phaseLabel(p flash.Phase) string {
	if p == flash.Verifying {
		return "checking"
	}
	return "writing"
}

func (c *cliProgress) report(p flash.Progress) {
	now := time.Now()
	if p.Phase != c.phase {
		c.end()
		c.phase, c.start = p.Phase, now
		if !c.ui.Interactive {
			c.ui.Info("%s…", phaseLabel(p.Phase))
		}
	}
	if !c.ui.Interactive || (now.Sub(c.last) < 200*time.Millisecond && p.Done < p.Total) {
		return
	}
	c.last = now

	pct := p.Percent()
	filled := pct * 24 / 100
	line := fmt.Sprintf("  %-8s %3d%%  [%s%s]  %s of %s", phaseLabel(p.Phase), pct,
		strings.Repeat("=", filled), strings.Repeat(" ", 24-filled), humanBytes(p.Done), humanBytes(p.Total))
	if elapsed := now.Sub(c.start).Seconds(); elapsed > 1 && p.Done > 0 {
		rate := float64(p.Done) / elapsed
		line += fmt.Sprintf("  %s/s", humanBytes(int64(rate)))
		if p.Done < p.Total {
			line += "  " + formatDuration(time.Duration(float64(p.Total-p.Done)/rate)*time.Second) + " left"
		}
	}
	fmt.Fprintf(c.ui.Err, "\r%s\x1b[K", line)
	c.open = true
}

// end finishes the progress line, so what follows starts on a fresh one.
func (c *cliProgress) end() {
	if c.open {
		fmt.Fprintln(c.ui.Err)
		c.open = false
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d >= time.Hour {
		return fmt.Sprintf("%d:%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
	}
	return fmt.Sprintf("%d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

// guiWrite is the desktop flow: choose an image if none was dropped, choose
// where it goes, confirm, and write with a progress bar.
func (a *app) guiWrite(kit dialogKit, path string, verify bool) error {
	const title = "Write a disk image"
	if path == "" {
		var ok bool
		path, ok = kit.openFile("Choose a disk image, or drop one here", "Disk images", diskImagePatterns)
		if !ok || path == "" {
			return nil
		}
	}
	img, err := inspectImage(path)
	if err != nil {
		kit.errorBox("Cannot write this file", fmtErr(err))
		return nil
	}

	scanner := newScanner()
	var choices []device.Target
	for {
		targets, err := scanner.Targets()
		if err != nil {
			kit.errorBox("Cannot list the drives", fmtErr(err))
			return nil
		}
		choices = choices[:0]
		var refused []string
		for _, t := range targets {
			if p := writeProblem(img, t, scanner); p != "" {
				refused = append(refused, fmt.Sprintf("%s (%s): %s", t.Path, t.Name(), p))
				continue
			}
			choices = append(choices, t)
		}
		if len(choices) > 0 {
			break
		}
		body := fmt.Sprintf("No drive or partition can take %s (%s).", img.Name(), humanBytes(img.Size))
		if len(refused) > 0 {
			body += "\n\nNot offered:\n" + strings.Join(refused, "\n")
		}
		if !kit.confirm("No drive to write to", body+"\n\nPlug in a USB drive, then choose Yes to look again.") {
			return nil
		}
	}

	rows := make([][]string, 0, len(choices))
	for _, t := range choices {
		name := t.Path
		if !t.Whole() {
			name = "    " + name
		}
		rows = append(rows, []string{t.Path, name, humanBytes(t.Size), t.Name() + ", " + t.Details()})
	}
	prompt := fmt.Sprintf("Where should %s (%s) be written?\nChoose a whole drive, or a single partition.", img.Name(), humanBytes(img.Size))
	choice, ok := kit.pick(title, prompt, []string{"Drive or partition", "Size", "What is there now"}, rows)
	if !ok || choice == "" {
		return nil
	}
	var target device.Target
	for _, t := range choices {
		if t.Path == choice {
			target = t
		}
	}
	if target.Path == "" {
		return nil
	}

	body := fmt.Sprintf("Write %s (%s, %s)\n\nto %s: %s, %s\n%s",
		img.Name(), humanBytes(img.Size), img.Describe(),
		target.Path, target.Name(), humanBytes(target.Size), target.Details())
	for _, w := range writeWarnings(img, target) {
		body += "\n\n" + w
	}
	body += "\n\nEverything on " + target.Path + " will be erased."
	if !kit.confirm("Erase "+target.Path+" and write?", body) {
		return nil
	}

	bar := kit.progress(title, "Starting… you may be asked for your password.")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-bar.Cancelled():
			cancel()
		case <-ctx.Done():
		}
	}()

	lastPct, lastPhase := -1, flash.Phase("")
	started := false
	err = a.performWrite(ctx, img, target, verify, true, func(p flash.Progress) {
		started = true
		// Never 100 until the end: reaching it can close the dialog early.
		pct := min(p.Percent(), 99)
		if pct == lastPct && p.Phase == lastPhase {
			return
		}
		lastPct, lastPhase = pct, p.Phase
		text := fmt.Sprintf("Writing %s to %s\n%s of %s", img.Name(), target.Path, humanBytes(p.Done), humanBytes(p.Total))
		if p.Phase == flash.Verifying {
			text = fmt.Sprintf("Checking that %s holds the image\n%s of %s", target.Path, humanBytes(p.Done), humanBytes(p.Total))
		}
		bar.Update(pct, text)
	})
	cancelled := ctx.Err() != nil
	bar.Close()

	switch {
	case err != nil && cancelled && !started:
		kit.info("Cancelled", "Nothing was written.")
	case err != nil && cancelled:
		kit.info("Stopped", target.Path+" now holds part of the image. Write it again, or reformat it, before using it.")
	case err != nil:
		kit.errorBox("Writing failed", fmtErr(err))
	default:
		msg := img.Name() + " was written to " + target.Path
		if verify {
			msg += " and checked"
		}
		msg += "."
		if target.External() {
			msg += "\n\nIt is safe to remove the drive."
		}
		kit.info("Done", msg)
	}
	return nil
}
