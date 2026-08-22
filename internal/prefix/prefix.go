// Package prefix manages the isolated Wine prefixes Oliver launches into.
package prefix

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Meta is the record Oliver keeps beside each prefix so later runs, and the
// user, can see how it was set up.
type Meta struct {
	Name        string    `json:"name"`
	Source      string    `json:"source_directory"`
	Arch        string    `json:"arch"`
	WindowsVer  string    `json:"windows_version,omitempty"`
	LastBackend string    `json:"last_backend,omitempty"`
	Created     time.Time `json:"created"`
	LastRun     time.Time `json:"last_run,omitempty"`
}

// Prefix is one isolated Windows environment.
//
// The layout deliberately matches Steam's: the prefix directory holds a `pfx`
// subdirectory that is the actual WINEPREFIX. Wine and Proton therefore agree
// on where the prefix lives, and a user can switch a program between the two
// without rebuilding its environment.
type Prefix struct {
	Name string
	// Dir is the container, and is what Proton wants as STEAM_COMPAT_DATA_PATH.
	Dir string
	// WinePrefix is the Wine prefix proper, and is what WINEPREFIX must be.
	WinePrefix string
	Meta       Meta
}

// Exists reports whether the prefix has already been initialised.
func (p *Prefix) Exists() bool {
	_, err := os.Stat(filepath.Join(p.WinePrefix, "system.reg"))
	return err == nil
}

// Manager owns the directory that holds every prefix.
type Manager struct {
	Root string
}

// DefaultRoot returns the XDG data location for prefixes.
func DefaultRoot() string {
	if d := os.Getenv("OLIVER_PREFIX_ROOT"); d != "" {
		return d
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(os.TempDir(), "oliver", "prefixes")
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "oliver", "prefixes")
}

// NewManager returns a manager rooted at the default location.
func NewManager() *Manager { return &Manager{Root: DefaultRoot()} }

var unsafeName = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// NameFor derives a stable prefix name from the directory an executable lives
// in. Keying on the directory rather than the executable means a program's
// launcher, its main binary and its uninstaller all share one environment,
// while two separate installs never collide.
func NameFor(exePath string) string {
	abs, err := filepath.Abs(exePath)
	if err != nil {
		abs = exePath
	}
	dir := filepath.Dir(abs)

	label := filepath.Base(dir)
	// A program sitting loose in a download or home directory is better
	// identified by its own name than by the folder it happens to be in.
	switch strings.ToLower(label) {
	case "", ".", "/", "downloads", "desktop", "documents", "tmp", "temp", "bin":
		label = strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
	}
	label = strings.Trim(unsafeName.ReplaceAllString(label, "-"), "-")
	if label == "" {
		label = "app"
	}
	if len(label) > 40 {
		label = label[:40]
	}

	sum := sha256.Sum256([]byte(dir))
	return strings.ToLower(label) + "-" + hex.EncodeToString(sum[:])[:6]
}

// For returns the prefix an executable should use. An explicit name (or an
// absolute path) overrides the derived one.
func (m *Manager) For(exePath, explicit string) *Prefix {
	if explicit != "" {
		if filepath.IsAbs(explicit) {
			return m.at(filepath.Base(explicit), explicit)
		}
		return m.at(explicit, filepath.Join(m.Root, explicit))
	}
	name := NameFor(exePath)
	return m.at(name, filepath.Join(m.Root, name))
}

func (m *Manager) at(name, dir string) *Prefix {
	p := &Prefix{
		Name:       name,
		Dir:        dir,
		WinePrefix: filepath.Join(dir, "pfx"),
	}
	p.Meta = p.readMeta()
	return p
}

func (p *Prefix) metaPath() string { return filepath.Join(p.Dir, "oliver.json") }

func (p *Prefix) readMeta() Meta {
	var m Meta
	data, err := os.ReadFile(p.metaPath())
	if err != nil {
		return m
	}
	_ = json.Unmarshal(data, &m)
	return m
}

// SaveMeta records how the prefix was set up.
func (p *Prefix) SaveMeta() error {
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return err
	}
	p.Meta.Name = p.Name
	data, err := json.MarshalIndent(p.Meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p.metaPath(), append(data, '\n'), 0o644)
}

// Prepare creates the directory skeleton. Wine itself creates the prefix
// contents on first launch; this only has to exist beforehand.
func (p *Prefix) Prepare() error {
	return os.MkdirAll(p.WinePrefix, 0o755)
}

// userDirs are the folders Wine links from a prefix out into the real home
// directory.
var userDirs = []string{
	"Desktop", "My Documents", "Documents", "Downloads",
	"My Music", "Music", "My Pictures", "Pictures",
	"My Videos", "Videos", "Templates",
}

// Sandbox replaces the symlinks Wine points at the user's real home
// directories with ordinary folders inside the prefix.
//
// Without this a Windows program's "Documents" is the user's actual Documents
// folder, so an installer or an uninstaller can write to — or delete — real
// files. It also stops programs scattering shortcuts across the real desktop.
// Returns the folders it isolated.
func (p *Prefix) Sandbox() ([]string, error) {
	usersDir := filepath.Join(p.WinePrefix, "drive_c", "users")
	entries, err := os.ReadDir(usersDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var isolated []string
	for _, user := range entries {
		if !user.IsDir() || user.Name() == "Public" {
			continue
		}
		for _, d := range userDirs {
			target := filepath.Join(usersDir, user.Name(), d)
			st, err := os.Lstat(target)
			if err != nil || st.Mode()&os.ModeSymlink == 0 {
				continue // absent, or already a real directory
			}
			if err := os.Remove(target); err != nil {
				return isolated, fmt.Errorf("unlinking %s: %w", target, err)
			}
			if err := os.MkdirAll(target, 0o755); err != nil {
				return isolated, fmt.Errorf("creating %s: %w", target, err)
			}
			isolated = append(isolated, d)
		}
	}
	sort.Strings(isolated)
	return isolated, nil
}

// List returns every prefix under the manager's root, newest use first.
func (m *Manager) List() ([]*Prefix, error) {
	entries, err := os.ReadDir(m.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []*Prefix
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		out = append(out, m.at(e.Name(), filepath.Join(m.Root, e.Name())))
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i].Meta.LastRun, out[j].Meta.LastRun
		if a.Equal(b) {
			return out[i].Name < out[j].Name
		}
		return a.After(b)
	})
	return out, nil
}

// Remove deletes a prefix. It refuses to touch anything outside the managed
// root, so a mistyped name can never delete unrelated files.
func (m *Manager) Remove(name string) error {
	if name == "" || strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
		return fmt.Errorf("invalid prefix name %q", name)
	}
	dir := filepath.Join(m.Root, name)
	root, err := filepath.Abs(m.Root)
	if err != nil {
		return err
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		return fmt.Errorf("refusing to remove %s: outside the prefix root", abs)
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("no such prefix %q", name)
	}
	return os.RemoveAll(abs)
}

// DiskUsage totals the bytes a prefix occupies.
func (p *Prefix) DiskUsage() int64 {
	var total int64
	_ = filepath.WalkDir(p.Dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if info, err := d.Info(); err == nil && info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return total
}
