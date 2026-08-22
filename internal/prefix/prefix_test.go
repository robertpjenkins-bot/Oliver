package prefix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNameForIsStableAndCollisionFree(t *testing.T) {
	a := NameFor("/games/Cool Game/game.exe")
	b := NameFor("/games/Cool Game/launcher.exe")
	if a != b {
		t.Errorf("executables in one directory should share a prefix: %q vs %q", a, b)
	}
	if NameFor("/games/Cool Game/game.exe") != a {
		t.Error("prefix names must be stable across calls")
	}

	// Two different installs that happen to share a folder name must not
	// collide, or one program would inherit the other's environment.
	c := NameFor("/other/Cool Game/game.exe")
	if a == c {
		t.Errorf("distinct installs collided on %q", a)
	}

	// A loose download is named after the program, not the folder.
	d := NameFor("/home/u/Downloads/Setup.exe")
	if got := d[:len(d)-7]; got != "setup" {
		t.Errorf("expected a program-derived name, got %q", d)
	}
}

func TestNameForSanitisesAwkwardNames(t *testing.T) {
	n := NameFor("/games/Ünicode & Spaces!/app.exe")
	for _, r := range n {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '.', r == '_':
		default:
			t.Fatalf("unsafe character %q in prefix name %q", r, n)
		}
	}
}

func TestForHonoursExplicitNamesAndPaths(t *testing.T) {
	m := &Manager{Root: t.TempDir()}

	p := m.For("/games/x/game.exe", "mybottle")
	if p.Name != "mybottle" || filepath.Dir(p.Dir) != m.Root {
		t.Errorf("named prefix resolved to %+v", p)
	}
	if p.WinePrefix != filepath.Join(p.Dir, "pfx") {
		t.Errorf("WinePrefix = %q, want <dir>/pfx", p.WinePrefix)
	}

	abs := filepath.Join(t.TempDir(), "elsewhere")
	p = m.For("/games/x/game.exe", abs)
	if p.Dir != abs {
		t.Errorf("absolute prefix path ignored: %q", p.Dir)
	}
}

func TestSandboxReplacesHomeSymlinksWithRealDirectories(t *testing.T) {
	m := &Manager{Root: t.TempDir()}
	p := m.For("/games/x/game.exe", "test")
	if err := p.Prepare(); err != nil {
		t.Fatal(err)
	}

	// Recreate what Wine does: link the prefix's user folders at the real ones.
	realHome := t.TempDir()
	realDocs := filepath.Join(realHome, "Documents")
	if err := os.MkdirAll(realDocs, 0o755); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(realDocs, "taxes.txt")
	if err := os.WriteFile(secret, []byte("private"), 0o644); err != nil {
		t.Fatal(err)
	}

	userDir := filepath.Join(p.WinePrefix, "drive_c", "users", "tester")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"Documents", "Desktop", "Downloads"} {
		target := realHome
		if d == "Documents" {
			target = realDocs
		}
		if err := os.Symlink(target, filepath.Join(userDir, d)); err != nil {
			t.Fatal(err)
		}
	}

	isolated, err := p.Sandbox()
	if err != nil {
		t.Fatalf("Sandbox: %v", err)
	}
	if len(isolated) != 3 {
		t.Errorf("isolated %v, want 3 folders", isolated)
	}

	for _, d := range []string{"Documents", "Desktop", "Downloads"} {
		target := filepath.Join(userDir, d)
		st, err := os.Lstat(target)
		if err != nil {
			t.Fatalf("%s missing after sandboxing: %v", d, err)
		}
		if st.Mode()&os.ModeSymlink != 0 {
			t.Errorf("%s is still a symlink into the real home directory", d)
		}
		if !st.IsDir() {
			t.Errorf("%s is not a directory", d)
		}
	}

	// The real file must be unreachable from inside the prefix now.
	if _, err := os.Stat(filepath.Join(userDir, "Documents", "taxes.txt")); err == nil {
		t.Error("the prefix can still reach the real Documents folder")
	}
	// ...and untouched on disk.
	if _, err := os.Stat(secret); err != nil {
		t.Errorf("sandboxing damaged the real file: %v", err)
	}
}

func TestSandboxLeavesRealDirectoriesAlone(t *testing.T) {
	m := &Manager{Root: t.TempDir()}
	p := m.For("/games/x/game.exe", "test")
	userDir := filepath.Join(p.WinePrefix, "drive_c", "users", "tester")
	docs := filepath.Join(userDir, "Documents")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(docs, "savegame.dat")
	if err := os.WriteFile(keep, []byte("progress"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := p.Sandbox(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Errorf("sandboxing destroyed existing prefix data: %v", err)
	}
}

func TestSandboxOnMissingPrefixIsHarmless(t *testing.T) {
	m := &Manager{Root: t.TempDir()}
	p := m.For("/games/x/game.exe", "absent")
	if got, err := p.Sandbox(); err != nil || got != nil {
		t.Errorf("Sandbox on a fresh prefix = %v, %v", got, err)
	}
}

func TestRemoveRefusesToEscapeTheRoot(t *testing.T) {
	root := t.TempDir()
	m := &Manager{Root: root}

	outside := filepath.Join(filepath.Dir(root), "important")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(outside) })

	for _, bad := range []string{"..", "../important", "/etc", "", "a/b"} {
		if err := m.Remove(bad); err == nil {
			t.Errorf("Remove(%q) was allowed", bad)
		}
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatal("Remove escaped the prefix root")
	}

	p := m.For("/games/x/game.exe", "doomed")
	if err := p.Prepare(); err != nil {
		t.Fatal(err)
	}
	if err := m.Remove("doomed"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if _, err := os.Stat(p.Dir); !os.IsNotExist(err) {
		t.Error("prefix was not removed")
	}
}

func TestMetaRoundTripAndListing(t *testing.T) {
	m := &Manager{Root: t.TempDir()}
	p := m.For("/games/x/game.exe", "meta")
	p.Meta.Arch = "win32"
	p.Meta.LastBackend = "system-wine"
	p.Meta.Source = "/games/x"
	if err := p.SaveMeta(); err != nil {
		t.Fatal(err)
	}

	reloaded := m.For("/games/x/game.exe", "meta")
	if reloaded.Meta.Arch != "win32" || reloaded.Meta.LastBackend != "system-wine" {
		t.Errorf("metadata did not round-trip: %+v", reloaded.Meta)
	}

	list, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].Name != "meta" {
		t.Errorf("List() = %v", list)
	}
}

func TestExistsTracksInitialisation(t *testing.T) {
	m := &Manager{Root: t.TempDir()}
	p := m.For("/games/x/game.exe", "fresh")
	if p.Exists() {
		t.Error("an uncreated prefix reported as existing")
	}
	if err := p.Prepare(); err != nil {
		t.Fatal(err)
	}
	if p.Exists() {
		t.Error("an empty directory is not an initialised prefix")
	}
	// Wine marks a prefix as initialised by writing its registry.
	if err := os.WriteFile(filepath.Join(p.WinePrefix, "system.reg"), []byte("WINE REGISTRY"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !p.Exists() {
		t.Error("an initialised prefix reported as missing")
	}
}
