package minecraft

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Instance is one Minecraft installation mods can go into: a Prism Launcher
// instance, an official launcher profile, a mods folder the user named.
type Instance struct {
	ID       string `json:"id"`
	Launcher string `json:"launcher"`
	Name     string `json:"name"`
	// Minecraft and Loader are what the instance runs. Loader is empty for
	// plain Minecraft. Known is false when the launcher keeps them somewhere
	// this package cannot read, and then both are empty.
	Minecraft     string `json:"minecraft"`
	Loader        string `json:"loader"`
	LoaderVersion string `json:"loaderVersion,omitempty"`
	Known         bool   `json:"known"`
	ModsDir       string `json:"modsDir"`
}

// Label describes what the instance runs: "NeoForge 21.1.77, Minecraft 1.21.1".
func (in Instance) Label() string {
	switch {
	case !in.Known:
		return "unknown Minecraft version"
	case in.Loader == "":
		return "Minecraft " + in.Minecraft + " without mods"
	}
	s := LoaderName(in.Loader)
	if in.LoaderVersion != "" {
		s += " " + in.LoaderVersion
	}
	if in.Minecraft != "" {
		s += ", Minecraft " + in.Minecraft
	}
	return s
}

// FindInstances lists the Minecraft instances on this computer, followed by
// the extra mods folders given.
func FindInstances(extraModsDirs ...string) []Instance {
	home, _ := os.UserHomeDir()
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" && home != "" {
		data = filepath.Join(home, ".local", "share")
	}
	flatpak := func(app string, parts ...string) string {
		return filepath.Join(append([]string{home, ".var", "app", app}, parts...)...)
	}

	var all []Instance
	if home != "" {
		// Prism Launcher and the launchers it grew from share one layout.
		for _, l := range []struct{ name, dir, cfg string }{
			{"Prism Launcher", filepath.Join(data, "PrismLauncher"), "prismlauncher.cfg"},
			{"Prism Launcher", flatpak("org.prismlauncher.PrismLauncher", "data", "PrismLauncher"), "prismlauncher.cfg"},
			{"PolyMC", filepath.Join(data, "PolyMC"), "polymc.cfg"},
			{"PolyMC", flatpak("org.polymc.PolyMC", "data", "PolyMC"), "polymc.cfg"},
			{"MultiMC", filepath.Join(data, "multimc"), "multimc.cfg"},
			{"MultiMC", filepath.Join(home, "MultiMC"), "multimc.cfg"},
		} {
			all = append(all, multiMCInstances(l.name, l.dir, l.cfg)...)
		}
		for _, dir := range []string{filepath.Join(home, ".minecraft"), flatpak("com.mojang.Minecraft", ".minecraft")} {
			all = append(all, launcherProfiles(dir)...)
		}
		for _, dir := range []string{flatpak("com.atlauncher.ATLauncher", "data"), filepath.Join(data, "atlauncher"), filepath.Join(home, "ATLauncher")} {
			all = append(all, atLauncherInstances(dir)...)
		}
		for _, dir := range []string{
			filepath.Join(data, "ModrinthApp"), filepath.Join(data, "com.modrinth.theseus"),
			flatpak("com.modrinth.ModrinthApp", "data", "ModrinthApp"),
		} {
			all = append(all, modrinthProfiles(dir)...)
		}
		all = append(all, curseForgeInstances(filepath.Join(home, "curseforge", "minecraft", "Instances"))...)
	}
	for _, dir := range extraModsDirs {
		all = append(all, Instance{
			Launcher: "Folder",
			Name:     filepath.Base(filepath.Dir(dir)),
			ModsDir:  dir,
		})
	}

	seen := map[string]bool{}
	var out []Instance
	for _, in := range all {
		in.ID = instanceID(in)
		if seen[in.ID] {
			continue
		}
		seen[in.ID] = true
		out = append(out, in)
	}
	return out
}

func instanceID(in Instance) string {
	sum := sha1.Sum([]byte(in.Launcher + "\x00" + in.Name + "\x00" + in.ModsDir))
	return hex.EncodeToString(sum[:6])
}

