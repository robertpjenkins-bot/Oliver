package backend

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Scanner locates installed runtimes. Every path it consults is a field, so
// tests can point it at a synthetic system tree.
type Scanner struct {
	Home     string
	PathDirs []string
	// ExtraRoots are user-configured directories to search for runtimes.
	ExtraRoots []string
	// SystemRoots are absolute prefixes such as /opt and /usr/lib. Overridable
	// so tests never touch the real filesystem.
	SystemRoots []string
	// FlatpakRoots hold flatpak app installations.
	FlatpakRoots []string
	// Probe allows running `<wine> --version` when the version cannot be read
	// from disk. Tests disable it.
	Probe bool
	// ProbeTimeout bounds each version probe.
	ProbeTimeout time.Duration
	// HostArch is the machine architecture, used to decide whether an x86
	// emulation layer is required.
	HostArch string
}

// NewScanner builds a scanner configured for the current user and machine.
func NewScanner() *Scanner {
	home, _ := os.UserHomeDir()
	return &Scanner{
		Home:         home,
		PathDirs:     filepath.SplitList(os.Getenv("PATH")),
		SystemRoots:  []string{"/opt", "/usr/lib/wine", "/usr/lib64/wine", "/usr/local"},
		FlatpakRoots: []string{"/var/lib/flatpak/app", filepath.Join(home, ".local/share/flatpak/app")},
		Probe:        true,
		ProbeTimeout: 4 * time.Second,
		HostArch:     runtime.GOARCH,
	}
}

// Discover returns every runtime found, best first.
func (s *Scanner) Discover() []*Backend {
	var all []*Backend
	all = append(all, s.scanPath()...)
	all = append(all, s.scanSteam()...)
	all = append(all, s.scanLutris()...)
	all = append(all, s.scanBottles()...)
	all = append(all, s.scanHeroic()...)
	all = append(all, s.scanPlayOnLinux()...)
	all = append(all, s.scanCrossOver()...)
	all = append(all, s.scanOpt()...)
	all = append(all, s.scanFlatpak()...)
	all = append(all, s.scanExtraRoots()...)

	all = dedupe(all)

	// A Proton build cannot be launched sensibly outside Steam without umu.
	// Record that once, on every Proton backend, so scoring can weigh it.
	if umu := s.findUMU(); umu != "" {
		for _, b := range all {
			if b.IsProton() {
				b.Launcher = []string{umu}
				b.Notes = append(b.Notes, "launched through umu-run")
			}
		}
	} else {
		for _, b := range all {
			if b.IsProton() {
				b.Notes = append(b.Notes, "umu-launcher not installed; falling back to a manual Steam runtime environment")
			}
		}
	}

	// On non-x86 hosts an x86 Wine build needs a translation layer.
	if s.HostArch != "amd64" && s.HostArch != "386" {
		if box := s.findBox(); box != "" {
			for _, b := range all {
				if b.Kind == KindWine && len(b.Launcher) == 0 {
					b.Launcher = []string{box}
					b.Kind = KindBox
					b.Notes = append(b.Notes, "x86 translation via "+filepath.Base(box))
				}
			}
		}
	}

	Sort(all)
	return all
}

// dedupe removes runtimes that resolve to the same executable, keeping the
// first (best-described) entry.
func dedupe(bs []*Backend) []*Backend {
	seen := map[string]bool{}
	out := bs[:0]
	for _, b := range bs {
		key := b.Exec
		if r, err := filepath.EvalSymlinks(b.Exec); err == nil {
			key = r
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, b)
	}
	return out
}

// systemWineNames are the binaries distributions ship, in rough order of how
// current they tend to be.
var systemWineNames = []string{
	"wine", "wine-stable", "wine-staging", "wine-development", "wine-devel", "wine64",
}

func (s *Scanner) scanPath() []*Backend {
	var out []*Backend
	for _, name := range systemWineNames {
		p := s.lookPath(name)
		if p == "" {
			continue
		}
		root := wineRootFor(p)
		b := &Backend{
			ID:     "system-" + name,
			Name:   name,
			Kind:   KindWine,
			Source: SourceSystem,
			Exec:   p,
			Root:   root,
		}
		b.Version = s.resolveWineVersion(p, root, name)
		b.Variant = DetectVariant(name + " " + b.Version.Raw)
		s.applyBitness(b, root, name)
		out = append(out, b)
	}
	return out
}

