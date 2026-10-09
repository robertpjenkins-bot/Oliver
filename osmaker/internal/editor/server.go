// Package editor serves the OS Maker designer: a web page drawn in the
// person's browser, backed by a small local server that saves designs and
// runs ISO builds.
package editor

import (
	"bytes"
	"context"
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/robertpjenkins-bot/oliver/osmaker/internal/archiso"
	"github.com/robertpjenkins-bot/oliver/osmaker/internal/builder"
	"github.com/robertpjenkins-bot/oliver/osmaker/internal/design"
)

//go:embed web
var webFS embed.FS

// Server is one running designer.
type Server struct {
	token    string
	addr     string
	dataDir  string
	openPath string // design file passed on the command line, if any

	mu       sync.Mutex
	lastSeen time.Time
	job      *job
}

type job struct {
	mu      sync.Mutex
	log     bytes.Buffer
	running bool
	ok      bool
	err     string
	iso     string
	dir     string
	cancel  context.CancelFunc
}

func (j *job) Write(p []byte) (int, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	// Keep the tail only: a full build prints tens of thousands of lines.
	const max = 2 << 20
	if j.log.Len()+len(p) > max {
		rest := j.log.Bytes()[j.log.Len()/2:]
		j.log = *bytes.NewBuffer(append([]byte("…\n"), rest...))
	}
	return j.log.Write(p)
}

// New prepares a designer that keeps its autosave in dataDir.
func New(dataDir, openPath string) (*Server, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, err
	}
	return &Server{token: hex.EncodeToString(b), dataDir: dataDir, openPath: openPath, lastSeen: time.Now()}, nil
}

// Listen binds to a local port and returns the URL to open.
func (s *Server) Listen(port int) (net.Listener, string, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, "", err
	}
	s.addr = ln.Addr().String()
	return ln, fmt.Sprintf("http://%s/#%s", s.addr, s.token), nil
}

// Serve runs until ctx ends or, when idleExit is set, until the page has
// been closed for a while and no build is running.
func (s *Server) Serve(ctx context.Context, ln net.Listener, idleExit bool) error {
	srv := &http.Server{Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	if idleExit {
		go func() {
			for range time.Tick(5 * time.Second) {
				s.mu.Lock()
				idle := time.Since(s.lastSeen) > 45*time.Second
				busy := s.job != nil && s.job.isRunning()
				s.mu.Unlock()
				if idle && !busy {
					stop()
					return
				}
			}
		}()
	}
	err := srv.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (j *job) isRunning() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.running
}

// Handler is the designer's HTTP interface.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	web, _ := fs.Sub(webFS, "web")
	mux.Handle("GET /", http.FileServerFS(web))

	api := func(pattern string, h func(w http.ResponseWriter, r *http.Request) (any, error)) {
		mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
			if !s.authorized(r) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			s.mu.Lock()
			s.lastSeen = time.Now()
			s.mu.Unlock()
			v, err := h(w, r)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				v = map[string]string{"error": err.Error()}
			}
			json.NewEncoder(w).Encode(v)
		})
	}

	api("GET /api/info", s.info)
	api("GET /api/design", s.loadDesign)
	api("PUT /api/autosave", s.autosave)
	api("POST /api/validate", func(w http.ResponseWriter, r *http.Request) (any, error) {
		_, err := readDesign(r)
		return map[string]bool{"ok": true}, err
	})
	api("POST /api/password", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var req struct{ Password string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, err
		}
		if req.Password == "" {
			return map[string]string{"hash": ""}, nil
		}
		return map[string]string{"hash": design.HashPassword(req.Password)}, nil
	})
	api("POST /api/heartbeat", func(w http.ResponseWriter, r *http.Request) (any, error) {
		return map[string]bool{"ok": true}, nil
	})
	api("POST /api/build", s.startBuild)
	api("GET /api/build", s.buildStatus)
	api("POST /api/build/cancel", func(w http.ResponseWriter, r *http.Request) (any, error) {
		s.mu.Lock()
		j := s.job
		s.mu.Unlock()
		if j != nil && j.cancel != nil {
			j.cancel()
		}
		return map[string]bool{"ok": true}, nil
	})
	api("POST /api/open-folder", func(w http.ResponseWriter, r *http.Request) (any, error) {
		var req struct{ Path string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return nil, err
		}
		if fi, err := os.Stat(req.Path); err != nil || !fi.IsDir() {
			return nil, fmt.Errorf("%s is not a folder", req.Path)
		}
		return map[string]bool{"ok": true}, exec.Command("xdg-open", req.Path).Start()
	})
	return mux
}

