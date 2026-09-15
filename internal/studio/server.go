// Package studio is the game maker: a local web application for designing
// retro video games, and the library behind Oliver's studio commands.
//
// It runs on the machine it is used from. There is no account, no cloud and no
// project living on someone else's disk: games are files in a folder, the
// editor is served from this process, and when the window is closed everything
// that was made is still there.
package studio

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/gamekit"
)

// Extension is what a project file is called. The double extension keeps games
// recognisable in a folder of other JSON.
const Extension = ".game.json"

// Server hosts the studio.
type Server struct {
	Dir       string
	Assistant *gamekit.Assistant
	// Log receives one line per interesting event, for the terminal the studio
	// was started from.
	Log func(format string, args ...any)
}

// DefaultDir is where games are kept: alongside everything else Oliver owns,
// so backing up one directory backs up the lot.
func DefaultDir() string {
	if d := os.Getenv("OLIVER_GAMES_DIR"); d != "" {
		return d
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "oliver", "games")
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "oliver", "games")
}

// New builds a studio over a directory of games.
func New(dir string) *Server {
	if dir == "" {
		dir = DefaultDir()
	}
	return &Server{
		Dir:       dir,
		Assistant: gamekit.NewAssistant(),
		Log:       func(string, ...any) {},
	}
}

// Handler returns the studio's HTTP routes.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.page)
	mux.HandleFunc("/studio.css", s.asset("studio.css", "text/css; charset=utf-8"))
	mux.HandleFunc("/studio.js", s.asset("studio.js", "text/javascript; charset=utf-8"))
	mux.HandleFunc("/engine.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		_, _ = w.Write([]byte(gamekit.EngineJS()))
	})
	mux.HandleFunc("/api/status", s.guard(s.status))
	mux.HandleFunc("/api/games", s.guard(s.games))
	mux.HandleFunc("/api/game", s.guard(s.game))
	mux.HandleFunc("/api/design", s.guard(s.design))
	mux.HandleFunc("/api/assist", s.guard(s.assist))
	mux.HandleFunc("/api/export", s.guard(s.export))
	mux.HandleFunc("/play/", s.play)
	return mux
}

// guard keeps other pages out.
//
// The studio listens on the loopback interface, but any page in any browser on
// this machine can still send it a request. Requiring a header that a
// cross-origin form cannot set — and that a cross-origin fetch may only set
// after a preflight this server never approves — is what stops a web page
// somewhere rewriting someone's games.
func (s *Server) guard(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Oliver-Studio") != "1" {
			http.Error(w, "this endpoint is for the studio itself", http.StatusForbidden)
			return
		}
		h(w, r)
	}
}

func (s *Server) page(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(studioHTML))
}

func (s *Server) asset(name, contentType string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		data, err := webAssets.ReadFile("web/" + name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write(data)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, format string, args ...any) {
	writeJSON(w, code, map[string]string{"error": fmt.Sprintf(format, args...)})
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) {
	assistant := "offline designer"
	if s.Assistant.Available() {
		assistant = s.Assistant.Model
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"dir":       s.Dir,
		"assistant": assistant,
		"online":    s.Assistant.Available(),
		"genres":    gamekit.Genres,
	})
}

// GameInfo is one row of the studio's game list.
type GameInfo struct {
	Name     string    `json:"name"`
	Title    string    `json:"title"`
	Genre    string    `json:"genre"`
	Levels   int       `json:"levels"`
	Modified time.Time `json:"modified"`
}

// List returns every game in the studio's directory, newest first.
func (s *Server) List() ([]GameInfo, error) {
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []GameInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), Extension) {
			continue
		}
		name := strings.TrimSuffix(e.Name(), Extension)
		info := GameInfo{Name: name, Title: name}
		if st, err := e.Info(); err == nil {
			info.Modified = st.ModTime()
		}
		if g, err := gamekit.Load(filepath.Join(s.Dir, e.Name())); err == nil {
			info.Title = g.Name
			info.Genre = g.Genre
			info.Levels = len(g.Levels)
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Modified.After(out[j].Modified) })
	return out, nil
}

func (s *Server) games(w http.ResponseWriter, r *http.Request) {
	list, err := s.List()
	if err != nil {
		fail(w, http.StatusInternalServerError, "%v", err)
		return
	}
	if list == nil {
		list = []GameInfo{}
	}
	writeJSON(w, http.StatusOK, list)
}

// Path returns the file a game is stored in. Names are restricted to what can
// safely be a file name, so a name from the browser can never reach outside
// the studio's own directory.
func (s *Server) Path(name string) (string, error) {
	clean := safeName(name)
	if clean == "" {
		return "", fmt.Errorf("%q is not a usable game name", name)
	}
	return filepath.Join(s.Dir, clean+Extension), nil
}

