package webui

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/device"
	"github.com/robertpjenkins-bot/oliver/internal/imaging"
)

// harness is a synthetic machine with the server in front of it. The drives
// are files, so a write through the API really writes, and the test can read
// back what landed.
type harness struct {
	t      *testing.T
	root   string
	server *Server
	http   *httptest.Server
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "dev"), 0o755); err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, root: root}
	h.mounts("")

	h.server = &Server{
		Token: "test-token",
		Home:  root,
		Scanner: func() *device.Scanner {
			return &device.Scanner{
				SysBlock: filepath.Join(root, "sys", "block"),
				Dev:      filepath.Join(root, "dev"),
				Mounts:   filepath.Join(root, "mounts"),
				Access:   func(string) (bool, bool) { return true, true },
			}
		},
	}
	h.http = httptest.NewServer(h.server.Handler())
	t.Cleanup(h.http.Close)
	return h
}

func (h *harness) drive(name string, size int64, removable bool, model string) string {
	h.t.Helper()
	base := filepath.Join(h.root, "sys", "block", name)
	if err := os.MkdirAll(filepath.Join(base, "device"), 0o755); err != nil {
		h.t.Fatal(err)
	}
	write := func(path, content string) {
		if err := os.WriteFile(path, []byte(content+"\n"), 0o644); err != nil {
			h.t.Fatal(err)
		}
	}
	write(filepath.Join(base, "size"), fmt.Sprint(size/512))
	flag := "0"
	if removable {
		flag = "1"
	}
	write(filepath.Join(base, "removable"), flag)
	write(filepath.Join(base, "device", "model"), model)

	node := filepath.Join(h.root, "dev", name)
	f, err := os.Create(node)
	if err != nil {
		h.t.Fatal(err)
	}
	defer f.Close()
	if err := f.Truncate(size); err != nil {
		h.t.Fatal(err)
	}
	return node
}

func (h *harness) mounts(lines string) {
	h.t.Helper()
	if err := os.WriteFile(filepath.Join(h.root, "mounts"), []byte(lines), 0o644); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) mount(name, target string) {
	h.t.Helper()
	existing, _ := os.ReadFile(filepath.Join(h.root, "mounts"))
	h.mounts(string(existing) + filepath.Join(h.root, "dev", name) + " " + target + " ext4 rw 0 0\n")
}

// image writes an image file and returns its path.
func (h *harness) image(name string, size int64) string {
	h.t.Helper()
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i*31 + 7)
	}
	path := filepath.Join(h.root, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		h.t.Fatal(err)
	}
	return path
}

// do sends an authorised request, the way the page does.
func (h *harness) do(method, path string, body any) (*http.Response, map[string]any) {
	h.t.Helper()
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			h.t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, h.http.URL+path, reader)
	if err != nil {
		h.t.Fatal(err)
	}
	req.Header.Set("X-Oliver-Token", h.server.Token)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)

	out := map[string]any{}
	if len(bytes.TrimSpace(raw)) > 0 && raw[0] == '{' {
		_ = json.Unmarshal(raw, &out)
	} else {
		out["_raw"] = string(raw)
	}
	return res, out
}

func (h *harness) list(path string) []map[string]any {
	h.t.Helper()
	req, _ := http.NewRequest("GET", h.http.URL+path, nil)
	req.Header.Set("X-Oliver-Token", h.server.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatal(err)
	}
	defer res.Body.Close()
	var out []map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		h.t.Fatalf("decoding %s: %v", path, err)
	}
	return out
}

// await waits for a job to finish and returns its final state.
func (h *harness) await(id string) map[string]any {
	h.t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		for _, job := range h.list("/api/jobs") {
			if job["id"] == id && job["finished"] == true {
				return job
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	h.t.Fatalf("job %s never finished", id)
	return nil
}

func contentsOf(t *testing.T, path string, n int) []byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	buf := make([]byte, n)
	f.ReadAt(buf, 0)
	return buf
}

/* ---- the door ---------------------------------------------------------- */

func TestEveryRouteNeedsTheToken(t *testing.T) {
	h := newHarness(t)
	h.drive("sdb", 64<<20, true, "stick")

	for _, route := range []struct{ method, path string }{
		{"GET", "/api/drives"},
		{"GET", "/api/browse"},
		{"GET", "/api/image?path=/etc/hostname"},
		{"GET", "/api/jobs"},
		{"POST", "/api/write"},
		{"POST", "/api/erase"},
		{"POST", "/api/format"},
	} {
		req, _ := http.NewRequest(route.method, h.http.URL+route.path, strings.NewReader("{}"))
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		res.Body.Close()
		if res.StatusCode != http.StatusForbidden {
			t.Errorf("%s %s without a token = %d, want 403", route.method, route.path, res.StatusCode)
		}
	}
}

func TestAWrongTokenIsRefused(t *testing.T) {
	h := newHarness(t)
	req, _ := http.NewRequest("GET", h.http.URL+"/api/drives", nil)
	req.Header.Set("X-Oliver-Token", "test-tokeN")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", res.StatusCode)
	}
}

