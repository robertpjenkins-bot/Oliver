// Package mcreator turns an MCreator workspace, as exported to a shareable
// ZIP, back into something Gradle can build, and builds it.
//
// A shareable ZIP is deliberately incomplete. MCreator leaves out build.gradle,
// the Gradle wrapper and every build output, because it rewrites them from its
// generator templates whenever a workspace is opened. What remains is the
// workspace description, the generated source code and resources, and — when
// exported "with run dir" — the worlds played while testing. This package
// restores the missing files from an MCreator installation, so the result is
// the same jar MCreator's own "Export the mod for distribution" produces.
package mcreator

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"unicode"
)

// Project is an MCreator workspace read from a shareable ZIP.
type Project struct {
	ZipPath string

	ModID     string
	ModName   string
	Version   string
	Generator string // e.g. "neoforge-1.21.1"
	Package   string // Java package of the generated code
	APIs      []string
	Worlds    []World

	root    string // where the workspace sits inside the ZIP: "" or "folder/"
	sources []*zip.File
	zr      *zip.ReadCloser
}

// World is a saved game found in the workspace's run directory.
type World struct {
	Name  string `json:"name"` // the folder name under saves/
	Files int    `json:"files"`
	Size  int64  `json:"size"`

	prefix string // ZIP path of the world folder, with trailing slash
}

// ErrNotWorkspace means the file is a ZIP but not an MCreator export.
var ErrNotWorkspace = errors.New("not an MCreator workspace")

// workspaceFile is the part of the .mcreator file this package uses.
type workspaceFile struct {
	Settings struct {
		ModID                string   `json:"modid"`
		ModName              string   `json:"modName"`
		Version              string   `json:"version"`
		CurrentGenerator     string   `json:"currentGenerator"`
		ModElementsPackage   string   `json:"modElementsPackage"`
		MCreatorDependencies []string `json:"mcreatorDependencies"`
	} `json:"workspaceSettings"`
}

// Open reads a shareable ZIP. Close the project when done with it.
func Open(zipPath string) (*Project, error) {
	if strings.EqualFold(path.Ext(zipPath), ".jar") {
		return nil, fmt.Errorf("%w: this is already a .jar — drop the .zip MCreator exported instead", ErrNotWorkspace)
	}
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		if errors.Is(err, zip.ErrFormat) {
			return nil, fmt.Errorf("%w: the file is not a ZIP archive", ErrNotWorkspace)
		}
		return nil, err
	}
	p, err := read(zr)
	if err != nil {
		zr.Close()
		return nil, err
	}
	p.ZipPath = zipPath
	p.zr = zr
	return p, nil
}

// Close releases the ZIP file.
func (p *Project) Close() error {
	if p.zr == nil {
		return nil
	}
	return p.zr.Close()
}

func read(zr *zip.ReadCloser) (*Project, error) {
	// The workspace file sits at the top of an MCreator export. People also
	// re-zip the workspace folder themselves, which puts it one level down,
	// so take the shallowest one.
	var ws *zip.File
	depth := -1
	for _, f := range zr.File {
		if f.FileInfo().IsDir() || path.Ext(f.Name) != ".mcreator" {
			continue
		}
		d := strings.Count(f.Name, "/")
		switch {
		case depth < 0 || d < depth:
			ws, depth = f, d
		case d == depth:
			return nil, fmt.Errorf("%w: the ZIP holds more than one workspace (%s and %s)", ErrNotWorkspace, ws.Name, f.Name)
		}
	}
	if ws == nil {
		return nil, fmt.Errorf("%w: there is no .mcreator file inside. In MCreator, use File → Export workspace to a shareable ZIP", ErrNotWorkspace)
	}

	var desc workspaceFile
	rc, err := ws.Open()
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(io.LimitReader(rc, 64<<20)).Decode(&desc)
	rc.Close()
	if err != nil {
		return nil, fmt.Errorf("%w: %s could not be read: %v", ErrNotWorkspace, ws.Name, err)
	}

	s := desc.Settings
	if s.ModID == "" {
		return nil, fmt.Errorf("%w: %s does not name a mod ID", ErrNotWorkspace, ws.Name)
	}
	p := &Project{
		ModID:     s.ModID,
		ModName:   s.ModName,
		Version:   s.Version,
		Generator: normalizeGenerator(s.CurrentGenerator),
		Package:   s.ModElementsPackage,
		APIs:      s.MCreatorDependencies,
		root:      path.Dir(ws.Name) + "/",
	}
	if p.root == "./" {
		p.root = ""
	}
	if p.ModName == "" {
		p.ModName = p.ModID
	}
	if p.Version == "" {
		p.Version = "1.0"
	}
	if p.Package == "" {
		p.Package = "net.mcreator." + p.ModID
	}
	sort.Strings(p.APIs)

	worlds := map[string]*World{}
	for _, f := range zr.File {
		rel, ok := strings.CutPrefix(f.Name, p.root)
		if !ok || f.FileInfo().IsDir() {
			continue
		}
		if strings.HasPrefix(rel, "src/") || strings.HasPrefix(rel, "lib/") {
			p.sources = append(p.sources, f)
			continue
		}
		if prefix, name, ok := worldOf(rel); ok {
			w := worlds[prefix]
			if w == nil {
				w = &World{Name: name, prefix: p.root + prefix}
				worlds[prefix] = w
			}
			w.Files++
			w.Size += int64(f.UncompressedSize64)
		}
	}
	if !p.hasSources() {
		return nil, fmt.Errorf("%w: the ZIP has no src folder, so there is no mod code to build", ErrNotWorkspace)
	}

	// A folder under saves/ without a level.dat is not a world Minecraft
	// would list, so it is not offered.
	for prefix, w := range worlds {
		if !hasFile(zr, w.prefix+"level.dat") {
			delete(worlds, prefix)
		}
	}
	for _, w := range worlds {
		p.Worlds = append(p.Worlds, *w)
	}
	sort.Slice(p.Worlds, func(i, j int) bool {
		return strings.ToLower(p.Worlds[i].Name) < strings.ToLower(p.Worlds[j].Name)
	})
	return p, nil
}

