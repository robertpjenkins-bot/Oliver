package mcreator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/robertpjenkins-bot/oliver/internal/mcreatortest"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestYAMLLookup(t *testing.T) {
	doc := `name: NeoForge for @minecraft # the display name
buildfileversion: 21.1.265
import:
  - datapack-1.21.1
gradle:
  run_client: runClient
  export_file: "build/libs/modid-1.0.jar"
base_templates:
  - template: modbase/mod.java.ftl
    name: "@SRCROOT/@JavaModName.java"
quoted: 'it''s'
`
	cases := []struct {
		path []string
		want string
		ok   bool
	}{
		{[]string{"name"}, "NeoForge for @minecraft", true},
		{[]string{"buildfileversion"}, "21.1.265", true},
		{[]string{"gradle", "export_file"}, "build/libs/modid-1.0.jar", true},
		{[]string{"gradle", "run_client"}, "runClient", true},
		{[]string{"quoted"}, "it's", true},
		{[]string{"gradle", "missing"}, "", false},
		{[]string{"export_file"}, "", false}, // only nested, never top level
		{[]string{"name", "deeper"}, "", false},
	}
	for _, c := range cases {
		got, ok := yamlLookup(doc, c.path...)
		if got != c.want || ok != c.ok {
			t.Errorf("yamlLookup(%v) = %q, %v; want %q, %v", c.path, got, ok, c.want, c.ok)
		}
	}
}

func TestYAMLBlockScalar(t *testing.T) {
	doc := `name: GeckoLib
neoforge-1.21.1:
  gradle: |
    repositories {
        maven { url = 'https://dl.cloudsmith.io/public/geckolib3/geckolib/maven/' }
    }

    dependencies {
        implementation "software.bernie.geckolib:geckolib:4.7"
    }
  update_files:
    - lib/geckolib.jar
forge-1.20.1:
  gradle: |-
    dependencies {}
`
	got, ok := yamlLookup(doc, "neoforge-1.21.1", "gradle")
	want := "repositories {\n    maven { url = 'https://dl.cloudsmith.io/public/geckolib3/geckolib/maven/' }\n}\n\ndependencies {\n    implementation \"software.bernie.geckolib:geckolib:4.7\"\n}\n"
	if !ok || got != want {
		t.Errorf("block = %q, %v\nwant  %q", got, ok, want)
	}
	if got, _ := yamlLookup(doc, "forge-1.20.1", "gradle"); got != "dependencies {}" {
		t.Errorf("stripped block = %q", got)
	}
	if _, ok := yamlLookup(doc, "fabric-1.21.1", "gradle"); ok {
		t.Error("found a generator that is not in the file")
	}
}

func TestRenderTemplate(t *testing.T) {
	values := map[string]string{"generator.getGeneratorBuildFileVersion()": "47.3.0", "modid": "ruby"}

	got, err := renderTemplate(`version = "${ generator.getGeneratorBuildFileVersion() }" // ${modid}, ${r"${project.version}"}`, values)
	if err != nil {
		t.Fatal(err)
	}
	if want := `version = "47.3.0" // ruby, ${project.version}`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	for _, bad := range []string{"${w.hasSounds()}", "<#if x>y</#if>", "${modid"} {
		if _, err := renderTemplate(bad, values); err == nil {
			t.Errorf("renderTemplate(%q) succeeded; want an error", bad)
		}
	}
}

func TestOpenReadsWorkspace(t *testing.T) {
	p, err := Open(mcreatortest.ShareableZip(t, t.TempDir(), nil))
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	if p.ModID != "ruby_tools" || p.ModName != "Ruby Tools" || p.Version != "1.2" || p.Generator != "neoforge-1.21.1" {
		t.Errorf("read %+v", p)
	}
	if p.Package != "net.mcreator.ruby_tools" {
		t.Errorf("package defaulted to %q", p.Package)
	}
	if got := p.JavaModName(); got != "RubyToolsMod" {
		t.Errorf("JavaModName = %q", got)
	}
	if got := p.JarName(); got != "ruby_tools-1.2-neoforge-1.21.1.jar" {
		t.Errorf("JarName = %q", got)
	}
	if got := p.GeneratorLabel(); got != "NeoForge 1.21.1" {
		t.Errorf("GeneratorLabel = %q", got)
	}
	if !p.MakesJar() || !p.CanBundleWorld() {
		t.Error("a NeoForge mod should make a jar that can carry a world")
	}

	// A folder without level.dat is not a world.
	var names []string
	for _, w := range p.Worlds {
		names = append(names, w.Name)
	}
	if strings.Join(names, ",") != "Creative Test,New World" {
		t.Fatalf("worlds = %v", names)
	}
	if w, _ := p.World("New World"); w.Files != 3 || w.Size != int64(len("LEVEL")+len("REGION")+len("LOCK")) {
		t.Errorf("New World = %+v", w)
	}
}

