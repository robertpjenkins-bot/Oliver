// Package minecraft finds the Minecraft installations on this computer — the
// instances of Prism Launcher, the official launcher and the others — reads
// mod jars, and puts mods into an instance's mods folder or takes them out.
package minecraft

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Loaders this package knows, by the names launchers and mods use.
const (
	NeoForge = "neoforge"
	Forge    = "forge"
	Fabric   = "fabric"
	Quilt    = "quilt"
)

// LoaderName is a loader's name for people.
func LoaderName(loader string) string {
	switch loader {
	case NeoForge:
		return "NeoForge"
	case Forge:
		return "Forge"
	case Fabric:
		return "Fabric"
	case Quilt:
		return "Quilt"
	case "":
		return "no mod loader"
	}
	return loader
}

// Mod is what a mod jar says about itself.
type Mod struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	// Loaders are the mod loaders the jar has metadata for. Most jars have
	// one; a multi-loader jar has several.
	Loaders []string `json:"loaders"`
	// Minecraft holds, per loader, the Minecraft versions the mod accepts,
	// written in that loader's own range syntax. Missing means any.
	Minecraft map[string]string `json:"minecraft"`
	// Requires lists other mods it cannot run without.
	Requires []Dependency `json:"requires,omitempty"`
}

// Dependency is a mod another mod needs.
type Dependency struct {
	ID       string `json:"id"`
	Versions string `json:"versions,omitempty"`
}

// ErrNotMod means the file is not a mod jar any supported loader would load.
var ErrNotMod = errors.New("not a mod")

// builtIn are dependencies the loader itself satisfies.
var builtIn = map[string]bool{
	"minecraft": true, "java": true, "javafml": true, "lowcodefml": true,
	"neoforge": true, "forge": true, "fml": true,
	"fabricloader": true, "fabric-loader": true,
	"quilt_loader": true,
}

// ReadMod reads a mod jar's metadata.
func ReadMod(path string) (*Mod, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		if errors.Is(err, zip.ErrFormat) {
			return nil, fmt.Errorf("%w: the file is not a jar", ErrNotMod)
		}
		return nil, err
	}
	defer zr.Close()

	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	read := func(name string) (string, bool) {
		f := files[name]
		if f == nil {
			return "", false
		}
		rc, err := f.Open()
		if err != nil {
			return "", false
		}
		defer rc.Close()
		data, err := io.ReadAll(io.LimitReader(rc, 4<<20))
		return string(data), err == nil
	}

	m := &Mod{Minecraft: map[string]string{}}
	if src, ok := read("META-INF/neoforge.mods.toml"); ok {
		m.readModsTOML(src, NeoForge)
	}
	if src, ok := read("META-INF/mods.toml"); ok {
		m.readModsTOML(src, Forge)
	}
	if src, ok := read("fabric.mod.json"); ok {
		m.readFabric(src)
	}
	if src, ok := read("quilt.mod.json"); ok {
		m.readQuilt(src)
	}
	if len(m.Loaders) == 0 || m.ID == "" {
		return nil, fmt.Errorf("%w: the jar has no Forge, NeoForge, Fabric or Quilt mod information in it", ErrNotMod)
	}

	// Forge fills this placeholder in from the manifest when it loads.
	if strings.Contains(m.Version, "${file.jarVersion}") {
		m.Version = manifestVersion(read)
	}
	if m.Name == "" {
		m.Name = m.ID
	}
	sort.Slice(m.Requires, func(i, j int) bool { return m.Requires[i].ID < m.Requires[j].ID })
	return m, nil
}

func (m *Mod) addLoader(loader string) {
	for _, l := range m.Loaders {
		if l == loader {
			return
		}
	}
	m.Loaders = append(m.Loaders, loader)
}

func (m *Mod) addRequirement(id, versions string) {
	if builtIn[id] {
		return
	}
	for _, d := range m.Requires {
		if d.ID == id {
			return
		}
	}
	m.Requires = append(m.Requires, Dependency{ID: id, Versions: versions})
}

