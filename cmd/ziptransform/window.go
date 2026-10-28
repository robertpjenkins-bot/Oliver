package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/mcreator"
	"github.com/robertpjenkins-bot/oliver/internal/ui"
)

// The window is a page served from this process to the browser, opened as an
// app window where the browser supports one. That is what makes real drag
// and drop possible with no GUI toolkit linked in, so the binary stays a
// single static file like Oliver's.

//go:embed window.html
var windowHTML []byte

//go:embed ziptransform.svg
var iconSVG string

// idleTimeout is how long the window may go unheard from before the program
// exits. Browsers slow the timers of hidden pages to one a minute, so this has
// to be comfortably longer than that.
const idleTimeout = 3 * time.Minute

const windowSynopsis = `ziptransform [window] [project.zip]

Opens the Zip Transform window. Drop an MCreator project ZIP onto it, say
whether to include a world, and it builds the jar.`

func cmdWindow(u *ui.UI, args []string) error {
	fs := newFlagSet(u, "window", windowSynopsis)
	var bf buildFlags
	bf.register(fs)
	noBrowser := fs.Bool("no-browser", false, "print the address instead of opening a window")
	if err := parseFlagsAnywhere(fs, args); err != nil {
		return err
	}
	if bf.mcreator != "" {
		if _, err := bf.toolkit(u); err != nil {
			return err
		}
	}

	s, err := newServer()
	if err != nil {
		return err
	}
	defer s.cleanup()
	if fs.NArg() > 0 {
		if _, err := s.openPath(fs.Arg(0)); err != nil {
			// Still open the window: it can show the problem, and take
			// another file.
			s.startupError = err.Error()
		}
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.host = ln.Addr().String()
	address := "http://" + s.host + "/?t=" + s.token

	srv := &http.Server{Handler: s, ReadHeaderTimeout: 10 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go s.watchdog(ctx, stop)
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()

	u.Info("Zip Transform is open at %s", address)
	if !*noBrowser && !openWindow(address) {
		u.Info("open that address in a web browser to use it")
	}
	if err := srv.Serve(ln); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	s.cancelBuilds()
	return nil
}

// server is the backend of the window.
type server struct {
	token string
	host  string
	tmp   string

	mu           sync.Mutex
	sessions     map[string]*session
	preloaded    string // the session for the file given on the command line
	startupError string
	lastSeen     time.Time
}

// session is one project ZIP dropped onto the window.
type session struct {
	id      string
	project *mcreator.Project
	build   *buildState
}

// buildState is a build in progress or finished, as the window polls it.
type buildState struct {
	mu     sync.Mutex
	lines  []string
	done   bool
	err    string
	hint   string
	jar    string
	world  string
	cancel context.CancelFunc
}

func (b *buildState) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, line := range strings.Split(strings.TrimRight(string(p), "\n"), "\n") {
		b.lines = append(b.lines, strings.TrimRight(line, "\r"))
	}
	return len(p), nil
}

func newServer() (*server, error) {
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		return nil, err
	}
	tmp, err := os.MkdirTemp("", "ziptransform-")
	if err != nil {
		return nil, err
	}
	return &server{
		token:    hex.EncodeToString(token),
		tmp:      tmp,
		sessions: map[string]*session{},
		lastSeen: time.Now(),
	}, nil
}

func (s *server) cleanup() {
	s.mu.Lock()
	for _, sess := range s.sessions {
		sess.project.Close()
	}
	s.mu.Unlock()
	os.RemoveAll(s.tmp)
}

func (s *server) cancelBuilds() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sess := range s.sessions {
		if b := sess.build; b != nil && b.cancel != nil {
			b.cancel()
		}
	}
}

// watchdog exits once the window has gone, but never during a build.
func (s *server) watchdog(ctx context.Context, stop func()) {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			s.mu.Lock()
			idle := time.Since(s.lastSeen) > idleTimeout && !s.building()
			s.mu.Unlock()
			if idle {
				stop()
				return
			}
		}
	}
}

