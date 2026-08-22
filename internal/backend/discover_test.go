package backend

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeSystem builds a synthetic install tree covering the runner layouts we
// claim to support, so discovery is tested without touching the real machine.
type fakeSystem struct {
	t    *testing.T
	root string
}

func newFakeSystem(t *testing.T) *fakeSystem {
	t.Helper()
	return &fakeSystem{t: t, root: t.TempDir()}
}

func (f *fakeSystem) path(parts ...string) string {
	return filepath.Join(append([]string{f.root}, parts...)...)
}

func (f *fakeSystem) mkdir(parts ...string) string {
	f.t.Helper()
	p := f.path(parts...)
	if err := os.MkdirAll(p, 0o755); err != nil {
		f.t.Fatalf("mkdir %s: %v", p, err)
	}
	return p
}

func (f *fakeSystem) write(content string, parts ...string) string {
	f.t.Helper()
	p := f.path(parts...)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		f.t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		f.t.Fatalf("write %s: %v", p, err)
	}
	return p
}

func (f *fakeSystem) exe(parts ...string) string {
	f.t.Helper()
	p := f.write("#!/bin/sh\n", parts...)
	if err := os.Chmod(p, 0o755); err != nil {
		f.t.Fatalf("chmod: %v", err)
	}
	return p
}

// wineTree lays down a wine installation with the given bitness support.
func (f *fakeSystem) wineTree(base []string, has32, has64 bool) {
	f.t.Helper()
	f.exe(append(append([]string{}, base...), "bin", "wine")...)
	if has32 {
		f.mkdir(append(append([]string{}, base...), "lib", "wine", "i386-windows")...)
		f.mkdir(append(append([]string{}, base...), "lib", "wine", "i386-unix")...)
	}
	if has64 {
		f.mkdir(append(append([]string{}, base...), "lib", "wine", "x86_64-windows")...)
		f.mkdir(append(append([]string{}, base...), "lib", "wine", "x86_64-unix")...)
	}
}

// protonTree lays down a Proton installation.
func (f *fakeSystem) protonTree(base []string, versionLine string) {
	f.t.Helper()
	f.exe(append(append([]string{}, base...), "proton")...)
	f.write(versionLine, append(append([]string{}, base...), "version")...)
	f.mkdir(append(append([]string{}, base...), "files", "lib", "wine", "i386-unix")...)
	f.mkdir(append(append([]string{}, base...), "files", "lib", "wine", "x86_64-unix")...)
}

func (f *fakeSystem) scanner() *Scanner {
	return &Scanner{
		Home:        f.path("home", "user"),
		PathDirs:    []string{f.path("usr", "bin"), f.path("home", "user", ".local", "bin")},
		SystemRoots: []string{f.path("opt")},
		Probe:       false,
		HostArch:    "amd64",
	}
}

func byID(bs []*Backend) map[string]*Backend {
	m := make(map[string]*Backend, len(bs))
	for _, b := range bs {
		m[b.ID] = b
	}
	return m
}

func TestDiscoverFindsEveryRunnerLayout(t *testing.T) {
	f := newFakeSystem(t)
	home := []string{"home", "user"}

	// System wine on PATH, 64-bit only (the modern wow64-less packaging).
	f.wineTree([]string{"usr"}, false, true)
	f.exe("usr", "bin", "wine")

	// Steam with Proton in the default library plus a second library on
	// another disk, referenced from libraryfolders.vdf.
	steam := append(append([]string{}, home...), ".local", "share", "Steam")
	f.protonTree(append(append([]string{}, steam...), "steamapps", "common", "Proton 9.0"), "1700000000 proton-9.0-4c")
	f.protonTree(append(append([]string{}, steam...), "compatibilitytools.d", "GE-Proton9-20"), "1710000000 GE-Proton9-20")
	f.protonTree([]string{"mnt", "games", "SteamLibrary", "steamapps", "common", "Proton Experimental"}, "1720000000 experimental")
	f.write(`"libraryfolders"
{
	"0"
	{
		"path"		"`+f.path(steam...)+`"
	}
	"1"
	{
		"path"		"`+f.path("mnt", "games", "SteamLibrary")+`"
	}
}`, append(append([]string{}, steam...), "steamapps", "libraryfolders.vdf")...)

	// Third-party managers.
	f.wineTree(append(append([]string{}, home...), ".local", "share", "lutris", "runners", "wine", "wine-ge-8-26"), true, true)
	f.wineTree(append(append([]string{}, home...), ".local", "share", "bottles", "runners", "soda-9.0-1"), true, true)
	f.wineTree(append(append([]string{}, home...), ".config", "heroic", "tools", "wine", "Wine-GE-Proton8-26"), true, true)
	f.wineTree(append(append([]string{}, home...), ".PlayOnLinux", "wine", "linux-amd64", "7.0"), true, true)
	f.wineTree([]string{"opt", "wine-staging"}, true, true)
	f.exe("opt", "cxoffice", "bin", "wine")

	got := f.scanner().Discover()
	m := byID(got)

	want := []string{
		"system-wine",
		"proton-proton-9-0",
		"proton-ge-proton9-20",
		"proton-proton-experimental",
		"lutris-wine-ge-8-26",
		"bottles-soda-9-0-1",
		"heroic-wine-ge-proton8-26",
		"pol-linux-amd64-7-0",
		"opt-wine-staging",
		"crossover-cxoffice",
	}
	for _, id := range want {
		if _, ok := m[id]; !ok {
			t.Errorf("discovery missed %s; found: %v", id, keys(m))
		}
	}
}