func TestARequestFromAnotherSiteIsRefused(t *testing.T) {
	h := newHarness(t)
	h.drive("sdb", 64<<20, true, "stick")

	// A page on another origin that somehow learned the token still must not
	// be able to drive this server.
	req, _ := http.NewRequest("POST", h.http.URL+"/api/erase", strings.NewReader(`{"device":"sdb","confirm":"sdb"}`))
	req.Header.Set("X-Oliver-Token", h.server.Token)
	req.Header.Set("Origin", "https://example.invalid")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403", res.StatusCode)
	}
	if got := contentsOf(t, filepath.Join(h.root, "dev", "sdb"), 512); !bytes.Equal(got, make([]byte, 512)) {
		t.Error("the drive was erased by a cross-origin request")
	}
}

func TestAHostnamePointedAtLoopbackIsRefused(t *testing.T) {
	h := newHarness(t)
	req, _ := http.NewRequest("GET", h.http.URL+"/api/drives", nil)
	req.Header.Set("X-Oliver-Token", h.server.Token)
	req.Host = "drives.example.invalid"
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want 403 for a rebound hostname", res.StatusCode)
	}
}

func TestThePageCarriesTheTokenAndNothingElseDoes(t *testing.T) {
	h := newHarness(t)

	res, err := http.Get(h.http.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("the page was served without a token: %d", res.StatusCode)
	}

	res, err = http.Get(h.http.URL + "/?t=" + h.server.Token)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if strings.Contains(string(body), tokenPlaceholder) {
		t.Error("the placeholder was served instead of the token")
	}
	if !strings.Contains(string(body), h.server.Token) {
		t.Error("the page cannot talk to the server: it carries no token")
	}
	if csp := res.Header.Get("Content-Security-Policy"); !strings.Contains(csp, "default-src 'none'") {
		t.Errorf("no content policy on the page: %q", csp)
	}
}

/* ---- the rules --------------------------------------------------------- */

func TestDrivesCarryTheReasonTheyCannotBeUsed(t *testing.T) {
	h := newHarness(t)
	h.drive("sda", 64<<20, false, "System SSD")
	h.drive("sdb", 64<<20, true, "Cruzer")
	h.mount("sda", "/")

	drives := h.list("/api/drives")
	if len(drives) != 2 {
		t.Fatalf("listed %d drives", len(drives))
	}
	byName := map[string]map[string]any{}
	for _, d := range drives {
		byName[d["name"].(string)] = d
	}
	if byName["sda"]["reason"] != "system_disk" || byName["sda"]["system_disk"] != true {
		t.Errorf("the system disk is not marked as one: %+v", byName["sda"])
	}
	if byName["sda"]["contents"] != "the running system" {
		t.Errorf("contents = %v", byName["sda"]["contents"])
	}
	if byName["sdb"]["reason"] != nil {
		t.Errorf("a plain removable drive was refused: %+v", byName["sdb"])
	}
}

func TestTheSystemDiskCannotBeWrittenToHoweverTheRequestIsDressedUp(t *testing.T) {
	h := newHarness(t)
	node := h.drive("sda", 64<<20, false, "System SSD")
	h.mount("sda", "/")
	image := h.image("test.iso", 1<<20)

	// Every permission the API offers, all at once.
	res, body := h.do("POST", "/api/write", map[string]any{
		"device": "sda", "image": image, "confirm": "sda",
		"allow_fixed": true, "unmount": true, "verify": true,
	})
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.StatusCode)
	}
	if body["reason"] != "system_disk" || body["overridable"] != false {
		t.Errorf("response = %+v", body)
	}
	if got := contentsOf(t, node, 4096); !bytes.Equal(got, make([]byte, 4096)) {
		t.Error("the system disk was written to")
	}
}

