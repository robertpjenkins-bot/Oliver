package main

// Etros Mac Installer as a page in Safari (macweb.html) instead of text in
// Terminal: a small web server on this Mac only (127.0.0.1, with a secret
// in every address so no other page can drive it), which lists the drives
// and partitions, takes the choice, and shows the install as it goes. The
// work itself is a macBackend (mac_darwin.go), so the page can be checked
// anywhere with a stand-in.

import (
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed macweb.html
var macWebPage string

type macBackend interface {
	targets() ([]macTarget, error)
	roomFree(part string) int64 // macOS's free room, for a "room" target
	wifiName() string           // the Wi-Fi this Mac is on
	checks() []macCheck         // what's looked at before anything is changed
	install(x macTarget, etros int64, ssid, key string, rep Reporter) error
}

// webState is how the install is going, for the page to show.
type webState struct {
	mu       sync.Mutex
	Running  bool     `json:"running"`
	Finished bool     `json:"finished"`
	Error    string   `json:"error"`
	Now      string   `json:"stage"`
	Lines    []string `json:"lines"`
	Done     int64    `json:"done"`
	Total    int64    `json:"total"`
	Kind     string   `json:"kind"`
	Ollie    bool     `json:"ollie"`
}

func (w *webState) Stage(s string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	logf("STAGE %s", s)
	w.Now, w.Done, w.Total = s, 0, 0
	w.Lines = append(w.Lines, "▸ "+s)
}

func (w *webState) Log(s string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	logf("  %s", s)
	w.Lines = append(w.Lines, s)
	if len(w.Lines) > 400 {
		w.Lines = w.Lines[len(w.Lines)-400:]
	}
}

func (w *webState) Progress(done, total int64) {
	w.mu.Lock()
	w.Done, w.Total = done, total
	w.mu.Unlock()
}

type webTarget struct {
	N        int    `json:"n"`
	Disk     string `json:"disk"`
	DiskName string `json:"diskName"`
	Internal bool   `json:"internal"`
	DiskSize int64  `json:"diskSize"`
	ID       string `json:"id"`
	What     string `json:"what"`
	Size     int64  `json:"size"`
	Action   string `json:"action"`
	Why      string `json:"why"`
	Kind     string `json:"kind"`
	Free     int64  `json:"free,omitempty"`
}

const (
	macMinGB   = (macBootSize + macMinShare) >> 30
	macOllieGB = (macBootSize + macMinShare + macOllieRoom) >> 30
)

// macWebHandler serves the page and its API under /SECRET/; busy tells
// whether Etros is being put on a drive.
func macWebHandler(b macBackend, secret string, quit func()) (h http.Handler, busy func() bool) {
	state := &webState{}
	var targets []macTarget
	mux := http.NewServeMux()
	base := "/" + secret + "/"
	reply := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(v)
	}
	fail := func(w http.ResponseWriter, code int, msg string) {
		w.WriteHeader(code)
		reply(w, map[string]string{"error": msg})
	}
	mux.HandleFunc(base, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != base {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		page := strings.ReplaceAll(macWebPage, "{{VERSION}}", version)
		page = strings.ReplaceAll(page, "{{MIN_GB}}", fmt.Sprint(macMinGB))
		page = strings.ReplaceAll(page, "{{OLLIE_GB}}", fmt.Sprint(macOllieGB))
		fmt.Fprint(w, page)
	})
	mux.HandleFunc(base+"drives", func(w http.ResponseWriter, r *http.Request) {
		ts, err := b.targets()
		if err != nil {
			fail(w, 500, err.Error())
			return
		}
		state.mu.Lock()
		if !state.Running {
			targets = ts
		}
		state.mu.Unlock()
		out := []webTarget{}
		for i, x := range ts {
			t := webTarget{N: i, Disk: x.disk.id, DiskName: x.disk.name, Internal: x.disk.internal, DiskSize: x.disk.size,
				ID: x.part.id, What: x.what, Size: x.part.size, Action: x.action, Why: x.why, Kind: x.kind}
			if x.kind == "whole" {
				t.ID, t.Size = x.disk.id, x.disk.size
			}
			for _, e := range x.extra {
				t.Size += e.size // BATOCERA with its SHARE
			}
			if x.kind == "room" && x.why == "" {
				t.Free = b.roomFree(x.part.id)
			}
			out = append(out, t)
		}
		reply(w, map[string]any{"targets": out, "wifi": b.wifiName()})
	})
	mux.HandleFunc(base+"checks", func(w http.ResponseWriter, r *http.Request) {
		reply(w, b.checks())
	})
	mux.HandleFunc(base+"install", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			fail(w, 405, "POST only")
			return
		}
		var req struct {
			N       int    `json:"n"`
			GB      int64  `json:"gb"`
			SSID    string `json:"ssid"`
			Key     string `json:"key"`
			Confirm string `json:"confirm"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			fail(w, 400, "bad request")
			return
		}
		if strings.ToUpper(strings.TrimSpace(req.Confirm)) != "ERASE" {
			fail(w, 400, "Type ERASE to go ahead.")
			return
		}
		state.mu.Lock()
		defer state.mu.Unlock()
		if state.Running {
			fail(w, 409, "Etros is being put on already.")
			return
		}
		if req.N < 0 || req.N >= len(targets) || targets[req.N].why != "" {
			fail(w, 400, "That can't be used for Etros.")
			return
		}
		x := targets[req.N]
		var etros int64
		switch x.kind {
		case "whole":
			etros = x.disk.size
		case "part", "ready":
			etros = x.part.size
			for _, e := range x.extra {
				etros += e.size
			}
		case "room":
			max := b.roomFree(x.part.id) - 20<<30 // leave macOS some room
			etros = req.GB << 30
			if etros < macBootSize+macMinShare || etros > max {
				fail(w, 400, fmt.Sprintf("Choose between %d and %d GB.", macMinGB, max>>30))
				return
			}
		}
		// (field by field: the lock is held)
		state.Running, state.Finished, state.Error, state.Now, state.Lines = true, false, "", "", nil
		state.Done, state.Total = 0, 0
		state.Kind, state.Ollie = x.kind, etros >= macBootSize+macMinShare+macOllieRoom
		go func() {
			err := b.install(x, etros, req.SSID, req.Key, state)
			state.mu.Lock()
			defer state.mu.Unlock()
			state.Running, state.Finished = false, true
			if err != nil {
				logf("ERROR: Mac: %v", err)
				state.Error = err.Error()
			}
		}()
		reply(w, map[string]bool{"ok": true})
	})
	mux.HandleFunc(base+"status", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		defer state.mu.Unlock()
		reply(w, state)
	})
	mux.HandleFunc(base+"quit", func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		running := state.Running
		state.mu.Unlock()
		if r.Method != http.MethodPost || running {
			fail(w, 409, "not now")
			return
		}
		reply(w, map[string]bool{"ok": true})
		go func() { time.Sleep(300 * time.Millisecond); quit() }()
	})
	return mux, func() bool {
		state.mu.Lock()
		defer state.mu.Unlock()
		return state.Running
	}
}

// macIdleQuit is how long the installer waits, with its page closed and
// nothing being put on a drive, before it ends: the app has no window, and
// macOS only opens it again once it has ended.
var macIdleQuit = 2 * time.Minute

// macPageFile is where the running installer says where its page is.
func macPageFile() string { return filepath.Join(defaultCache(), "mac-installer-page.txt") }

// serveMacWeb runs the page until it's closed with its Quit button, and
// opens it with open (in Safari).
func serveMacWeb(b macBackend, open func(url string) error) error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	sec := make([]byte, 16)
	rand.Read(sec)
	secret := hex.EncodeToString(sec)
	done := make(chan struct{})
	var once sync.Once
	quit := func() { once.Do(func() { close(done) }) }
	h, busy := macWebHandler(b, secret, quit)
	var mu sync.Mutex
	last := time.Now()
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		last = time.Now()
		mu.Unlock()
		h.ServeHTTP(w, r)
	})}
	go srv.Serve(ln)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-time.After(macIdleQuit / 8):
			}
			mu.Lock()
			idle := time.Since(last)
			mu.Unlock()
			if idle > macIdleQuit && !busy() {
				logf("the page has been closed for %v: ending", idle.Round(time.Second))
				quit()
				return
			}
		}
	}()
	url := fmt.Sprintf("http://%s/%s/", ln.Addr(), secret)
	logf("Mac installer page: %s", url)
	// For the app opened again while this is running (the app's launcher
	// script opens this page instead of starting a second installer).
	pageFile := macPageFile()
	os.MkdirAll(filepath.Dir(pageFile), 0o755)
	os.WriteFile(pageFile, []byte(fmt.Sprintf("%d %s\n", os.Getpid(), url)), 0o600)
	defer os.Remove(pageFile)
	if err := open(url); err != nil {
		return errors.Join(fmt.Errorf("couldn't open Safari: %w", err), srv.Close())
	}
	<-done
	return srv.Close()
}
