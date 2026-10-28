package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/robertpjenkins-bot/oliver/internal/mcreatortest"
)

// rubyJar writes the NeoForge 26.1.2 jar MCreator would build for Ruby Tools.
func rubyJar(t *testing.T, dir, file string) string {
	t.Helper()
	path := filepath.Join(dir, file)
	mcreatortest.WriteZip(t, path, map[string]string{
		"META-INF/neoforge.mods.toml": mcreatortest.NeoForgeModsTOML("ruby_tools", "Ruby Tools", "1.2", "26.1.2"),
	})
	return path
}

func TestInstallPicksTheInstanceThatFits(t *testing.T) {
	m := newMachine(t)
	mods := mcreatortest.PrismInstance(t, m.home, "Ruby Test", "26.1.2", "net.neoforged", "26.1.2.12")
	mcreatortest.PrismInstance(t, m.home, "Fabric Fun", "1.21.1", "net.fabricmc.fabric-loader", "0.16.5")
	jar := rubyJar(t, filepath.Join(m.home, "Downloads"), "ruby_tools-1.2-neoforge-26.1.2.jar")

	if err := m.run("install", jar); err != nil {
		t.Fatalf("%v\n%s", err, m.err)
	}
	if !m.exists(mods, "ruby_tools-1.2-neoforge-26.1.2.jar") {
		t.Error("the mod is not in the instance's mods folder")
	}
	if !strings.Contains(m.out.String(), `added to "Ruby Test" (Prism Launcher), switched on`) {
		t.Errorf("output: %s", m.out)
	}
}

func TestInstallWithoutAFittingInstance(t *testing.T) {
	m := newMachine(t)
	mcreatortest.PrismInstance(t, m.home, "Fabric Fun", "1.21.1", "net.fabricmc.fabric-loader", "0.16.5")
	jar := rubyJar(t, m.home, "ruby.jar")

	err := m.run("install", jar)
	if err == nil || !strings.Contains(err.Error(), "needs NeoForge, Minecraft 26.1.2") {
		t.Errorf("err = %v", err)
	}
	err = m.run("install", jar, "-instance", "fabric fun")
	if err == nil || !strings.Contains(err.Error(), "This is Fabric") {
		t.Errorf("forced into the wrong instance: %v", err)
	}
}

func TestInstallAsksWhenSeveralFit(t *testing.T) {
	m := newMachine(t)
	mcreatortest.PrismInstance(t, m.home, "Ruby Test", "26.1.2", "net.neoforged", "26.1.2.12")
	second := mcreatortest.PrismInstance(t, m.home, "Ruby Survival", "26.1.2", "net.neoforged", "26.1.2.12")
	jar := rubyJar(t, m.home, "ruby.jar")

	if err := m.run("install", jar); err == nil || !strings.Contains(err.Error(), "choose one with -instance") {
		t.Errorf("err = %v", err)
	}
	if err := m.run("install", "-instance", "Ruby Survival", jar); err != nil {
		t.Fatal(err)
	}
	if !m.exists(second, "ruby.jar") {
		t.Error("not installed into the named instance")
	}
}

func TestInstancesCommand(t *testing.T) {
	m := newMachine(t)
	mods := mcreatortest.PrismInstance(t, m.home, "Ruby Test", "26.1.2", "net.neoforged", "26.1.2.12")
	os.MkdirAll(mods, 0o755)
	rubyJar(t, mods, "a.jar")
	rubyJar(t, mods, "b.jar.disabled")
	if err := m.run("instances"); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Ruby Test", "Prism Launcher", "NeoForge 26.1.2.12, Minecraft 26.1.2", "1 on, 1 off"} {
		if !strings.Contains(m.out.String(), want) {
			t.Errorf("lacks %q:\n%s", want, m.out)
		}
	}

	extra := filepath.Join(m.home, "server", "mods")
	os.MkdirAll(extra, 0o755)
	if err := m.run("instances", "-folder", extra); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.out.String(), "Folder") || loadConfig().ModsFolders[0] != extra {
		t.Errorf("the folder was not added:\n%s", m.out)
	}
}

func TestBuildAndInstall(t *testing.T) {
	m := newMachine(t)
	mods := mcreatortest.PrismInstance(t, m.home, "Ruby Test", "26.1.2", "net.neoforged", "26.1.2.12")
	t.Setenv("PREBUILT_JAR", rubyJar(t, m.home, "prebuilt.jar"))
	if err := m.run("build", m.zip, "-no-world", "-install", "auto"); err != nil {
		t.Fatalf("%v\n%s", err, m.err)
	}
	if !m.exists(mods, "ruby_tools-1.2-neoforge-1.21.1.jar") {
		t.Errorf("the built mod was not added:\n%s", m.err)
	}
}