func TestConfirmationIsCheckedByTheServerNotThePage(t *testing.T) {
	h := newHarness(t)
	node := h.drive("sdb", 64<<20, true, "stick")
	image := h.image("test.iso", 1<<20)

	for _, confirm := range []string{"", "yes", "sdc", "/dev/sdb"} {
		res, body := h.do("POST", "/api/write", map[string]any{
			"device": "sdb", "image": image, "confirm": confirm,
		})
		if res.StatusCode == http.StatusOK {
			t.Fatalf("a write went ahead on confirmation %q", confirm)
		}
		if !strings.Contains(fmt.Sprint(body["error"]), "naming the drive") {
			t.Errorf("confirm %q: unhelpful error %v", confirm, body["error"])
		}
	}
	if got := contentsOf(t, node, 4096); !bytes.Equal(got, make([]byte, 4096)) {
		t.Error("the drive was written to without confirmation")
	}
}

func TestAFixedDiskAndAMountedDriveEachNeedPermission(t *testing.T) {
	h := newHarness(t)
	h.drive("sdb", 64<<20, false, "Internal disk")
	h.drive("sdc", 64<<20, true, "stick")
	h.mount("sdc", "/media/user/DATA")
	image := h.image("test.iso", 1<<20)

	res, body := h.do("POST", "/api/write", map[string]any{"device": "sdb", "image": image, "confirm": "sdb"})
	if res.StatusCode != http.StatusForbidden || body["reason"] != "fixed_disk" {
		t.Errorf("fixed disk: status %d, body %+v", res.StatusCode, body)
	}
	if body["overridable"] != true {
		t.Error("a fixed disk should be reported as overridable, so the page can offer the choice")
	}

	res, body = h.do("POST", "/api/write", map[string]any{"device": "sdc", "image": image, "confirm": "sdc"})
	if res.StatusCode != http.StatusForbidden || body["reason"] != "mounted" {
		t.Errorf("mounted drive: status %d, body %+v", res.StatusCode, body)
	}

	// With permission, the fixed disk is allowed.
	res, _ = h.do("POST", "/api/write", map[string]any{
		"device": "sdb", "image": image, "confirm": "sdb", "allow_fixed": true,
	})
	if res.StatusCode != http.StatusOK {
		t.Errorf("permission did not permit the write: %d", res.StatusCode)
	}
}

/* ---- doing the work ---------------------------------------------------- */

func TestWritingThroughTheAPIPutsTheImageOnTheDriveAndVerifiesIt(t *testing.T) {
	h := newHarness(t)
	node := h.drive("sdb", 64<<20, true, "stick")
	image := h.image("test.iso", 2<<20)

	res, job := h.do("POST", "/api/write", map[string]any{
		"device": "sdb", "image": image, "confirm": "sdb", "verify": true,
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %+v", res.StatusCode, job)
	}
	final := h.await(job["id"].(string))

	if final["error"] != nil {
		t.Fatalf("the write failed: %v", final["error"])
	}
	if final["verified"] != true {
		t.Errorf("the write was not verified: %+v", final)
	}
	if digest, _ := final["digest"].(string); len(digest) != 64 {
		t.Errorf("no digest reported: %v", final["digest"])
	}
	if !strings.Contains(fmt.Sprint(final["summary"]), "read back and compared") {
		t.Errorf("summary = %v", final["summary"])
	}

	want, _ := os.ReadFile(image)
	if got := contentsOf(t, node, len(want)); !bytes.Equal(got, want) {
		t.Error("the drive does not hold the image")
	}
}

func TestProgressIsStreamedUntilTheJobEnds(t *testing.T) {
	h := newHarness(t)
	h.drive("sdb", 64<<20, true, "stick")
	image := h.image("test.iso", 4<<20)

	_, job := h.do("POST", "/api/write", map[string]any{
		"device": "sdb", "image": image, "confirm": "sdb", "verify": true,
	})
	id := job["id"].(string)

	// The page's EventSource cannot set a header, so the stream takes its
	// token in the query string.
	res, err := http.Get(h.http.URL + "/api/jobs/" + id + "/events?t=" + h.server.Token)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type = %q", ct)
	}

	var states []map[string]any
	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line, ok := strings.CutPrefix(scanner.Text(), "data: ")
		if !ok {
			continue
		}
		var st map[string]any
		if err := json.Unmarshal([]byte(line), &st); err != nil {
			t.Fatalf("stream sent something that is not JSON: %q", line)
		}
		states = append(states, st)
	}
	if len(states) == 0 {
		t.Fatal("the stream said nothing")
	}
	last := states[len(states)-1]
	if last["finished"] != true {
		t.Errorf("the stream ended before the job did: %+v", last)
	}
	if last["error"] != nil {
		t.Errorf("job failed: %v", last["error"])
	}
	// The stream must close itself once the job is done, or a page would hold
	// the connection open for ever. Reaching here at all proves it did.
}

