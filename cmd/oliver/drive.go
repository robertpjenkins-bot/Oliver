package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/device"
	"github.com/robertpjenkins-bot/oliver/internal/imaging"
	"github.com/robertpjenkins-bot/oliver/internal/ui"
)

const writeSynopsis = `oliver write <image.iso> [drive]

Writes a disc image to a drive, and checks that it arrived intact.

  oliver write ubuntu.iso              choose the drive from a list
  oliver write ubuntu.iso /dev/sdb     name it outright
  oliver write ubuntu.iso -no-verify   skip the read-back check

Everything already on the drive is destroyed. Oliver will not write to the
disk this system is running from, and asks before writing to anything that is
not removable.

After writing, the drive is read back and compared with the image. This is the
part worth waiting for: a failing USB stick accepts every byte you send it and
hands back something else, and the first you would otherwise hear of it is a
machine that will not boot.`

const eraseSynopsis = `oliver erase [drive]

Erases a drive and leaves an empty one behind.

  oliver erase                      choose the drive from a list
  oliver erase /dev/sdb -name WORK  erase it and name the new volume
  oliver erase /dev/sdb -raw        erase it, leave it unformatted
  oliver erase /dev/sdb -full       overwrite every byte first

By default this clears the partition tables and filesystem signatures, then
creates one empty FAT32 volume filling the drive — which is what makes a stick
that has had an image written to it usable again for ordinary files.

-full overwrites the whole drive with zeroes first. That takes as long as a
write of the same size, and is what to do before passing a drive on to
somebody else. It is not a secure erase: flash media keeps spare blocks that
no write from here can reach.`

const formatSynopsis = `oliver format [drive] -name NAME

Puts one empty FAT32 volume on a drive, filling it.

  oliver format /dev/sdb -name BACKUP

FAT32 is the deliberate choice: Linux, macOS, Windows and the firmware of a
machine being installed can all read it. The price is that no single file may
exceed four gigabytes.

This writes a fresh partition table and filesystem. It does not overwrite the
data that was there — use "oliver erase -full" for that.`

const repairSynopsis = `oliver repair [drive]

Works out why a drive is behaving oddly, and puts it right.

  oliver repair                 choose the drive from a list
  oliver repair /dev/sdb        name it outright
  oliver repair /dev/sdb -check look, report, change nothing

The usual complaint this answers: a stick that used to be 64 GB now shows up
as 2 GB, or turns up read-only, or a partitioning tool refuses to touch it.
None of that is the drive failing. It is the few sectors that describe the
drive disagreeing with the drive — normally because an installation image was
written over them, and an image describes a disc.

Repair says what it found, then clears those sectors and writes a fresh
full-size FAT32 volume, so every tool sees the whole drive again.

It does not recover files. There is no undoing a repair: what it fixes is the
drive, and the way it fixes it is by overwriting what is on it. If the files
matter, copy them off first — a drive that mounts read-only will usually still
let you read them.`

const unprotectSynopsis = `oliver unprotect [drive]

Works out why a drive will not take writes, and clears what can be cleared.

  oliver unprotect              choose the drive from a list
  oliver unprotect /dev/sdb     name it outright
  oliver unprotect /dev/sdb -check   look, report, change nothing

"The disk is write-protected" has four causes and they are not equally
fixable. Two are flags: the kernel's read-only flag on the device, and a
filesystem mounted read-only. Both clear in a second, and clearing them
changes nothing on the drive.

The third is the filesystem itself — an installation image leaves one that
cannot be written to by its nature. "oliver repair" is what clears that.

The fourth is the drive refusing: a lock switch on the side of the stick, or a
controller that has put worn-out flash beyond writing for good. Nothing here
can change either, and Oliver says so rather than pretending.`

const verifySynopsis = `oliver verify <image.iso> <drive>

Compares a drive with the image that should be on it.

  oliver verify ubuntu.iso /dev/sdb

Reads back as many bytes as the image holds and compares their checksum with
the image's. Nothing is written. Use it on a stick that has been in a drawer,
or one that was written somewhere else.`

// newScanner builds the block device scanner. It is a variable so tests can
// describe a machine's drives without touching the real ones.
var newScanner = device.NewScanner