// multiMCInstances reads the instances of Prism Launcher, PolyMC or MultiMC.
//
// Each instance is a folder with instance.cfg (its name) and mmc-pack.json
// (the Minecraft and loader versions, as components); the game itself runs
// in its .minecraft or minecraft subfolder.
func multiMCInstances(launcher, dataDir, cfg string) []Instance {
	instances := filepath.Join(dataDir, "instances")
	if d := iniValue(filepath.Join(dataDir, cfg), "InstanceDir"); d != "" {
		if filepath.IsAbs(d) {
			instances = d
		} else {
			instances = filepath.Join(dataDir, d)
		}
	}
	entries, err := os.ReadDir(instances)
	if err != nil {
		return nil
	}
	var out []Instance
	for _, e := range entries {
		dir := filepath.Join(instances, e.Name())
		if _, err := os.Stat(filepath.Join(dir, "instance.cfg")); err != nil {
			continue
		}
		in := Instance{Launcher: launcher, Name: iniValue(filepath.Join(dir, "instance.cfg"), "name")}
		if in.Name == "" {
			in.Name = e.Name()
		}

		var pack struct {
			Components []struct {
				UID     string `json:"uid"`
				Version string `json:"version"`
			} `json:"components"`
		}
		if readJSON(filepath.Join(dir, "mmc-pack.json"), &pack) {
			in.Known = true
			for _, c := range pack.Components {
				switch c.UID {
				case "net.minecraft":
					in.Minecraft = c.Version
				case "net.neoforged":
					in.Loader, in.LoaderVersion = NeoForge, c.Version
				case "net.minecraftforge":
					in.Loader, in.LoaderVersion = Forge, c.Version
				case "net.fabricmc.fabric-loader":
					in.Loader, in.LoaderVersion = Fabric, c.Version
				case "org.quiltmc.quilt-loader":
					in.Loader, in.LoaderVersion = Quilt, c.Version
				}
			}
		}

		// The launcher uses .minecraft when it exists and creates minecraft
		// otherwise.
		game := filepath.Join(dir, ".minecraft")
		if _, err := os.Stat(game); err != nil {
			game = filepath.Join(dir, "minecraft")
		}
		in.ModsDir = filepath.Join(game, "mods")
		out = append(out, in)
	}
	return out
}