// building reports whether any build is running. The caller holds s.mu.
func (s *server) building() bool {
	for _, sess := range s.sessions {
		if b := sess.build; b != nil {
			b.mu.Lock()
			running := !b.done
			b.mu.Unlock()
			if running {
				return true
			}
		}
	}
	return false
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only this machine's browser, through the address handed to it, may
	// drive the server. The Host check defeats DNS rebinding; the token stops
	// other web pages from making requests to it.
	if r.Host != s.host {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.URL.Path == "/" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.Write(windowHTML)
		return
	}
	if r.URL.Path == "/icon.svg" {
		w.Header().Set("Content-Type", "image/svg+xml")
		io.WriteString(w, iconSVG)
		return
	}
	token := r.Header.Get("X-Token")
	if token == "" {
		token = r.URL.Query().Get("t")
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(s.token)) != 1 {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	s.mu.Lock()
	s.lastSeen = time.Now()
	s.mu.Unlock()

	switch r.URL.Path {
	case "/api/ping":
	case "/api/bye":
		// The window closed or reloaded. Leave a few seconds for a reload to
		// say hello again before treating it as gone.
		s.mu.Lock()
		s.lastSeen = time.Now().Add(-idleTimeout + 5*time.Second)
		s.mu.Unlock()
	case "/api/setup":
		s.handleSetup(w, r)
	case "/api/mcreator":
		s.handleMCreator(w, r)
	case "/api/open":
		s.handleOpen(w, r)
	case "/api/project":
		s.handleProject(w, r)
	case "/api/build":
		s.handleBuild(w, r)
	case "/api/status":
		s.handleStatus(w, r)
	case "/api/cancel":
		s.handleCancel(w, r)
	case "/api/download":
		s.handleDownload(w, r)
	default:
		http.NotFound(w, r)
	}
}

// setupInfo is what the window shows about the build machinery.
type setupInfo struct {
	Installs   []mcreator.Install `json:"installs"`
	Java       string             `json:"java,omitempty"`
	JavaError  string             `json:"javaError,omitempty"`
	Generators []string           `json:"generators"`
	Project    *projectInfo       `json:"project,omitempty"`
	Error      string             `json:"error,omitempty"`
}

func (s *server) handleSetup(w http.ResponseWriter, r *http.Request) {
	t := mcreator.FindToolkit(loadConfig().MCreator)
	defer t.Close()
	info := setupInfo{Installs: t.Installs, Generators: t.Generators()}
	if info.Installs == nil {
		info.Installs = []mcreator.Install{}
	}
	if java, err := t.Java(); err == nil {
		info.Java = java
	} else {
		info.JavaError = err.Error()
	}

	s.mu.Lock()
	info.Error, s.startupError = s.startupError, ""
	sess := s.sessions[s.preloaded]
	s.mu.Unlock()
	if sess != nil {
		pi := describeProject(sess, t)
		info.Project = &pi
	}
	writeJSON(w, info)
}

func (s *server) handleMCreator(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir string `json:"dir"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	dir := strings.TrimSpace(req.Dir)
	if strings.HasPrefix(dir, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			dir = filepath.Join(home, dir[2:])
		}
	}
	inst, err := mcreator.CheckInstall(dir)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	cfg := loadConfig()
	cfg.MCreator = inst.Dir
	if err := cfg.save(); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("could not remember the folder: %w", err))
		return
	}
	s.handleSetup(w, r)
}

// projectInfo is what the window shows about a dropped ZIP.
type projectInfo struct {
	ID             string      `json:"id"`
	File           string      `json:"file"`
	Name           string      `json:"name"`
	ModID          string      `json:"modid"`
	Version        string      `json:"version"`
	Generator      string      `json:"generator"`
	Label          string      `json:"label"`
	JarName        string      `json:"jarName"`
	MakesJar       bool        `json:"makesJar"`
	CanBundleWorld bool        `json:"canBundleWorld"`
	Worlds         []worldInfo `json:"worlds"`
	// Problem is a reason the build cannot work, found before trying it.
	Problem string `json:"problem,omitempty"`
}

type worldInfo struct {
	mcreator.World
	SizeText   string `json:"sizeText"`
	InstallsAs string `json:"installsAs"`
}

func describeProject(sess *session, t *mcreator.Toolkit) projectInfo {
	p := sess.project
	info := projectInfo{
		ID:             sess.id,
		File:           filepath.Base(p.ZipPath),
		Name:           p.ModName,
		ModID:          p.ModID,
		Version:        p.Version,
		Generator:      p.Generator,
		Label:          p.GeneratorLabel(),
		JarName:        p.JarName(),
		MakesJar:       p.MakesJar(),
		CanBundleWorld: p.CanBundleWorld(),
		Worlds:         []worldInfo{},
	}
	for _, w := range p.Worlds {
		info.Worlds = append(info.Worlds, worldInfo{World: w, SizeText: mcreator.HumanSize(w.Size), InstallsAs: mcreator.InstalledWorldName(p, w)})
	}

	switch {
	case !p.MakesJar():
		info.Problem = p.GeneratorLabel() + " projects do not build into a jar. Use File → Export the mod for distribution in MCreator instead."
	default:
		if _, err := t.Generator(p.Generator); err != nil {
			info.Problem = sentence(err)
		} else if _, err := t.Java(); err != nil {
			info.Problem = sentence(err)
		} else {
			for _, api := range p.APIs {
				if _, err := t.APIGradle(api, p.Generator); err != nil {
					info.Problem = sentence(err)
					break
				}
			}
		}
	}
	return info
}

// openPath starts a session for a ZIP already on disk.
func (s *server) openPath(path string) (*session, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	p, err := mcreator.Open(abs)
	if err != nil {
		return nil, err
	}
	sess := s.addSession(p)
	s.mu.Lock()
	s.preloaded = sess.id
	s.mu.Unlock()
	return sess, nil
}

func (s *server) addSession(p *mcreator.Project) *session {
	id := make([]byte, 8)
	_, _ = rand.Read(id)
	sess := &session{id: hex.EncodeToString(id), project: p}
	s.mu.Lock()
	s.sessions[sess.id] = sess
	s.mu.Unlock()
	return sess
}

// handleOpen receives a ZIP dropped onto the window.
func (s *server) handleOpen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	name, _ := url.QueryUnescape(r.Header.Get("X-File-Name"))
	name = filepath.Base(filepath.Clean("/" + name))
	if name == "/" || name == "." {
		name = "project.zip"
	}

	dir, err := os.MkdirTemp(s.tmp, "upload-")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	dest := filepath.Join(dir, name)
	f, err := os.Create(dest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	_, err = io.Copy(f, http.MaxBytesReader(w, r.Body, 8<<30))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusBadRequest, fmt.Errorf("receiving the file: %w", err))
		return
	}

	p, err := mcreator.Open(dest)
	if err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	sess := s.addSession(p)
	t := mcreator.FindToolkit(loadConfig().MCreator)
	defer t.Close()
	writeJSON(w, describeProject(sess, t))
}

// handleProject describes an open project again, for when the MCreator
// folder changed and the checks need re-running.
func (s *server) handleProject(w http.ResponseWriter, r *http.Request) {
	sess := s.session(r.URL.Query().Get("id"))
	if sess == nil {
		writeError(w, http.StatusNotFound, errors.New("that project is no longer open; drop it again"))
		return
	}
	t := mcreator.FindToolkit(loadConfig().MCreator)
	defer t.Close()
	writeJSON(w, describeProject(sess, t))
}

func (s *server) session(id string) *session {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessions[id]
}

// buildOf returns the latest build of a session, if it has one.
func (s *server) buildOf(id string) *buildState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess := s.sessions[id]; sess != nil {
		return sess.build
	}
	return nil
}

// handleBuild starts a build in the background.
func (s *server) handleBuild(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID    string `json:"id"`
		World string `json:"world"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	sess := s.session(req.ID)
	if sess == nil {
		writeError(w, http.StatusNotFound, errors.New("that project is no longer open; drop it again"))
		return
	}

	s.mu.Lock()
	if b := sess.build; b != nil {
		b.mu.Lock()
		running := !b.done
		b.mu.Unlock()
		if running {
			s.mu.Unlock()
			writeError(w, http.StatusConflict, errors.New("this project is already building"))
			return
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	b := &buildState{cancel: cancel}
	sess.build = b
	s.mu.Unlock()

	out := filepath.Join(s.tmp, "jar-"+sess.id, sess.project.JarName())
	go func() {
		defer cancel()
		t := mcreator.FindToolkit(loadConfig().MCreator)
		defer t.Close()
		res, err := mcreator.Build(ctx, sess.project, t, mcreator.BuildOptions{
			World:   req.World,
			Output:  out,
			Log:     b,
			TempDir: s.tmp,
		})
		b.mu.Lock()
		defer b.mu.Unlock()
		b.done = true
		switch {
		case errors.Is(err, context.Canceled):
			b.err = "The build was stopped."
		case err != nil:
			b.err = sentence(err)
			var ge *mcreator.GradleError
			if errors.As(err, &ge) {
				b.hint = ge.Hint()
			}
		default:
			b.jar = res.Jar
			b.world = res.World
		}
	}()
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *server) handleStatus(w http.ResponseWriter, r *http.Request) {
	b := s.buildOf(r.URL.Query().Get("id"))
	if b == nil {
		writeError(w, http.StatusNotFound, errors.New("no build is running for that project"))
		return
	}
	var from int
	fmt.Sscan(r.URL.Query().Get("from"), &from)

	b.mu.Lock()
	defer b.mu.Unlock()
	if from < 0 || from > len(b.lines) {
		from = len(b.lines)
	}
	status := map[string]any{
		"lines": b.lines[from:],
		"next":  len(b.lines),
		"done":  b.done,
	}
	if b.done {
		if b.err != "" {
			status["error"] = b.err
			status["hint"] = b.hint
		} else {
			status["jar"] = filepath.Base(b.jar)
			status["world"] = b.world
			if st, err := os.Stat(b.jar); err == nil {
				status["size"] = mcreator.HumanSize(st.Size())
			}
		}
	}
	writeJSON(w, status)
}

func (s *server) handleCancel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID string `json:"id"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	if b := s.buildOf(req.ID); b != nil {
		b.cancel()
	}
	writeJSON(w, map[string]bool{"ok": true})
}

func (s *server) handleDownload(w http.ResponseWriter, r *http.Request) {
	b := s.buildOf(r.URL.Query().Get("id"))
	if b == nil {
		http.NotFound(w, r)
		return
	}
	b.mu.Lock()
	jar := b.jar
	b.mu.Unlock()
	if jar == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/java-archive")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q; filename*=UTF-8''%s", filepath.Base(jar), url.PathEscape(filepath.Base(jar))))
	http.ServeFile(w, r, jar)
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return false
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": sentence(err)})
}

// sentence turns an error into something to show a person: capitalised and
// ending in a full stop.
func sentence(err error) string {
	msg := strings.TrimPrefix(err.Error(), mcreator.ErrNotWorkspace.Error()+": ")
	if errors.Is(err, mcreator.ErrNotWorkspace) && msg == err.Error() {
		msg = "This is not an MCreator project ZIP."
	}
	if msg == "" {
		return msg
	}
	msg = strings.ToUpper(msg[:1]) + msg[1:]
	if !strings.HasSuffix(msg, ".") {
		msg += "."
	}
	return msg
}

// openWindow shows the page. A Chromium-family browser can open it as an app
// window, without tabs or an address bar, which is the closest thing to a
// native window; any other browser opens it as a tab.
func openWindow(address string) bool {
	if env := os.Getenv("ZIPTRANSFORM_BROWSER"); env != "" {
		return exec.Command(env, address).Start() == nil
	}
	for _, browser := range []string{"chromium", "chromium-browser", "google-chrome", "google-chrome-stable", "brave-browser", "microsoft-edge", "vivaldi"} {
		if path, err := exec.LookPath(browser); err == nil {
			cmd := exec.Command(path, "--app="+address, "--window-size=720,760")
			if cmd.Start() == nil {
				go cmd.Wait()
				return true
			}
		}
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", address)
	case "darwin":
		cmd = exec.Command("open", address)
	default:
		cmd = exec.Command("xdg-open", address)
	}
	if cmd.Start() != nil {
		return false
	}
	go cmd.Wait()
	return true
}