// wineRootFor walks up from <root>/bin/wine to <root>.
func wineRootFor(exe string) string {
	dir := filepath.Dir(exe)
	if filepath.Base(dir) == "bin" {
		return filepath.Dir(dir)
	}
	return dir
}

func (s *Scanner) lookPath(name string) string {
	for _, dir := range s.PathDirs {
		if dir == "" {
			continue
		}
		p := filepath.Join(dir, name)
		if isExecutable(p) {
			return p
		}
	}
	return ""
}

func isExecutable(p string) bool {
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return false
	}
	return st.Mode().Perm()&0o111 != 0
}

func isDir(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

// steamRoots lists every plausible Steam installation, including the flatpak.
func (s *Scanner) steamRoots() []string {
	if s.Home == "" {
		return nil
	}
	cands := []string{
		filepath.Join(s.Home, ".steam/steam"),
		filepath.Join(s.Home, ".steam/root"),
		filepath.Join(s.Home, ".local/share/Steam"),
		filepath.Join(s.Home, ".var/app/com.valvesoftware.Steam/data/Steam"),
	}
	var out []string
	seen := map[string]bool{}
	for _, c := range cands {
		if !isDir(c) {
			continue
		}
		real := c
		if r, err := filepath.EvalSymlinks(c); err == nil {
			real = r
		}
		if seen[real] {
			continue
		}
		seen[real] = true
		out = append(out, c)
	}
	return out
}

var vdfPath = regexp.MustCompile(`"path"\s*"([^"]+)"`)

// steamLibraries returns every Steam library directory, following
// libraryfolders.vdf so Proton installed on a second drive is still found.
func (s *Scanner) steamLibraries() []string {
	var libs []string
	seen := map[string]bool{}
	add := func(p string) {
		if p == "" || seen[p] || !isDir(p) {
			return
		}
		seen[p] = true
		libs = append(libs, p)
	}
	for _, root := range s.steamRoots() {
		add(root)
		data, err := os.ReadFile(filepath.Join(root, "steamapps", "libraryfolders.vdf"))
		if err != nil {
			continue
		}
		for _, m := range vdfPath.FindAllStringSubmatch(string(data), -1) {
			// VDF escapes backslashes; harmless on Linux paths but cheap to undo.
			add(strings.ReplaceAll(m[1], `\\`, `\`))
		}
	}
	return libs
}

func (s *Scanner) scanSteam() []*Backend {
	var out []*Backend
	for _, lib := range s.steamLibraries() {
		out = append(out, s.scanProtonDir(filepath.Join(lib, "steamapps", "common"), SourceSteam)...)
	}
	for _, root := range s.steamRoots() {
		out = append(out, s.scanProtonDir(filepath.Join(root, "compatibilitytools.d"), SourceCompatTools)...)
	}
	out = append(out, s.scanProtonDir("/usr/share/steam/compatibilitytools.d", SourceCompatTools)...)
	if s.Home != "" {
		out = append(out, s.scanProtonDir(filepath.Join(s.Home, ".local/share/Steam/compatibilitytools.d"), SourceCompatTools)...)
	}
	return out
}

// scanProtonDir treats every immediate subdirectory containing a `proton`
// entry script as a Proton installation.
func (s *Scanner) scanProtonDir(dir string, src Source) []*Backend {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []*Backend
	for _, e := range entries {
		if !e.IsDir() && e.Type()&os.ModeSymlink == 0 {
			continue
		}
		root := filepath.Join(dir, e.Name())
		script := filepath.Join(root, "proton")
		if !isExecutable(script) && !fileExists(script) {
			continue
		}
		if b := s.protonBackend(root, script, e.Name(), src); b != nil {
			out = append(out, b)
		}
	}
	return out
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func (s *Scanner) protonBackend(root, script, dirName string, src Source) *Backend {
	name := dirName
	// Proton writes "<build timestamp> <version name>" into `version`.
	if data, err := os.ReadFile(filepath.Join(root, "version")); err == nil {
		fields := strings.Fields(strings.TrimSpace(string(data)))
		if len(fields) > 0 {
			name = fields[len(fields)-1]
		}
	}
	b := &Backend{
		ID:      slug("proton-" + dirName),
		Name:    dirName,
		Kind:    KindProton,
		Source:  src,
		Exec:    script,
		Root:    root,
		Version: ParseVersion(name),
		Variant: DetectVariant(dirName + " " + name),
		Has64:   true,
	}
	// Proton ships its Wine tree under files/ (or dist/ on older builds).
	for _, sub := range []string{"files", "dist"} {
		if isDir(filepath.Join(root, sub, "lib")) || isDir(filepath.Join(root, sub, "lib64")) {
			s.applyBitness(b, filepath.Join(root, sub), dirName)
			break
		}
	}
	if !b.Has32 && !b.Has64 {
		// Every shipped Proton supports both; assume so when the layout is new.
		b.Has32, b.Has64 = true, true
	}
	if strings.Contains(strings.ToLower(dirName), "experimental") {
		b.Variant = VariantExperimental
	}
	return b
}

// runnerDirs is the shared shape used by Lutris, Bottles and Heroic: a
// directory of runner installations, each with bin/wine somewhere inside.
func (s *Scanner) scanRunnerDir(dir string, src Source, idPrefix string) []*Backend {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []*Backend
	for _, e := range entries {
		root := filepath.Join(dir, e.Name())
		if !isDir(root) {
			continue
		}
		// A Proton-style runner inside a manager's runner directory.
		if script := filepath.Join(root, "proton"); fileExists(script) {
			if b := s.protonBackend(root, script, e.Name(), src); b != nil {
				out = append(out, b)
			}
			continue
		}
		exe, base := findWineIn(root)
		if exe == "" {
			continue
		}
		b := &Backend{
			ID:      slug(idPrefix + "-" + e.Name()),
			Name:    e.Name(),
			Kind:    KindWine,
			Source:  src,
			Exec:    exe,
			Root:    base,
			Version: ParseVersion(e.Name()),
			Variant: DetectVariant(e.Name()),
		}
		if b.Version.IsZero() {
			b.Version = s.resolveWineVersion(exe, base, e.Name())
		}
		s.applyBitness(b, base, e.Name())
		out = append(out, b)
	}
	return out
}

// findWineIn locates a wine binary within a runner installation, returning the
// binary and the tree root it belongs to.
func findWineIn(root string) (exe string, base string) {
	// Managers nest the wine tree differently; check the known layouts.
	for _, sub := range []string{"", "files", "dist"} {
		b := filepath.Join(root, sub)
		for _, name := range []string{"wine", "wine64"} {
			p := filepath.Join(b, "bin", name)
			if isExecutable(p) {
				return p, b
			}
		}
	}
	return "", ""
}

func (s *Scanner) scanLutris() []*Backend {
	if s.Home == "" {
		return nil
	}
	var out []*Backend
	for _, d := range []string{
		filepath.Join(s.Home, ".local/share/lutris/runners/wine"),
		filepath.Join(s.Home, ".var/app/net.lutris.Lutris/data/lutris/runners/wine"),
	} {
		out = append(out, s.scanRunnerDir(d, SourceLutris, "lutris")...)
	}
	return out
}

func (s *Scanner) scanBottles() []*Backend {
	if s.Home == "" {
		return nil
	}
	var out []*Backend
	for _, d := range []string{
		filepath.Join(s.Home, ".local/share/bottles/runners"),
		filepath.Join(s.Home, ".var/app/com.usebottles.bottles/data/bottles/runners"),
	} {
		out = append(out, s.scanRunnerDir(d, SourceBottles, "bottles")...)
	}
	return out
}

func (s *Scanner) scanHeroic() []*Backend {
	if s.Home == "" {
		return nil
	}
	var out []*Backend
	for _, base := range []string{
		filepath.Join(s.Home, ".config/heroic/tools"),
		filepath.Join(s.Home, ".var/app/com.heroicgameslauncher.hgl/config/heroic/tools"),
	} {
		out = append(out, s.scanRunnerDir(filepath.Join(base, "wine"), SourceHeroic, "heroic")...)
		out = append(out, s.scanRunnerDir(filepath.Join(base, "proton"), SourceHeroic, "heroic")...)
	}
	return out
}

func (s *Scanner) scanPlayOnLinux() []*Backend {
	if s.Home == "" {
		return nil
	}
	var out []*Backend
	for _, arch := range []string{"linux-amd64", "linux-x86"} {
		out = append(out, s.scanRunnerDir(filepath.Join(s.Home, ".PlayOnLinux/wine", arch), SourcePlayOnLinux, "pol-"+arch)...)
	}
	return out
}

func (s *Scanner) scanCrossOver() []*Backend {
	var cands []string
	for _, sysRoot := range s.SystemRoots {
		cands = append(cands,
			filepath.Join(sysRoot, "cxoffice"),
			filepath.Join(sysRoot, "crossover"),
		)
	}
	if s.Home != "" {
		cands = append(cands, filepath.Join(s.Home, ".cxoffice"))
	}
	var out []*Backend
	for _, root := range cands {
		exe := filepath.Join(root, "bin", "wine")
		if !isExecutable(exe) {
			continue
		}
		b := &Backend{
			ID:     slug("crossover-" + filepath.Base(root)),
			Name:   "CrossOver (" + filepath.Base(root) + ")",
			Kind:   KindCrossOver,
			Source: SourceCrossOver,
			Exec:   exe,
			Root:   root,
		}
		b.Version = s.resolveWineVersion(exe, root, "")
		s.applyBitness(b, root, "")
		out = append(out, b)
	}
	return out
}

// scanOpt picks up hand-installed trees such as /opt/wine-staging.
func (s *Scanner) scanOpt() []*Backend {
	var out []*Backend
	for _, sysRoot := range s.SystemRoots {
		entries, err := os.ReadDir(sysRoot)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if !strings.HasPrefix(strings.ToLower(name), "wine") && !strings.HasPrefix(strings.ToLower(name), "proton") {
				continue
			}
			root := filepath.Join(sysRoot, name)
			if !isDir(root) {
				continue
			}
			if script := filepath.Join(root, "proton"); fileExists(script) {
				if b := s.protonBackend(root, script, name, SourceOpt); b != nil {
					out = append(out, b)
				}
				continue
			}
			exe, base := findWineIn(root)
			if exe == "" {
				continue
			}
			b := &Backend{
				ID:      slug("opt-" + name),
				Name:    name,
				Kind:    KindWine,
				Source:  SourceOpt,
				Exec:    exe,
				Root:    base,
				Version: ParseVersion(name),
				Variant: DetectVariant(name),
			}
			if b.Version.IsZero() {
				b.Version = s.resolveWineVersion(exe, base, name)
			}
			s.applyBitness(b, base, name)
			out = append(out, b)
		}
	}
	return out
}

// scanFlatpak finds the Wine flatpak, which must be launched through flatpak
// rather than by executing its binary directly.
func (s *Scanner) scanFlatpak() []*Backend {
	var out []*Backend
	for _, root := range s.FlatpakRoots {
		appRoot := filepath.Join(root, "org.winehq.Wine", "current", "active", "files")
		exe := filepath.Join(appRoot, "bin", "wine")
		if !isExecutable(exe) {
			continue
		}
		b := &Backend{
			ID:       "flatpak-winehq",
			Name:     "org.winehq.Wine (flatpak)",
			Kind:     KindWine,
			Source:   SourceFlatpak,
			Exec:     exe,
			Root:     appRoot,
			Launcher: []string{"flatpak", "run", "--command=wine", "org.winehq.Wine"},
			Version:  s.resolveWineVersion(exe, appRoot, ""),
		}
		s.applyBitness(b, appRoot, "")
		b.Notes = append(b.Notes, "flatpak sandbox may not see files outside your home directory")
		out = append(out, b)
	}
	return out
}

func (s *Scanner) scanExtraRoots() []*Backend {
	var out []*Backend
	for _, root := range s.ExtraRoots {
		if script := filepath.Join(root, "proton"); fileExists(script) {
			if b := s.protonBackend(root, script, filepath.Base(root), SourceOpt); b != nil {
				out = append(out, b)
			}
			continue
		}
		if exe, base := findWineIn(root); exe != "" {
			b := &Backend{
				ID:      slug("custom-" + filepath.Base(root)),
				Name:    filepath.Base(root),
				Kind:    KindWine,
				Source:  SourceOpt,
				Exec:    exe,
				Root:    base,
				Version: s.resolveWineVersion(exe, base, filepath.Base(root)),
				Variant: DetectVariant(filepath.Base(root)),
			}
			s.applyBitness(b, base, filepath.Base(root))
			out = append(out, b)
		}
		// A directory of runners, as configured by the user.
		out = append(out, s.scanRunnerDir(root, SourceOpt, "custom")...)
	}
	return out
}

// findUMU locates umu-run, the launcher that makes Proton usable outside Steam.
func (s *Scanner) findUMU() string {
	for _, n := range []string{"umu-run", "umu-launcher"} {
		if p := s.lookPath(n); p != "" {
			return p
		}
	}
	if s.Home != "" {
		p := filepath.Join(s.Home, ".local/bin/umu-run")
		if isExecutable(p) {
			return p
		}
	}
	return ""
}

func (s *Scanner) findBox() string {
	for _, n := range []string{"box64", "box86"} {
		if p := s.lookPath(n); p != "" {
			return p
		}
	}
	return ""
}

// applyBitness determines 32- and 64-bit capability by inspecting the tree's
// library layout. Wine builds that dropped 32-bit support are the most common
// reason a working-looking runner fails on an old application.
func (s *Scanner) applyBitness(b *Backend, root, hint string) {
	has32, has64, known := probeBitness(root)
	if !known {
		// Fall back to what the binary name tells us.
		switch {
		case strings.HasSuffix(b.Exec, "wine64"):
			has64, known = true, true
		case strings.Contains(strings.ToLower(hint), "win32"):
			has32, known = true, true
		}
	}
	if !known {
		// Unverifiable: assume capable rather than block a launch, but say so.
		b.Has32, b.Has64 = true, true
		b.Notes = append(b.Notes, "32/64-bit support could not be verified from disk layout")
		return
	}
	b.Has32, b.Has64 = has32, has64
	if has64 && !has32 {
		b.Notes = append(b.Notes, "no 32-bit support: cannot run older 32-bit applications")
	}
}

// bitnessMarkers map a directory under a Wine tree's lib root to the
// architecture its presence proves.
var bitnessMarkers = []struct {
	rel string
	is3 bool
}{
	{"lib/wine/i386-unix", true},
	{"lib/wine/i386-windows", true},
	{"lib32/wine", true},
	{"lib/i386-linux-gnu/wine", true},
	{"lib/wine/x86_64-unix", false},
	{"lib/wine/x86_64-windows", false},
	{"lib64/wine", false},
	{"lib/x86_64-linux-gnu/wine", false},
}

func probeBitness(root string) (has32, has64, known bool) {
	if root == "" {
		return false, false, false
	}
	for _, m := range bitnessMarkers {
		if isDir(filepath.Join(root, m.rel)) {
			known = true
			if m.is3 {
				has32 = true
			} else {
				has64 = true
			}
		}
	}
	return has32, has64, known
}

var slugUnsafe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	s = strings.ToLower(s)
	s = slugUnsafe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

// resolveWineVersion reads a version from the tree, and only asks the binary
// itself as a last resort.
func (s *Scanner) resolveWineVersion(exe, root, hint string) Version {
	if v := ParseVersion(hint); !v.IsZero() {
		return v
	}
	if root != "" {
		if v := ParseVersion(filepath.Base(root)); !v.IsZero() {
			return v
		}
	}
	if !s.Probe {
		return Version{}
	}
	timeout := s.ProbeTimeout
	if timeout <= 0 {
		timeout = 4 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "--version")
	// A version probe must never create or touch a prefix.
	cmd.Env = append(os.Environ(), "WINEDEBUG=-all", "DISPLAY=", "WINEDLLOVERRIDES=mscoree=d;mshtml=d")
	out, err := cmd.Output()
	if err != nil {
		return Version{}
	}
	return ParseVersion(strings.TrimSpace(string(out)))
}