// guard holds the answers to "may this drive be written to", gathered from
// flags before anything is opened.
type guard struct {
	allowFixed bool
	unmount    bool
	force      bool
}

func (a *app) cmdWrite(args []string) error {
	fs := a.newFlagSet("write", writeSynopsis)
	noVerify := fs.Bool("no-verify", false, "do not read the drive back afterwards")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	block := fs.Int("block-size", imaging.DefaultBlockSize/1024, "transfer size in KiB")
	g := guardFlags(fs)
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("usage: oliver write <image.iso> [drive]")
	}

	img, err := imaging.Inspect(fs.Arg(0))
	if err != nil {
		return err
	}
	// Everything from here to the result is addressed to the person running
	// the command, so it goes to stderr and leaves stdout to -json.
	u := a.ui.OnStderr()
	u.Printf("%s", u.Bold("Image"))
	u.Printf("  %s", fs.Arg(0))
	u.Printf("  %s", img.Describe())
	for _, w := range img.Warnings {
		u.Printf("")
		for _, line := range wrap(w, 72) {
			u.Printf("  %s", u.Yellow(line))
		}
	}
	u.Printf("")

	dev, err := a.chooseDrive("write to", fs.Arg(1), *g)
	if err != nil {
		return err
	}
	if dev.Size > 0 && img.Size > dev.Size {
		return fmt.Errorf("%s holds %s, which is smaller than the %s image",
			dev.Path, humanBytes(dev.Size), humanBytes(img.Size))
	}

	if err := checkBusy(dev, "write to", *g); err != nil {
		return err
	}
	if err := a.confirmDestruction(dev, fmt.Sprintf("Write %s to %s?", baseName(fs.Arg(0)), dev.Path), *g); err != nil {
		return err
	}
	if err := a.release(dev, *g); err != nil {
		return err
	}

	ctx, stop := interruptible()
	defer stop()

	rep := newReporter(u)
	res, err := imaging.Write(ctx, fs.Arg(0), dev.Path, imaging.Options{
		Verify:    !*noVerify,
		BlockSize: *block * 1024,
		Progress:  rep.report,
	})
	rep.close()
	if err != nil {
		return writeError(err, dev)
	}

	if *asJSON {
		return a.printJSON(res)
	}
	a.reportWrite(res, dev, *noVerify)
	return nil
}

func (a *app) cmdErase(args []string) error {
	fs := a.newFlagSet("erase", eraseSynopsis)
	full := fs.Bool("full", false, "overwrite every byte rather than only the partition tables")
	raw := fs.Bool("raw", false, "leave the drive unformatted afterwards")
	name := fs.String("name", "", "name for the new volume (up to 11 characters)")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	g := guardFlags(fs)
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	// A bad name must stop the command now, not after the drive is empty.
	label, err := imaging.NormaliseLabel(*name)
	if err != nil {
		return err
	}

	dev, err := a.chooseDrive("erase", fs.Arg(0), *g)
	if err != nil {
		return err
	}

	question := fmt.Sprintf("Erase %s?", dev.Path)
	if *full {
		question = fmt.Sprintf("Erase every byte of %s?", dev.Path)
	}
	if err := checkBusy(dev, "erase", *g); err != nil {
		return err
	}
	if err := a.confirmDestruction(dev, question, *g); err != nil {
		return err
	}
	if err := a.release(dev, *g); err != nil {
		return err
	}

	ctx, stop := interruptible()
	defer stop()

	u := a.ui
	rep := newReporter(u)
	res, err := imaging.Wipe(ctx, dev.Path, imaging.WipeOptions{Full: *full, Progress: rep.report})
	rep.close()
	if err != nil {
		return writeError(err, dev)
	}
	u.Info("%s erased %s of %s in %s", u.Green("✓"), humanBytes(res.Bytes), dev.Path, shortDuration(res.Duration))

	if *raw {
		if *asJSON {
			return a.printJSON(res)
		}
		u.Info("%s", u.Dim("The drive now has no partition table. Format it before use."))
		return nil
	}
	return a.format(dev, label, *asJSON)
}