func safeName(name string) string {
	name = strings.TrimSuffix(strings.TrimSpace(name), Extension)
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == ' ':
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// Load reads one game by name.
func (s *Server) Load(name string) (*gamekit.Game, error) {
	path, err := s.Path(name)
	if err != nil {
		return nil, err
	}
	return gamekit.Load(path)
}

// Save writes a game, returning the name it was stored under.
func (s *Server) Save(name string, g *gamekit.Game) (string, error) {
	if name == "" {
		name = g.Slug()
	}
	path, err := s.Path(name)
	if err != nil {
		return "", err
	}
	if err := g.Save(path); err != nil {
		return "", err
	}
	return strings.TrimSuffix(filepath.Base(path), Extension), nil
}

func (s *Server) game(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	switch r.Method {
	case http.MethodGet:
		g, err := s.Load(name)
		if err != nil {
			fail(w, http.StatusNotFound, "%v", err)
			return
		}
		writeJSON(w, http.StatusOK, g)
	case http.MethodPut, http.MethodPost:
		var g gamekit.Game
		if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
			fail(w, http.StatusBadRequest, "this is not a game: %v", err)
			return
		}
		g.Normalize()
		saved, err := s.Save(name, &g)
		if err != nil {
			fail(w, http.StatusInternalServerError, "%v", err)
			return
		}
		s.Log("saved %s", saved)
		writeJSON(w, http.StatusOK, map[string]any{
			"name":     saved,
			"problems": g.Validate(),
		})
	case http.MethodDelete:
		path, err := s.Path(name)
		if err != nil {
			fail(w, http.StatusBadRequest, "%v", err)
			return
		}
		if err := os.Remove(path); err != nil {
			fail(w, http.StatusInternalServerError, "%v", err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"deleted": name})
	default:
		fail(w, http.StatusMethodNotAllowed, "%s is not allowed here", r.Method)
	}
}

func (s *Server) design(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Prompt string `json:"prompt"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	s.Log("designing: %s", req.Prompt)
	g, note, err := s.Assistant.Create(ctx, req.Prompt)
	if err != nil {
		fail(w, http.StatusInternalServerError, "%v", err)
		return
	}
	name, err := s.Save("", g)
	if err != nil {
		fail(w, http.StatusInternalServerError, "%v", err)
		return
	}
	s.Log("made %s (%s)", g.Name, note)
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "game": g, "note": note})
}

func (s *Server) assist(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string        `json:"name"`
		Instruction string        `json:"instruction"`
		Game        *gamekit.Game `json:"game"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	g := req.Game
	if g == nil {
		loaded, err := s.Load(req.Name)
		if err != nil {
			fail(w, http.StatusNotFound, "%v", err)
			return
		}
		g = loaded
	}
	g.Normalize()

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()
	edited, note, err := s.Assistant.Edit(ctx, g, req.Instruction)
	if err != nil {
		fail(w, http.StatusInternalServerError, "%v", err)
		return
	}
	name, err := s.Save(req.Name, edited)
	if err != nil {
		fail(w, http.StatusInternalServerError, "%v", err)
		return
	}
	s.Log("%s: %s", name, note)
	writeJSON(w, http.StatusOK, map[string]any{
		"name":     name,
		"game":     edited,
		"note":     note,
		"problems": edited.Validate(),
	})
}

func (s *Server) export(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, http.StatusBadRequest, "%v", err)
		return
	}
	g, err := s.Load(req.Name)
	if err != nil {
		fail(w, http.StatusNotFound, "%v", err)
		return
	}
	out := req.Path
	if out == "" {
		out = filepath.Join(s.Dir, "exports", g.Slug()+".html")
	}
	written, err := g.WriteHTML(out)
	if err != nil {
		fail(w, http.StatusInternalServerError, "%v", err)
		return
	}
	s.Log("exported %s", written)
	writeJSON(w, http.StatusOK, map[string]string{"path": written})
}

// play serves a game as a finished page, which is also exactly what an export
// produces: the play-test window and the exported file are the same thing, so
// a game cannot behave one way in the studio and another once it is shared.
func (s *Server) play(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/play/")
	g, err := s.Load(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	page, err := g.ExportHTML()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(page)
}

// Serve starts the studio and blocks until the context is cancelled. It
// returns the address it is listening on through ready, so a caller can open a
// browser at it.
func (s *Server) Serve(ctx context.Context, addr string, ready func(url string)) error {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return fmt.Errorf("creating the games folder: %w", err)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("the studio could not start listening on %s: %w", addr, err)
	}
	srv := &http.Server{
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	if ready != nil {
		ready("http://" + ln.Addr().String() + "/")
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// OpenBrowser asks the desktop to open a URL. A studio that starts and then
// tells someone to paste an address has already lost them.
func OpenBrowser(url string) error {
	for _, candidate := range [][]string{
		{"xdg-open", url},
		{"gio", "open", url},
		{"x-www-browser", url},
		{"firefox", url},
		{"chromium", url},
		{"google-chrome", url},
	} {
		if _, err := exec.LookPath(candidate[0]); err != nil {
			continue
		}
		cmd := exec.Command(candidate[0], candidate[1:]...)
		cmd.Stdout, cmd.Stderr = nil, nil
		if err := cmd.Start(); err == nil {
			go func() { _ = cmd.Wait() }()
			return nil
		}
	}
	return fmt.Errorf("no browser could be opened; visit %s", url)
}