func TestOneJobPerDriveAtATime(t *testing.T) {
	h := newHarness(t)
	h.drive("sdb", 192<<20, true, "stick")
	image := h.image("test.iso", 128<<20)

	res, first := h.do("POST", "/api/write", map[string]any{"device": "sdb", "image": image, "confirm": "sdb"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("first write refused: %+v", first)
	}
	res, second := h.do("POST", "/api/erase", map[string]any{"device": "sdb", "confirm": "sdb"})
	if res.StatusCode != http.StatusConflict {
		t.Errorf("a second job on the same drive returned %d: %+v", res.StatusCode, second)
	}
	if !strings.Contains(fmt.Sprint(second["error"]), "already being") {
		t.Errorf("error = %v", second["error"])
	}
	h.await(first["id"].(string))
}

func TestCancellingStopsTheWork(t *testing.T) {
	h := newHarness(t)
	h.drive("sdb", 192<<20, true, "stick")
	image := h.image("big.iso", 128<<20)

	_, job := h.do("POST", "/api/write", map[string]any{"device": "sdb", "image": image, "confirm": "sdb"})
	id := job["id"].(string)
	if res, _ := h.do("POST", "/api/jobs/"+id+"/cancel", nil); res.StatusCode != http.StatusOK {
		t.Fatalf("cancel returned %d", res.StatusCode)
	}
	final := h.await(id)
	if final["error"] == nil {
		t.Error("a cancelled write reported success")
	}
}

func TestEraseLeavesANamedVolumeAndRefusesAnImpossibleName(t *testing.T) {
	h := newHarness(t)
	node := h.drive("sdb", 512<<20, true, "stick")

	res, body := h.do("POST", "/api/erase", map[string]any{
		"device": "sdb", "confirm": "sdb", "name": "far too long a name",
	})
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("an impossible volume name was accepted: %d %+v", res.StatusCode, body)
	}
	if got := contentsOf(t, node, 4096); !bytes.Equal(got, make([]byte, 4096)) {
		t.Error("the drive was erased before the name was checked")
	}

	_, job := h.do("POST", "/api/erase", map[string]any{"device": "sdb", "confirm": "sdb", "name": "work"})
	final := h.await(job["id"].(string))
	if final["error"] != nil {
		t.Fatalf("erase failed: %v", final["error"])
	}
	if !strings.Contains(fmt.Sprint(final["summary"]), `FAT32 "WORK"`) {
		t.Errorf("summary = %v", final["summary"])
	}
	part := contentsOf(t, node, (2048+1)*512)[2048*512:]
	if string(part[82:90]) != "FAT32   " || string(part[71:82]) != "WORK       " {
		t.Errorf("the volume is %q labelled %q", part[82:90], part[71:82])
	}
}

func TestFormatMakesAVolume(t *testing.T) {
	h := newHarness(t)
	node := h.drive("sdb", 512<<20, true, "stick")

	_, job := h.do("POST", "/api/format", map[string]any{"device": "sdb", "confirm": "sdb", "name": "backup"})
	final := h.await(job["id"].(string))
	if final["error"] != nil {
		t.Fatalf("format failed: %v", final["error"])
	}
	if got := imaging.Probe(node); got.Type != "partitioned" {
		t.Errorf("the drive probes as %v", got)
	}
}

/* ---- looking around ---------------------------------------------------- */

