package mcreator

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// BuildOptions controls a build.
type BuildOptions struct {
	World   string    // the world to bundle, by folder name; empty for none
	Output  string    // where to write the jar; empty for beside the ZIP
	Log     io.Writer // receives progress lines and Gradle's output
	TempDir string    // parent of the build folder; empty for the system default
}

// Result describes a finished build.
type Result struct {
	Jar   string
	World string // the folder the bundled world installs as, if one was bundled
}

// Build turns the project into a mod jar.
func Build(ctx context.Context, p *Project, t *Toolkit, opts BuildOptions) (*Result, error) {
	log := opts.Log
	if log == nil {
		log = io.Discard
	}
	logf := func(format string, args ...any) { fmt.Fprintf(log, "» "+format+"\n", args...) }

	if p.Generator == "" {
		return nil, errors.New("the workspace does not say which Minecraft version it is for")
	}
	if !p.MakesJar() {
		return nil, fmt.Errorf("%s projects do not build into a jar; use File → Export the mod for distribution in MCreator instead", p.GeneratorLabel())
	}
	var world World
	if opts.World != "" {
		if !p.CanBundleWorld() {
			return nil, fmt.Errorf("a world cannot be bundled with a %s plugin", p.GeneratorLabel())
		}
		w, ok := p.World(opts.World)
		if !ok {
			return nil, fmt.Errorf("the ZIP has no world called %q", opts.World)
		}
		world = w
	}

	g, err := t.Generator(p.Generator)
	if err != nil {
		return nil, err
	}
	java := g.Java
	if java == "" {
		if java, err = t.Java(); err != nil {
			return nil, err
		}
	}
	if opts.Output == "" {
		opts.Output = filepath.Join(filepath.Dir(p.ZipPath), p.JarName())
	}
	logf("building %s %s for %s", p.ModName, p.Version, p.GeneratorLabel())
	logf("build files from %s", g.From)
	logf("Java from %s", java)

	work, err := os.MkdirTemp(opts.TempDir, "ziptransform-"+safeFileName(p.ModID)+"-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(work)

	if err := prepare(p, t, g, work); err != nil {
		return nil, err
	}
	res := &Result{}
	if opts.World != "" {
		logf("bundling the world %q (%s)", world.Name, HumanSize(world.Size))
		if err := bundleWorld(p, world, work); err != nil {
			return nil, fmt.Errorf("bundling the world: %w", err)
		}
		res.World = InstalledWorldName(p, world)
	}

	logf("running Gradle — the first build downloads Minecraft and can take several minutes")
	tail := &tailWriter{max: 40}
	cmd := gradleCommand(ctx, work, java, t.GradleHome(), jvmArgs(work))
	cmd.Stdout = io.MultiWriter(log, tail)
	cmd.Stderr = cmd.Stdout
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, &GradleError{Err: err, Tail: tail.lines()}
	}

	built, err := exportedFile(work, g)
	if err != nil {
		return nil, err
	}
	if err := copyAtomic(built, opts.Output); err != nil {
		return nil, err
	}
	res.Jar = opts.Output
	logf("wrote %s", opts.Output)
	return res, nil
}

// prepare lays the workspace out the way MCreator would before building: the
// project's own sources, and the generator's build files over the top.
func prepare(p *Project, t *Toolkit, g *Generator, work string) error {
	// Only src (and lib, where mod APIs keep jars) is taken from the ZIP. That
	// is all a build needs, and nothing else in a ZIP from someone else gets a
	// chance to run: Gradle executes build scripts, not Java sources.
	for _, f := range p.sources {
		rel := strings.TrimPrefix(f.Name, p.root)
		if !filepath.IsLocal(rel) {
			return fmt.Errorf("the ZIP contains an unsafe file name: %q", f.Name)
		}
		if err := extractFile(f, filepath.Join(work, filepath.FromSlash(rel))); err != nil {
			return err
		}
	}

	values := templateValues(p, g)
	for name, content := range g.files {
		if strings.HasPrefix(name, "run/") {
			continue // settings for the test client, not needed to build
		}
		if isTemplated(name) {
			rendered, err := renderTemplate(string(content), values)
			if err != nil {
				return fmt.Errorf("%s from the %s generator %w", name, g.Name, err)
			}
			content = []byte(rendered)
		}
		mode := os.FileMode(0o644)
		if name == "gradlew" || strings.HasSuffix(name, ".sh") {
			mode = 0o755
		}
		dest := filepath.Join(work, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, content, mode); err != nil {
			return err
		}
	}

	// mcreator.gradle carries the dependencies of the mod APIs the project
	// uses. MCreator regenerates it rather than export it.
	var apis []string
	for _, api := range p.APIs {
		gradle, err := t.APIGradle(api, g.Name)
		if err != nil {
			return err
		}
		apis = append(apis, gradle)
	}
	return os.WriteFile(filepath.Join(work, "mcreator.gradle"), []byte(strings.Join(apis, "\n\n")), 0o644)
}

// gradleCommand runs the build the way MCreator does: with a chosen JDK as the
// only toolchain Gradle may use, and MCreator's Gradle cache.
func gradleCommand(ctx context.Context, work, java, gradleHome, jvm string) *exec.Cmd {
	script := filepath.Join(work, "gradlew")
	if runtime.GOOS == "windows" {
		script += ".bat"
	}
	args := []string{
		"build", "--no-daemon", "--console=plain",
		"-Porg.gradle.java.installations.auto-detect=false",
		"-Porg.gradle.java.installations.paths=" + filepath.ToSlash(java),
		"-Dorg.gradle.jvmargs=" + jvm,
	}
	if gradleHome != "" {
		args = append(args, "--gradle-user-home", gradleHome)
	}
	cmd := exec.CommandContext(ctx, script, args...)
	cmd.Dir = work
	cmd.Env = gradleEnv(os.Environ(), java)
	return cmd
}

