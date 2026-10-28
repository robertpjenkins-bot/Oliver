package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/mcreatortest"
	"github.com/robertpjenkins-bot/oliver/internal/ui"
)

// machine is a home directory with MCreator installed in it and a project ZIP
// in Downloads.
type machine struct {
	t       *testing.T
	home    string
	install string
	zip     string
	capture string
	out     *bytes.Buffer
	err     *bytes.Buffer
}

func newMachine(t *testing.T) *machine {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake Gradle is a shell script")
	}
	home := mcreatortest.Isolate(t)
	t.Setenv("ZIPTRANSFORM_CONFIG", "")
	m := &machine{
		t:       t,
		home:    home,
		install: mcreatortest.FakeInstall(t, filepath.Join(home, "MCreator20263"), "2026.3"),
		zip:     mcreatortest.ShareableZip(t, filepath.Join(home, "Downloads"), nil),
		capture: filepath.Join(home, "capture"),
		out:     &bytes.Buffer{},
		err:     &bytes.Buffer{},
	}
	os.MkdirAll(m.capture, 0o755)
	t.Setenv("CAPTURE", m.capture)
	return m
}

func (m *machine) run(args ...string) error {
	m.t.Helper()
	m.out.Reset()
	m.err.Reset()
	return run(&ui.UI{Out: m.out, Err: m.err}, args)
}

func (m *machine) exists(parts ...string) bool {
	_, err := os.Stat(filepath.Join(parts...))
	return err == nil
}

func TestBuildWritesJarBesideZip(t *testing.T) {
	m := newMachine(t)
	if err := m.run("build", m.zip, "-no-world"); err != nil {
		t.Fatalf("%v\n%s", err, m.err)
	}
	jar := filepath.Join(m.home, "Downloads", "ruby_tools-1.2-neoforge-1.21.1.jar")
	if data, err := os.ReadFile(jar); err != nil || string(data) != "the jar\n" {
		t.Fatalf("jar = %q, %v", data, err)
	}
	if !strings.Contains(m.out.String(), "built "+jar) {
		t.Errorf("output: %s", m.out)
	}
	if m.exists(m.capture, "work", "src", "main", "resources", "ziptransform-world") {
		t.Error("a world was bundled after -no-world")
	}
}

func TestBuildWithWorldFlagAfterTheZip(t *testing.T) {
	m := newMachine(t)
	out := filepath.Join(m.home, "mods", "ruby.jar")
	if err := m.run("build", m.zip, "-world", "New World", "-o", out); err != nil {
		t.Fatalf("%v\n%s", err, m.err)
	}
	if !m.exists(out) {
		t.Fatal("no jar at -o")
	}
	if !m.exists(m.capture, "work", "src", "main", "resources", "ziptransform-world", "ruby_tools", "index.txt") {
		t.Error("the world was not bundled")
	}
	if !strings.Contains(m.err.String(), `in the folder "New World (ruby_tools)"`) {
		t.Errorf("does not say where the world appears:\n%s", m.err)
	}
}

func TestBuildWithoutTerminalDoesNotGuessAWorld(t *testing.T) {
	m := newMachine(t)
	if err := m.run("build", m.zip); err != nil {
		t.Fatalf("%v\n%s", err, m.err)
	}
	if !strings.Contains(m.err.String(), "not bundling a world") {
		t.Errorf("did not say the world was left out:\n%s", m.err)
	}
}

func TestBuildUnknownWorld(t *testing.T) {
	m := newMachine(t)
	err := m.run("build", m.zip, "-world", "Skyblock")
	if err == nil || !strings.Contains(err.Error(), `"Creative Test", "New World"`) {
		t.Errorf("err = %v", err)
	}
}

func TestMCreatorFolderIsRemembered(t *testing.T) {
	m := newMachine(t)
	hidden := mcreatortest.FakeInstall(t, filepath.Join(m.home, "tools", "mc-old"), "2025.1")

	if err := m.run("setup", "-mcreator", hidden); err != nil {
		t.Fatal(err)
	}
	if err := m.run("setup"); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(m.out.String(), "\n")
	if !strings.Contains(lines[0], hidden) || !strings.Contains(lines[0], "2025.1") {
		t.Errorf("the remembered folder is not the one used:\n%s", m.out)
	}

	if err := m.run("setup", "-mcreator", filepath.Join(m.home, "Downloads")); err == nil {
		t.Error("accepted a folder that is not MCreator")
	}
}

func TestSetupWithoutMCreator(t *testing.T) {
	m := newMachine(t)
	os.RemoveAll(m.install)
	if err := m.run("setup"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.out.String(), "MCreator was not found") {
		t.Errorf("output:\n%s", m.out)
	}
	err := m.run("build", m.zip, "-no-world")
	if err == nil || !strings.Contains(err.Error(), "MCreator was not found") {
		t.Errorf("build without MCreator: %v", err)
	}
}