func TestOpenWorkspaceInsideAFolder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rezipped.zip")
	mcreatortest.WriteZip(t, path, map[string]string{
		"Ruby Tools/ruby_tools.mcreator":                       mcreatortest.WorkspaceJSON,
		"Ruby Tools/src/main/java/A.java":                      "class A {}",
		"Ruby Tools/runs/client/saves/Survival/level.dat":      "L",
		"Ruby Tools/runs/client/saves/Survival/data/raids.dat": "R",
		"Other/src/main/java/B.java":                           "class B {}",
	})
	p, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if p.root != "Ruby Tools/" {
		t.Errorf("root = %q", p.root)
	}
	if len(p.Worlds) != 1 || p.Worlds[0].Name != "Survival" || p.Worlds[0].Files != 2 {
		t.Errorf("worlds = %+v", p.Worlds)
	}
	for _, f := range p.sources {
		if !strings.HasPrefix(f.Name, "Ruby Tools/") {
			t.Errorf("took a source from outside the workspace: %s", f.Name)
		}
	}
}

func TestOpenRejectsOtherFiles(t *testing.T) {
	dir := t.TempDir()
	notZip := filepath.Join(dir, "notes.zip")
	os.WriteFile(notZip, []byte("plain text"), 0o644)
	jar := filepath.Join(dir, "mod.jar")
	mcreatortest.WriteZip(t, jar, map[string]string{"META-INF/MANIFEST.MF": ""})
	noWorkspace := filepath.Join(dir, "photos.zip")
	mcreatortest.WriteZip(t, noWorkspace, map[string]string{"cat.jpg": "meow"})
	noSources := filepath.Join(dir, "empty.zip")
	mcreatortest.WriteZip(t, noSources, map[string]string{"x.mcreator": mcreatortest.WorkspaceJSON})
	noModID := filepath.Join(dir, "broken.zip")
	mcreatortest.WriteZip(t, noModID, map[string]string{"x.mcreator": `{"workspaceSettings": {}}`, "src/a": "a"})

	for _, path := range []string{notZip, jar, noWorkspace, noSources, noModID} {
		p, err := Open(path)
		if err == nil {
			p.Close()
			t.Errorf("Open(%s) succeeded", filepath.Base(path))
			continue
		}
		if !errors.Is(err, ErrNotWorkspace) {
			t.Errorf("Open(%s) = %v; want ErrNotWorkspace", filepath.Base(path), err)
		}
	}
}

func TestOldWorkspacesAreForge(t *testing.T) {
	if got := normalizeGenerator("1.12.2"); got != "forge-1.12.2" {
		t.Errorf("normalizeGenerator = %q", got)
	}
	p := &Project{Generator: "datapack-1.21.1"}
	if p.MakesJar() || p.CanBundleWorld() {
		t.Error("a data pack does not build into a jar")
	}
	p.Generator = "spigot-1.21.1"
	if !p.MakesJar() || p.CanBundleWorld() {
		t.Error("a server plugin is a jar without a world list")
	}
}

