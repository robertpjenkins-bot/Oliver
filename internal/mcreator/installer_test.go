package mcreator

import (
	"archive/zip"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// These tests compile the world installer with a real JDK and run it from a
// jar, the way a mod loader would, against stand-ins for the loader classes
// it looks up by name.

// Forge's loader API, as far as the installer uses it. GAMEDIR has a body so
// the constant is an anonymous subclass, as real enum constants can be.
const fakeFMLPaths = `package net.minecraftforge.fml.loading;

import java.nio.file.Path;
import java.nio.file.Paths;

public enum FMLPaths {
	GAMEDIR {
	};

	public Path get() {
		return Paths.get(System.getProperty("gamedir"));
	}
}
`

const fakeFMLEnvironment = `package net.minecraftforge.fml.loading;

public class FMLEnvironment {
	public static final Dist dist = Dist.valueOf(System.getProperty("side", "CLIENT"));

	public enum Dist { CLIENT, DEDICATED_SERVER }
}
`

const fakeMain = `package net.mcreator.rubytools;

public class Main {
	public static void main(String[] args) {
		ZipTransformWorld.install();
	}
}
`

type modJar struct {
	t    *testing.T
	java string
	path string
}

// buildModJar compiles the installer, optionally with the fake loader, and
// packs it into a jar together with a bundled world.
func buildModJar(t *testing.T, withLoader bool) *modJar {
	t.Helper()
	javac, err := exec.LookPath("javac")
	if err != nil {
		t.Skip("javac is not installed")
	}
	// --release 17 needs JDK 17 or newer, the oldest Java a supported Forge uses.
	if out, err := exec.Command(javac, "-version").CombinedOutput(); err != nil || javaMajor(string(out)) < 17 {
		t.Skipf("javac is older than 17: %s", out)
	}
	java, err := exec.LookPath("java")
	if err != nil {
		t.Skip("java is not installed")
	}

	dir := t.TempDir()
	sources := map[string]string{
		"net/mcreator/rubytools/ZipTransformWorld.java": installerSource("net.mcreator.rubytools", "ruby_tools", "New World (ruby_tools)"),
		"net/mcreator/rubytools/Main.java":              fakeMain,
	}
	if withLoader {
		sources["net/minecraftforge/fml/loading/FMLPaths.java"] = fakeFMLPaths
		sources["net/minecraftforge/fml/loading/FMLEnvironment.java"] = fakeFMLEnvironment
	}
	args := []string{"--release", "17", "-d", filepath.Join(dir, "classes")}
	for name, src := range sources {
		p := filepath.Join(dir, "src", filepath.FromSlash(name))
		os.MkdirAll(filepath.Dir(p), 0o755)
		if err := os.WriteFile(p, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		args = append(args, p)
	}
	if out, err := exec.Command(javac, args...).CombinedOutput(); err != nil {
		t.Fatalf("javac: %v\n%s", err, out)
	}

	entries := map[string]string{
		"ziptransform-world/ruby_tools/index.txt":              "level.dat\nregion/r.0.0.mca\n",
		"ziptransform-world/ruby_tools/world/level.dat":        "LEVEL",
		"ziptransform-world/ruby_tools/world/region/r.0.0.mca": "REGION",
		"ziptransform-world/other_mod/index.txt":               "level.dat\n",
		"ziptransform-world/other_mod/world/level.dat":         "SOMEONE ELSE'S",
		"META-INF/MANIFEST.MF":                                 "Manifest-Version: 1.0\n",
	}
	classes := filepath.Join(dir, "classes")
	filepath.WalkDir(classes, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(classes, p)
			entries[filepath.ToSlash(rel)] = readFile(t, p)
		}
		return err
	})
	jar := filepath.Join(dir, "ruby_tools.jar")
	f, err := os.Create(jar)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, content := range entries {
		w, _ := zw.Create(name)
		w.Write([]byte(content))
	}
	zw.Close()
	f.Close()
	return &modJar{t: t, java: java, path: jar}
}

// javaMajor reads the major version from "javac 21.0.2".
func javaMajor(version string) int {
	for _, field := range strings.Fields(version) {
		if major, _, _ := strings.Cut(field, "."); major != "" && major[0] >= '0' && major[0] <= '9' {
			n, err := strconv.Atoi(major)
			if err == nil {
				return n
			}
		}
	}
	return 0
}

// launch starts the "game" with the mod on the classpath.
func (m *modJar) launch(cwd string, props ...string) string {
	m.t.Helper()
	args := append(props, "-cp", m.path, "net.mcreator.rubytools.Main")
	cmd := exec.Command(m.java, args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		m.t.Fatalf("java: %v\n%s", err, out)
	}
	return string(out)
}

func TestInstallerAddsTheWorldOnce(t *testing.T) {
	mod := buildModJar(t, true)
	game := t.TempDir()
	world := filepath.Join(game, "saves", "New World (ruby_tools)")

	out := mod.launch(t.TempDir(), "-Dgamedir="+game)
	if !strings.Contains(out, "added the world") {
		t.Errorf("no success message:\n%s", out)
	}
	if got := readFile(t, filepath.Join(world, "region", "r.0.0.mca")); got != "REGION" {
		t.Errorf("region file = %q", got)
	}
	if got := readFile(t, filepath.Join(world, "level.dat")); got != "LEVEL" {
		t.Errorf("level.dat = %q (read another mod's world?)", got)
	}
	if _, err := os.Stat(world + ".installing"); err == nil {
		t.Error("the staging folder was left behind")
	}

	// Play the world, then start the game again: the progress must survive.
	os.WriteFile(filepath.Join(world, "level.dat"), []byte("PLAYED"), 0o644)
	mod.launch(t.TempDir(), "-Dgamedir="+game)
	if got := readFile(t, filepath.Join(world, "level.dat")); got != "PLAYED" {
		t.Errorf("the installed world was overwritten: %q", got)
	}
}

func TestInstallerSkipsDedicatedServers(t *testing.T) {
	mod := buildModJar(t, true)
	game := t.TempDir()
	mod.launch(t.TempDir(), "-Dgamedir="+game, "-Dside=DEDICATED_SERVER")
	if _, err := os.Stat(filepath.Join(game, "saves")); err == nil {
		t.Error("a server got a saves folder")
	}
}

func TestInstallerWithoutKnownLoaderUsesWorkingDirectory(t *testing.T) {
	mod := buildModJar(t, false)
	game := t.TempDir()
	mod.launch(game)
	if got := readFile(t, filepath.Join(game, "saves", "New World (ruby_tools)", "level.dat")); got != "LEVEL" {
		t.Errorf("level.dat = %q", got)
	}
}

// Forge and NeoForge load mods as Java modules, where resources inside a
// module's packages are hidden from lookups. The bundled world must stay
// reachable there too.
func TestInstallerWorksFromAModule(t *testing.T) {
	mod := buildModJar(t, true)
	game := t.TempDir()
	cmd := exec.Command(mod.java, "-Dgamedir="+game, "-p", mod.path, "-m", "ruby.tools/net.mcreator.rubytools.Main")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("java: %v\n%s", err, out)
	}
	if got := readFile(t, filepath.Join(game, "saves", "New World (ruby_tools)", "region", "r.0.0.mca")); got != "REGION" {
		t.Errorf("region file = %q", got)
	}
}