func TestInfo(t *testing.T) {
	m := newMachine(t)
	if err := m.run("info", m.zip); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Ruby Tools", "NeoForge 1.21.1 (neoforge-1.21.1)", "ruby_tools-1.2-neoforge-1.21.1.jar", "Creative Test", "New World"} {
		if !strings.Contains(m.out.String(), want) {
			t.Errorf("info lacks %q:\n%s", want, m.out)
		}
	}
}

// window drives the window's server the way the page does.
type window struct {
	t   *testing.T
	s   *server
	url string
}

func openTestWindow(t *testing.T) *window {
	t.Helper()
	s, err := newServer()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.cleanup)
	ts := httptest.NewServer(s)
	t.Cleanup(ts.Close)
	s.host = strings.TrimPrefix(ts.URL, "http://")
	return &window{t: t, s: s, url: ts.URL}
}

func (w *window) do(method, path string, body io.Reader, header map[string]string) (int, []byte) {
	w.t.Helper()
	req, err := http.NewRequest(method, w.url+path, body)
	if err != nil {
		w.t.Fatal(err)
	}
	req.Header.Set("X-Token", w.s.token)
	for k, v := range header {
		if v == "" {
			req.Header.Del(k)
		} else {
			req.Header.Set(k, v)
		}
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		w.t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	return res.StatusCode, data
}

func (w *window) json(method, path string, body any, into any) int {
	w.t.Helper()
	var r io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		r = bytes.NewReader(data)
	}
	code, data := w.do(method, path, r, nil)
	if into != nil {
		if err := json.Unmarshal(data, into); err != nil {
			w.t.Fatalf("%s %s: %v\n%s", method, path, err, data)
		}
	}
	return code
}

func (w *window) upload(path string) (int, projectInfo, string) {
	w.t.Helper()
	f, err := os.Open(path)
	if err != nil {
		w.t.Fatal(err)
	}
	defer f.Close()
	code, data := w.do("POST", "/api/open", f, map[string]string{"X-File-Name": filepath.Base(path)})
	var info projectInfo
	var fail struct{ Error string }
	json.Unmarshal(data, &info)
	json.Unmarshal(data, &fail)
	return code, info, fail.Error
}

func TestWindowRefusesStrangers(t *testing.T) {
	newMachine(t)
	w := openTestWindow(t)

	if code, body := w.do("GET", "/", nil, map[string]string{"X-Token": ""}); code != 200 || !bytes.Contains(body, []byte("Drop your MCreator project ZIP here")) {
		t.Errorf("page: %d", code)
	}
	if code, _ := w.do("GET", "/api/setup", nil, map[string]string{"X-Token": ""}); code != http.StatusForbidden {
		t.Errorf("no token: %d", code)
	}
	if code, _ := w.do("GET", "/api/setup", nil, map[string]string{"X-Token": "guess"}); code != http.StatusForbidden {
		t.Errorf("wrong token: %d", code)
	}

	// A page on another site that resolves its own name to 127.0.0.1 sends
	// its own Host header.
	req, _ := http.NewRequest("GET", w.url+"/api/setup", nil)
	req.Host = "attacker.example:80"
	req.Header.Set("X-Token", w.s.token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("foreign Host: %d", res.StatusCode)
	}
}

func TestWindowBuildsDroppedZip(t *testing.T) {
	m := newMachine(t)
	w := openTestWindow(t)

	var setup setupInfo
	if code := w.json("GET", "/api/setup", nil, &setup); code != 200 {
		t.Fatalf("setup: %d", code)
	}
	if len(setup.Installs) != 1 || setup.Installs[0].Dir != m.install || setup.Java == "" {
		t.Errorf("setup = %+v", setup)
	}

	code, p, msg := w.upload(m.zip)
	if code != 200 {
		t.Fatalf("open: %d %s", code, msg)
	}
	if p.Name != "Ruby Tools" || p.Label != "NeoForge 1.21.1" || p.Problem != "" || !p.CanBundleWorld || p.File != "Ruby Tools.zip" {
		t.Errorf("project = %+v", p)
	}
	if len(p.Worlds) != 2 || p.Worlds[1].Name != "New World" || p.Worlds[1].InstallsAs != "New World (ruby_tools)" {
		t.Errorf("worlds = %+v", p.Worlds)
	}

	if code := w.json("POST", "/api/build", map[string]string{"id": p.ID, "world": "New World"}, nil); code != 200 {
		t.Fatalf("build: %d", code)
	}
	var status struct {
		Lines []string
		Next  int
		Done  bool
		Error string
		Jar   string
		World string
	}
	var log []string
	for deadline := time.Now().Add(20 * time.Second); !status.Done; {
		if time.Now().After(deadline) {
			t.Fatalf("build did not finish:\n%s", strings.Join(log, "\n"))
		}
		w.json("GET", "/api/status?id="+p.ID+"&from="+itoa(status.Next), nil, &status)
		log = append(log, status.Lines...)
		time.Sleep(20 * time.Millisecond)
	}
	if status.Error != "" || status.Jar != "ruby_tools-1.2-neoforge-1.21.1.jar" || status.World != "New World (ruby_tools)" {
		t.Fatalf("status = %+v\n%s", status, strings.Join(log, "\n"))
	}
	if !strings.Contains(strings.Join(log, "\n"), "BUILD SUCCESSFUL") {
		t.Errorf("Gradle output did not reach the window:\n%s", strings.Join(log, "\n"))
	}

	code, jar := w.do("GET", "/api/download?id="+p.ID+"&t="+w.s.token, nil, map[string]string{"X-Token": ""})
	if code != 200 || string(jar) != "the jar\n" {
		t.Errorf("download: %d %q", code, jar)
	}
}