func TestBrowseShowsFoldersAndImagesAndNothingElse(t *testing.T) {
	h := newHarness(t)
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(h.root, "Downloads"), 0o755))
	must(os.WriteFile(filepath.Join(h.root, "ubuntu.iso"), []byte("x"), 0o644))
	must(os.WriteFile(filepath.Join(h.root, "notes.txt"), []byte("x"), 0o644))
	must(os.WriteFile(filepath.Join(h.root, ".hidden.iso"), []byte("x"), 0o644))

	_, body := h.do("GET", "/api/browse", nil)
	entries, _ := body["entries"].([]any)
	names := map[string]bool{}
	for _, e := range entries {
		names[e.(map[string]any)["name"].(string)] = true
	}
	if !names["Downloads"] || !names["ubuntu.iso"] {
		t.Errorf("the folder or the image is missing: %v", names)
	}
	if names["notes.txt"] {
		t.Error("a file that is not an image was offered")
	}
	if names[".hidden.iso"] {
		t.Error("a hidden file was offered")
	}
	if body["dir"] != h.root {
		t.Errorf("dir = %v, want the home directory", body["dir"])
	}
}

func TestAnImageIsDescribedBeforeItIsWritten(t *testing.T) {
	h := newHarness(t)
	path := filepath.Join(h.root, "test.iso")
	img := make([]byte, 24*2048)
	pvd := img[16*2048:]
	pvd[0] = 1
	copy(pvd[1:], "CD001")
	copy(pvd[40:72], "TEST VOLUME                     ")
	if err := os.WriteFile(path, img, 0o644); err != nil {
		t.Fatal(err)
	}

	_, body := h.do("GET", "/api/image?path="+path, nil)
	if body["format"] != "ISO 9660 image" || body["label"] != "TEST VOLUME" {
		t.Errorf("image = %+v", body)
	}
	if warnings, _ := body["warnings"].([]any); len(warnings) == 0 {
		t.Error("an ISO with no partition table and no boot catalogue drew no warning")
	}

	res, _ := h.do("GET", "/api/image?path="+filepath.Join(h.root, "nothing.iso"), nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("a missing image returned %d", res.StatusCode)
	}
}

/* ---- the page as a file ------------------------------------------------ */

func TestAPageOpenedFromAFileCanTalkToTheServer(t *testing.T) {
	h := newHarness(t)
	h.drive("sdb", 64<<20, true, "stick")

	// Every browser labels a page loaded from disk "null". That is the saved
	// copy of this same page, and it must work.
	req, _ := http.NewRequest("GET", h.http.URL+"/api/drives", nil)
	req.Header.Set("X-Oliver-Token", h.server.Token)
	req.Header.Set("Origin", "null")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	// Without this header the browser will not let the page read the answer.
	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "null" {
		t.Errorf("Access-Control-Allow-Origin = %q, want \"null\"", got)
	}
	if !strings.Contains(res.Header.Get("Vary"), "Origin") {
		t.Errorf("the answer does not vary by origin: %q", res.Header.Get("Vary"))
	}
}

func TestThePermissionQuestionIsAnsweredForAFileAndRefusedForASite(t *testing.T) {
	h := newHarness(t)

	// A preflight carries no token; it is asking permission to send one.
	req, _ := http.NewRequest("OPTIONS", h.http.URL+"/api/write", nil)
	req.Header.Set("Origin", "null")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "x-oliver-token")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("preflight from a file = %d, want 204", res.StatusCode)
	}
	if !strings.Contains(strings.ToLower(res.Header.Get("Access-Control-Allow-Headers")), "x-oliver-token") {
		t.Errorf("the token header was not permitted: %q", res.Header.Get("Access-Control-Allow-Headers"))
	}

	// A site, though, is refused the permission outright — which is what stops
	// a page you happen to be visiting from even trying.
	req, _ = http.NewRequest("OPTIONS", h.http.URL+"/api/write", nil)
	req.Header.Set("Origin", "https://example.invalid")
	req.Header.Set("Access-Control-Request-Method", "POST")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("preflight from a site = %d, want 403", res.StatusCode)
	}
	if res.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Error("a site was told it may send requests")
	}
}