func (a *app) cmdFormat(args []string) error {
	fs := a.newFlagSet("format", formatSynopsis)
	name := fs.String("name", "", "name for the volume (up to 11 characters)")
	bootable := fs.Bool("bootable", false, "mark the partition active")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	g := guardFlags(fs)
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	label, err := imaging.NormaliseLabel(*name)
	if err != nil {
		return err
	}

	dev, err := a.chooseDrive("format", fs.Arg(0), *g)
	if err != nil {
		return err
	}
	if err := checkBusy(dev, "format", *g); err != nil {
		return err
	}
	if err := a.confirmDestruction(dev, fmt.Sprintf("Format %s as FAT32?", dev.Path), *g); err != nil {
		return err
	}
	if err := a.release(dev, *g); err != nil {
		return err
	}
	if *bootable {
		return a.formatWith(dev, imaging.FormatOptions{Label: label, Bootable: true}, *asJSON)
	}
	return a.format(dev, label, *asJSON)
}

func (a *app) cmdVerify(args []string) error {
	fs := a.newFlagSet("verify", verifySynopsis)
	asJSON := fs.Bool("json", false, "print the result as JSON")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: oliver verify <image.iso> <drive>")
	}

	dev, err := newScanner().Find(fs.Arg(1))
	if err != nil {
		return err
	}
	if !dev.Readable {
		return fmt.Errorf("cannot read %s. %s", dev.Path, device.AccessAdvice(dev))
	}

	ctx, stop := interruptible()
	defer stop()

	u := a.ui
	rep := newReporter(u)
	res, err := imaging.Verify(ctx, fs.Arg(0), dev.Path, imaging.Options{Progress: rep.report})
	rep.close()
	if err != nil {
		if errors.Is(err, imaging.ErrVerifyFailed) && *asJSON {
			// Report it, but still fail: a script must not read a corrupt
			// drive as a success because it asked for JSON.
			_ = a.printJSON(res)
		}
		return err
	}
	if *asJSON {
		return a.printJSON(res)
	}
	u.Info("%s %s holds %s, byte for byte", u.Green("✓"), dev.Path, baseName(fs.Arg(0)))
	u.Info("  %s", u.Dim("sha256 "+res.Source))
	return nil
}

func (a *app) cmdUnprotect(args []string) error {
	fs := a.newFlagSet("unprotect", unprotectSynopsis)
	checkOnly := fs.Bool("check", false, "report what is protecting it and change nothing")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	g := guardFlags(fs)
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	scanner := newScanner()
	var dev *device.Device
	var err error
	if *checkOnly || fs.Arg(0) != "" {
		dev, err = scanner.Find(fs.Arg(0))
	} else {
		dev, err = a.pickDrive(scanner, "unprotect")
	}
	if err != nil {
		return err
	}
	// A drive the running system lives on is not one to unlock, whatever else
	// is true of it.
	if dev.SystemCritical() && !*checkOnly {
		return withFlagHint(device.CheckWritable(dev, "unprotect", g.allowFixed))
	}

	protections := scanner.WriteProtection(dev)
	if *asJSON && *checkOnly {
		return a.printJSON(protections)
	}

	u := a.ui.OnStderr()
	u.Printf("%s", u.Bold("Drive"))
	u.Printf("  %s", dev.Describe())
	u.Printf("")

	if len(protections) == 0 {
		// Nothing in software is in the way. Either it takes writes already,
		// or the refusal is coming from the drive.
		if dev.Writable && !dev.ReadOnly {
			u.Printf("%s Nothing is write-protecting this drive.", u.Green("✓"))
			if !dev.Writable {
				u.Printf("  %s", u.Dim(device.AccessAdvice(dev)))
			}
			return nil
		}
		hardware := device.HardwareProtection(dev)
		a.printProtection(hardware)
		return nil
	}

	for _, p := range protections {
		a.printProtection(p)
	}
	if *checkOnly {
		u.Printf("%s", u.Dim("Clear what can be cleared with: oliver unprotect "+dev.Path))
		return nil
	}

	clearable := 0
	for _, p := range protections {
		if p.Clearable && p.Kind == device.KernelFlag {
			clearable++
		}
	}
	if clearable == 0 {
		u.Printf("%s", u.Dim("None of these is a flag Oliver can clear."))
		return nil
	}
	if !g.force && !u.Confirm(fmt.Sprintf("Clear the read-only flag on %s?", dev.Path), true) {
		return errCancelled
	}

	done, err := device.Unprotect(dev, protections)
	for _, line := range done {
		a.ui.Info("%s %s", a.ui.Green("✓"), line)
	}
	if err != nil {
		return err
	}

	// Look again: the drive has the last word on whether it worked.
	after := newScanner()
	fresh, findErr := after.Find(dev.Path)
	if findErr == nil && len(after.WriteProtection(fresh)) == 0 && !fresh.ReadOnly {
		a.ui.Info("%s %s now takes writes", a.ui.Green("✓"), dev.Path)
		return nil
	}
	a.ui.Info("%s", a.ui.Yellow("the flag was cleared but the drive is still refusing"))
	a.printProtection(device.HardwareProtection(dev))
	return nil
}