func TestFindToolkit(t *testing.T) {
	home := mcreatortest.Isolate(t)
	older := mcreatortest.FakeInstall(t, filepath.Join(home, "MCreator20252"), "2025.2")
	newer := mcreatortest.FakeInstall(t, filepath.Join(home, "apps", "somewhere", "mc"), "2026.3")
	notInstall := filepath.Join(home, "MCreatorWorkspaces", "ruby_tools")
	os.MkdirAll(notInstall, 0o755)

	// The launcher MCreator writes to the desktop on first start is how an
	// installation in an unusual place is found.
	desktop := filepath.Join(home, "Desktop")
	os.MkdirAll(desktop, 0o755)
	entry := "[Desktop Entry]\nExec=/bin/bash -c 'cd \"" + newer + "\" && ./mcreator.sh'\nType=Application\nName=MCreator\n"
	os.WriteFile(filepath.Join(desktop, "mcreator.desktop"), []byte(entry), 0o644)

	tk := FindToolkit()
	defer tk.Close()
	if len(tk.Installs) != 2 {
		t.Fatalf("installs = %+v", tk.Installs)
	}
	if tk.Installs[0].Dir != newer || tk.Installs[0].Version != "2026.3" || tk.Installs[1].Dir != older {
		t.Errorf("not newest first: %+v", tk.Installs)
	}
	if java, err := tk.Java(); err != nil || java != filepath.Join(newer, "jdk") {
		t.Errorf("Java = %q, %v", java, err)
	}
	if got := tk.GradleHome(); got != filepath.Join(home, ".mcreator", "gradle") {
		t.Errorf("GradleHome = %q", got)
	}

	// A folder the user names comes first, whatever its version.
	pinned := FindToolkit(older)
	defer pinned.Close()
	if pinned.Installs[0].Dir != older {
		t.Errorf("pinned install not first: %+v", pinned.Installs)
	}

	if _, err := CheckInstall(notInstall); err == nil {
		t.Error("a workspace folder was accepted as an MCreator installation")
	}
}

func TestGeneratorFromPlugin(t *testing.T) {
	home := mcreatortest.Isolate(t)
	install := mcreatortest.FakeInstall(t, filepath.Join(home, "MCreator20263"), "2026.3")
	tk := FindToolkit()
	defer tk.Close()

	g, err := tk.Generator("neoforge-1.21.1")
	if err != nil {
		t.Fatal(err)
	}
	if g.BuildFileVersion != "21.1.265" || g.ExportFile != "build/libs/modid-1.0.jar" || g.MinecraftVersion() != "1.21.1" {
		t.Errorf("generator = %+v", g)
	}
	if g.Java != filepath.Join(install, "jdk") {
		t.Errorf("generator Java = %q", g.Java)
	}
	for _, f := range []string{"build.gradle", "gradlew", "gradle/wrapper/gradle-wrapper.properties"} {
		if _, ok := g.files[f]; !ok {
			t.Errorf("workspace base is missing %s", f)
		}
	}

	_, err = tk.Generator("forge-1.20.1")
	var missing *MissingGeneratorError
	if !errors.As(err, &missing) {
		t.Fatalf("missing generator: %v", err)
	}
	if !strings.Contains(err.Error(), "datapack-1.21.1, neoforge-1.21.1") {
		t.Errorf("error does not list what can be built: %v", err)
	}

	empty := FindToolkit()
	empty.Installs = nil
	if _, err := empty.Generator("neoforge-1.21.1"); !errors.Is(err, ErrNoMCreator) {
		t.Errorf("without MCreator: %v", err)
	}
}

func TestAPIFromUserPlugin(t *testing.T) {
	home := mcreatortest.Isolate(t)
	mcreatortest.FakeInstall(t, filepath.Join(home, "MCreator20263"), "2026.3")
	mcreatortest.WriteZip(t, filepath.Join(home, ".mcreator", "plugins", "geckolib.zip"), map[string]string{
		"apis/geckolib.yaml": "name: GeckoLib\nneoforge-1.21.1:\n  gradle: |\n    dependencies { implementation 'geckolib' }\n",
	})
	tk := FindToolkit()
	defer tk.Close()
	got, err := tk.APIGradle("geckolib", "neoforge-1.21.1")
	if err != nil || got != "dependencies { implementation 'geckolib' }\n" {
		t.Errorf("APIGradle = %q, %v", got, err)
	}
	if _, err := tk.APIGradle("geckolib", "forge-1.20.1"); err == nil {
		t.Error("found an API implementation for the wrong generator")
	}
}

// buildFixture is a project and an MCreator that a fake Gradle can build.
type buildFixture struct {
	project *Project
	toolkit *Toolkit
	capture string
	out     string
	java    string
}

func newBuildFixture(t *testing.T, extra map[string]string) *buildFixture {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fake Gradle is a shell script")
	}
	home := mcreatortest.Isolate(t)
	install := mcreatortest.FakeInstall(t, filepath.Join(home, "MCreator20263"), "2026.3")
	p, err := Open(mcreatortest.ShareableZip(t, filepath.Join(home, "Downloads"), extra))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { p.Close() })
	tk := FindToolkit()
	t.Cleanup(tk.Close)

	capture := filepath.Join(home, "capture")
	os.MkdirAll(capture, 0o755)
	t.Setenv("CAPTURE", capture)
	t.Setenv("JAVA_TOOL_OPTIONS", "-Dsomething=bad")
	t.Setenv("CI", "true")
	return &buildFixture{p, tk, capture, filepath.Join(home, "out", p.JarName()), filepath.Join(install, "jdk")}
}