func TestAFileWithoutTheTokenStillGetsNothing(t *testing.T) {
	h := newHarness(t)
	node := h.drive("sdb", 64<<20, true, "stick")

	// Origin "null" is not a way in: it only says where the page came from.
	req, _ := http.NewRequest("POST", h.http.URL+"/api/erase",
		strings.NewReader(`{"device":"sdb","confirm":"sdb"}`))
	req.Header.Set("Origin", "null")
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", res.StatusCode)
	}
	if got := contentsOf(t, node, 512); !bytes.Equal(got, make([]byte, 512)) {
		t.Error("the drive was erased")
	}
}

func TestTheSavedPageAsksForAnAddressInsteadOfAssumingOne(t *testing.T) {
	page := Page()

	// Saved to a file, the placeholder is still in it: that is how the page
	// knows it was not served, and must ask where Oliver is.
	if !strings.Contains(page, tokenPlaceholder) {
		t.Error("the saved page carries no placeholder, so it cannot tell it was opened on its own")
	}
	for _, needed := range []string{`id="connect"`, "parseLink", "oliver ui"} {
		if !strings.Contains(page, needed) {
			t.Errorf("the saved page has no %s, so there is no way to point it at a running Oliver", needed)
		}
	}
	// Nothing may be fetched from anywhere else: this has to work offline.
	for _, forbidden := range []string{"http://fonts.", "https://fonts.", "cdn.", "<script src=", "<link rel=\"stylesheet\""} {
		if strings.Contains(page, forbidden) {
			t.Errorf("the page loads something external (%s); it must be self-contained", forbidden)
		}
	}
}

/* ---- write protection -------------------------------------------------- */

// protect marks a drive read-only the way the kernel does.
func (h *harness) protect(name string) {
	h.t.Helper()
	path := filepath.Join(h.root, "sys", "block", name, "ro")
	if err := os.WriteFile(path, []byte("1\n"), 0o644); err != nil {
		h.t.Fatal(err)
	}
}

func TestADriveThatWillNotTakeWritesSaysWhyOnItsCard(t *testing.T) {
	h := newHarness(t)
	h.drive("sdb", 64<<20, true, "Cruzer")
	h.protect("sdb")

	drives := h.list("/api/drives")
	protections, _ := drives[0]["protections"].([]any)
	if len(protections) != 1 {
		t.Fatalf("the card says nothing about it: %+v", drives[0])
	}
	p := protections[0].(map[string]any)
	if p["kind"] != "kernel-flag" || p["clearable"] != true {
		t.Errorf("protection = %+v", p)
	}
}

func TestClearingTheFlagNeedsNoConfirmationBecauseItChangesNoData(t *testing.T) {
	h := newHarness(t)
	node := h.drive("sdb", 64<<20, true, "Cruzer")
	h.protect("sdb")
	before := contentsOf(t, node, 512)

	// No confirm field: unprotecting destroys nothing, so demanding the
	// ritual for a destructive call would be theatre.
	// The ioctl behind this only answers for a real block device, and these
	// drives are files.
	original := device.ClearReadOnly
	t.Cleanup(func() { device.ClearReadOnly = original })
	var cleared []string
	device.ClearReadOnly = func(path string) error {
		cleared = append(cleared, path)
		return nil
	}

	res, body := h.do("POST", "/api/unprotect", map[string]any{"device": "sdb"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d: %+v", res.StatusCode, body)
	}
	if len(cleared) != 1 || cleared[0] != node {
		t.Errorf("cleared %v, want the drive itself", cleared)
	}
	if !bytes.Equal(contentsOf(t, node, 512), before) {
		t.Error("unprotecting wrote to the drive")
	}
}

func TestTheSystemDiskCannotBeUnprotectedEither(t *testing.T) {
	h := newHarness(t)
	h.drive("sda", 64<<20, false, "System SSD")
	h.mount("sda", "/")

	res, body := h.do("POST", "/api/unprotect", map[string]any{"device": "sda"})
	if res.StatusCode != http.StatusForbidden || body["reason"] != "system_disk" {
		t.Errorf("status %d, body %+v", res.StatusCode, body)
	}
}

func TestAnUnprotectedDriveReportsNothingInTheWay(t *testing.T) {
	h := newHarness(t)
	h.drive("sdb", 64<<20, true, "Cruzer")

	drives := h.list("/api/drives")
	if p, ok := drives[0]["protections"]; ok && p != nil {
		t.Errorf("a writable drive was reported as protected: %+v", p)
	}
}
