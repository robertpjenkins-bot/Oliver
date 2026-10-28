package mcreator

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// Install is an MCreator installation found on this machine.
type Install struct {
	Dir     string `json:"dir"`
	Version string `json:"version,omitempty"` // e.g. "2026.3", when it could be read
	Java    string `json:"java,omitempty"`    // the JDK MCreator ships with, if present
}

// Toolkit is everything a build borrows from MCreator: the generator
// templates, the plugins that provide them, and a JDK.
type Toolkit struct {
	Installs []Install
	// UserDir is MCreator's per-user folder, ~/.mcreator, which holds user
	// plugins and the Gradle cache MCreator's own builds have filled.
	UserDir string

	sources []pluginSource
	loaded  bool
}

// Generator is one MCreator generator: the build files it lays down in a
// workspace, and where its build puts the finished mod.
type Generator struct {
	Name             string
	BuildFileVersion string
	ExportFile       string // workspace-relative, e.g. build/libs/modid-1.0.jar
	From             string // the plugin it was read from
	// Java is the JDK of the MCreator the generator came with, which is the
	// Java it was made to build with. Empty for a user-installed plugin.
	Java string

	files map[string][]byte // workspace base, keyed by workspace-relative path
}

// MinecraftVersion is the version part of the generator name.
func (g *Generator) MinecraftVersion() string {
	_, v, _ := strings.Cut(g.Name, "-")
	return v
}

// pluginSource is a plugin directory or archive.
type pluginSource struct {
	name   string
	fsys   fs.FS
	closer io.Closer
	java   string // the JDK of the installation it ships with
}

// FindToolkit locates MCreator. Directories in extra are tried first, so a
// folder the user pointed at wins over anything found by searching.
func FindToolkit(extra ...string) *Toolkit {
	t := &Toolkit{UserDir: userDir()}
	seen := map[string]bool{}
	add := func(dir string) {
		if dir == "" {
			return
		}
		inst, ok := inspectInstall(dir)
		if !ok {
			return
		}
		key := inst.Dir
		if resolved, err := filepath.EvalSymlinks(key); err == nil {
			key = resolved
		}
		if seen[key] {
			return
		}
		seen[key] = true
		t.Installs = append(t.Installs, inst)
	}

	for _, dir := range extra {
		add(dir)
	}
	pinned := len(t.Installs)

	add(os.Getenv("MCREATOR_HOME"))
	for _, dir := range desktopEntryInstalls() {
		add(dir)
	}
	for _, dir := range candidateInstalls() {
		add(dir)
	}

	// Newest first: it is the most likely to know the workspace's generator.
	found := t.Installs[pinned:]
	sort.SliceStable(found, func(i, j int) bool {
		return compareVersions(found[i].Version, found[j].Version) > 0
	})
	return t
}

// CheckInstall reports whether dir is an MCreator installation, and why not.
func CheckInstall(dir string) (Install, error) {
	if inst, ok := inspectInstall(dir); ok {
		return inst, nil
	}
	if _, err := os.Stat(dir); err != nil {
		return Install{}, fmt.Errorf("%s does not exist", dir)
	}
	return Install{}, fmt.Errorf("%s does not look like an MCreator folder (it should contain plugins and lib folders)", dir)
}

func inspectInstall(dir string) (Install, bool) {
	dir = filepath.Clean(dir)
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	if st, err := os.Stat(filepath.Join(dir, "plugins")); err != nil || !st.IsDir() {
		return Install{}, false
	}
	jar := filepath.Join(dir, "lib", "mcreator.jar")
	if _, err := os.Stat(jar); err != nil {
		// Without the main jar, only a plugins folder full of generators
		// identifies the place.
		if m, _ := filepath.Glob(filepath.Join(dir, "plugins", "generator-*")); len(m) == 0 {
			return Install{}, false
		}
	}
	inst := Install{Dir: dir, Version: installVersion(jar)}
	for _, home := range []string{filepath.Join(dir, "jdk"), filepath.Join(dir, "jdk", "Contents", "Home")} {
		if isJavaHome(home) {
			inst.Java = home
			break
		}
	}
	return inst, true
}

// installVersion reads the release MCreator records inside its own jar.
func installVersion(jar string) string {
	zr, err := zip.OpenReader(jar)
	if err != nil {
		return ""
	}
	defer zr.Close()
	for _, f := range zr.File {
		if f.Name != "mcreator.conf" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return ""
		}
		defer rc.Close()
		sc := bufio.NewScanner(io.LimitReader(rc, 1<<16))
		for sc.Scan() {
			if v, ok := strings.CutPrefix(strings.TrimSpace(sc.Text()), "mcreator="); ok {
				return strings.TrimSpace(v)
			}
		}
	}
	return ""
}

