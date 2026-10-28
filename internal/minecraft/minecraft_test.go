package minecraft

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/robertpjenkins-bot/oliver/internal/mcreatortest"
)

func neoForgeJar(t *testing.T, dir, file, modid, version, minecraft string) string {
	t.Helper()
	path := filepath.Join(dir, file)
	mcreatortest.WriteZip(t, path, map[string]string{
		"META-INF/neoforge.mods.toml":      mcreatortest.NeoForgeModsTOML(modid, "Ruby Tools", version, minecraft),
		"net/mcreator/rubytools/Mod.class": "class",
	})
	return path
}

func fabricJar(t *testing.T, dir, file, modid, minecraft string) string {
	t.Helper()
	path := filepath.Join(dir, file)
	mcreatortest.WriteZip(t, path, map[string]string{
		"fabric.mod.json": mcreatortest.FabricModJSON(modid, "Ruby Tools", "1.0", minecraft),
	})
	return path
}

func TestParseTOML(t *testing.T) {
	tables := parseTOML(mcreatortest.NeoForgeModsTOML("ruby_tools", "Ruby \\\"Tools\\\"", "1.2", "26.1.2") + `
[[mixins]]
config = "ruby.mixins.json" # trailing comment
authors = [
  "a = b",
]
odd = 'it''s literal'
`)
	var names []string
	for _, tb := range tables {
		names = append(names, tb.name)
	}
	if want := []string{"", "mods", "dependencies.ruby_tools", "dependencies.ruby_tools", "mixins"}; !reflect.DeepEqual(names, want) {
		t.Fatalf("tables = %q", names)
	}
	if got := tables[1].values["displayName"]; got != `Ruby "Tools"` {
		t.Errorf("displayName = %q", got)
	}
	if got := tables[1].values["description"]; got != "A mod made in MCreator.\nIt has two lines.\n" {
		t.Errorf("multi-line string = %q", got)
	}
	if tables[0].values["modLoader"] != "javafml" || tables[3].values["versionRange"] != "[26.1.2]" {
		t.Errorf("values: %v / %v", tables[0].values, tables[3].values)
	}
	if got := tables[4].values["config"]; got != "ruby.mixins.json" {
		t.Errorf("value with comment = %q", got)
	}
}

func TestReadNeoForgeMod(t *testing.T) {
	m, err := ReadMod(neoForgeJar(t, t.TempDir(), "ruby.jar", "ruby_tools", "1.2", "26.1.2"))
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != "ruby_tools" || m.Name != "Ruby Tools" || m.Version != "1.2" {
		t.Errorf("mod = %+v", m)
	}
	if !reflect.DeepEqual(m.Loaders, []string{NeoForge}) || m.Minecraft[NeoForge] != "[26.1.2]" || len(m.Requires) != 0 {
		t.Errorf("mod = %+v", m)
	}
	if got := m.Needs(); got != "NeoForge, Minecraft 26.1.2" {
		t.Errorf("Needs = %q", got)
	}
}

