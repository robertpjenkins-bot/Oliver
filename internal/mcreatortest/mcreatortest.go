// Package mcreatortest builds the MCreator files Zip Transform reads, for
// tests: shareable workspace ZIPs, and an MCreator installation whose Gradle
// is a shell script that records how it was run.
package mcreatortest

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// WriteZip creates a ZIP holding the given files.
func WriteZip(t testing.TB, path string, files map[string]string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(files[name])); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

// WorkspaceJSON is the .mcreator file of the "Ruby Tools" test mod.
const WorkspaceJSON = `{
  "mod_elements": [],
  "workspaceSettings": {
    "modid": "ruby_tools",
    "modName": "Ruby Tools",
    "version": "1.2",
    "currentGenerator": "neoforge-1.21.1",
    "mcreatorDependencies": []
  },
  "mcreatorVersion": 202600341718
}`

// MainClass is the main mod class MCreator generates for NeoForge, cut down
// to the parts Zip Transform looks at.
const MainClass = `package net.mcreator.rubytools;

@Mod("ruby_tools") public class RubyToolsMod {

	public static final String MODID = "ruby_tools";

	public RubyToolsMod(IEventBus modEventBus) {
		// Start of user code block mod constructor
		// End of user code block mod constructor

		NeoForge.EVENT_BUS.register(this);
	}
}
`

// ShareableZip writes "Ruby Tools.zip" into dir, laid out the way MCreator
// exports a workspace with its run directory: two worlds, a folder under
// saves that is not one, and the files MCreator leaves out missing. extra
// adds files; an empty value removes one.
func ShareableZip(t testing.TB, dir string, extra map[string]string) string {
	t.Helper()
	files := map[string]string{
		"ruby_tools.mcreator":                                          WorkspaceJSON,
		"elements/RubySword.mod.json":                                  `{}`,
		"src/main/java/net/mcreator/rubytools/RubyToolsMod.java":       MainClass,
		"src/main/java/net/mcreator/rubytools/item/RubySwordItem.java": "package net.mcreator.rubytools.item;\n",
		"src/main/resources/META-INF/neoforge.mods.toml":               "modId=\"ruby_tools\"\n",
		"src/main/resources/assets/ruby_tools/textures/item/ruby.png":  "PNG",
		"run/options.txt":                      "",
		"run/saves/New World/level.dat":        "LEVEL",
		"run/saves/New World/region/r.0.0.mca": "REGION",
		"run/saves/New World/session.lock":     "LOCK",
		"run/saves/Not A World/notes.txt":      "no level.dat here",
		"run/saves/Creative Test/level.dat":    "LEVEL2",
		"gradle.properties":                    "org.gradle.jvmargs=-Xmx64m\n",
	}
	for k, v := range extra {
		if v == "" {
			delete(files, k)
		} else {
			files[k] = v
		}
	}
	path := filepath.Join(dir, "Ruby Tools.zip")
	WriteZip(t, path, files)
	return path
}

// FakeGradlew stands in for Gradle. It writes its arguments to $CAPTURE/args,
// the relevant environment to $CAPTURE/env and a copy of the workspace to
// $CAPTURE/work, then "builds" the jar — a copy of $PREBUILT_JAR when that
// is set — or fails as a download would when $FAIL_BUILD is set.
const FakeGradlew = `#!/bin/sh
printf '%s\n' "$@" > "$CAPTURE/args"
printf 'JAVA_HOME=%s\nJAVA_TOOL_OPTIONS=%s\nCI=%s\n' "$JAVA_HOME" "$JAVA_TOOL_OPTIONS" "$CI" > "$CAPTURE/env"
cp -R . "$CAPTURE/work"
if [ -n "$FAIL_BUILD" ]; then
	echo "> Could not resolve net.neoforged:neoforge:21.1.265."
	echo "BUILD FAILED"
	exit 1
fi
mkdir -p build/libs
if [ -n "$PREBUILT_JAR" ]; then cp "$PREBUILT_JAR" build/libs/modid-1.0.jar; else echo "the jar" > build/libs/modid-1.0.jar; fi
echo "the sources" > build/libs/modid-1.0-sources.jar
echo "BUILD SUCCESSFUL"
`