// compareVersions orders dotted version strings numerically. An unknown
// version sorts below every known one.
func compareVersions(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "":
		return -1
	case b == "":
		return 1
	}
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			if x > y {
				return 1
			}
			return -1
		}
	}
	return 0
}

func userDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".mcreator")
}

// GradleHome is where MCreator keeps Gradle's caches. Building with the same
// one means everything MCreator already downloaded is reused.
func (t *Toolkit) GradleHome() string {
	if t.UserDir == "" {
		return ""
	}
	return filepath.Join(t.UserDir, "gradle")
}

// candidateInstalls lists the places MCreator is usually unpacked or installed.
func candidateInstalls() []string {
	home, _ := os.UserHomeDir()
	var patterns []string
	if home != "" {
		for _, base := range []string{"", "Downloads", "Desktop", "Applications", "Programs", "Games", "apps", "opt", ".local/share", ".local/opt"} {
			patterns = append(patterns,
				filepath.Join(home, base, "MCreator*"),
				filepath.Join(home, base, "mcreator*"),
				// A tarball unpacked into a folder named after the download.
				filepath.Join(home, base, "MCreator*", "MCreator*"))
		}
	}
	patterns = append(patterns, "/opt/MCreator*", "/opt/mcreator*", "/usr/share/mcreator*", "/usr/lib/mcreator*", "/usr/local/share/mcreator*")
	if runtime.GOOS == "windows" {
		for _, env := range []string{"ProgramFiles", "ProgramFiles(x86)", "LOCALAPPDATA"} {
			if d := os.Getenv(env); d != "" {
				patterns = append(patterns, filepath.Join(d, "Pylo", "MCreator*"), filepath.Join(d, "MCreator*"))
			}
		}
	}

	var dirs []string
	for _, p := range patterns {
		m, _ := filepath.Glob(p)
		dirs = append(dirs, m...)
	}
	return dirs
}

// execCD matches the launcher MCreator writes on first start:
// Exec=/bin/bash -c 'cd "/home/u/MCreator20263" && ./mcreator.sh'
var execCD = regexp.MustCompile(`cd\s+"([^"]+)"|cd\s+'([^']+)'|cd\s+(\S+)`)

// desktopEntryInstalls finds MCreator through the desktop launchers it
// creates, which record where it was unpacked — the one reliable trace of an
// installation that could be anywhere.
func desktopEntryInstalls() []string {
	home, _ := os.UserHomeDir()
	var dirs []string
	if home != "" {
		dirs = append(dirs, filepath.Join(home, "Desktop"))
		if out, err := exec.Command("xdg-user-dir", "DESKTOP").Output(); err == nil {
			if d := strings.TrimSpace(string(out)); d != "" {
				dirs = append(dirs, d)
			}
		}
		dirs = append(dirs, filepath.Join(home, ".local", "share", "applications"))
	}
	dirs = append(dirs, "/usr/share/applications", "/usr/local/share/applications")

	var found []string
	for _, d := range dirs {
		entries, _ := filepath.Glob(filepath.Join(d, "*.desktop"))
		for _, e := range entries {
			if !strings.Contains(strings.ToLower(filepath.Base(e)), "mcreator") {
				continue
			}
			found = append(found, installsFromDesktopEntry(e)...)
		}
	}
	return found
}

func installsFromDesktopEntry(file string) []string {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		value, ok := strings.CutPrefix(line, "Exec=")
		if !ok {
			value, ok = strings.CutPrefix(line, "Path=")
			if ok {
				dirs = append(dirs, value)
			}
			continue
		}
		for _, m := range execCD.FindAllStringSubmatch(value, -1) {
			for _, g := range m[1:] {
				if g != "" {
					dirs = append(dirs, g)
				}
			}
		}
		// Or a launcher that runs the script by its full path.
		for _, field := range strings.Fields(strings.NewReplacer(`"`, " ", `'`, " ").Replace(value)) {
			if strings.HasSuffix(field, "/mcreator.sh") || strings.HasSuffix(field, "/mcreator") {
				dirs = append(dirs, filepath.Dir(field))
			}
		}
	}
	return dirs
}

// plugins opens every plugin MCreator would load: those shipped with each
// installation, then the user's own.
func (t *Toolkit) plugins() []pluginSource {
	if t.loaded {
		return t.sources
	}
	t.loaded = true
	type pluginDir struct{ dir, java string }
	dirs := make([]pluginDir, 0, len(t.Installs)+1)
	for _, inst := range t.Installs {
		dirs = append(dirs, pluginDir{filepath.Join(inst.Dir, "plugins"), inst.Java})
	}
	if t.UserDir != "" {
		dirs = append(dirs, pluginDir{dir: filepath.Join(t.UserDir, "plugins")})
	}
	for _, d := range dirs {
		entries, err := os.ReadDir(d.dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			full := filepath.Join(d.dir, e.Name())
			switch {
			case e.IsDir():
				t.sources = append(t.sources, pluginSource{name: full, fsys: os.DirFS(full), java: d.java})
			case strings.HasSuffix(e.Name(), ".zip") || strings.HasSuffix(e.Name(), ".jar"):
				zr, err := zip.OpenReader(full)
				if err != nil {
					continue
				}
				t.sources = append(t.sources, pluginSource{name: full, fsys: zr, closer: zr, java: d.java})
			}
		}
	}
	return t.sources
}