func TestWindowAddsDroppedModToMinecraft(t *testing.T) {
	m := newMachine(t)
	mods := mcreatortest.PrismInstance(t, m.home, "Ruby Test", "26.1.2", "net.neoforged", "26.1.2.12")
	mcreatortest.PrismInstance(t, m.home, "Fabric Fun", "1.21.1", "net.fabricmc.fabric-loader", "0.16.5")
	w := openTestWindow(t)

	f, _ := os.Open(rubyJar(t, m.home, "ruby_tools-1.2-neoforge-26.1.2.jar"))
	defer f.Close()
	code, data := w.do("POST", "/api/mod/open", f, map[string]string{"X-File-Name": "ruby_tools-1.2-neoforge-26.1.2.jar"})
	var mod modInfo
	json.Unmarshal(data, &mod)
	if code != 200 || mod.Name != "Ruby Tools" || mod.Needs != "NeoForge, Minecraft 26.1.2" {
		t.Fatalf("open: %d %s", code, data)
	}

	var list struct{ Instances []instanceView }
	w.json("GET", "/api/instances?mod="+mod.ID, nil, &list)
	if len(list.Instances) != 2 || list.Instances[0].Name != "Ruby Test" || !list.Instances[0].Fit.OK || list.Instances[1].Fit.OK {
		t.Fatalf("instances = %+v", list.Instances)
	}
	if !strings.Contains(list.Instances[1].Fit.Reason, "This is Fabric") {
		t.Errorf("no reason for the misfit: %+v", list.Instances[1].Fit)
	}

	var res struct {
		File, Instance string
	}
	if code := w.json("POST", "/api/mod/install", map[string]string{"mod": mod.ID, "instance": list.Instances[0].ID}, &res); code != 200 {
		t.Fatalf("install: %d", code)
	}
	if res.Instance != "Ruby Test" || !m.exists(mods, res.File) {
		t.Errorf("install = %+v", res)
	}

	var toggled struct{ File string }
	w.json("POST", "/api/mod/toggle", map[string]any{"instance": list.Instances[0].ID, "file": res.File, "on": false}, &toggled)
	if toggled.File != res.File+".disabled" || !m.exists(mods, toggled.File) {
		t.Errorf("switching off: %+v", toggled)
	}

	// A mod dropped into the folder some other way is listed.
	rubyJar(t, mods, "copied-by-hand.jar")
	w.json("GET", "/api/instances", nil, &list)
	for _, in := range list.Instances {
		if in.Name == "Ruby Test" && len(in.Mods) != 2 {
			t.Errorf("mods listed: %+v", in.Mods)
		}
	}
}

func TestWindowRejectsJarsThatAreNotMods(t *testing.T) {
	m := newMachine(t)
	w := openTestWindow(t)
	lib := filepath.Join(m.home, "lib.jar")
	mcreatortest.WriteZip(t, lib, map[string]string{"A.class": "x"})
	f, _ := os.Open(lib)
	defer f.Close()
	code, data := w.do("POST", "/api/mod/open", f, map[string]string{"X-File-Name": "lib.jar"})
	if code != http.StatusUnprocessableEntity || !bytes.Contains(data, []byte("no Forge, NeoForge, Fabric or Quilt mod information")) {
		t.Errorf("lib.jar: %d %s", code, data)
	}
}

func TestWindowSendsBuiltModToMinecraft(t *testing.T) {
	m := newMachine(t)
	t.Setenv("PREBUILT_JAR", rubyJar(t, m.home, "prebuilt.jar"))
	w := openTestWindow(t)
	_, p, _ := w.upload(m.zip)
	w.json("POST", "/api/build", map[string]string{"id": p.ID}, nil)
	for deadline := time.Now().Add(20 * time.Second); ; {
		var status struct{ Done bool }
		w.json("GET", "/api/status?id="+p.ID, nil, &status)
		if status.Done {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("build did not finish")
		}
		time.Sleep(20 * time.Millisecond)
	}
	var mod modInfo
	if code := w.json("POST", "/api/mod/from-build", map[string]string{"id": p.ID}, &mod); code != 200 || mod.ModID != "ruby_tools" {
		t.Errorf("from build: %d %+v", code, mod)
	}
}

func TestWindowWithJarFromFileManager(t *testing.T) {
	m := newMachine(t)
	w := openTestWindow(t)
	if _, err := w.s.openModPath(rubyJar(t, m.home, "ruby.jar")); err != nil {
		t.Fatal(err)
	}
	var setup setupInfo
	w.json("GET", "/api/setup", nil, &setup)
	if setup.Mod == nil || setup.Mod.ModID != "ruby_tools" {
		t.Errorf("the jar passed on the command line is not shown: %+v", setup.Mod)
	}
}