func (a *app) printProtection(p device.Protection) {
	u := a.ui.OnStderr()
	marker := u.Yellow("•")
	if !p.Clearable {
		marker = u.Red("•")
	}
	u.Printf("%s %s", marker, u.Bold(p.Summary))
	for _, line := range wrap(p.Detail, 72) {
		u.Printf("  %s", u.Dim(line))
	}
	u.Printf("")
}

func (a *app) cmdRepair(args []string) error {
	fs := a.newFlagSet("repair", repairSynopsis)
	checkOnly := fs.Bool("check", false, "report what is wrong and change nothing")
	name := fs.String("name", "", "name for the repaired volume (up to 11 characters)")
	asJSON := fs.Bool("json", false, "print the result as JSON")
	g := guardFlags(fs)
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	label, err := imaging.NormaliseLabel(*name)
	if err != nil {
		return err
	}

	// Looking costs nothing and changes nothing, so -check does not need the
	// permission a repair does.
	if *checkOnly {
		dev, err := newScanner().Find(fs.Arg(0))
		if err != nil {
			return err
		}
		return a.diagnose(dev, *asJSON)
	}

	dev, err := a.chooseDrive("repair", fs.Arg(0), *g)
	if err != nil {
		return err
	}
	diagnosis, err := imaging.Diagnose(dev.Path)
	if err != nil {
		return err
	}
	a.printFaults(diagnosis)

	u := a.ui.OnStderr()
	if diagnosis.Healthy() {
		u.Printf("")
		u.Printf("%s", u.Dim("Nothing here needs repairing. Repairing it anyway would erase it for no reason."))
		if !g.force && !u.Confirm("Erase and rebuild it regardless?", false) {
			return errCancelled
		}
	}

	if err := checkBusy(dev, "repair", *g); err != nil {
		return err
	}
	if err := a.confirmDestruction(dev, fmt.Sprintf("Repair %s?", dev.Path), *g); err != nil {
		return err
	}
	if err := a.release(dev, *g); err != nil {
		return err
	}

	ctx, stop := interruptible()
	defer stop()

	rep := newReporter(a.ui)
	wiped, err := imaging.Wipe(ctx, dev.Path, imaging.WipeOptions{Progress: rep.report})
	rep.close()
	if err != nil {
		return writeError(err, dev)
	}
	formatted, err := imaging.FormatFAT32(dev.Path, imaging.FormatOptions{Label: label})
	if err != nil {
		return err
	}

	result := &imaging.RepairResult{Before: diagnosis, Format: formatted, Cleared: wiped.Bytes}
	if *asJSON {
		return a.printJSON(result)
	}
	ui := a.ui
	volume := formatted.Label
	if volume == "" {
		volume = "unnamed"
	}
	ui.Info("%s repaired %s", ui.Green("✓"), dev.Path)
	ui.Info("  one FAT32 volume %q filling the drive, %s usable", volume, humanBytes(formatted.Usable))
	if recovered := result.Recovered(); recovered > 0 {
		ui.Info("  %s", ui.Green(fmt.Sprintf("%s that the drive was not admitting to is usable again",
			humanBytes(recovered))))
	}
	return nil
}

// diagnose reports on a drive without touching it.
func (a *app) diagnose(dev *device.Device, asJSON bool) error {
	diagnosis, err := imaging.Diagnose(dev.Path)
	if err != nil {
		return err
	}
	if asJSON {
		return a.printJSON(diagnosis)
	}
	a.printDiagnosis(diagnosis)
	if !diagnosis.Healthy() {
		a.ui.Info("")
		a.ui.Info("%s", a.ui.Dim(fmt.Sprintf("Put it right with: oliver repair %s", dev.Path)))
	}
	return nil
}

