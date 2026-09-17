package procoder

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// PackRequest asks for a project to be turned into an AppImage.
type PackRequest struct {
	URL    string `json:"url"`
	Name   string `json:"name"`
	Format string `json:"format"`
	// Out is where the result should be written. The page never sets it —
	// a page has no business naming paths on the disk — so it is for the
	// command line, and an empty value means "in the working directory,
	// named after the project".
	Out string `json:"-"`
}

// PackResult is what came of it.
type PackResult struct {
	Path string `json:"path"`
	Note string `json:"note,omitempty"`
}

// Server serves the launcher to a browser on this machine.
//
// It exists only so the Convert button can reach the parts of the job a page
// cannot do for itself. A page can write a standalone HTML file — that is a
// download — but it cannot build an AppImage, because that means writing a
// directory tree and running appimagetool. So the page asks, and Oliver does
// it.
type Server struct {
	// Pack builds an AppImage. When it is nil the page is told so, and it
	// offers the command to run in a terminal instead of a button that would
	// fail.
	Pack func(PackRequest) (PackResult, error)

	// Now is the clock, so tests do not have to wait for real minutes.
	Now func() time.Time

	mu       sync.Mutex
	activity Activity
}

// Activity is what the page has told the server about itself: when it last
// spoke, and whether it said goodbye on the way out.
//
// This is how Pro Coder knows it is still wanted. The obvious alternative —
// waiting for the browser process to exit — does not work, because a browser
// that is already running does not start a new process at all: it hands the
// address to the copy of itself that is already there and exits immediately.
// Taking that as "the window was closed" shuts the server down a moment before
// the window it just asked for finishes opening, and the window then shows
// "127.0.0.1 refused to connect".
type Activity struct {
	Held  int       // windows holding a connection open right now
	Empty time.Time // when the last of them let go
	Last  time.Time // the last time any page said anything at all
}

// How patient the server is.
const (
	HoldGrace  = 5 * time.Second // after the last window let go, in case of a reload
	Silence    = 3 * time.Minute // no word at all from a page that did load
	NoContact  = 2 * time.Minute // no page ever loaded
	HeartbeatS = 5               // seconds between the page's pings
)

// ShouldStop decides whether Pro Coder has been left alone.
//
// A held connection is the reliable signal and is checked first: while one is
// open there is a window, and when the window goes the connection goes with
// it, whatever the browser thinks about running code on the way out. The
// timing rules below it are the backstop for a browser that never managed to
// hold one open — there, the page's own heartbeat has to do, and it is given
// minutes rather than seconds, because a window left in the background has its
// timers throttled to about one a minute.
func ShouldStop(now, started time.Time, a Activity) (bool, string) {
	switch {
	case a.Held > 0:
		return false, ""
	case !a.Empty.IsZero():
		if now.Sub(a.Empty) > HoldGrace {
			return true, "the window was closed"
		}
	case a.Last.IsZero():
		if now.Sub(started) > NoContact {
			return true, "nothing opened it"
		}
	case now.Sub(a.Last) > Silence:
		return true, "the window went away"
	}
	return false, ""
}

// Activity reports what the page has been up to.
func (s *Server) Activity() Activity {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.activity
}

func (s *Server) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// seen records that the page is still there. Any request counts: a reload
// arriving after a goodbye means the page came back.
func (s *Server) seen() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activity.Last = s.now()
}

// hold records a window arriving and returns the function that records it
// leaving.
func (s *Server) hold() func() {
	s.mu.Lock()
	s.activity.Held++
	s.activity.Empty = time.Time{}
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.activity.Held--
		if s.activity.Held <= 0 {
			s.activity.Held = 0
			s.activity.Empty = s.now()
		}
	}
}

// Handler returns the routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.serveApp)
	mux.HandleFunc("/api/capabilities", s.serveCapabilities)
	mux.HandleFunc("/api/convert", s.serveConvert)
	mux.HandleFunc("/api/alive", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/api/hold", s.serveHold)
	return localOnly(s.noteActivity(mux))
}

func (s *Server) noteActivity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.seen()
		next.ServeHTTP(w, r)
	})
}

// serveHold is a connection the page opens and never closes. Its only purpose
// is to end when the window does.
func (s *Server) serveHold(w http.ResponseWriter, r *http.Request) {
	release := s.hold()
	defer release()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}
	// A reconnection delay shorter than the grace period, so that a page being
	// reloaded is back before its absence is acted on.
	fmt.Fprint(w, "retry: 1000\n\n")
	flusher.Flush()

	ping := time.NewTicker(20 * time.Second)
	defer ping.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ping.C:
			// Writing is also how a connection that died without telling
			// anybody gets noticed.
			if _, err := fmt.Fprint(w, ": still here\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) serveApp(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && r.URL.Path != "/index.html" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// The launcher frames other sites, so it must not be cached into a stale
	// version between Oliver releases.
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(App())
}

func (s *Server) serveCapabilities(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"appimage": s.Pack != nil,
		"host":     "oliver",
	})
}

func (s *Server) serveConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "use POST"})
		return
	}
	var req PackRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "could not read the request"})
		return
	}
	if s.Pack == nil {
		writeJSON(w, http.StatusNotImplemented, map[string]any{"error": "this copy cannot build AppImages"})
		return
	}
	// Recognising here as well as in the page is not duplication for its own
	// sake: the page can be reloaded, edited or replaced, and the address is
	// about to be written into a launcher script.
	if _, err := Recognise(req.URL); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}
	res, err := s.Pack(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// localOnly refuses requests that did not come from this machine.
//
// The listener is already bound to the loopback address, so nothing on the
// network can connect. The Host check covers the remaining case: a page on
// some other site telling a browser on this machine to fetch a name that
// resolves to 127.0.0.1. Convert writes files, so it is worth the four lines.
func localOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		switch strings.ToLower(strings.Trim(host, "[]")) {
		case "localhost", "127.0.0.1", "::1":
			next.ServeHTTP(w, r)
		default:
			http.Error(w, "Pro Coder only answers to this machine", http.StatusForbidden)
		}
	})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

// Listen opens a loopback listener, choosing a free port when none is given.
func Listen(port int) (net.Listener, string, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, "", fmt.Errorf("opening a local address: %w", err)
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		ln.Close()
		return nil, "", fmt.Errorf("unexpected listener address %v", ln.Addr())
	}
	return ln, fmt.Sprintf("http://127.0.0.1:%d/", addr.Port), nil
}
