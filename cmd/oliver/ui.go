package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/webui"
)

const uiSynopsis = `oliver ui

Opens Oliver's drives in a browser.

  oliver ui                       start it and open a browser
  oliver ui -no-open              start it and print the address
  oliver ui -port 8731            ask for a particular port
  oliver ui -save drives.html     write the page out as a file and stop

The page is served from this program on the loopback interface only, and does
nothing by itself: every drive is read, written and verified here, because no
page in a browser can open a raw device. The rules about which drives may be
touched are the same ones the commands use.

The page is one self-contained HTML file with nothing fetched from anywhere
else, so it works on a machine with no network. Saved with -save it can be
opened on its own — from a download, or off the stick it is sitting on — and
asks for the address of a running Oliver.

The address carries a one-time token. Anything reaching the port without it is
refused, so another program on this machine — or a web page you happen to have
open — cannot drive it. Closing Oliver ends the session; a write already under
way finishes first.`

func (a *app) cmdUI(args []string) error {
	fs := a.newFlagSet("ui", uiSynopsis)
	port := fs.Int("port", 0, "port to listen on (default: any free one)")
	noOpen := fs.Bool("no-open", false, "do not open a browser")
	save := fs.String("save", "", "write the page to this file and stop")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}

	if *save != "" {
		if err := os.WriteFile(*save, []byte(webui.Page()), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", *save, err)
		}
		a.ui.Info("%s wrote %s", a.ui.Green("✓"), *save)
		a.ui.Info("%s", a.ui.Dim("Open it in a browser. It will ask for the address `oliver ui` prints."))
		return nil
	}

	session, err := serveDrives(*port)
	if err != nil {
		return err
	}
	defer session.stop()

	url := session.url
	u := a.ui
	u.Info("%s %s", u.Bold("Oliver is serving its drives at"), u.Cyan(url))
	u.Info("%s", u.Dim("The link carries a one-time token; anything without it is refused."))
	u.Info("%s", u.Dim("Press Ctrl-C to stop."))

	ctx, stop := interruptible()
	defer stop()

	if !*noOpen {
		openBrowser(url)
	}

	select {
	case err := <-session.errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		u.Info("")
		u.Info("%s", u.Dim("stopping"))
		return session.stop()
	}
}

// session is a running drives server.
type session struct {
	url    string
	server *http.Server
	errs   chan error
}

// serveDrives starts the server on loopback and returns the address to open.
//
// Loopback only is the whole security boundary: a program that writes boot
// records must not be reachable from the network, whatever else is
// misconfigured on the machine.
func serveDrives(port int) (*session, error) {
	drives := webui.New()
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, fmt.Errorf("could not start the server: %w", err)
	}
	s := &session{
		url: fmt.Sprintf("http://%s/?t=%s", listener.Addr().String(), drives.Token),
		server: &http.Server{
			Handler:           drives.Handler(),
			ReadHeaderTimeout: 10 * time.Second,
			// No write timeout: the progress stream stays open for as long as
			// a write takes, which for a five gigabyte image is minutes.
		},
		errs: make(chan error, 1),
	}
	go func() { s.errs <- s.server.Serve(listener) }()
	return s, nil
}

// stop shuts the server down, waiting for what is in flight so that a write in
// progress is not cut off half way.
func (s *session) stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// openBrowser asks the desktop to open the page. Failing to is not an error:
// the address has already been printed.
func openBrowser(url string) {
	for _, name := range []string{"xdg-open", "gio", "kde-open", "open"} {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		args := []string{url}
		if name == "gio" {
			args = []string{"open", url}
		}
		cmd := exec.Command(path, args...)
		if err := cmd.Start(); err == nil {
			go func() { _ = cmd.Wait() }()
			return
		}
	}
}