func (a *app) printDiagnosis(d *imaging.Diagnosis) {
	u := a.ui.OnStderr()
	u.Printf("%s", u.Bold("Drive"))
	u.Printf("  %s  %s", d.Path, humanBytes(d.Capacity))
	if !d.Volume.Empty() {
		u.Printf("  %s %s", u.Dim("holds"), d.Volume.String())
	}
	if d.Described > 0 {
		u.Printf("  %s %s", u.Dim("described by its partition table:"), humanBytes(d.Described))
	}
	u.Printf("")
	a.printFaults(d)
}

// printFaults reports the findings alone, for when the drive itself is about
// to be described again by the confirmation.
func (a *app) printFaults(d *imaging.Diagnosis) {
	u := a.ui.OnStderr()
	if d.Healthy() {
		u.Printf("%s %s", u.Green("✓"), "Nothing wrong with how this drive describes itself.")
		return
	}
	for _, f := range d.Faults {
		u.Printf("%s %s", u.Yellow("•"), u.Bold(f.Summary))
		for _, line := range wrap(f.Detail, 72) {
			u.Printf("  %s", u.Dim(line))
		}
		u.Printf("")
	}
}

func guardFlags(fs *flag.FlagSet) *guard {
	g := &guard{}
	fs.BoolVar(&g.allowFixed, "allow-fixed", false, "permit a drive that is not removable")
	fs.BoolVar(&g.unmount, "unmount", false, "unmount anything on the drive first")
	fs.BoolVar(&g.force, "f", false, "do not ask for confirmation")
	return g
}

// chooseDrive resolves the drive to operate on, either from the argument or by
// asking, and refuses the ones that must not be touched.
func (a *app) chooseDrive(action, arg string, g guard) (*device.Device, error) {
	scanner := newScanner()

	var dev *device.Device
	var err error
	if arg != "" {
		dev, err = scanner.Find(arg)
		if err != nil {
			return nil, err
		}
	} else {
		dev, err = a.pickDrive(scanner, action)
		if err != nil {
			return nil, err
		}
	}

	if err := device.CheckWritable(dev, action, g.allowFixed); err != nil {
		return nil, withFlagHint(err)
	}
	return dev, nil
}

// withFlagHint adds the way past a refusal, in the terms this front end
// offers: a flag to type. The rule itself is shared with every other caller,
// so only the phrasing lives here.
func withFlagHint(err error) error {
	reason, ok := device.RefusedFor(err)
	if !ok {
		return err
	}
	switch reason {
	case device.SystemDisk:
		return fmt.Errorf("%w.\nThere is no flag for this. Do it from a live USB, where this disk is not in use", err)
	case device.FixedDisk:
		return fmt.Errorf("%w.\nIf it really is the drive you mean — an internal disk you are reimaging, say — "+
			"pass -allow-fixed. Check the model and size first", err)
	case device.Mounted:
		return fmt.Errorf("%w.\nClose anything using it and unmount it, or pass -unmount to have Oliver do it", err)
	}
	return err
}

// pickDrive shows the drives and asks which one. Drives that hold the running
// system are listed but cannot be chosen, because seeing them is how a user
// confirms the list is the one they are looking at.
func (a *app) pickDrive(scanner *device.Scanner, action string) (*device.Device, error) {
	devices, err := scanner.List()
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("no drives found")
	}

	u := a.ui.OnStderr()
	var choices []*device.Device
	t := u.NewTable("#", "DEVICE", "SIZE", "TYPE", "CONTAINS", "MODEL")
	for _, d := range devices {
		number := ""
		if d.SystemCritical() {
			number = u.Red("––")
		} else {
			choices = append(choices, d)
			number = u.Bold(strconv.Itoa(len(choices)))
		}
		kind := "fixed"
		if d.Removable {
			kind = u.Green("removable")
		}
		contents := describeContents(d)
		if d.SystemCritical() {
			contents = u.Red("the running system")
		}
		t.Row(number, d.Path, humanBytes(d.Size), kind, contents, d.Model)
	}
	t.Render()
	u.Printf("")

	if len(choices) == 0 {
		return nil, fmt.Errorf("every drive on this machine holds the running system; there is nothing safe to %s", action)
	}
	if !u.Interactive {
		return nil, fmt.Errorf("no drive given, and there is no terminal to ask at; name one: oliver ... /dev/sdX")
	}

	answer, ok := u.Ask(fmt.Sprintf("Which drive should Oliver %s? [1-%d, or q to stop]", action, len(choices)))
	if !ok || answer == "" || strings.EqualFold(answer, "q") {
		return nil, errCancelled
	}
	n, err := strconv.Atoi(answer)
	if err != nil || n < 1 || n > len(choices) {
		return nil, fmt.Errorf("%q is not one of the choices", answer)
	}
	return choices[n-1], nil
}