func TestBuildRestoresWhatTheExportLeftOut(t *testing.T) {
	f := newBuildFixture(t, map[string]string{
		// Things a ZIP from someone else could carry that must never run.
		"settings.gradle":           "println 'from the zip'",
		"buildSrc/build.gradle":     "println 'from the zip'",
		"gradle/wrapper/evil.jar":   "x",
		"src/main/resources/x.json": "{}",
	})
	var log strings.Builder
	res, err := Build(context.Background(), f.project, f.toolkit, BuildOptions{Output: f.out, Log: &log})
	if err != nil {
		t.Fatalf("%v\n%s", err, log.String())
	}
	if res.Jar != f.out || res.World != "" {
		t.Errorf("result = %+v", res)
	}
	if got := readFile(t, f.out); got != "the jar\n" {
		t.Errorf("copied the wrong jar: %q", got)
	}

	work := filepath.Join(f.capture, "work")
	build := readFile(t, filepath.Join(work, "build.gradle"))
	if !strings.Contains(build, `version = "21.1.265"`) || strings.Contains(build, "${") {
		t.Errorf("build.gradle not rendered:\n%s", build)
	}
	if got := readFile(t, filepath.Join(work, "settings.gradle")); got != "rootProject.name = 'base'\n" {
		t.Errorf("settings.gradle came from the ZIP: %q", got)
	}
	for _, absent := range []string{"buildSrc", "gradle/wrapper/evil.jar", "run", "elements", "ruby_tools.mcreator"} {
		if _, err := os.Stat(filepath.Join(work, absent)); err == nil {
			t.Errorf("%s was copied into the build", absent)
		}
	}
	for _, present := range []string{"mcreator.gradle", "src/main/java/net/mcreator/rubytools/RubyToolsMod.java", "src/main/resources/x.json", "gradle/wrapper/gradle-wrapper.properties"} {
		if _, err := os.Stat(filepath.Join(work, present)); err != nil {
			t.Errorf("%s is missing from the build", present)
		}
	}

	args := readFile(t, filepath.Join(f.capture, "args"))
	for _, want := range []string{
		"build\n", "--no-daemon\n",
		"-Porg.gradle.java.installations.auto-detect=false\n",
		"-Porg.gradle.java.installations.paths=" + f.java + "\n",
		"--gradle-user-home\n" + filepath.Join(os.Getenv("HOME"), ".mcreator", "gradle") + "\n",
		"-XX:+UseG1GC -Duser.language=en -Declipse.application=net.mcreator\n",
	} {
		if !strings.Contains(args, want) {
			t.Errorf("Gradle arguments lack %q:\n%s", want, args)
		}
	}
	if strings.Contains(args, "-Xmx1G") {
		t.Errorf("the generator's heap was not replaced:\n%s", args)
	}

	env := readFile(t, filepath.Join(f.capture, "env"))
	if want := "JAVA_HOME=" + f.java + "\nJAVA_TOOL_OPTIONS=\nCI=\n"; env != want {
		t.Errorf("environment:\n%s\nwant:\n%s", env, want)
	}
	if !strings.Contains(log.String(), "BUILD SUCCESSFUL") {
		t.Errorf("Gradle output not passed on:\n%s", log.String())
	}
}

func TestBuildBundlesWorld(t *testing.T) {
	f := newBuildFixture(t, nil)
	res, err := Build(context.Background(), f.project, f.toolkit, BuildOptions{World: "New World", Output: f.out})
	if err != nil {
		t.Fatal(err)
	}
	if res.World != "New World (ruby_tools)" {
		t.Errorf("world installs as %q", res.World)
	}

	work := filepath.Join(f.capture, "work")
	res_ := filepath.Join(work, "src", "main", "resources", "ziptransform-world", "ruby_tools")
	if got := readFile(t, filepath.Join(res_, "index.txt")); got != "level.dat\nregion/r.0.0.mca\n" {
		t.Errorf("index = %q", got)
	}
	if got := readFile(t, filepath.Join(res_, "world", "region", "r.0.0.mca")); got != "REGION" {
		t.Errorf("region file = %q", got)
	}
	if _, err := os.Stat(filepath.Join(res_, "world", "session.lock")); err == nil {
		t.Error("the session lock was bundled")
	}

	dir := filepath.Join(work, "src", "main", "java", "net", "mcreator", "rubytools")
	main := readFile(t, filepath.Join(dir, "RubyToolsMod.java"))
	want := "\t\t// Start of user code block mod constructor\n\t\tZipTransformWorld.install(); // added by Zip Transform\n\t\t// End of user code block mod constructor\n"
	if !strings.Contains(main, want) {
		t.Errorf("main class not hooked:\n%s", main)
	}
	installer := readFile(t, filepath.Join(dir, "ZipTransformWorld.java"))
	for _, want := range []string{"package net.mcreator.rubytools;", `"/ziptransform-world/ruby_tools/"`, `"New World (ruby_tools)"`} {
		if !strings.Contains(installer, want) {
			t.Errorf("installer lacks %s", want)
		}
	}

	// The project in the ZIP is untouched, so a second build starts clean.
	if _, err := Build(context.Background(), f.project, f.toolkit, BuildOptions{Output: f.out}); err != nil {
		t.Fatal(err)
	}
}