func (p *Project) hasSources() bool {
	for _, f := range p.sources {
		if strings.HasPrefix(strings.TrimPrefix(f.Name, p.root), "src/") {
			return true
		}
	}
	return false
}

// worldOf reports whether a workspace path lies inside a saved world, and if
// so returns the world folder (with trailing slash) and its name.
//
// Worlds live under the run directory Gradle starts Minecraft in: run/ for
// most generators, runs/<side>/ for some.
func worldOf(rel string) (prefix, name string, ok bool) {
	if !strings.HasPrefix(rel, "run/") && !strings.HasPrefix(rel, "runs/") {
		return "", "", false
	}
	parts := strings.Split(rel, "/")
	for i := 1; i+2 < len(parts); i++ {
		if parts[i] == "saves" && parts[i+1] != "" {
			return strings.Join(parts[:i+2], "/") + "/", parts[i+1], true
		}
	}
	return "", "", false
}

func hasFile(zr *zip.ReadCloser, name string) bool {
	for _, f := range zr.File {
		if f.Name == name {
			return true
		}
	}
	return false
}

// World returns the named world.
func (p *Project) World(name string) (World, bool) {
	for _, w := range p.Worlds {
		if w.Name == name {
			return w, true
		}
	}
	return World{}, false
}

// normalizeGenerator mirrors MCreator: workspaces older than 2020.3 named only
// the Minecraft version, and were all Forge.
func normalizeGenerator(name string) string {
	if name != "" && !strings.Contains(name, "-") {
		return "forge-" + name
	}
	return name
}

// Flavor is the generator family: forge, neoforge, fabric, datapack, ...
func (p *Project) Flavor() string {
	flavor, _, _ := strings.Cut(p.Generator, "-")
	return flavor
}

// MakesJar reports whether this kind of workspace builds into a jar. Data
// packs, resource packs and Bedrock add-ons export as archives of their own.
func (p *Project) MakesJar() bool {
	switch p.Flavor() {
	case "forge", "neoforge", "fabric", "quilt", "spigot", "paper":
		return true
	}
	return false
}

// CanBundleWorld reports whether a world can be shipped inside this jar.
// Server plugins have no singleplayer world list to put it in.
func (p *Project) CanBundleWorld() bool {
	switch p.Flavor() {
	case "forge", "neoforge", "fabric", "quilt":
		return true
	}
	return false
}

// GeneratorLabel describes the generator for people: "NeoForge 1.21.1".
func (p *Project) GeneratorLabel() string {
	flavor, version, _ := strings.Cut(p.Generator, "-")
	names := map[string]string{
		"forge":        "Minecraft Forge",
		"neoforge":     "NeoForge",
		"fabric":       "Fabric",
		"quilt":        "Quilt",
		"spigot":       "Spigot",
		"paper":        "Paper",
		"datapack":     "Data pack",
		"resourcepack": "Resource pack",
		"addon":        "Bedrock add-on",
	}
	name, ok := names[flavor]
	if !ok {
		name = flavor
	}
	if version == "" {
		return name
	}
	return name + " " + version
}

// JavaModName is the class name MCreator gives the mod's main class.
func (p *Project) JavaModName() string {
	var b strings.Builder
	for _, word := range strings.Split(p.ModID, "_") {
		if word == "" {
			continue
		}
		r := []rune(strings.ToLower(word))
		r[0] = unicode.ToUpper(r[0])
		b.WriteString(string(r))
	}
	return b.String() + "Mod"
}

// JarName is the file name MCreator suggests when exporting this mod.
func (p *Project) JarName() string {
	return safeFileName(p.ModID+"-"+p.Version+"-"+p.Generator) + ".jar"
}

// safeFileName replaces characters that are awkward or invalid in file names.
func safeFileName(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r < 0x20, strings.ContainsRune(`/\:*?"<>|`, r):
			return '_'
		}
		return r
	}, s)
}
