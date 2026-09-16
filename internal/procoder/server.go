package procoder

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
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
}

// Handler returns the routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.serveApp)
	mux.HandleFunc("/api/capabilities", s.serveCapabilities)
	mux.HandleFunc("/api/convert", s.serveConvert)
	return localOnly(mux)
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
