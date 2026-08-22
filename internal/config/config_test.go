package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	c, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.Profile != "auto" || c.WinVer != "auto" {
		t.Errorf("defaults not applied: %+v", c)
	}
}

func TestLoadRejectsMalformedConfig(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte("{ not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A typo must be reported, not silently ignored: quietly falling back to
	// defaults would launch programs with settings the user did not choose.
	if _, err := Load(p); err == nil {
		t.Fatal("expected an error for a malformed configuration file")
	}
}

func TestSaveAndReload(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "config.json")
	c := Example()
	c.SetPath(p)
	c.Backend = "staging"
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}

	back, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if back.Backend != "staging" || len(back.Rules) != len(c.Rules) {
		t.Errorf("round-trip lost data: %+v", back)
	}
}

func TestRuleMatching(t *testing.T) {
	c := &Config{Rules: []Rule{
		{Match: "**/Photoshop*/*.exe", Backend: "staging"},
		{Match: "*.msi", Backend: "wine"},
		{Match: "Game.exe", Backend: "proton"},
	}}

	cases := []struct {
		path string
		want string
	}{
		{"/home/u/Adobe/Photoshop 2021/photoshop.exe", "staging"},
		{"/home/u/thing.msi", "wine"},
		{"/anywhere/at/all/Game.exe", "proton"},
		{"/home/u/other.exe", ""},
		// A bare-name pattern must not match a same-named file's directory.
		{"/home/u/Game.exe/notreally", ""},
	}
	for _, c2 := range cases {
		got := c.RuleFor(c2.path)
		switch {
		case c2.want == "" && got != nil:
			t.Errorf("%s unexpectedly matched %q", c2.path, got.Match)
		case c2.want != "" && got == nil:
			t.Errorf("%s matched nothing, want %q", c2.path, c2.want)
		case c2.want != "" && got != nil && got.Backend != c2.want:
			t.Errorf("%s matched %q, want backend %q", c2.path, got.Match, c2.want)
		}
	}
}

func TestFirstMatchingRuleWins(t *testing.T) {
	c := &Config{Rules: []Rule{
		{Match: "**/special/game.exe", Backend: "first"},
		{Match: "game.exe", Backend: "second"},
	}}
	if r := c.RuleFor("/x/special/game.exe"); r == nil || r.Backend != "first" {
		t.Errorf("expected the earlier rule to win, got %+v", r)
	}
}

func TestGlobMatchingIsCaseInsensitiveAndSegmentAware(t *testing.T) {
	// Windows users do not think about case, and `*` should not leak across
	// directory boundaries the way `**` does.
	if !matchGlob("SETUP.EXE", "/x/setup.exe") {
		t.Error("expected case-insensitive matching")
	}
	if matchGlob("/x/*.exe", "/x/y/z.exe") {
		t.Error("* must not cross a path separator")
	}
	if !matchGlob("/x/**/z.exe", "/x/y/deeper/z.exe") {
		t.Error("** should cross path separators")
	}
	if matchGlob("", "/x/a.exe") {
		t.Error("an empty pattern must not match")
	}
}
