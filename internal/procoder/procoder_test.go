package procoder

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecognise(t *testing.T) {
	cases := []struct {
		in       string
		kind     Kind
		play     string
		page     string
		fileName string
	}{
		{"https://scratch.mit.edu/projects/60917032/", KindScratch,
			"https://scratch.mit.edu/projects/60917032/embed",
			"https://scratch.mit.edu/projects/60917032", "scratch-60917032"},
		{"scratch.mit.edu/projects/60917032", KindScratch,
			"https://scratch.mit.edu/projects/60917032/embed",
			"https://scratch.mit.edu/projects/60917032", "scratch-60917032"},
		{"60917032", KindScratch,
			"https://scratch.mit.edu/projects/60917032/embed",
			"https://scratch.mit.edu/projects/60917032", "scratch-60917032"},
		{"https://turbowarp.org/60917032", KindTurboWarp,
			"https://turbowarp.org/60917032/embed",
			"https://turbowarp.org/60917032", "turbowarp-60917032"},
		{"https://arcade.makecode.com/_gVFuJf2Y3cVW", KindMakeCode,
			"https://arcade.makecode.com/---run?id=_gVFuJf2Y3cVW",
			"https://arcade.makecode.com/_gVFuJf2Y3cVW", "arcade-gVFuJf2Y3cVW"},
		{"https://makecode.com/_gVFuJf2Y3cVW", KindMakeCode,
			"https://arcade.makecode.com/---run?id=_gVFuJf2Y3cVW",
			"https://arcade.makecode.com/_gVFuJf2Y3cVW", "arcade-gVFuJf2Y3cVW"},
		{"https://arcade.makecode.com/62716-83606-45471-11832", KindMakeCode,
			"https://arcade.makecode.com/---run?id=62716-83606-45471-11832",
			"https://arcade.makecode.com/62716-83606-45471-11832", "arcade-62716-83606-45471-11832"},
		{"example.org/game.html", KindWeb,
			"https://example.org/game.html", "https://example.org/game.html", "example-org"},
	}
	for _, c := range cases {
		got, err := Recognise(c.in)
		if err != nil {
			t.Errorf("Recognise(%q): %v", c.in, err)
			continue
		}
		if got.Kind != c.kind {
			t.Errorf("Recognise(%q).Kind = %q, want %q", c.in, got.Kind, c.kind)
		}
		if got.Play != c.play {
			t.Errorf("Recognise(%q).Play = %q, want %q", c.in, got.Play, c.play)
		}
		if got.Page != c.page {
			t.Errorf("Recognise(%q).Page = %q, want %q", c.in, got.Page, c.page)
		}
		if got.Name() != c.fileName {
			t.Errorf("Recognise(%q).Name() = %q, want %q", c.in, got.Name(), c.fileName)
		}
	}
}

// A converted project ends up as an address inside a launcher script and a
// frame, so anything that is not a web page has to be turned away here rather
// than somewhere further down.
func TestRecogniseRejects(t *testing.T) {
	for _, in := range []string{"", "   ", "javascript:alert(1)", "data:text/html,<b>", "file:///etc/passwd"} {
		if got, err := Recognise(in); err == nil {
			t.Errorf("Recognise(%q) = %+v, want an error", in, got)
		}
	}
}

func TestWrapperHTML(t *testing.T) {
	target, err := Recognise("https://scratch.mit.edu/projects/60917032")
	if err != nil {
		t.Fatal(err)
	}
	page, err := WrapperHTML(target, `Ada's "game"`)
	if err != nil {
		t.Fatal(err)
	}
	html := string(page)
	if !strings.Contains(html, target.Play) {
		t.Errorf("wrapper does not play the project:\n%s", html)
	}
	if !strings.Contains(html, `href="https://scratch.mit.edu/projects/60917032"`) {
		t.Errorf("wrapper has no link back to the project:\n%s", html)
	}
	if strings.Contains(html, `Ada's "game"`) {
		t.Error("the name was written into the page without escaping")
	}
	if !strings.Contains(html, "Ada&#39;s") {
		t.Errorf("the name is missing from the page:\n%s", html)
	}
}

func TestServerServesTheApp(t *testing.T) {
	srv := &Server{}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:1/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET / = %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "Pro Coder") {
		t.Error("the page served is not Pro Coder")
	}
}

func TestServerCapabilitiesFollowThePacker(t *testing.T) {
	for _, withPacker := range []bool{false, true} {
		srv := &Server{}
		if withPacker {
			srv.Pack = func(PackRequest) (PackResult, error) { return PackResult{Path: "x"}, nil }
		}
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://localhost:1/api/capabilities", nil))
		var body struct {
			AppImage bool `json:"appimage"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.AppImage != withPacker {
			t.Errorf("appimage capability = %v with packer %v", body.AppImage, withPacker)
		}
	}
}

func TestServerConvert(t *testing.T) {
	var asked PackRequest
	srv := &Server{Pack: func(r PackRequest) (PackResult, error) {
		asked = r
		return PackResult{Path: "/tmp/thing.AppImage"}, nil
	}}

	post := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:1/api/convert", strings.NewReader(body))
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}

	rec := post(`{"url":"https://scratch.mit.edu/projects/60917032","name":"Game","format":"appimage"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("convert = %d: %s", rec.Code, rec.Body)
	}
	if asked.Name != "Game" || asked.URL == "" {
		t.Errorf("packer asked for %+v", asked)
	}

	// The page is not the only thing that can reach this handler, so the
	// address is checked again here.
	if rec := post(`{"url":"javascript:alert(1)","format":"appimage"}`); rec.Code != http.StatusBadRequest {
		t.Errorf("converting a javascript: URL = %d, want 400", rec.Code)
	}
	if rec := post(`not json`); rec.Code != http.StatusBadRequest {
		t.Errorf("converting nonsense = %d, want 400", rec.Code)
	}
}

// Only this machine may drive Pro Coder: the listener is on loopback, and a
// page elsewhere pointing a name at 127.0.0.1 is turned away by the Host it
// sends.
func TestServerRefusesOtherHosts(t *testing.T) {
	srv := &Server{}
	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:1/", nil)
	req.Host = "procoder.example.com"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("request from another host = %d, want 403", rec.Code)
	}
}

func TestListenIsLoopbackOnly(t *testing.T) {
	ln, addr, err := Listen(0)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	if !strings.HasPrefix(addr, "http://127.0.0.1:") {
		t.Errorf("Listen gave %q, want a loopback address", addr)
	}
}
