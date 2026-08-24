package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/robertpjenkins-bot/oliver/internal/runner"
)

const logSynopsis = `oliver log [name]

Shows the output of the last program Oliver ran.

A program started from a file manager has no terminal, so when it fails there
is nothing on screen to act on. Every run is recorded; this prints the record.`

func (a *app) cmdLog(args []string) error {
	fs := a.newFlagSet("log", logSynopsis)
	list := fs.Bool("list", false, "list recorded runs instead of printing one")
	tail := fs.Int("n", 0, "print only the last n lines")
	problems := fs.Bool("problems", false, "print only the lines that look like errors")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	if *list {
		logs, err := runner.Logs()
		if err != nil {
			return err
		}
		if len(logs) == 0 {
			a.ui.Printf("No runs have been logged yet.")
			return nil
		}
		for _, l := range logs {
			name := filepath.Base(l)
			info, err := os.Stat(l)
			if err != nil {
				continue
			}
			a.ui.Printf("%s  %s  %s", info.ModTime().Format("2006-01-02 15:04"),
				humanBytes(info.Size()), name[:len(name)-len(".log")])
		}
		return nil
	}

	path := ""
	if fs.NArg() > 0 {
		path = runner.LogPathFor(fs.Arg(0))
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("no log for %q (try `oliver log -list`)", fs.Arg(0))
		}
	} else {
		var err error
		if path, err = runner.LatestLog(); err != nil {
			return err
		}
	}

	if *problems {
		lines := runner.Diagnose(path, 40)
		if len(lines) == 0 {
			a.ui.Printf("%s", a.ui.Dim("nothing in "+filepath.Base(path)+" looks like an error"))
			return nil
		}
		for _, l := range lines {
			a.ui.Printf("%s", l)
		}
		return nil
	}

	if *tail > 0 {
		for _, l := range runner.Tail(path, *tail) {
			a.ui.Printf("%s", l)
		}
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(a.ui.Out, f)
	return err
}