// describeContents says what is on a drive now, read from the drive itself.
func describeContents(d *device.Device) string {
	if len(d.Partitions) == 0 {
		return imaging.Describe(d.Path)
	}
	var parts []string
	for _, p := range d.Partitions {
		if v := imaging.Probe(p.Path); !v.Empty() {
			parts = append(parts, v.String())
		}
	}
	if len(parts) == 0 {
		if len(d.Partitions) == 1 {
			return "1 partition"
		}
		return fmt.Sprintf("%d partitions", len(d.Partitions))
	}
	return strings.Join(parts, ", ")
}

// confirmDestruction spells out what is about to be lost and requires the
// drive's own name to be typed back.
func (a *app) confirmDestruction(d *device.Device, question string, g guard) error {
	u := a.ui.OnStderr()
	u.Printf("%s", u.Bold("Drive"))
	u.Printf("  %s", d.Describe())
	if c := describeContents(d); c != "" && c != "empty" && c != "unreadable" {
		u.Printf("  %s %s", u.Dim("contains"), c)
	}
	for _, p := range d.Partitions {
		mounted := ""
		if len(p.Mounts) > 0 {
			mounted = u.Yellow("  mounted at " + p.Mounts[0].Target)
		}
		u.Printf("    %s %s %s%s", u.Dim(p.Path), humanBytes(p.Size), imaging.Probe(p.Path).String(), mounted)
	}
	u.Printf("")
	u.Printf("  %s", u.Red("Everything on this drive will be destroyed."))
	u.Printf("")

	if g.force {
		return nil
	}
	if !u.Interactive {
		return fmt.Errorf("refusing to destroy %s without confirmation, and there is no terminal to ask at; pass -f if you are sure", d.Path)
	}
	// Typing the drive's name is the difference between agreeing and
	// acknowledging which drive was agreed to.
	if !u.ConfirmWord(fmt.Sprintf("%s Type %s to continue:", question, u.Bold(d.Name)), d.Name) {
		return errCancelled
	}
	return nil
}

// checkBusy refuses a drive the system is still reading from. It runs before
// the confirmation prompt, so a write that cannot proceed is not preceded by a
// question about destroying a drive.
func checkBusy(d *device.Device, action string, g guard) error {
	if g.unmount {
		return nil
	}
	return withFlagHint(device.CheckIdle(d, action))
}

// release unmounts what is on the drive, once the user has agreed to destroy
// it. Nothing is unmounted before that agreement.
func (a *app) release(d *device.Device, g guard) error {
	if !d.Busy() {
		return nil
	}
	a.ui.Info("· unmounting %d filesystem(s) on %s", len(d.Mounts), d.Path)
	return device.Unmount(d)
}

func (a *app) format(d *device.Device, label string, asJSON bool) error {
	return a.formatWith(d, imaging.FormatOptions{Label: label}, asJSON)
}