// authorized rejects requests from other web pages: the token is only known
// to the page this server opened, and the Host check stops DNS rebinding.
func (s *Server) authorized(r *http.Request) bool {
	return r.Host == s.addr && r.Header.Get("X-OSMaker-Token") == s.token
}

func readDesign(r *http.Request) (*design.Design, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 32<<20)
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		return nil, err
	}
	return design.Parse(buf.Bytes())
}

func (s *Server) autosavePath() string { return filepath.Join(s.dataDir, "autosave"+design.FileExt) }

func (s *Server) info(w http.ResponseWriter, r *http.Request) (any, error) {
	home, _ := os.UserHomeDir()
	return map[string]any{
		"catalog":    design.Catalog,
		"categories": design.Categories,
		"keyboards":  design.Keyboards,
		"itemTypes":  design.ItemTypes,
		"canvas":     map[string]int{"w": design.CanvasW, "h": design.CanvasH},
		"methods":    builder.Detect(),
		"outDir":     filepath.Join(home, "OS Maker"),
		"default":    design.Default(),
	}, nil
}

// loadDesign returns the design to start with: a file named on the command
// line, else the autosave, else a fresh one.
func (s *Server) loadDesign(w http.ResponseWriter, r *http.Request) (any, error) {
	for _, p := range []string{s.openPath, s.autosavePath()} {
		if p == "" {
			continue
		}
		if d, err := design.Load(p); err == nil {
			return d, nil
		}
	}
	return design.Default(), nil
}

func (s *Server) autosave(w http.ResponseWriter, r *http.Request) (any, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 32<<20)
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r.Body); err != nil {
		return nil, err
	}
	// Save even a half-finished design; only building needs a valid one.
	var probe design.Design
	if err := json.Unmarshal(buf.Bytes(), &probe); err != nil {
		return nil, err
	}
	tmp := s.autosavePath() + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return nil, err
	}
	return map[string]bool{"ok": true}, os.Rename(tmp, s.autosavePath())
}

func (s *Server) startBuild(w http.ResponseWriter, r *http.Request) (any, error) {
	var req struct {
		Design     json.RawMessage `json:"design"`
		OutDir     string          `json:"outDir"`
		Method     string          `json:"method"`
		FolderOnly bool            `json:"folderOnly"`
	}
	r.Body = http.MaxBytesReader(nil, r.Body, 32<<20)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	d, err := design.Parse(req.Design)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.OutDir) == "" || !filepath.IsAbs(req.OutDir) {
		return nil, errors.New("choose a full folder path to build in, like /home/you/OS Maker")
	}
	dir := filepath.Join(req.OutDir, d.System.Hostname)

	var method builder.Method
	if !req.FolderOnly {
		for _, m := range builder.Detect() {
			if m.Kind == req.Method {
				method = m
			}
		}
		if method.Kind == "" {
			return nil, errors.New("this computer cannot build ISOs yet: install Docker or Podman, or use Arch Linux with archiso")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.job != nil && s.job.isRunning() {
		return nil, errors.New("a build is already running")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	if err := archiso.Write(d, dir); err != nil {
		return nil, err
	}
	j := &job{dir: dir, running: !req.FolderOnly}
	fmt.Fprintf(j, "Build folder written to %s\n", dir)
	s.job = j
	if req.FolderOnly {
		j.ok = true
		return map[string]string{"dir": dir}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	j.cancel = cancel
	go func() {
		defer cancel()
		iso, err := builder.Run(ctx, method, dir, j)
		j.mu.Lock()
		defer j.mu.Unlock()
		j.running = false
		if err != nil {
			j.err = err.Error()
			return
		}
		j.ok, j.iso = true, iso
	}()
	return map[string]string{"dir": dir}, nil
}

func (s *Server) buildStatus(w http.ResponseWriter, r *http.Request) (any, error) {
	s.mu.Lock()
	j := s.job
	s.mu.Unlock()
	if j == nil {
		return map[string]any{"running": false}, nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	return map[string]any{
		"running": j.running,
		"ok":      j.ok,
		"error":   j.err,
		"iso":     j.iso,
		"dir":     j.dir,
		"log":     j.log.String(),
	}, nil
}