// Close releases the plugin archives.
func (t *Toolkit) Close() {
	for _, s := range t.sources {
		if s.closer != nil {
			s.closer.Close()
		}
	}
	t.sources, t.loaded = nil, false
}

// ErrNoMCreator means no MCreator installation was found.
var ErrNoMCreator = errors.New("MCreator was not found on this computer")

// Generator loads a generator by name from the installed plugins.
func (t *Toolkit) Generator(name string) (*Generator, error) {
	if len(t.Installs) == 0 {
		return nil, ErrNoMCreator
	}
	for _, src := range t.plugins() {
		data, err := fs.ReadFile(src.fsys, name+"/generator.yaml")
		if err != nil {
			continue
		}
		g := &Generator{Name: name, From: src.name, Java: src.java, files: map[string][]byte{}}
		g.BuildFileVersion, _ = yamlLookup(string(data), "buildfileversion")
		g.ExportFile, _ = yamlLookup(string(data), "gradle", "export_file")

		base := name + "/workspacebase"
		err = fs.WalkDir(src.fsys, base, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			content, err := fs.ReadFile(src.fsys, p)
			if err != nil {
				return err
			}
			g.files[strings.TrimPrefix(p, base+"/")] = content
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("reading the %s generator from %s: %w", name, src.name, err)
		}
		if _, ok := g.files["build.gradle"]; !ok {
			return nil, fmt.Errorf("the %s generator in %s has no build.gradle", name, src.name)
		}
		return g, nil
	}
	return nil, &MissingGeneratorError{Name: name, Available: t.Generators()}
}

// MissingGeneratorError means the installed MCreator cannot build for the
// workspace's Minecraft version.
type MissingGeneratorError struct {
	Name      string
	Available []string
}

func (e *MissingGeneratorError) Error() string {
	msg := fmt.Sprintf("the MCreator installed here does not have the %s generator this project was made with", e.Name)
	if len(e.Available) > 0 {
		msg += "; it can build for " + strings.Join(e.Available, ", ")
	}
	return msg + ". Install the MCreator version the project was made in, or open it in MCreator and switch its generator"
}

// Generators lists the generators the installed plugins provide.
func (t *Toolkit) Generators() []string {
	seen := map[string]bool{}
	var names []string
	for _, src := range t.plugins() {
		entries, err := fs.ReadDir(src.fsys, ".")
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() || seen[e.Name()] {
				continue
			}
			if _, err := fs.Stat(src.fsys, e.Name()+"/generator.yaml"); err == nil {
				seen[e.Name()] = true
				names = append(names, e.Name())
			}
		}
	}
	sort.Strings(names)
	return names
}

// APIGradle returns the Gradle configuration a mod API (GeckoLib and the
// like) adds to a workspace using the given generator.
func (t *Toolkit) APIGradle(api, generator string) (string, error) {
	for _, src := range t.plugins() {
		data, err := fs.ReadFile(src.fsys, path.Join("apis", api+".yaml"))
		if err != nil {
			continue
		}
		if gradle, ok := yamlLookup(string(data), generator, "gradle"); ok {
			return gradle, nil
		}
	}
	return "", fmt.Errorf("the project uses the %q mod API, and no installed MCreator plugin provides it for %s", api, generator)
}

// Java picks the JDK to build with: the one bundled with MCreator, since it is
// what MCreator itself builds with, otherwise the system's. A build prefers
// the JDK of the installation its generator came from; see Generator.Java.
func (t *Toolkit) Java() (string, error) {
	for _, inst := range t.Installs {
		if inst.Java != "" {
			return inst.Java, nil
		}
	}
	if home := os.Getenv("JAVA_HOME"); home != "" && isJavaHome(home) {
		return home, nil
	}
	if java, err := exec.LookPath("java"); err == nil {
		if resolved, err := filepath.EvalSymlinks(java); err == nil {
			java = resolved
		}
		home := filepath.Dir(filepath.Dir(java))
		if isJavaHome(home) {
			return home, nil
		}
	}
	return "", errors.New("no Java was found: install MCreator (it comes with its own Java) or a JDK")
}

func isJavaHome(dir string) bool {
	for _, name := range []string{"java", "java.exe"} {
		if st, err := os.Stat(filepath.Join(dir, "bin", name)); err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}