func (a *app) formatWith(d *device.Device, opt imaging.FormatOptions, asJSON bool) error {
	// The volume this writes describes itself as having 512-byte sectors. On a
	// drive that addresses larger ones it would be laid out wrong, and the
	// failure would appear later as a drive nothing can read.
	if n := d.LogicalBlockSize; n != 0 && n != 512 {
		return fmt.Errorf(
			"%s addresses %d-byte sectors, and Oliver writes FAT32 volumes with 512-byte ones.\n"+
				"Formatting it here would produce a drive nothing can read. Use mkfs.vfat for this drive, "+
				"or -raw to erase it and leave the filesystem to something else",
			d.Path, n)
	}
	res, err := imaging.FormatFAT32(d.Path, opt)
	if err != nil {
		return err
	}
	if asJSON {
		return a.printJSON(res)
	}
	u := a.ui
	name := res.Label
	if name == "" {
		name = "unnamed"
	}
	u.Info("%s formatted %s as FAT32 %q, %s usable in %s clusters",
		u.Green("✓"), d.Path, name, humanBytes(res.Usable), humanBytes(int64(res.ClusterBytes)))
	return nil
}

func (a *app) reportWrite(res *imaging.Result, d *device.Device, skippedVerify bool) {
	u := a.ui
	u.Info("%s wrote %s to %s in %s (%s/s)",
		u.Green("✓"), humanBytes(res.Bytes), d.Path, shortDuration(res.Duration), humanBytes(int64(res.Rate())))
	if res.Verified {
		u.Info("%s read back and compared: the drive holds the image exactly", u.Green("✓"))
		u.Info("  %s", u.Dim("sha256 "+res.Source))
	} else if skippedVerify {
		u.Info("%s", u.Yellow("· not verified, because -no-verify was given"))
	}
	u.Info("%s", u.Dim("The drive can be removed now; everything has been flushed to it."))
}

// writeError explains the two failures that are about the drive rather than
// about the program.
func writeError(err error, d *device.Device) error {
	switch {
	case errors.Is(err, context.Canceled):
		return fmt.Errorf("stopped. %s has been partly written and holds neither the old contents nor the new; "+
			"erase it or write to it again before using it", d.Path)
	case errors.Is(err, imaging.ErrVerifyFailed):
		return fmt.Errorf("%w.\n"+
			"The write reported success, so this is the drive rather than the transfer: cheap or worn "+
			"flash media accepts every byte and returns something else. Try another drive, and do not "+
			"trust this one with anything that has to boot", err)
	}
	return err
}

// reporter turns the imaging package's progress calls into a bar per phase.
type reporter struct {
	ui    *ui.UI
	bar   *ui.Bar
	phase imaging.Phase
	last  imaging.Progress
}

func newReporter(u *ui.UI) *reporter { return &reporter{ui: u} }

func (r *reporter) report(p imaging.Progress) {
	if p.Phase != r.phase {
		r.close()
		r.phase = p.Phase
		r.bar = r.ui.NewBar(label(p.Phase))
	}
	r.last = p
	detail := fmt.Sprintf("%s of %s  %s/s", humanBytes(p.Done), humanBytes(p.Total), humanBytes(int64(p.Rate())))
	if remaining := estimate(p); remaining > 0 {
		detail += "  " + shortDuration(remaining) + " left"
	}
	r.bar.Update(p.Done, p.Total, detail)
}

// close finishes the current bar, leaving its summary line behind.
func (r *reporter) close() {
	if r.bar == nil {
		return
	}
	r.bar.Done(fmt.Sprintf("%s in %s (%s/s)",
		humanBytes(r.last.Done), shortDuration(r.last.Elapsed), humanBytes(int64(r.last.Rate()))))
	r.bar = nil
	r.phase = ""
}

func label(p imaging.Phase) string {
	switch p {
	case imaging.PhaseWriting:
		return "writing  "
	case imaging.PhaseVerifying:
		return "verifying"
	case imaging.PhaseErasing:
		return "erasing  "
	}
	return string(p)
}

func estimate(p imaging.Progress) time.Duration {
	rate := p.Rate()
	if rate <= 0 || p.Done >= p.Total {
		return 0
	}
	return time.Duration(float64(p.Total-p.Done)/rate) * time.Second
}

func shortDuration(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Hour:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
}

// interruptible returns a context that is cancelled by Ctrl-C, so a write stops
// between blocks rather than in the middle of one.
func interruptible() (context.Context, func()) {
	return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
}

// errCancelled is the user declining. It is reported without the word "error",
// since nothing went wrong.
var errCancelled = errors.New("cancelled")

func baseName(path string) string {
	if i := strings.LastIndexByte(path, '/'); i >= 0 {
		return path[i+1:]
	}
	return path
}
