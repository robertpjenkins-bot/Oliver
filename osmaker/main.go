// Command osmaker is OS Maker 2.0: design a desktop by dragging things onto
// a screen, pick the apps it comes with, and build it as a bootable Arch
// Linux ISO.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/robertpjenkins-bot/oliver/osmaker/internal/archiso"
	"github.com/robertpjenkins-bot/oliver/osmaker/internal/builder"
	"github.com/robertpjenkins-bot/oliver/osmaker/internal/design"
	"github.com/robertpjenkins-bot/oliver/osmaker/internal/editor"
)

var version = "dev"

const usage = `OS Maker 2.0 - design your own operating system and build it as an ISO.

usage:
  osmaker [design.osmaker]          open the designer (in your web browser)
  osmaker build <design> [-o dir]   build an ISO from a saved design
  osmaker folder <design> [-o dir]  only write the build folder (build.sh and files)
  osmaker version

designer options:
  --port N        use this port instead of a random one
  --no-browser    print the address instead of opening a browser
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "osmaker:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "build", "folder":
			return buildCmd(args[0] == "folder", args[1:])
		case "version", "--version":
			fmt.Println("OS Maker", version)
			return nil
		case "help", "-h", "--help":
			fmt.Print(usage)
			return nil
		}
	}
	return editCmd(args)
}

func editCmd(args []string) error {
	fs := flag.NewFlagSet("osmaker", flag.ContinueOnError)
	port := fs.Int("port", 0, "")
	noBrowser := fs.Bool("no-browser", false, "")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	if err := fs.Parse(args); err != nil {
		return err
	}
	open := fs.Arg(0)
	if open != "" {
		if _, err := design.Load(open); err != nil {
			return fmt.Errorf("%s: %w", open, err)
		}
	}

	srv, err := editor.New(dataDir(), open)
	if err != nil {
		return err
	}
	ln, url, err := srv.Listen(*port)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("OS Maker %s is running at\n  %s\nClose the page (or press Ctrl+C) to quit.\n", version, url)
	idleExit := false
	if !*noBrowser {
		if err := openBrowser(url); err != nil {
			fmt.Fprintln(os.Stderr, "could not open a browser:", err, "- open the address above yourself")
		} else {
			// Started from a menu there is no terminal to Ctrl+C, so quit
			// once the page has gone away.
			idleExit = true
		}
	}
	return srv.Serve(ctx, ln, idleExit)
}

func dataDir() string {
	if d := os.Getenv("XDG_DATA_HOME"); d != "" {
		return filepath.Join(d, "osmaker")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "osmaker")
}

// openBrowser prefers a browser's app mode, which gives the designer a window
// of its own without tabs and an address bar.
func openBrowser(url string) error {
	for _, b := range []string{"chromium", "chromium-browser", "google-chrome-stable", "google-chrome", "brave-browser", "brave", "microsoft-edge-stable", "vivaldi-stable"} {
		if p, err := exec.LookPath(b); err == nil {
			return detach(exec.Command(p, "--app="+url, "--new-window"))
		}
	}
	for _, b := range []string{"xdg-open", "firefox", "sensible-browser"} {
		if p, err := exec.LookPath(b); err == nil {
			return detach(exec.Command(p, url))
		}
	}
	return errors.New("no web browser found")
}

func detach(cmd *exec.Cmd) error {
	// The AppImage's own library paths must not leak into the browser.
	env := os.Environ()[:0:0]
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "LD_LIBRARY_PATH=") && !strings.HasPrefix(kv, "LD_PRELOAD=") {
			env = append(env, kv)
		}
	}
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait()
	return nil
}

func buildCmd(folderOnly bool, args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	home, _ := os.UserHomeDir()
	out := fs.String("o", filepath.Join(home, "OS Maker"), "folder to build in")
	method := fs.String("with", "", "archiso, docker or podman (default: the first available)")
	// Accept the design before or after the flags.
	var file string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		file, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if file == "" {
		file = fs.Arg(0)
	}
	if file == "" {
		return errors.New("which design? osmaker build my-os.osmaker")
	}
	d, err := design.Load(file)
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	dir := filepath.Join(*out, d.System.Hostname)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := archiso.Write(d, dir); err != nil {
		return err
	}
	fmt.Println("build folder:", dir)
	if folderOnly {
		fmt.Println("on Arch Linux, build it with:  sudo bash", filepath.Join(dir, "build.sh"))
		return nil
	}

	methods := builder.Detect()
	var m builder.Method
	for _, c := range methods {
		if *method == "" || c.Kind == *method {
			m = c
			break
		}
	}
	if m.Kind == "" {
		return errors.New("this computer cannot build ISOs: install Docker or Podman, or use Arch Linux (then: sudo bash " + filepath.Join(dir, "build.sh") + ")")
	}
	fmt.Println("building with", m.Desc)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	iso, err := builder.Run(ctx, m, dir, os.Stdout)
	if err != nil {
		return err
	}
	fmt.Println("\nyour OS is ready:", iso)
	return nil
}