// FakeInstall lays out an MCreator installation in dir the way the Linux
// download unpacks — lib/mcreator.jar, plugins/generator-*.zip and a bundled
// jdk — and returns dir.
func FakeInstall(t testing.TB, dir, version string) string {
	t.Helper()
	WriteZip(t, filepath.Join(dir, "lib", "mcreator.jar"), map[string]string{
		"mcreator.conf": "mcreator=" + version + "\n",
	})
	WriteZip(t, filepath.Join(dir, "plugins", "generator-1.21.1.zip"), map[string]string{
		"plugin.json": `{"id": "generator-1.21.1"}`,
		"neoforge-1.21.1/generator.yaml": `name: NeoForge for @minecraft (@buildfileversion)
status: lts
buildfileversion: 21.1.265

# gradle task definitions
gradle:
  run_client: runClient
  export_file: "build/libs/modid-1.0.jar"
`,
		"neoforge-1.21.1/workspacebase/build.gradle": `plugins {
    id 'net.neoforged.moddev' version '2.0.148'
}
base.archivesName = "modid"
neoForge {
    version = "${generator.getGeneratorBuildFileVersion()}"
}
apply from: 'mcreator.gradle'
`,
		"neoforge-1.21.1/workspacebase/settings.gradle":                          "rootProject.name = 'base'\n",
		"neoforge-1.21.1/workspacebase/gradle.properties":                        "org.gradle.jvmargs=-Xmx1G -XX:+UseG1GC\norg.gradle.daemon=true\n",
		"neoforge-1.21.1/workspacebase/gradlew":                                  FakeGradlew,
		"neoforge-1.21.1/workspacebase/gradle/wrapper/gradle-wrapper.properties": "distributionUrl=x\n",
		"neoforge-1.21.1/workspacebase/run/options.txt":                          "lang:en_us\n",
		"datapack-1.21.1/generator.yaml":                                         "buildfileversion: 1\n",
		"datapack-1.21.1/workspacebase/build.gradle":                             "",
	})
	java := filepath.Join(dir, "jdk", "bin", "java")
	if err := os.MkdirAll(filepath.Dir(java), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(java, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// Isolate points every place MCreator could be found at an empty home
// directory, and returns it.
func Isolate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("MCREATOR_HOME", "")
	t.Setenv("JAVA_HOME", "")
	t.Setenv("PATH", "/usr/bin:/bin") // the fake Gradle needs a shell and cp
	return home
}

// NeoForgeModsTOML is the metadata MCreator writes into a NeoForge mod.
func NeoForgeModsTOML(modid, name, version, minecraft string) string {
	return `modLoader="javafml"
loaderVersion="[4,)"
license="Not specified"

[[mods]]
modId="` + modid + `"
version="` + version + `"
displayName="` + name + `"
description="""
A mod made in MCreator.
It has two lines.
"""

# Start of user code block mod configuration
# End of user code block mod configuration

[[dependencies.` + modid + `]]
    modId="neoforge"
    type="required"
    versionRange="[21.1.265,)"
    ordering="AFTER"
    side="BOTH"

[[dependencies.` + modid + `]]
    modId="minecraft"
    type="required"
    versionRange="[` + minecraft + `]"
    ordering="AFTER"
    side="BOTH"
`
}

// FabricModJSON is the metadata MCreator writes into a Fabric mod.
func FabricModJSON(modid, name, version, minecraft string) string {
	return `{
  "schemaVersion": 1,
  "id": "` + modid + `",
  "version": "` + version + `",
  "name": "` + name + `",
  "environment": "*",
  "depends": {
	"fabricloader": ">=0.19.3",
	"minecraft": "~` + minecraft + `",
	"java": ">=25",
	"fabric-api": "*"
  }
}`
}

// PrismInstance creates a Prism Launcher instance as the Flatpak keeps them,
// which is how Prism is usually installed on a Steam Deck, and returns its
// mods folder. loader is a component UID such as "net.neoforged", or empty.
func PrismInstance(t testing.TB, home, name, minecraft, loaderUID, loaderVersion string) string {
	t.Helper()
	dir := filepath.Join(home, ".var", "app", "org.prismlauncher.PrismLauncher", "data", "PrismLauncher", "instances", name)
	components := `{"uid": "net.minecraft", "version": "` + minecraft + `", "important": true}`
	if loaderUID != "" {
		components += `, {"uid": "` + loaderUID + `", "version": "` + loaderVersion + `"}`
	}
	for file, content := range map[string]string{
		"instance.cfg":  "[General]\nConfigVersion=1.2\nInstanceType=OneSix\nname=" + name + "\n",
		"mmc-pack.json": `{"components": [` + components + `], "formatVersion": 1}`,
	} {
		p := filepath.Join(dir, file)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return filepath.Join(dir, "minecraft", "mods")
}