func TestReadForgeMod(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.jar")
	mcreatortest.WriteZip(t, path, map[string]string{
		"META-INF/MANIFEST.MF": "Manifest-Version: 1.0\nImplementation-Version: 3.4.5\n",
		"META-INF/mods.toml": `modLoader="javafml"
[[mods]]
modId="oldmod"
version="${file.jarVersion}"
[[dependencies.oldmod]]
    modId="forge"
    mandatory=true
    versionRange="[47,)"
[[dependencies.oldmod]]
    modId="minecraft"
    mandatory=true
    versionRange="[1.20.1,1.21)"
[[dependencies.oldmod]]
    modId="geckolib"
    mandatory=true
    versionRange="[4.4,)"
[[dependencies.oldmod]]
    modId="jei"
    mandatory=false
`,
	})
	m, err := ReadMod(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Version != "3.4.5" || m.Name != "oldmod" || !reflect.DeepEqual(m.Loaders, []string{Forge}) {
		t.Errorf("mod = %+v", m)
	}
	if !reflect.DeepEqual(m.Requires, []Dependency{{ID: "geckolib", Versions: "[4.4,)"}}) {
		t.Errorf("requires = %+v", m.Requires)
	}
}

func TestReadFabricAndQuiltMods(t *testing.T) {
	m, err := ReadMod(fabricJar(t, t.TempDir(), "f.jar", "ruby_tools", "1.21.1"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.Loaders, []string{Fabric}) || m.Minecraft[Fabric] != "~1.21.1" {
		t.Errorf("mod = %+v", m)
	}
	if !reflect.DeepEqual(m.Requires, []Dependency{{ID: "fabric-api", Versions: "*"}}) {
		t.Errorf("requires = %+v", m.Requires)
	}

	path := filepath.Join(t.TempDir(), "q.jar")
	mcreatortest.WriteZip(t, path, map[string]string{
		"quilt.mod.json": `{"quilt_loader": {"id": "quiltmod", "version": "2.0", "metadata": {"name": "Quilt Mod"},
			"depends": ["qsl", {"id": "minecraft", "versions": [">=1.20", "1.21.x"]}, {"id": "emi", "optional": true}]}}`,
	})
	q, err := ReadMod(path)
	if err != nil {
		t.Fatal(err)
	}
	if q.Name != "Quilt Mod" || q.Minecraft[Quilt] != ">=1.20 || 1.21.x" || !reflect.DeepEqual(q.Requires, []Dependency{{ID: "qsl", Versions: "*"}}) {
		t.Errorf("quilt mod = %+v", q)
	}
}

func TestReadModRejectsOtherJars(t *testing.T) {
	dir := t.TempDir()
	library := filepath.Join(dir, "lib.jar")
	mcreatortest.WriteZip(t, library, map[string]string{"com/example/A.class": "x"})
	text := filepath.Join(dir, "notes.jar")
	os.WriteFile(text, []byte("hello"), 0o644)
	for _, p := range []string{library, text} {
		if _, err := ReadMod(p); !errors.Is(err, ErrNotMod) {
			t.Errorf("ReadMod(%s) = %v", filepath.Base(p), err)
		}
	}
}

func TestAccepts(t *testing.T) {
	cases := []struct {
		loader, versions, mc string
		ok, understood       bool
	}{
		{NeoForge, "[26.1.2]", "26.1.2", true, true},
		{NeoForge, "[26.1.2]", "26.1.1", false, true},
		{NeoForge, "[1.21.1]", "1.21.1", true, true},
		{Forge, "[1.20.1,1.21)", "1.20.4", true, true},
		{Forge, "[1.20.1,1.21)", "1.21", false, true},
		{Forge, "[1.20,1.20.1],[1.21,)", "1.21.4", true, true},
		{Forge, "(1.20,1.21]", "1.20", false, true},
		{Forge, "1.20.1", "1.21", true, true}, // a bare Maven version constrains nothing
		{Forge, "", "1.21", true, true},
		{Fabric, "~1.21.1", "1.21.4", true, true},
		{Fabric, "~1.21.1", "1.21", false, true},
		{Fabric, "~1.21.1", "1.22", false, true},
		{Fabric, ">=1.20 <1.21", "1.20.6", true, true},
		{Fabric, ">=1.20 <1.21", "1.21", false, true},
		{Fabric, "1.20.x || 1.21.x", "1.21.1", true, true},
		{Fabric, "1.21.1", "1.21.1", true, true},
		{Fabric, "*", "26.1", true, true},
		{Quilt, "^1.20", "1.21.1", true, true},
		{Fabric, "~1.21.1", "24w14a", false, false},
		{NeoForge, "[1.21.1]", "1.21.1-rc1", true, true},
	}
	for _, c := range cases {
		ok, understood := Accepts(c.loader, c.versions, c.mc)
		if ok != c.ok || understood != c.understood {
			t.Errorf("Accepts(%s, %q, %s) = %v, %v; want %v, %v", c.loader, c.versions, c.mc, ok, understood, c.ok, c.understood)
		}
	}
}

func TestDescribeRange(t *testing.T) {
	for in, want := range map[string]string{
		"[26.1.2]":    "26.1.2",
		"~1.21.1":     "1.21.1 or a later 1.21",
		"[1.20,)":     "1.20 or later",
		">=1.20":      "1.20 or later",
		"*":           "any version",
		"[1.20,1.21)": "[1.20,1.21)",
	} {
		if got := DescribeRange(in); got != want {
			t.Errorf("DescribeRange(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFindInstances(t *testing.T) {
	home := mcreatortest.Isolate(t)
	mcreatortest.PrismInstance(t, home, "Ruby Test", "26.1.2", "net.neoforged", "26.1.2.12")
	mcreatortest.PrismInstance(t, home, "Vanilla", "1.21.1", "", "")

	// Prism installed natively, with its instances moved elsewhere.
	native := filepath.Join(home, ".local", "share", "PrismLauncher")
	os.MkdirAll(native, 0o755)
	os.WriteFile(filepath.Join(native, "prismlauncher.cfg"), []byte("InstanceDir=/nonexistent\n"), 0o644)

	// The official launcher with a NeoForge profile and the default one.
	mc := filepath.Join(home, ".minecraft")
	os.MkdirAll(filepath.Join(mc, "versions", "neoforge-21.1.77"), 0o755)
	os.WriteFile(filepath.Join(mc, "versions", "neoforge-21.1.77", "neoforge-21.1.77.json"), []byte(`{"id": "neoforge-21.1.77", "inheritsFrom": "1.21.1"}`), 0o644)
	os.WriteFile(filepath.Join(mc, "launcher_profiles.json"), []byte(`{"profiles": {
		"a": {"name": "NeoForge", "type": "custom", "lastVersionId": "neoforge-21.1.77", "gameDir": "`+filepath.Join(home, "games", "neo")+`"},
		"b": {"name": "", "type": "latest-release", "lastVersionId": "latest-release"},
		"c": {"name": "", "type": "latest-snapshot", "lastVersionId": "latest-snapshot"}
	}}`), 0o644)

	// A Modrinth App profile whose versions live in the app's database.
	os.MkdirAll(filepath.Join(home, ".local", "share", "ModrinthApp", "profiles", "Adventure", "mods"), 0o755)

	// CurseForge.
	cf := filepath.Join(home, "curseforge", "minecraft", "Instances", "Pack")
	os.MkdirAll(cf, 0o755)
	os.WriteFile(filepath.Join(cf, "minecraftinstance.json"), []byte(`{"name": "Big Pack", "gameVersion": "1.20.1", "baseModLoader": {"name": "forge-47.3.0"}}`), 0o644)

	custom := filepath.Join(home, "server", "mods")
	got := map[string]Instance{}
	for _, in := range FindInstances(custom) {
		got[in.Launcher+"/"+in.Name] = in
		if in.ID == "" {
			t.Errorf("%s has no ID", in.Name)
		}
	}

	check := func(key, label, modsDir string) {
		t.Helper()
		in, ok := got[key]
		if !ok {
			t.Errorf("%s not found among %v", key, keys(got))
			return
		}
		if in.Label() != label || in.ModsDir != modsDir {
			t.Errorf("%s = %q in %s; want %q in %s", key, in.Label(), in.ModsDir, label, modsDir)
		}
	}
	prism := filepath.Join(home, ".var", "app", "org.prismlauncher.PrismLauncher", "data", "PrismLauncher", "instances")
	check("Prism Launcher/Ruby Test", "NeoForge 26.1.2.12, Minecraft 26.1.2", filepath.Join(prism, "Ruby Test", "minecraft", "mods"))
	check("Prism Launcher/Vanilla", "Minecraft 1.21.1 without mods", filepath.Join(prism, "Vanilla", "minecraft", "mods"))
	check("Minecraft Launcher/NeoForge", "NeoForge, Minecraft 1.21.1", filepath.Join(home, "games", "neo", "mods"))
	check("Minecraft Launcher/Latest release", "Minecraft latest without mods", filepath.Join(mc, "mods"))
	check("Modrinth App/Adventure", "unknown Minecraft version", filepath.Join(home, ".local", "share", "ModrinthApp", "profiles", "Adventure", "mods"))
	check("CurseForge/Big Pack", "Forge 47.3.0, Minecraft 1.20.1", filepath.Join(cf, "mods"))
	check("Folder/server", "unknown Minecraft version", custom)
	if len(got) != 7 {
		t.Errorf("found %d instances: %v", len(got), keys(got))
	}
}

func keys(m map[string]Instance) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestInstallSwitchesOffTheOlderCopy(t *testing.T) {
	dir := t.TempDir()
	mods := filepath.Join(dir, "instance", "mods") // not created yet
	downloads := filepath.Join(dir, "Downloads")
	os.MkdirAll(downloads, 0o755)

	old := neoForgeJar(t, downloads, "ruby_tools-1.1-neoforge-26.1.2.jar", "ruby_tools", "1.1", "26.1.2")
	m, _ := ReadMod(old)
	if _, err := Install(old, m, mods); err != nil {
		t.Fatal(err)
	}
	other := neoForgeJar(t, downloads, "other.jar", "other_mod", "1", "26.1.2")
	om, _ := ReadMod(other)
	Install(other, om, mods)

	newer := neoForgeJar(t, downloads, "ruby_tools-1.2-neoforge-26.1.2.jar", "ruby_tools", "1.2", "26.1.2")
	m, _ = ReadMod(newer)
	res, err := Install(newer, m, mods)
	if err != nil {
		t.Fatal(err)
	}
	if res.File != "ruby_tools-1.2-neoforge-26.1.2.jar" || !reflect.DeepEqual(res.TurnedOff, []string{"ruby_tools-1.1-neoforge-26.1.2.jar"}) {
		t.Errorf("result = %+v", res)
	}

	var state []string
	for _, im := range ListMods(mods) {
		state = append(state, im.File)
	}
	want := []string{"other.jar", "ruby_tools-1.1-neoforge-26.1.2.jar.disabled", "ruby_tools-1.2-neoforge-26.1.2.jar"}
	if !reflect.DeepEqual(state, want) {
		t.Errorf("mods folder = %v", state)
	}

	// Installing the same build again just replaces it.
	if res, err := Install(newer, m, mods); err != nil || len(res.TurnedOff) != 0 {
		t.Errorf("reinstall = %+v, %v", res, err)
	}
	if entries, _ := os.ReadDir(mods); len(entries) != 3 {
		t.Errorf("reinstall left %d files", len(entries))
	}
}

func TestInstallTurnsOnADisabledFile(t *testing.T) {
	dir := t.TempDir()
	src := neoForgeJar(t, dir, "ruby.jar.disabled", "ruby_tools", "1.2", "26.1.2")
	m, err := ReadMod(src)
	if err != nil {
		t.Fatal(err)
	}
	mods := filepath.Join(dir, "mods")
	os.MkdirAll(mods, 0o755)
	os.WriteFile(filepath.Join(mods, "ruby.jar.disabled"), []byte("old"), 0o644)

	res, err := Install(src, m, mods)
	if err != nil || res.File != "ruby.jar" {
		t.Fatalf("Install = %+v, %v", res, err)
	}
	list := ListMods(mods)
	if len(list) != 1 || !list[0].Enabled || list[0].Mod == nil {
		t.Errorf("mods = %+v", list)
	}
}

func TestSetEnabled(t *testing.T) {
	mods := t.TempDir()
	neoForgeJar(t, mods, "ruby.jar", "ruby_tools", "1.2", "26.1.2")

	off, err := SetEnabled(mods, "ruby.jar", false)
	if err != nil || off != "ruby.jar.disabled" {
		t.Fatalf("off = %q, %v", off, err)
	}
	if list := ListMods(mods); len(list) != 1 || list[0].Enabled || list[0].Name() != "Ruby Tools" {
		t.Errorf("after switching off: %+v", list)
	}
	on, err := SetEnabled(mods, off, true)
	if err != nil || on != "ruby.jar" {
		t.Fatalf("on = %q, %v", on, err)
	}
	for _, bad := range []string{"../ruby.jar", "", "notes.txt"} {
		if _, err := SetEnabled(mods, bad, false); err == nil {
			t.Errorf("SetEnabled(%q) succeeded", bad)
		}
	}
}

func TestCheck(t *testing.T) {
	dir := t.TempDir()
	neo, _ := ReadMod(neoForgeJar(t, dir, "n.jar", "ruby_tools", "1.2", "26.1.2"))
	fab, _ := ReadMod(fabricJar(t, dir, "f.jar", "ruby_fabric", "1.21.1"))
	api := []InstalledMod{{File: "fabric-api.jar", Enabled: true, Mod: &Mod{ID: "fabric-api"}}}

	cases := []struct {
		name      string
		mod       *Mod
		in        Instance
		installed []InstalledMod
		ok        bool
		unsure    bool
		reason    string
		missing   []string
	}{
		{"fits", neo, Instance{Known: true, Loader: NeoForge, Minecraft: "26.1.2"}, nil, true, false, "", nil},
		{"wrong loader", neo, Instance{Known: true, Loader: Fabric, Minecraft: "26.1.2"}, nil, false, false, "This is Fabric. The mod needs NeoForge, Minecraft 26.1.2.", nil},
		{"wrong version", neo, Instance{Known: true, Loader: NeoForge, Minecraft: "1.21.1"}, nil, false, false, "This is Minecraft 1.21.1. The mod needs Minecraft 26.1.2.", nil},
		{"vanilla", neo, Instance{Known: true, Minecraft: "26.1.2"}, nil, false, false, "This is plain Minecraft, with no mod loader. The mod needs NeoForge, Minecraft 26.1.2.", nil},
		{"unknown", neo, Instance{}, nil, true, true, "Zip Transform cannot tell which Minecraft this is. Make sure it is NeoForge, Minecraft 26.1.2.", nil},
		{"needs Fabric API", fab, Instance{Known: true, Loader: Fabric, Minecraft: "1.21.4"}, nil, true, false, "", []string{"fabric-api"}},
		{"has Fabric API", fab, Instance{Known: true, Loader: Fabric, Minecraft: "1.21.4"}, api, true, false, "", nil},
		{"Quilt runs Fabric mods", fab, Instance{Known: true, Loader: Quilt, Minecraft: "1.21.1"}, api, true, false, "", nil},
	}
	for _, c := range cases {
		f := Check(c.mod, c.in, c.installed)
		if f.OK != c.ok || f.Unsure != c.unsure || f.Reason != c.reason || !reflect.DeepEqual(f.Missing, c.missing) {
			t.Errorf("%s: %+v", c.name, f)
		}
	}
	if !strings.Contains(ModName("fabric-api"), "Fabric API") || ModName("ruby") != "ruby" {
		t.Error("ModName")
	}
}