// launcherProfiles reads the official launcher's profiles. A profile names a
// version; for a modded one, that version's JSON says which Minecraft it
// builds on. Profiles without their own game directory share .minecraft.
func launcherProfiles(mcDir string) []Instance {
	var file struct {
		Profiles map[string]struct {
			Name          string `json:"name"`
			Type          string `json:"type"`
			LastVersionID string `json:"lastVersionId"`
			GameDir       string `json:"gameDir"`
		} `json:"profiles"`
	}
	if !readJSON(filepath.Join(mcDir, "launcher_profiles.json"), &file) {
		return nil
	}
	var out []Instance
	for _, p := range file.Profiles {
		in := Instance{Launcher: "Minecraft Launcher", Name: p.Name, Known: true}
		switch p.Type {
		case "latest-release":
			in.Name, in.Minecraft = "Latest release", "latest"
		case "latest-snapshot":
			continue // snapshots cannot run mods
		default:
			in.Loader, in.Minecraft = versionInfo(mcDir, p.LastVersionID)
			if in.Name == "" {
				in.Name = p.LastVersionID
			}
		}
		game := mcDir
		if p.GameDir != "" {
			game = p.GameDir
		}
		in.ModsDir = filepath.Join(game, "mods")
		out = append(out, in)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// versionInfo works out the loader and Minecraft version of an installed
// launcher version such as "neoforge-21.1.77", "1.20.1-forge-47.3.0" or
// "fabric-loader-0.16.5-1.21.1".
func versionInfo(mcDir, id string) (loader, minecraft string) {
	lower := strings.ToLower(id)
	switch {
	case strings.Contains(lower, "neoforge"):
		loader = NeoForge
	case strings.Contains(lower, "forge"):
		loader = Forge
	case strings.Contains(lower, "fabric-loader"):
		loader = Fabric
	case strings.Contains(lower, "quilt-loader"):
		loader = Quilt
	}
	var v struct {
		InheritsFrom string `json:"inheritsFrom"`
	}
	if readJSON(filepath.Join(mcDir, "versions", id, id+".json"), &v) && v.InheritsFrom != "" {
		return loader, v.InheritsFrom
	}
	if loader == "" {
		return "", id // a plain Minecraft version is its own name
	}
	return loader, "" // the version is not installed, so it cannot be told
}

func atLauncherInstances(dataDir string) []Instance {
	entries, err := os.ReadDir(filepath.Join(dataDir, "instances"))
	if err != nil {
		return nil
	}
	var out []Instance
	for _, e := range entries {
		dir := filepath.Join(dataDir, "instances", e.Name())
		var meta struct {
			ID       string `json:"id"`
			Launcher struct {
				Name          string `json:"name"`
				LoaderVersion struct {
					Type    string `json:"type"`
					Version string `json:"version"`
				} `json:"loaderVersion"`
			} `json:"launcher"`
		}
		if !readJSON(filepath.Join(dir, "instance.json"), &meta) {
			continue
		}
		in := Instance{
			Launcher:      "ATLauncher",
			Name:          meta.Launcher.Name,
			Minecraft:     meta.ID,
			Loader:        strings.ToLower(meta.Launcher.LoaderVersion.Type),
			LoaderVersion: meta.Launcher.LoaderVersion.Version,
			Known:         true,
			ModsDir:       filepath.Join(dir, "mods"),
		}
		if in.Name == "" {
			in.Name = e.Name()
		}
		out = append(out, in)
	}
	return out
}

// modrinthProfiles lists Modrinth App profiles. Current versions of the app
// keep a profile's versions in a database, so those come out as unknown.
func modrinthProfiles(dataDir string) []Instance {
	entries, err := os.ReadDir(filepath.Join(dataDir, "profiles"))
	if err != nil {
		return nil
	}
	var out []Instance
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(dataDir, "profiles", e.Name())
		in := Instance{Launcher: "Modrinth App", Name: e.Name(), ModsDir: filepath.Join(dir, "mods")}
		var meta struct {
			Metadata struct {
				Name          string `json:"name"`
				GameVersion   string `json:"game_version"`
				Loader        string `json:"loader"`
				LoaderVersion struct {
					ID string `json:"id"`
				} `json:"loader_version"`
			} `json:"metadata"`
		}
		if readJSON(filepath.Join(dir, "profile.json"), &meta) && meta.Metadata.GameVersion != "" {
			m := meta.Metadata
			in.Known, in.Minecraft, in.LoaderVersion = true, m.GameVersion, m.LoaderVersion.ID
			if m.Loader != "vanilla" {
				in.Loader = m.Loader
			}
			if m.Name != "" {
				in.Name = m.Name
			}
		}
		out = append(out, in)
	}
	return out
}

func curseForgeInstances(dir string) []Instance {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Instance
	for _, e := range entries {
		inst := filepath.Join(dir, e.Name())
		var meta struct {
			Name          string `json:"name"`
			GameVersion   string `json:"gameVersion"`
			BaseModLoader struct {
				Name string `json:"name"` // "neoforge-21.1.77"
			} `json:"baseModLoader"`
		}
		if !readJSON(filepath.Join(inst, "minecraftinstance.json"), &meta) {
			continue
		}
		in := Instance{Launcher: "CurseForge", Name: meta.Name, Minecraft: meta.GameVersion, Known: true, ModsDir: filepath.Join(inst, "mods")}
		loader, version, _ := strings.Cut(meta.BaseModLoader.Name, "-")
		in.Loader, in.LoaderVersion = strings.ToLower(loader), version
		if in.Name == "" {
			in.Name = e.Name()
		}
		out = append(out, in)
	}
	return out
}

// iniValue reads one key from an INI file, whatever its section.
func iniValue(path, key string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if ok && strings.TrimSpace(k) == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func readJSON(path string, v any) bool {
	data, err := os.ReadFile(path)
	return err == nil && json.Unmarshal(data, v) == nil
}