// gradleEnv removes the variables MCreator removes, any of which can point
// Gradle at the wrong Java or the wrong cache, and sets JAVA_HOME.
func gradleEnv(env []string, java string) []string {
	drop := map[string]bool{
		"JAVA_HOME": true, "JRE_HOME": true, "JDK_HOME": true, "CLASSPATH": true,
		"_JAVA_OPTIONS": true, "JAVA_OPTS": true, "JAVA_TOOL_OPTIONS": true,
		"JAVACMD": true, "JDK_JAVA_OPTIONS": true,
		"GRADLE_HOME": true, "GRADLE_OPTS": true, "GRADLE_USER_HOME": true,
		"CI": true, // https://github.com/neoforged/ModDevGradle/pull/320
	}
	out := make([]string, 0, len(env)+1)
	for _, kv := range env {
		k, _, _ := strings.Cut(kv, "=")
		if !drop[k] {
			out = append(out, kv)
		}
	}
	return append(out, "JAVA_HOME="+java)
}

// jvmArgs keeps the generator's own Gradle JVM options but sets the heap the
// way MCreator does, which overrides the generator's choice: 3 GB, or less on
// a machine without the memory for it.
func jvmArgs(work string) string {
	args := []string{}
	if data, err := os.ReadFile(filepath.Join(work, "gradle.properties")); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if v, ok := strings.CutPrefix(strings.TrimSpace(line), "org.gradle.jvmargs="); ok {
				for _, a := range strings.Fields(v) {
					if !strings.HasPrefix(a, "-Xmx") {
						args = append(args, a)
					}
				}
			}
		}
	}
	heap := 3072
	if total := totalMemoryMB(); total > 0 && total-1024 < heap {
		heap = max(total-1024, 1024)
	}
	args = append([]string{"-Xmx" + strconv.Itoa(heap) + "m"}, args...)
	// MCreator's error decoding relies on English output, and Mod Dev Gradle
	// behaves only when it believes it is running inside an IDE.
	return strings.Join(append(args, "-Duser.language=en", "-Declipse.application=net.mcreator"), " ")
}

func totalMemoryMB() int {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "MemTotal:"); ok {
			kb, _ := strconv.Atoi(strings.TrimSuffix(strings.TrimSpace(v), " kB"))
			return kb / 1024
		}
	}
	return 0
}

// exportedFile finds the jar the generator's build produces.
func exportedFile(work string, g *Generator) (string, error) {
	if g.ExportFile != "" {
		p := filepath.Join(work, filepath.FromSlash(g.ExportFile))
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	// Fall back to the one jar in build/libs that is not sources or javadoc.
	jars, _ := filepath.Glob(filepath.Join(work, "build", "libs", "*.jar"))
	var found []string
	for _, j := range jars {
		base := filepath.Base(j)
		if !strings.HasSuffix(base, "-sources.jar") && !strings.HasSuffix(base, "-javadoc.jar") {
			found = append(found, j)
		}
	}
	if len(found) == 1 {
		return found[0], nil
	}
	return "", fmt.Errorf("the build finished but did not produce %s", g.ExportFile)
}

// copyAtomic copies src to dst through a temporary file, so a failed copy
// never leaves a truncated jar where a good one is expected.
func copyAtomic(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".ziptransform-*.jar")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, in); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}

// GradleError is a failed Gradle build, with the end of its output.
type GradleError struct {
	Err  error
	Tail []string
}

func (e *GradleError) Error() string {
	return "the Gradle build failed (" + e.Err.Error() + ")"
}

func (e *GradleError) Unwrap() error { return e.Err }

// Hint explains the likely cause in plain terms, from the build's output.
func (e *GradleError) Hint() string {
	out := strings.Join(e.Tail, "\n")
	switch {
	case containsAny(out, "UnknownHostException", "Could not resolve", "Could not GET", "Connection refused", "timed out", "Network is unreachable"):
		return "Gradle could not download what it needs. The first build of a mod needs an internet connection."
	case containsAny(out, "Compilation failed", "error: cannot find symbol", ": error:"):
		return "The mod's code did not compile. Open the project in MCreator and use Build → Build workspace to see which element is broken."
	case containsAny(out, "Unsupported class file major version", "invalid source release", "toolchain"):
		return "The Java version does not match this Minecraft version. Building with the Java that comes with MCreator avoids this."
	case containsAny(out, "OutOfMemoryError", "Java heap space"):
		return "Gradle ran out of memory. Close other programs and try again."
	case containsAny(out, "No space left on device"):
		return "The disk is full."
	}
	return ""
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// tailWriter keeps the last lines written to it.
type tailWriter struct {
	max     int
	buf     []string
	partial string
}

func (t *tailWriter) Write(p []byte) (int, error) {
	parts := strings.Split(t.partial+string(p), "\n")
	t.partial = parts[len(parts)-1]
	for _, line := range parts[:len(parts)-1] {
		t.buf = append(t.buf, strings.TrimRight(line, "\r"))
		if len(t.buf) > t.max {
			t.buf = t.buf[1:]
		}
	}
	return len(p), nil
}

func (t *tailWriter) lines() []string {
	if t.partial != "" {
		return append(t.buf, t.partial)
	}
	return t.buf
}

// HumanSize formats a byte count.
func HumanSize(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d bytes", n)
}
