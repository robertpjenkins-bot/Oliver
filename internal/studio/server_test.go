package studio

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/gamekit"
)

func newTestStudio(t *testing.T) (*Server, http.Handler) {
	t.Helper()
	s := New(t.TempDir())
	s.Assistant = &gamekit.Assistant{} // no key: the offline designer does the work
	return s, s.Handler()
}

func do(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(data)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("X-Oliver-Studio", "1")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// The whole round trip a person makes in the studio: describe a game, change
// it, play it, export it.
func TestStudioRoundTrip(t *testing.T) {
	s, h := newTestStudio(t)

	rec := do(t, h, http.MethodPost, "/api/design", map[string]string{"prompt": "a space shooter with alien drones"})
	if rec.Code != http.StatusOK {
		t.Fatalf("design: %d %s", rec.Code, rec.Body)
	}
	var made struct {
		Name string        `json:"name"`
		Game *gamekit.Game `json:"game"`
		Note string        `json:"note"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &made); err != nil {
		t.Fatal(err)
	}
	if made.Game == nil || made.Game.Genre != gamekit.GenreShooter {
		t.Fatalf("design produced %+v", made.Game)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, made.Name+Extension)); err != nil {
		t.Errorf("the game was not written to disk: %v", err)
	}

	rec = do(t, h, http.MethodGet, "/api/games", nil)
	if !strings.Contains(rec.Body.String(), made.Name) {
		t.Errorf("the game list does not mention %q: %s", made.Name, rec.Body)
	}

	rec = do(t, h, http.MethodPost, "/api/assist", map[string]any{
		"name": made.Name, "instruction": "add a boss",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("assist: %d %s", rec.Code, rec.Body)
	}
	after, err := s.Load(made.Name)
	if err != nil {
		t.Fatal(err)
	}
	if after.Object("boss") == nil {
		t.Error("the instruction was not applied to the saved game")
	}

	// The play page and an export are the same bytes: what is play-tested is
	// exactly what is handed to someone else.
	req := httptest.NewRequest(http.MethodGet, "/play/"+made.Name, nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "OliverEngine.start") {
		t.Fatalf("play page: %d, %d bytes", rec.Code, rec.Body.Len())
	}
	exported, err := after.ExportHTML()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(exported, rec.Body.Bytes()) {
		t.Error("the play page and the export are not the same game")
	}

	rec = do(t, h, http.MethodPost, "/api/export", map[string]string{"name": made.Name})
	var out struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Stat(out.Path); err != nil || st.Size() < 10000 {
		t.Errorf("export wrote %q: %v", out.Path, err)
	}
}

// Saving from the editor must reject anything that is not a game, and report
// the problems of one that is a game but a broken one.
func TestStudioSaveReportsProblems(t *testing.T) {
	_, h := newTestStudio(t)

	rec := do(t, h, http.MethodPut, "/api/game?name=broken", map[string]any{
		"format": gamekit.Format,
		"name":   "Broken",
		"levels": []any{},
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("save: %d %s", rec.Code, rec.Body)
	}
	var res struct {
		Problems []string `json:"problems"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatal(err)
	}
	if len(res.Problems) == 0 {
		t.Error("a game with no levels and no player was saved without complaint")
	}
}

// A name from the browser must never be able to write outside the studio's own
// folder.
func TestGameNamesStayInTheFolder(t *testing.T) {
	s := New("/games")
	for _, name := range []string{"../escape", "/etc/passwd", "..", "a/b/c", ""} {
		path, err := s.Path(name)
		if err != nil {
			continue
		}
		if filepath.Dir(path) != "/games" {
			t.Errorf("%q resolved to %q, which is outside the games folder", name, path)
		}
	}
}

// Every API route refuses a request that did not come from the studio's own
// page, so another page in the same browser cannot reach in.
func TestAPIRefusesOtherPages(t *testing.T) {
	_, h := newTestStudio(t)
	for _, path := range []string{"/api/status", "/api/games", "/api/game?name=x", "/api/design", "/api/assist", "/api/export"} {
		req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader([]byte("{}")))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s answered %d without the studio's header", path, rec.Code)
		}
	}
}

func TestListReadsTheFolder(t *testing.T) {
	s := New(t.TempDir())
	g := gamekit.Design("a forest platformer")
	if _, err := s.Save("", g); err != nil {
		t.Fatal(err)
	}
	// Anything that is not a game is ignored rather than breaking the list.
	if err := os.WriteFile(filepath.Join(s.Dir, "notes.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, "rubbish"+Extension), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("listed %d games, want 2 (the game and the unreadable one)", len(list))
	}
	found := false
	for _, info := range list {
		if info.Title == g.Name && info.Levels == len(g.Levels) {
			found = true
		}
	}
	if !found {
		t.Errorf("the saved game is missing from %+v", list)
	}
}

// A studio opened from a desktop icon has no terminal to stop it, so it has to
// stop itself once the window it was opened for has gone.
func TestStudioStopsWhenNobodyIsUsingIt(t *testing.T) {
	s := New(t.TempDir())
	s.Assistant = &gamekit.Assistant{}
	s.IdleTimeout = 200 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	addr := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, "127.0.0.1:0", func(url string) { addr <- url }) }()

	url := <-addr
	// While the editor is open it says so, and the studio stays up.
	deadline := time.Now().Add(700 * time.Millisecond)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequest(http.MethodGet, url+"api/status", nil)
		req.Header.Set("X-Oliver-Studio", "1")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("the studio stopped while it was still in use: %v", err)
		}
		resp.Body.Close()
		time.Sleep(50 * time.Millisecond)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the studio kept running after everyone had gone")
	}
}

// Left to itself, the studio runs until it is stopped: somebody working in a
// terminal should not lose their editor because they stopped to think.
func TestStudioKeepsRunningWithoutAnIdleTimeout(t *testing.T) {
	s := New(t.TempDir())
	s.Assistant = &gamekit.Assistant{}

	ctx, cancel := context.WithCancel(context.Background())
	addr := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- s.Serve(ctx, "127.0.0.1:0", func(url string) { addr <- url }) }()
	url := <-addr

	time.Sleep(300 * time.Millisecond)
	select {
	case <-done:
		t.Fatal("the studio stopped on its own")
	default:
	}
	req, _ := http.NewRequest(http.MethodGet, url+"api/status", nil)
	req.Header.Set("X-Oliver-Studio", "1")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("the studio is not answering: %v", err)
	}
	resp.Body.Close()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the studio did not stop when it was told to")
	}
}
