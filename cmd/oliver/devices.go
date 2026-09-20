package main

import (
	"fmt"
	"strings"

	"github.com/robertpjenkins-bot/oliver/internal/device"
	"github.com/robertpjenkins-bot/oliver/internal/prefix"
)

const deviceSynopsis = `oliver device [list|attach|detach] ...

Exposes a real disk to a Windows program.

  oliver device list                        show this machine's drives
  oliver device attach <prefix> <device>    give a prefix access to a drive
  oliver device detach <prefix> <letter>    take it away again

Windows disk utilities expect to open drives directly. By default a Wine prefix
has no access to any of them, which is why such programs show an empty device
list. Attaching one exposes exactly that drive and nothing else.

Two things this cannot do: it will not attach a disk the running system is
using, and it cannot grant you permissions you do not have — if your user
cannot write to the device node, nor can the program.`

func (a *app) cmdDevice(args []string) error {
	fs := a.newFlagSet("device", deviceSynopsis)
	asJSON := fs.Bool("json", false, "print as JSON")
	letter := fs.String("letter", "", "drive letter to use (default: the first free one)")
	force := fs.Bool("f", false, "skip the confirmation prompt")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	sub := "list"
	if fs.NArg() > 0 {
		sub = fs.Arg(0)
	}

	switch sub {
	case "list", "ls":
		return a.deviceList(*asJSON)
	case "attach", "add":
		if fs.NArg() < 3 {
			return fmt.Errorf("usage: oliver device attach <prefix> <device>")
		}
		return a.deviceAttach(fs.Arg(1), fs.Arg(2), *letter, *force)
	case "detach", "remove", "rm":
		if fs.NArg() < 3 {
			return fmt.Errorf("usage: oliver device detach <prefix> <letter>")
		}
		return a.deviceDetach(fs.Arg(1), fs.Arg(2))
	default:
		return fmt.Errorf("unknown subcommand %q (want list, attach or detach)", sub)
	}
}

func (a *app) manager() *prefix.Manager {
	mgr := prefix.NewManager()
	if a.cfg.PrefixRoot != "" {
		mgr.Root = a.cfg.PrefixRoot
	}
	return mgr
}

func (a *app) deviceList(asJSON bool) error {
	devices, err := newScanner().List()
	if err != nil {
		return err
	}
	if asJSON {
		return a.printJSON(devices)
	}

	u := a.ui
	if len(devices) == 0 {
		u.Printf("No block devices found.")
		return nil
	}

	t := u.NewTable("DEVICE", "SIZE", "TYPE", "CONTAINS", "ACCESS", "MODEL", "IN USE BY")
	for _, d := range devices {
		kind := "fixed"
		if d.Removable {
			kind = u.Green("removable")
		}
		access := u.Red("none")
		switch {
		case d.Writable:
			access = "read/write"
		case d.Readable:
			access = u.Yellow("read only")
		}
		inUse := ""
		if c := d.CriticalMounts(); len(c) > 0 {
			inUse = u.Red("the running system (" + strings.Join(c, ", ") + ")")
		} else if len(d.Mounts) > 0 {
			inUse = u.Dim(fmt.Sprintf("%d mounted", len(d.Mounts)))
		}
		t.Row(d.Path, humanBytes(d.Size), kind, describeContents(d), access, d.Model, inUse)
	}
	t.Render()
	u.Printf("")
	u.Printf("%s", u.Dim("Write an image to one with:  oliver write <image.iso>"))
	u.Printf("%s", u.Dim("Erase and reformat one with: oliver erase"))
	u.Printf("%s", u.Dim("Expose one to a program:     oliver device attach <prefix> <device>"))
	return nil
}

func (a *app) deviceAttach(prefixName, devicePath, letter string, force bool) error {
	scanner := newScanner()
	dev, err := scanner.Find(devicePath)
	if err != nil {
		return err
	}

	u := a.ui

	// The guard that matters: handing a disk utility the disk this system is
	// running from invites it to reformat the machine out from under itself.
	if dev.SystemCritical() {
		return fmt.Errorf(
			"refusing to attach %s: it holds the running system (%s).\n"+
				"A Windows disk utility given this drive can repartition or erase your installation.\n"+
				"If you genuinely need this, do it from a live USB where nothing is mounted",
			dev.Path, strings.Join(dev.CriticalMounts(), ", "))
	}

	p := a.manager().For("", prefixName)
	if !p.Exists() {
		return fmt.Errorf("prefix %q does not exist yet; run the program once first (`oliver prefix list` shows them)", prefixName)
	}

	if letter == "" {
		letter, err = p.FreeLetter()
		if err != nil {
			return err
		}
	}

	u.Printf("%s", u.Bold("About to attach:"))
	u.Printf("  %s", dev.Describe())
	if len(dev.Mounts) > 0 {
		u.Printf("  %s", u.Yellow(fmt.Sprintf("%d filesystem(s) from this device are currently mounted", len(dev.Mounts))))
		for _, m := range dev.Mounts {
			u.Printf("    %s", u.Dim(m.Source+" → "+m.Target))
		}
		u.Printf("  %s", u.Yellow("Unmount them before writing to the drive, or the write will fail or corrupt data."))
	}
	u.Printf("  %s %s:", u.Dim("as drive"), strings.ToUpper(letter))
	u.Printf("  %s %s", u.Dim("in prefix"), p.Dir)

	if !dev.Removable {
		u.Printf("")
		u.Printf("%s %s", u.Yellow("This is a fixed disk, not removable media."), "Check it is the drive you mean.")
	}

	if !force && !u.Confirm("Attach this device?", false) {
		u.Info("cancelled")
		return nil
	}

	mount := ""
	if len(dev.Mounts) == 1 {
		mount = dev.Mounts[0].Target
	}
	if err := p.MapDevice(letter, dev.Path, mount); err != nil {
		return err
	}
	u.Info("%s attached %s as %s:", u.Green("✓"), dev.Path, strings.ToUpper(letter))

	if advice := device.AccessAdvice(dev); advice != "" {
		u.Printf("")
		for _, line := range wrap(advice, 76) {
			u.Printf("%s", u.Yellow(line))
		}
	}
	return nil
}

func (a *app) deviceDetach(prefixName, letter string) error {
	p := a.manager().For("", prefixName)
	if err := p.UnmapDevice(letter); err != nil {
		return err
	}
	a.ui.Info("detached %s: from %s", strings.ToUpper(strings.TrimSuffix(letter, ":")), p.Name)
	return nil
}

// printPrefixDevices lists a prefix's attached drives, for `oliver prefix list`.
func (a *app) printPrefixDevices(p *prefix.Prefix) {
	mapped, err := p.Devices()
	if err != nil || len(mapped) == 0 {
		return
	}
	for _, m := range mapped {
		a.ui.Printf("    %s %s → %s", a.ui.Dim("drive"), strings.ToUpper(m.Letter)+":", m.Device)
	}
}