func keys(m map[string]*Backend) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestDiscoverDetectsMissing32BitSupport(t *testing.T) {
	f := newFakeSystem(t)
	f.wineTree([]string{"usr"}, false, true) // 64-bit only
	f.exe("usr", "bin", "wine")

	b := byID(f.scanner().Discover())["system-wine"]
	if b == nil {
		t.Fatal("system wine not discovered")
	}
	if b.Has32 {
		t.Error("expected 32-bit support to be detected as absent")
	}
	if !b.Has64 {
		t.Error("expected 64-bit support to be detected as present")
	}
	if b.Bitness() != "64-only" {
		t.Errorf("Bitness() = %q, want 64-only", b.Bitness())
	}
}

func TestProtonNotedWhenUmuMissingAndUsedWhenPresent(t *testing.T) {
	f := newFakeSystem(t)
	steam := []string{"home", "user", ".local", "share", "Steam"}
	f.protonTree(append(append([]string{}, steam...), "compatibilitytools.d", "GE-Proton9-20"), "1710000000 GE-Proton9-20")

	b := byID(f.scanner().Discover())["proton-ge-proton9-20"]
	if b == nil {
		t.Fatal("proton not discovered")
	}
	if len(b.Launcher) != 0 {
		t.Errorf("expected no launcher without umu, got %v", b.Launcher)
	}

	f.exe("usr", "bin", "umu-run")
	b = byID(f.scanner().Discover())["proton-ge-proton9-20"]
	if len(b.Launcher) == 0 || filepath.Base(b.Launcher[0]) != "umu-run" {
		t.Errorf("expected umu-run launcher, got %v", b.Launcher)
	}
}

func TestVersionParsingAndOrdering(t *testing.T) {
	cases := []struct {
		in     string
		fields []int
	}{
		{"wine-9.0", []int{9, 0}},
		{"wine-8.0.2 (Staging)", []int{8, 0, 2}},
		{"GE-Proton9-20", []int{9, 20}},
		{"Proton 9.0", []int{9, 0}},
		{"proton-9.0-4c", []int{9, 0, 4}},
		{"Proton Experimental", nil},
	}
	for _, c := range cases {
		got := ParseVersion(c.in)
		if len(got.Fields) != len(c.fields) {
			t.Errorf("ParseVersion(%q).Fields = %v, want %v", c.in, got.Fields, c.fields)
			continue
		}
		for i := range c.fields {
			if got.Fields[i] != c.fields[i] {
				t.Errorf("ParseVersion(%q).Fields = %v, want %v", c.in, got.Fields, c.fields)
				break
			}
		}
	}

	if ParseVersion("wine-9.0").Compare(ParseVersion("wine-8.21")) <= 0 {
		t.Error("9.0 should sort newer than 8.21")
	}
	if ParseVersion("GE-Proton9-20").Compare(ParseVersion("GE-Proton9-4")) <= 0 {
		t.Error("GE-Proton9-20 should sort newer than GE-Proton9-4")
	}
}

func TestDetectVariant(t *testing.T) {
	cases := map[string]Variant{
		"wine-staging":        VariantStaging,
		"GE-Proton9-20":       VariantGE,
		"Proton Experimental": VariantExperimental,
		"wine-9.0":            VariantStable,
		"wine-tkg-9.1":        VariantTKG,
		"wine-8.26-ge-1":      VariantGE,
	}
	for in, want := range cases {
		if got := DetectVariant(in); got != want {
			t.Errorf("DetectVariant(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFindBackendByFuzzyName(t *testing.T) {
	bs := []*Backend{
		{ID: "system-wine", Name: "wine"},
		{ID: "proton-ge-proton9-20", Name: "GE-Proton9-20"},
		{ID: "opt-wine-staging", Name: "wine-staging"},
	}
	if b, err := Find(bs, "ge"); err != nil || b.ID != "proton-ge-proton9-20" {
		t.Errorf("Find(ge) = %v, %v", b, err)
	}
	if b, err := Find(bs, "staging"); err != nil || b.ID != "opt-wine-staging" {
		t.Errorf("Find(staging) = %v, %v", b, err)
	}
	// "wine" is an exact name match on system-wine even though it is also a
	// substring of wine-staging.
	if b, err := Find(bs, "wine"); err != nil || b.ID != "system-wine" {
		t.Errorf("Find(wine) = %v, %v", b, err)
	}
	if _, err := Find(bs, "nonexistent"); err == nil {
		t.Error("expected an error for an unknown backend")
	}
}
