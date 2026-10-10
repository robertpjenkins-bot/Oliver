package design

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
)

func TestSHA512CryptKnownVectors(t *testing.T) {
	// The first is from the SHA-crypt specification; the second is openssl passwd -6.
	cases := []struct{ pw, salt, want string }{
		{"Hello world!", "saltstring", "$6$saltstring$svn8UoSVapNtMuq1ukKS4tPQd8iKwSMHWjl/O817G3uBnIFNjnQJuesI68u4OTLiBFdcbYEdFCoEOfaS35inz1"},
		{"p@ss word!", "abcdefgh", "$6$abcdefgh$jYf0uL6uQ0chIH4jx.NO0aiMu4WhHL9U.vVCZDcNJVs8sSVZBUhu.OmoOcrP8Us8MCpOQo8FJuaXJEaRB0B4H/"},
	}
	for _, c := range cases {
		if got := sha512Crypt(c.pw, c.salt); got != c.want {
			t.Errorf("sha512Crypt(%q, %q) = %s, want %s", c.pw, c.salt, got, c.want)
		}
	}
}

func TestHashPasswordShape(t *testing.T) {
	h := HashPassword("hunter2")
	if !reHash.MatchString(h) {
		t.Fatalf("hash %q does not look like a shadow hash", h)
	}
	if HashPassword("hunter2") == h {
		t.Fatal("two hashes of the same password share a salt")
	}
}

func TestDefaultIsValid(t *testing.T) {
	d := Default()
	d.Normalize()
	if err := d.Validate(); err != nil {
		t.Fatalf("default design is invalid: %v", err)
	}
}

func TestNormalizeInstallsPlacedAndPinnedApps(t *testing.T) {
	d := Default()
	d.Apps = nil
	d.Taskbar.Pinned = []string{"steam", "files"}
	d.Items = append(d.Items, Item{ID: "x", Type: "app", App: "gimp", X: 0, Y: 0, W: 50, H: 50})
	d.Normalize()
	for _, want := range []string{"steam", "gimp", "firefox"} {
		if !slices.Contains(d.Apps, want) {
			t.Errorf("apps %v missing %s", d.Apps, want)
		}
	}
	if slices.Contains(d.Apps, "files") {
		t.Errorf("base app listed in apps: %v", d.Apps)
	}
	if !d.NeedsMultilib() {
		t.Error("steam should need multilib")
	}
}

func TestValidateCatchesBadInput(t *testing.T) {
	cases := map[string]func(d *Design){
		"username":  func(d *Design) { d.System.Username = "Root Beer" },
		"reserved":  func(d *Design) { d.System.Username = "root" },
		"hostname":  func(d *Design) { d.System.Hostname = "My PC" },
		"osname":    func(d *Design) { d.System.OSName = "a;rm -rf /" },
		"timezone":  func(d *Design) { d.System.Timezone = "../../etc/passwd" },
		"keyboard":  func(d *Design) { d.System.Keyboard = "klingon" },
		"color":     func(d *Design) { d.Items[0].Color = "red" },
		"offscreen": func(d *Design) { d.Items[0].X = 1270 },
		"app": func(d *Design) {
			d.Items = append(d.Items, Item{ID: "z", Type: "app", App: "nope", W: 20, H: 20, Color: "#000000", TextColor: "#000000", FontSize: 12})
		},
		"nologin":   func(d *Design) { d.System.Autologin = false },
		"menulabel": func(d *Design) { d.Taskbar.MenuLabel = "two\nlines" },
		"fullname":  func(d *Design) { d.System.FullName = "a:b" },
		"image":     func(d *Design) { d.Desktop.Background = Background{Kind: "image", Image: "data:text/html;base64,AA=="} },
	}
	for name, mutate := range cases {
		d := Default()
		d.Normalize()
		mutate(d)
		if err := d.Validate(); err == nil {
			t.Errorf("%s: expected a validation error", name)
		}
	}
}

func TestParseRoundTrip(t *testing.T) {
	b, _ := json.Marshal(Default())
	d, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if d.System.OSName != "My OS" {
		t.Fatalf("got %+v", d.System)
	}
	if _, err := Parse([]byte("hello")); err == nil || !strings.Contains(err.Error(), "not an OS Maker design") {
		t.Fatalf("garbage parsed: %v", err)
	}
}

func TestCatalogIsConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range Catalog {
		if seen[a.ID] {
			t.Errorf("duplicate id %s", a.ID)
		}
		seen[a.ID] = true
		if !slices.Contains(Categories, a.Category) {
			t.Errorf("%s: unknown category %s", a.ID, a.Category)
		}
		if len(a.Packages) == 0 || a.Exec == "" || !reColor.MatchString(a.Color) {
			t.Errorf("%s: incomplete entry", a.ID)
		}
	}
}

func TestCatalogServicesAreUnits(t *testing.T) {
	for _, a := range Catalog {
		for _, svc := range a.Services {
			if !strings.HasSuffix(svc, ".service") || strings.ContainsAny(svc, "/ ") {
				t.Errorf("%s: bad service %q", a.ID, svc)
			}
		}
	}
}