// readModsTOML reads META-INF/(neoforge.)mods.toml. NeoForge's first
// releases still used mods.toml, so the loader is settled by which loader
// the mod declares a dependency on.
func (m *Mod) readModsTOML(src, loader string) {
	tables := parseTOML(src)
	id := ""
	for _, t := range tables {
		if t.name == "mods" {
			id = t.values["modId"]
			if m.ID == "" {
				m.ID, m.Name, m.Version = id, t.values["displayName"], t.values["version"]
			}
			break
		}
	}
	if id == "" {
		return
	}

	minecraft := ""
	for _, t := range tables {
		if t.name != "dependencies."+id {
			continue
		}
		dep := t.values["modId"]
		required := t.values["type"] == "" || strings.EqualFold(t.values["type"], "required")
		if mandatory, ok := t.values["mandatory"]; ok { // before NeoForge's "type"
			required = mandatory == "true"
		}
		switch dep {
		case "minecraft":
			minecraft = t.values["versionRange"]
		case "neoforge":
			loader = NeoForge
		case "forge":
			loader = Forge
		default:
			if required {
				m.addRequirement(dep, t.values["versionRange"])
			}
		}
	}
	m.addLoader(loader)
	if minecraft != "" {
		m.Minecraft[loader] = minecraft
	}
}

func (m *Mod) readFabric(src string) {
	var meta struct {
		ID      string                     `json:"id"`
		Name    string                     `json:"name"`
		Version string                     `json:"version"`
		Depends map[string]json.RawMessage `json:"depends"`
	}
	if json.Unmarshal([]byte(src), &meta) != nil || meta.ID == "" {
		return
	}
	if m.ID == "" {
		m.ID, m.Name, m.Version = meta.ID, meta.Name, meta.Version
	}
	m.addLoader(Fabric)
	for id, raw := range meta.Depends {
		versions := fabricVersions(raw)
		if id == "minecraft" {
			m.Minecraft[Fabric] = versions
			continue
		}
		m.addRequirement(id, versions)
	}
}

// fabricVersions flattens a Fabric version predicate, which may be a string
// or a list of alternatives, into one string with " || " between them.
func fabricVersions(raw json.RawMessage) string {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		return one
	}
	var many []string
	if json.Unmarshal(raw, &many) == nil {
		return strings.Join(many, " || ")
	}
	return "*"
}

func (m *Mod) readQuilt(src string) {
	var meta struct {
		Loader struct {
			ID       string `json:"id"`
			Version  string `json:"version"`
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Depends []json.RawMessage `json:"depends"`
		} `json:"quilt_loader"`
	}
	if json.Unmarshal([]byte(src), &meta) != nil || meta.Loader.ID == "" {
		return
	}
	l := meta.Loader
	if m.ID == "" {
		m.ID, m.Name, m.Version = l.ID, l.Metadata.Name, l.Version
	}
	m.addLoader(Quilt)
	for _, raw := range l.Depends {
		var dep struct {
			ID       string          `json:"id"`
			Versions json.RawMessage `json:"versions"`
			Optional bool            `json:"optional"`
		}
		if json.Unmarshal(raw, &dep.ID) != nil && json.Unmarshal(raw, &dep) != nil {
			continue
		}
		versions := "*"
		if len(dep.Versions) > 0 {
			versions = fabricVersions(dep.Versions)
		}
		switch {
		case dep.ID == "minecraft":
			m.Minecraft[Quilt] = versions
		case !dep.Optional:
			m.addRequirement(dep.ID, versions)
		}
	}
}

func manifestVersion(read func(string) (string, bool)) string {
	src, ok := read("META-INF/MANIFEST.MF")
	if !ok {
		return ""
	}
	sc := bufio.NewScanner(strings.NewReader(src))
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "Implementation-Version:"); ok {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// Needs describes what the mod runs on, for people: "NeoForge, Minecraft 26.1.2".
func (m *Mod) Needs() string {
	var parts []string
	for _, l := range m.Loaders {
		s := LoaderName(l)
		if r := m.Minecraft[l]; r != "" {
			s += ", Minecraft " + DescribeRange(r)
		}
		parts = append(parts, s)
	}
	return strings.Join(parts, " or ")
}

// knownNames are the people-facing names of mods commonly required by others.
var knownNames = map[string]string{
	"fabric-api":   "Fabric API",
	"fabric":       "Fabric API",
	"qsl":          "Quilted Fabric API",
	"geckolib":     "GeckoLib",
	"architectury": "Architectury API",
	"cloth-config": "Cloth Config",
	"curios":       "Curios API",
}

// ModName is a mod ID's name for people, where it is a well-known one.
func ModName(id string) string {
	if n, ok := knownNames[id]; ok {
		return n
	}
	return id
}