func TestWindowExplainsWrongFiles(t *testing.T) {
	m := newMachine(t)
	w := openTestWindow(t)

	photos := filepath.Join(m.home, "photos.zip")
	mcreatortest.WriteZip(t, photos, map[string]string{"cat.jpg": "meow"})
	code, _, msg := w.upload(photos)
	if code != http.StatusUnprocessableEntity || !strings.HasPrefix(msg, "There is no .mcreator file inside") {
		t.Errorf("photos.zip: %d %q", code, msg)
	}

	// A project for a Minecraft version this MCreator cannot build is
	// reported before anyone waits for a build.
	old := mcreatortest.ShareableZip(t, filepath.Join(m.home, "old"), map[string]string{
		"ruby_tools.mcreator": strings.Replace(mcreatortest.WorkspaceJSON, "neoforge-1.21.1", "forge-1.20.1", 1),
	})
	code, p, _ := w.upload(old)
	if code != 200 || !strings.Contains(p.Problem, "forge-1.20.1 generator") {
		t.Errorf("old project: %d %+v", code, p)
	}
}

func TestWindowWithZipFromFileManager(t *testing.T) {
	m := newMachine(t)
	w := openTestWindow(t)
	if _, err := w.s.openPath(m.zip); err != nil {
		t.Fatal(err)
	}
	var setup setupInfo
	w.json("GET", "/api/setup", nil, &setup)
	if setup.Project == nil || setup.Project.ModID != "ruby_tools" {
		t.Errorf("the ZIP passed on the command line is not shown: %+v", setup.Project)
	}
}

func TestWindowChoosesMCreatorFolder(t *testing.T) {
	m := newMachine(t)
	os.RemoveAll(m.install)
	elsewhere := mcreatortest.FakeInstall(t, filepath.Join(m.home, "stuff", "mc"), "2026.3")
	w := openTestWindow(t)

	var fail struct{ Error string }
	if code := w.json("POST", "/api/mcreator", map[string]string{"dir": "~/Downloads"}, &fail); code != http.StatusBadRequest || !strings.Contains(fail.Error, "does not look like an MCreator folder") {
		t.Errorf("wrong folder: %d %q", code, fail.Error)
	}
	var setup setupInfo
	if code := w.json("POST", "/api/mcreator", map[string]string{"dir": "~/stuff/mc"}, &setup); code != 200 || len(setup.Installs) == 0 || setup.Installs[0].Dir != elsewhere {
		t.Errorf("chosen folder: %d %+v", code, setup)
	}
	if loadConfig().MCreator != elsewhere {
		t.Error("the folder was not remembered")
	}
}

func TestSentence(t *testing.T) {
	cases := map[string]string{
		"the build failed":                            "The build failed.",
		"already a sentence.":                         "Already a sentence.",
		"not an MCreator workspace: there is no file": "There is no file.",
	}
	for in, want := range cases {
		if got := sentence(errors.New(in)); got != want {
			t.Errorf("sentence(%q) = %q, want %q", in, got, want)
		}
	}
}

func itoa(n int) string {
	b, _ := json.Marshal(n)
	return string(b)
}

// The window polls a build from the moment it starts, often before the build
// has written anything. An empty log must still be a list: the page once read
// null and stopped polling, stuck on "Starting…".
func TestWindowStatusBeforeAnyOutput(t *testing.T) {
	m := newMachine(t)
	w := openTestWindow(t)
	sess, err := w.s.openPath(m.zip)
	if err != nil {
		t.Fatal(err)
	}
	sess.build = &buildState{cancel: func() {}}

	code, body := w.do("GET", "/api/status?id="+sess.id+"&from=0", nil, nil)
	if code != 200 || !bytes.Contains(body, []byte(`"lines":[]`)) {
		t.Errorf("status = %d %s", code, body)
	}
}