// An older generator is built with the Java of the MCreator it came with:
// the newest MCreator's Java can be too new for an old Gradle.
func TestBuildUsesTheGeneratorsOwnJava(t *testing.T) {
	f := newBuildFixture(t, nil)
	home := os.Getenv("HOME")
	newer := mcreatortest.FakeInstall(t, filepath.Join(home, "MCreator20270"), "2027.1")
	os.Remove(filepath.Join(newer, "plugins", "generator-1.21.1.zip"))
	tk := FindToolkit()
	defer tk.Close()
	if tk.Installs[0].Dir != newer {
		t.Fatalf("installs = %+v", tk.Installs)
	}
	if _, err := Build(context.Background(), f.project, tk, BuildOptions{Output: f.out}); err != nil {
		t.Fatal(err)
	}
	if env := readFile(t, filepath.Join(f.capture, "env")); !strings.HasPrefix(env, "JAVA_HOME="+f.java+"\n") {
		t.Errorf("built with the wrong Java:\n%s", env)
	}
}

func TestBuildFailures(t *testing.T) {
	f := newBuildFixture(t, nil)
	ctx := context.Background()

	if _, err := Build(ctx, f.project, f.toolkit, BuildOptions{World: "Nether Test", Output: f.out}); err == nil {
		t.Error("built with a world that is not in the ZIP")
	}

	t.Setenv("FAIL_BUILD", "1")
	_, err := Build(ctx, f.project, f.toolkit, BuildOptions{Output: f.out})
	var ge *GradleError
	if !errors.As(err, &ge) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(ge.Hint(), "internet connection") {
		t.Errorf("hint = %q from %q", ge.Hint(), ge.Tail)
	}
	if _, err := os.Stat(f.out); err == nil {
		t.Error("a failed build left a jar behind")
	}

	f.project.Generator = "datapack-1.21.1"
	if _, err := Build(ctx, f.project, f.toolkit, BuildOptions{Output: f.out}); err == nil || !strings.Contains(err.Error(), "do not build into a jar") {
		t.Errorf("data pack: %v", err)
	}
}

func TestHookInstallerWithoutTrailingNewline(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Mod.java")
	os.WriteFile(path, []byte("    "+constructorMarker), 0o644)
	if err := hookInstaller(path); err != nil {
		t.Fatal(err)
	}
	if got, want := readFile(t, path), "    "+constructorMarker+"\n    ZipTransformWorld.install(); // added by Zip Transform\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFindMainClassNeedsTheMarker(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "A.java"), []byte("package a;\nclass A {}\n"), 0o644)
	if _, _, err := findMainClass(root); err == nil {
		t.Error("found a main class where there is none")
	}
}

func TestJavaString(t *testing.T) {
	cases := map[string]string{
		`New World`:   `"New World"`,
		`say "hi" \o`: `"say \"hi\" \\o"`,
		"Welt Ü":      `"Welt \u00dc"`,
		"⛏ 🌋":         `"\u26cf \ud83c\udf0b"`,
	}
	for in, want := range cases {
		if got := javaString(in); got != want {
			t.Errorf("javaString(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestHumanSize(t *testing.T) {
	for n, want := range map[int64]string{12: "12 bytes", 2048: "2 KB", 5 << 20: "5.0 MB", 3 << 30: "3.0 GB"} {
		if got := HumanSize(n); got != want {
			t.Errorf("HumanSize(%d) = %q, want %q", n, got, want)
		}
	}
}
