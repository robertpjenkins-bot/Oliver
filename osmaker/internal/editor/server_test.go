package editor

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/robertpjenkins-bot/oliver/osmaker/internal/design"
)

func newTestServer(t *testing.T) (*Server, *httptest.Server) {
	t.Helper()
	s, err := New(t.TempDir(), "")
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	s.addr = ts.Listener.Addr().String()
	return s, ts
}

func call(t *testing.T, ts *httptest.Server, token, method, path string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, ts.URL+path, &buf)
	req.Header.Set("X-OSMaker-Token", token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	json.NewDecoder(res.Body).Decode(&out)
	return res, out
}

func TestAPIRejectsOtherPages(t *testing.T) {
	_, ts := newTestServer(t)
	if res, _ := call(t, ts, "wrong", "GET", "/api/info", nil); res.StatusCode != http.StatusForbidden {
		t.Fatalf("got %d without the token", res.StatusCode)
	}
	res, err := http.Get(ts.URL + "/")
	if err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("page itself should load: %v %v", res, err)
	}
}

func TestAutosaveAndFolderBuild(t *testing.T) {
	s, ts := newTestServer(t)
	d := design.Default()
	if res, out := call(t, ts, s.token, "PUT", "/api/autosave", d); res.StatusCode != 200 {
		t.Fatalf("autosave: %v", out)
	}
	if _, err := os.Stat(s.autosavePath()); err != nil {
		t.Fatal(err)
	}
	if _, out := call(t, ts, s.token, "GET", "/api/design", nil); out["system"] == nil {
		t.Fatalf("design not returned: %v", out)
	}

	bad := design.Default()
	bad.System.Username = "Bad Name"
	if res, out := call(t, ts, s.token, "POST", "/api/validate", bad); res.StatusCode != 400 || out["error"] == "" {
		t.Fatalf("bad design accepted: %v", out)
	}

	out := t.TempDir()
	res, body := call(t, ts, s.token, "POST", "/api/build", map[string]any{"design": d, "outDir": out, "folderOnly": true})
	if res.StatusCode != 200 {
		t.Fatalf("build: %v", body)
	}
	if _, err := os.Stat(filepath.Join(out, "my-os", "build.sh")); err != nil {
		t.Fatal(err)
	}
	if _, st := call(t, ts, s.token, "GET", "/api/build", nil); st["ok"] != true || st["running"] != false {
		t.Fatalf("status: %v", st)
	}
}
