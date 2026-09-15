package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/gamekit"
	"github.com/robertpjenkins-bot/oliver/internal/studio"
)

const playSynopsis = `oliver play <game.html | game.game.json | name>

Plays a game made with Oliver Studio.

An exported game is a single HTML file that a browser can open on its own, so
this usually just opens it. --serve hosts it on this machine instead, which is
what to use when a browser refuses to run a local file, or to play it from
another device on the same network.`

func (a *app) cmdPlay(args []string) error {
	fs := a.newFlagSet("play", playSynopsis)
	serve := fs.Bool("serve", false, "host the game on this machine instead of opening the file")
	addr := fs.String("addr", "127.0.0.1:0", "with --serve: address to listen on")
	noOpen := fs.Bool("no-open", false, "do not open a browser")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return fmt.Errorf("say what to play")
	}

	target := fs.Arg(0)
	path, err := playablePath(target)
	if err != nil {
		return err
	}
	if !*serve {
		if *noOpen {
			a.ui.Printf("%s", path)
			return nil
		}
		return openPlayable(a, path)
	}
	return a.servePlayable(path, *addr, *noOpen)
}

// playablePath resolves what was asked for to an HTML file on disk: an
// exported game is used as it is, a project is exported first, and a bare name
// is looked up among the studio's games.
func playablePath(target string) (string, error) {
	if st, err := os.Stat(target); err == nil && !st.IsDir() {
		if strings.HasSuffix(strings.ToLower(target), ".html") {
			return target, nil
		}
		g, err := gamekit.Load(target)
		if err != nil {
			return "", err
		}
		return g.WriteHTML(filepath.Join(filepath.Dir(target), g.Slug()+".html"))
	}
	s := studio.New("")
	g, err := s.Load(target)
	if err != nil {
		return "", fmt.Errorf("no game called %q — list them with: oliver studio list", target)
	}
	return g.WriteHTML(filepath.Join(s.Dir, "played", g.Slug()+".html"))
}

func (a *app) servePlayable(path, addr string, noOpen bool) error {
	page, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(page)
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	url := "http://" + ln.Addr().String() + "/"
	a.ui.Printf("%s", url)
	if !noOpen {
		if err := studio.OpenBrowser(url); err != nil {
			a.ui.Warn("%v", err)
		}
	}
	a.ui.Info("%s", a.ui.Dim("press Ctrl-C to stop"))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
